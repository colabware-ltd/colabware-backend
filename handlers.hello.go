package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func hello(c *gin.Context) {
	jsonData := []byte(`{"msg":"this worked"}`)
	c.Data(http.StatusOK, "application/json", jsonData)
}
