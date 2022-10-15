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

func initializeRoutes(db Connection) {
	// Handle the index route
	authorized := router.Group("/api/user")
	authorized.Use(authorizeRequest())
	{
		authorized.POST("/project", db.postProject)
		authorized.POST("/project/:id/request", db.postRequest)
		authorized.GET("/project/:name/request/:id/bounty", db.postBounty)
		authorized.GET("/project/branches/:owner/:repo", getProjectBranches)
		authorized.POST("/request/:request/response", db.postRequestResponse)
		// authorized.POST("/project/:project/request/:request/vote", c.postRequestVote)
		authorized.GET("/logout", db.logout)
		authorized.GET("/", db.getUser)
		authorized.POST("/payment-intent", db.createPaymentIntent)
	}
	router.GET("/api", hello)
	router.GET("/api/project/:name", db.getProject)
	router.GET("/api/project/list", db.listProjects)
	router.GET("/api/project/:name/request/list", db.listRequests)
	router.GET("/api/project/:name/balance/:wallet", db.getBalance)
	router.GET("/api/project/:name/balances", db.getProjectBalances)	
	router.POST("/api/createwallet", db.postWallet)
	// router.POST("/api/transfer", c.transfer)
	router.GET("/api/login", loginHandler)
	router.GET("/api/auth", db.authHandler)
}
