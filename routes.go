package main

func initializeRoutes(c Connection) {
	// Handle the index route
	router.GET("/", hello)
	router.POST("/project", c.postProject)
}
