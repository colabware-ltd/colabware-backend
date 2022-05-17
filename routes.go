package main

func initializeRoutes() {
	// Handle the index route
	router.GET("/", hello)
	router.POST("/project", postProject)
}
