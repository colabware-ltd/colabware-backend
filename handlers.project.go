package main

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/colabware-ltd/colabware-backend/utilities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TODO: Create wallet for project upon creation. Maintainers should then
// be able to access this wallet. Wallet should hold maintainer tokens.
type Project struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
	Maintainers []primitive.ObjectID `json:"maintainers"`
	Token       Token    `json:"token"`
}

type Token struct {
	Name                 string         `json:"name"`
	Symbol               string         `json:"symbol"`
	Price                float32        `json:"price"`
	Supply               int64          `json:"supply"`
	MaintainerAllocation float32        `json:"maintainerAllocation"`
	Address              common.Address `json:"address"`
}


func (con Connection) postProject(c *gin.Context) {
	var p Project
	session := sessions.Default(c)
	userId := session.Get("user-id")

	// Deploy contract and store address; wait for execution to complete
	p.Token.Address = utilities.DeployToken(p.Token.Name, p.Token.Symbol, p.Token.Supply)
	log.Printf("Contract pending deploy: 0x%x\n", p.Token.Address)

	// Find ID of current user
	var user struct {
		ID primitive.ObjectID `bson:"_id, omitempty"`
	}
	e := con.Users.FindOne(context.TODO(), bson.M{"email": userId}).Decode(&user)
	if e != nil { 
		log.Printf("%v", e)
		return
	}

	// Add current user to project maintainers
	p.Maintainers = append(p.Maintainers, user.ID)
	if err := c.BindJSON(&p); err != nil {
		log.Printf("%v", err)
		return
	}
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
	c.IndentedJSON(http.StatusCreated, p)
}

func (con Connection) getProject(c *gin.Context) {
	name := c.Param("name")
	filterCursor, err := con.Projects.Find(context.TODO(), bson.M{"name": name})
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
	if len(projectsFiltered) == 0 {
		log.Printf("no item found")
		c.IndentedJSON(http.StatusOK, "no item found")
		return
	}

	log.Printf("%v", projectsFiltered[0])
	c.IndentedJSON(http.StatusFound, projectsFiltered[0])

	// TEST: Get project from Ethereum
	utilities.Fetch()
}

func (con Connection) listProjects(c *gin.Context) {
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
	log.Printf("%v", projectsFiltered)
	c.IndentedJSON(http.StatusFound, gin.H{"total": total, "results": projectsFiltered} )
}
