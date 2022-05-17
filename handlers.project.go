package main

import (
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

func postProject(c *gin.Context) {
	var p Project
	if err := c.BindJSON(&p); err != nil {
		return
	}

	c.IndentedJSON(http.StatusCreated, p)
}
