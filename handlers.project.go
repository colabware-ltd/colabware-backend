package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Project struct {
	Name          string        `json:"name"`
	Repository    string        `json:"repository"`
	Description   string        `json:"description"`
	Categories    []string      `json:"categories"`
	Maintainers   []string      `json:"maintainers"`
	RequestConfig RequestConfig `json:"requestConfig"`
}

type RequestConfig struct {
	MinTokens          int  `json:"minTokens"`
	MaintainerApproval bool `json:"maintainerApproval"`
}

func (con Connection) postProject(c *gin.Context) {
	var p Project
	if err := c.BindJSON(&p); err != nil {
		return
	}

	_, err := con.Projects.InsertOne(context.Background(), p)
	if err != nil {
		log.Fatal("can't create db stuff")
	}

	c.IndentedJSON(http.StatusCreated, p)
}
