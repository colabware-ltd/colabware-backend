package main

import (
	"context"
	"log"
	"strconv"
	"net/http"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/gin-contrib/sessions"
)

type Project struct {
	Name        string   `json:"name"`
	Repository  string   `json:"repository"`
	Description string   `json:"description"`
	Categories  []string `json:"categories"`
	Maintainers []string `json:"maintainers"`
	Token       Token    `json:"token"`
}

type Token struct {
	Name                 string  `json:"name"`
	Symbol               string  `json:"symbol"`
	Price                float32 `json:"price"`
	Supply               int     `json:"supply"`
	MaintainerAllocation float32 `json:"maintainerAllocation"`
}


func (con Connection) postProject(c *gin.Context) {
	var p Project
	session := sessions.Default(c)
	id := session.Get("user-id")
	if str, ok := id.(string); ok {
		p.Maintainers = append(p.Maintainers, str)
	}
	if err := c.BindJSON(&p); err != nil {
		log.Printf("%v", err)
		return
	}
	_, err := con.Projects.InsertOne(context.TODO(), p)
	// TODO: Update user object with created project
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
