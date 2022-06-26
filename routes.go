package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
)

// Middleware for authorizing request
func authorizeRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		v := session.Get("user-id")
		if v == nil {
			c.IndentedJSON(http.StatusUnauthorized, nil)
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
	}
	router.GET("/", hello)
	router.GET("/api/project/:name", c.getProject)
	// router.POST("/api/project/", c.postProject)
	router.GET("/login", loginHandler)
	router.GET("/auth", c.authHandler)
}