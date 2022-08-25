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
		authorized.GET("/logout", c.logout)
		authorized.GET("/", c.getUser)
	}
	router.GET("/api", hello)
	router.GET("/api/project/:name", c.getProject)
	router.GET("/api/project/list", c.listProjects)
	router.POST("/api/createwallet", c.postWallet)
	router.POST("/api/transfer", c.transfer)
	router.GET("/api/login", loginHandler)
	router.GET("/api/auth", c.authHandler)
}
