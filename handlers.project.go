package main

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/colabware-ltd/colabware-backend/api"
	"github.com/colabware-ltd/colabware-backend/eth"
	"github.com/colabware-ltd/colabware-backend/utilities"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TODO: Create wallet for project upon creation. Maintainers should then
// be able to access this wallet. Wallet should hold maintainer tokens.
type Project struct {
	ID             primitive.ObjectID   `json:"_id,omitempty" bson:"_id,omitempty"`
	Created        primitive.DateTime   `json:"created" bson:"created,omitempty"`
	Name           string               `json:"name" bson:"name,omitempty"`
	GitHub         GitHub               `json:"github" bson:"github,omitempty"`
	Description    string               `json:"description" bson:"description,omitempty"`
	Categories     []string             `json:"categories" bson:"categories,omitempty"`
	Maintainers    []primitive.ObjectID `json:"maintainers" bson:"maintainers,omitempty"`
	Token          Token                `json:"token" bson:"token,omitempty"`
	Address        string               `json:"address" bson:"address,omitempty"`
	Wallet         primitive.ObjectID   `json:"wallet" bson:"wallet,omitempty"`
	WalletAddress  string               `json:"wallet_address" bson:"wallet_address,omitempty"`
	Requests       []primitive.ObjectID `json:"requests" bson:"requests,omitempty"`
	RequestCount   uint64               `json:"request_count" bson:"request_count"`
	Roadmap        []primitive.ObjectID `json:"roadmap" bson:"roadmap,omitempty"`
	Status         string               `json:"status" bson:"status,omitempty"`
	ApprovalConfig ApprovalConfig       `json:"approval_config" bson:"approval_config,omitempty"`
	USDCBalance    int64                `json:"usdc_balance" bson:"usdc_balance,omitempty"`
	TokenHolders   []TokenHolding       `json:"token_holders" bson:"token_holders"`
}

type ApprovalConfig struct {
	TokensRequired     float32 `json:"tokens_required" bson:"tokens_required,omitempty"`
	MaintainerRequired bool    `json:"maintainer_required" bson:"maintainer_required,omitempty"`
}

type Token struct {
	Name             string  `json:"name"`
    Address          string  `json:"address" bson:"address,omitempty"`
	Symbol           string  `json:"symbol"`
	Price            float64 `json:"price"`
	TotalSupply      int64   `json:"total_supply"`
	MaintainerSupply int64   `json:"maintainer_supply"`
}

type GitHub struct {
	RepoOwner string                 `json:"repo_owner" bson:"repo_owner,omitempty"`
	RepoName  string                 `json:"repo_name" bson:"repo_name,omitempty"`
	Forks     []api.GitHubFork `json:"forks" bson:"forks,omitempty"`
}

func (t Token) getBigTotalSupply() *big.Int {
	i := big.NewInt(t.TotalSupply)
	return i.Mul(i, big.NewInt(utilities.ONE_TOKEN))
}

func (t Token) getBigMaintainerSupply() *big.Int {
	i := big.NewInt(t.MaintainerSupply)
	return i.Mul(i, big.NewInt(utilities.ONE_TOKEN))
}

func (con Connection) postProject(c *gin.Context) {
	var p Project
	if err := c.BindJSON(&p); err != nil {
		log.Printf("%v", err)
		return
	}
	// Convert Token supply to the right amount
	log.Printf("TotalTokenSupply: %v\n", p.Token.getBigTotalSupply())

	session := sessions.Default(c)
	// TODO: Update session to store db ID
	userId := session.Get("user-id")
	var user User
	e := con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if e != nil {
		log.Printf("%v", e)
		return
	}
	p.Maintainers = append(p.Maintainers, user.ID)
	p.Status = "pending"
	p.RequestCount = 0
	p.Created = primitive.NewDateTimeFromTime(time.Now())

	// TODO: Introduce as configuration option on frontend
	p.ApprovalConfig.TokensRequired = 0.5
	p.ApprovalConfig.MaintainerRequired = true
	
	p.TokenHolders = []TokenHolding{}

	// Check that user is a maintainer of the project repository
	isMaintainer, err := api.RepoMaintainer(client, p.GitHub.RepoOwner, p.GitHub.RepoName)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	log.Print(isMaintainer)

	if isMaintainer == false {
		log.Printf("User isn't authorized to complete this request.")
		c.IndentedJSON(http.StatusForbidden, gin.H{
			"message": "You must be an administrator of the selected repository to proceed with this request.",
		})
		return
	}

	// TODO: Add validation to check whether project with name exists
	result, err := con.Projects.InsertOne(context.TODO(), p)
	selector := bson.M{"_id": user.ID}
	update := bson.M{
		"$push": bson.M{"projects_maintained": result.InsertedID},
	}
	_, err = con.Users.UpdateOne(context.TODO(), selector, update)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	// Create wallet for project and get address; initial project tokens will be minted for this address.
	// TODO: Provide the alternative option to import a wallet
	// TODO: Return private key so we don't store this(?)
	walletId, wallet := con.createWallet(result.InsertedID.(primitive.ObjectID))

	// Deploy contract and store address; wait for execution to complete
	projectAddress := eth.DeployProject(p.Token.Name, p.Token.Symbol, *p.Token.getBigTotalSupply(), *p.Token.getBigMaintainerSupply(), wallet.Address, colabwareConf.EthNode, colabwareConf.EthKey, colabwareConf.EthChainId)
	log.Printf("Contract pending deploy: 0x%x\n", projectAddress)

	selector = bson.M{"_id": result.InsertedID.(primitive.ObjectID)}
	update = bson.M{
		"$set": bson.M{
			"wallet": walletId,
			"wallet_address":  wallet.Address,
			"address": projectAddress.Hex(),
		},
	}
	_, err = con.Projects.UpdateOne(context.TODO(), selector, update)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	c.IndentedJSON(http.StatusCreated, p)
}


func (con Connection) getProject(c *gin.Context) {
	name := c.Param("project")
	project, err := con.getProjectByName(name)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// Fetch balance of USDC
	balance, err := eth.FetchBalance(
		project.WalletAddress, 
		colabwareConf.MaticTestAddr,
		colabwareConf.EthNode, 
		colabwareConf.EthChainId,
	)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	project.USDCBalance = balance.Int64()

	// Get token holders
	project.TokenHolders, err = con.listTokenHolders(project.Token.Address)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// If user is authenticated, get forks from GitHub API
	if sessions.Default(c).Get("user-id") != nil {
		project.GitHub.Forks, err = api.RepoForks(
			client, 
			project.GitHub.RepoOwner, 
			project.GitHub.RepoName,
		)
		if err != nil {
			log.Printf("%v", err)
			c.IndentedJSON(http.StatusInternalServerError, project)
			return
		}
	}
	c.IndentedJSON(http.StatusFound, project)
}

func (con Connection) getProjectByName(name string) (*Project, error) {
	var project Project
	selector := bson.M{"name": name}
	err := con.Projects.FindOne(context.TODO(), selector).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	return &project, nil
}

func (con Connection) getProjectById(id primitive.ObjectID) (*Project, error) {
	var project Project
	selector := bson.M{"_id": id}
	err := con.Projects.FindOne(context.TODO(), selector).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	return &project, nil
}

func (con Connection) getProjectByTokenAddress(address string) (*Project, error) {
	var project Project
	selector := bson.M{"token.address": address}
	err := con.Projects.FindOne(context.TODO(), selector).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	return &project, nil
}

func (con Connection) getProjectByWalletID(id primitive.ObjectID) (*Project, error) {
	var project Project
	selector := bson.M{"wallet": id}
	err := con.Projects.FindOne(context.TODO(), selector).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	return &project, nil
}

func getProjectBranches(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")

	branches, err := api.RepoBranches(client, owner, repo)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	c.IndentedJSON(http.StatusFound, branches)
}

func (con Connection) getTokenBalanceOverview(c *gin.Context) {
	project := c.Param("project")

	maintainerBalance, maintainerReserved, investorBalance, err := eth.ProjectTokenBalances(
		project, 
		colabwareConf.EthNode,
	)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// Get Token balance for current user
	c.IndentedJSON(http.StatusFound, gin.H{
		"maintainer_balance":  utilities.BigIntToTokens(maintainerBalance),
		"maintainer_reserved": utilities.BigIntToTokens(maintainerReserved),
		"investor_balance":    utilities.BigIntToTokens(investorBalance),
	})
}

func (con Connection) getProjects(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")
	limitInt, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	pageInt, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	orderBy := c.DefaultQuery("orderBy", "created")
	filterBy := strings.Split(c.DefaultQuery("filterBy", ""), ",")
	desc, err := strconv.ParseBool(c.DefaultQuery("desc", "true"))
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	sortDirection := 1
	if desc {
		sortDirection = -1
	}

	options := options.Find()
	options.SetProjection(bson.M{"name": 1, "categories": 1, "description": 1, "_id": 0})
	options.SetLimit(limitInt)
	options.SetSkip(limitInt * (pageInt - 1))
	options.SetSort(bson.M{orderBy: sortDirection})
	
	projectSelector := bson.M{}
	if filterBy[0] != "" {
		projectSelector = bson.M{
			"categories": bson.M{
				"$in": filterBy,
			},
		}
	}

	total, err := con.Projects.CountDocuments(context.TODO(), projectSelector)
	filterCursor, err := con.Projects.Find(context.TODO(), projectSelector, options)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	var projectsFiltered []bson.M
	err = filterCursor.All(context.TODO(), &projectsFiltered)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	c.IndentedJSON(http.StatusFound, gin.H{"total": total, "results": projectsFiltered})
}

func (con Connection) getTokenHolding(c *gin.Context) {
	projectId, err := primitive.ObjectIDFromHex(c.Param("project"))
	if err != nil {
		log.Printf("%v", err)
		return
	}
	userId := sessions.Default(c).Get("user-id")
	var user User
	var tokenHolding TokenHolding

	// Find address of current user
	err = con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	maintainer, project, err := con.isMaintainer(user.ID, projectId)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	walletAddress := ""
	if (maintainer) {
		// Select project wallet if user is maintainer
		walletAddress = project.WalletAddress
	} else {
		// Select user's wallet if they are not a maintainer
		walletAddress = user.WalletAddress
	}

	selector := bson.M{
		"token_address": project.Token.Address, 
		"wallet_address": walletAddress,
	}
	err = con.TokenHoldings.FindOne(context.TODO(), selector).Decode(&tokenHolding)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	
	c.IndentedJSON(http.StatusFound, gin.H{
		"wallet_address": user.WalletAddress,
		"token_address": project.Token.Address,
		"balance": tokenHolding.Balance,
	})
}



func (con Connection) isMaintainer(userId primitive.ObjectID, projectId primitive.ObjectID) (bool, *Project, error) {
	project, err := con.getProjectById(projectId)
	if err != nil {
		log.Printf("%v", err)
		return false, nil, fmt.Errorf("%v", err)
	}

	isMaintainer := false
	for _, maintainer := range project.Maintainers {
		if maintainer == userId {
			isMaintainer = true
			break
		}
	}
	
	return isMaintainer, project, nil
}