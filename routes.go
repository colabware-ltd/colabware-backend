package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Middleware for authorizing request
func authorizeRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		v := session.Get("user-id")
		if v == nil {
			c.IndentedJSON(http.StatusUnauthorized, nil)
			log.Println("f")
			c.Abort()
		}
		c.Next()
	}
}

func initializeRoutes(c Connection) {
	// Handle the index route
	authorized := router.Group("/api/user")
	authorized.Use(authorizeRequest())
	{
		authorized.POST("/project", c.postProject)
		authorized.POST("/project/:id/request", c.postRequest)
		// authorized.POST("/project/:project/request/:request/vote", c.postRequestVote)
		// authorized.POST("/project/:project/request/:request/response", c.postRequestResponse)
		authorized.GET("/logout", c.logout)
		authorized.GET("/", c.getUser)
	}
	router.GET("/api", hello)
	router.GET("/api/project/:name", c.getProject)
	router.GET("/api/project/list", c.listProjects)
	router.GET("/api/project/:name/request/list", c.listRequests)
	router.GET("/api/project/:name/balance/:wallet", c.getBalance)
	router.GET("/api/project/:name/balances", c.getProjectBalances)	
	router.POST("/createwallet", c.postWallet)
	// router.POST("/transfer", c.transfer)
	router.GET("/login", loginHandler)
	router.GET("/auth", c.authHandler)
}
