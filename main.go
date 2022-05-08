package main

import (
	"github.com/gin-gonic/gin"
)

var router *gin.Engine

func main() {
	// Set Gin to production mode
	gin.SetMode(gin.ReleaseMode)

	// Set the router as the default one provided by Gin
	router = gin.Default()

	// Initialize the routes
	initializeRoutes()

	// Start serving the application
	router.Run(":9999")
}
