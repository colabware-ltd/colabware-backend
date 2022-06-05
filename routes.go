package main

func initializeRoutes(c Connection) {
	// Handle the index route
	router.GET("/", hello)
	router.POST("/api/project", c.postProject)
	router.GET("/api/project/:name", c.getProject)
}
