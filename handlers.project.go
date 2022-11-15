package main

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/colabware-ltd/colabware-backend/utilities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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
	Name           string               `json:"name" bson:"name,omitempty"`
	GitHub         GitHub               `json:"github" bson:"github,omitempty"`
	Description    string               `json:"description" bson:"description,omitempty"`
	Categories     []string             `json:"categories" bson:"categories,omitempty"`
	Maintainers    []primitive.ObjectID `json:"maintainers" bson:"maintainers,omitempty"`
	Token          Token                `json:"token" bson:"token,omitempty"`
	Address        string               `json:"address" bson:"address,omitempty"`
	Wallet         primitive.ObjectID   `json:"wallet" bson:"wallet,omitempty"`
	Requests       []primitive.ObjectID `json:"requests" bson:"requests,omitempty"`
	Roadmap        []primitive.ObjectID `json:"roadmap" bson:"roadmap,omitempty"`
	Status         string               `json:"status" bson:"status,omitempty"`
}

type Token struct {
	Name             string  `json:"name" bson:"name,omitempty"`
	Address          string  `json:"address" bson:"address,omitempty"`
	Symbol           string  `json:"symbol" bson:"symbol,omitempty"`
	Price            float32 `json:"price" bson:"price,omitempty"`
	TotalSupply      float64   `json:"total_supply" bson:"total_supply,omitempty"`
	MaintainerSupply float64   `json:"maintainer_supply" bson:"maintainer_supply,omitempty"`
}

type GitHub struct {
	RepoOwner string       `json:"repo_owner" bson:"repo_owner,omitempty"`
	RepoName  string       `json:"repo_name" bson:"repo_name,omitempty"`
	Forks     []GitHubFork `json:"forks" bson:"forks,omitempty"`
}

type GitHubFork struct {
	FullName string `json:"full_name,omitempty" bson:"full_name,omitempty"`
}

// type GitHubBranch struct {
// 	Name string `json:"name,omitempty" bson:"name,omitempty"`
// }

func (con Connection) postProject(c *gin.Context) {
	var p Project
	if err := c.BindJSON(&p); err != nil {
		log.Printf("%v", err)
		return
	}
	session := sessions.Default(c)
	// TODO: Update session to store db ID
	userId := session.Get("user-id")
	var user struct {
		ID primitive.ObjectID `bson:"_id, omitempty"`
	}
	e := con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if e != nil {
		log.Printf("%v", e)
		return
	}
	p.Maintainers = append(p.Maintainers, user.ID)
	p.Status = "pending"

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
	walletId, wallet := con.createWallet(result.InsertedID.(primitive.ObjectID))

	// Deploy contract and store address; wait for execution to complete
	projectAddress := utilities.DeployProject(p.Token.Name, p.Token.Symbol, *floatToBigInt(p.Token.TotalSupply), *floatToBigInt(p.Token.MaintainerSupply), wallet.Address, config.EthNode, config.EthKey)
	log.Printf("Contract pending deploy: 0x%x\n", projectAddress)

	selector = bson.M{"_id": result.InsertedID.(primitive.ObjectID)}
	update = bson.M{
		"$set": bson.M{
			"wallet":  walletId,
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
	var project Project
	selector := bson.M{"name": name}
	err := con.Projects.FindOne(context.TODO(), selector).Decode(&project)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// If user is authenticated, get forks from GitHub API
	if sessions.Default(c).Get("user-id") != nil {
		var resTarget []GitHubFork
		res, err := client.Get("https://api.github.com/repos/" + project.GitHub.RepoOwner + "/" + project.GitHub.RepoName + "/forks")
		if err != nil {
			log.Printf("%v", err)
			return
		}
		defer res.Body.Close()
		err = json.NewDecoder(res.Body).Decode(&resTarget)
		if err != nil {
			log.Printf("%v", err)
			return
		}
		project.GitHub.Forks = resTarget
	}
	c.IndentedJSON(http.StatusFound, project)
}

func getProjectBranches(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")

	var resTarget []struct {
		Name string `json:"name,omitempty" bson:"name,omitempty"`
	}
	res, err := client.Get("https://api.github.com/repos/" + owner + "/" + repo + "/branches")
	if err != nil {
		log.Printf("%v", err)
		return
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&resTarget)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	c.IndentedJSON(http.StatusFound, resTarget)
}

func (con Connection) getProjectBalances(c *gin.Context) {
	project := c.Param("project")

	client, err := ethclient.Dial(config.EthNode)
	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
		return
	}

	contract, err := contracts.NewProjectCaller(common.HexToAddress(project), client)
	if err != nil {
		log.Fatalf("Unable to create contract binding:%v\n", err)
		return
	}
	maintainerBalance, maintainerReserved, investorBalance, _ := contract.ListBalances(nil)

	if (maintainerBalance != nil && maintainerReserved != nil && investorBalance != nil) {
		maintainerBalance = new(big.Int).Div(maintainerBalance, big.NewInt(ONE_TOKEN))
		maintainerReserved = new(big.Int).Div(maintainerReserved, big.NewInt(ONE_TOKEN))
		investorBalance = new(big.Int).Div(investorBalance, big.NewInt(ONE_TOKEN))
	}

	// Get Token balance for current user
	c.IndentedJSON(http.StatusFound, gin.H{
		"maintainer_balance": maintainerBalance,
		"maintainer_reserved": maintainerReserved,
		"investor_balance":    investorBalance,
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

	options := options.Find()
	options.SetProjection(bson.M{"name": 1, "categories": 1, "description": 1, "_id": 0})
	options.SetLimit(limitInt)
	options.SetSkip(limitInt * (pageInt - 1))
	total, err := con.Projects.CountDocuments(context.TODO(), bson.M{})
	filterCursor, err := con.Projects.Find(context.TODO(), bson.M{}, options)
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
