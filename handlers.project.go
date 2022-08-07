package main

import (
	"context"
	"fmt"
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
	Address              common.Address `json:"address"`
}

type Token struct {
	Name             string  `json:"name"`
	Symbol           string  `json:"symbol"`
	Price            float32 `json:"price"`
	TotalSupply      int64   `json:"totalSupply"`
	MaintainerSupply int64   `json:"maintainerSupply"`
}


func (con Connection) postProject(c *gin.Context) {
	var p Project
	session := sessions.Default(c)
	userId := session.Get("user-id")

	if err := c.BindJSON(&p); err != nil {
		log.Printf("%v", err)
		return
	}

	// Deploy contract and store address; wait for execution to complete
	p.Address = utilities.DeployProject(p.Token.Name, p.Token.Symbol, p.Token.TotalSupply, p.Token.MaintainerSupply)
	log.Printf("Contract pending deploy: 0x%x\n", p.Address)
	
	// Find ID of current user
	var user struct {
		ID primitive.ObjectID `bson:"_id, omitempty"`
	}
	e := con.Users.FindOne(context.TODO(), bson.M{"email": userId}).Decode(&user)
	if e != nil { 
		log.Printf("%v", e)
		return
	}
	p.Maintainers = append(p.Maintainers, user.ID)

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

	var project Project
	err := con.Projects.FindOne(context.TODO(), bson.M{"name": name}).Decode(&project)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	c.IndentedJSON(http.StatusFound, project)

	// TEST: Get project from Ethereum
	totalSupply, err := utilities.FetchProject(project.Address)

	fmt.Printf("total:%d\n", totalSupply)
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
