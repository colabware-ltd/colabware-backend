package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var router *gin.Engine

type Connection struct {
	Projects *mongo.Collection
}

func initDB() *mongo.Client {
	// Connect to the database
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://172.18.0.2:27017"))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		// Can't connect to Mongo server
		log.Fatal(err)
	}
	return client
}

func main() {
	// Set Gin to production mode
	gin.SetMode(gin.ReleaseMode)

	// Set the router as the default one provided by Gin
	router = gin.Default()

	client := initDB()
	defer client.Disconnect(context.Background())
	conn := Connection{
		Projects: client.Database("colabware").Collection("projects"),
	}

	// Initialize the routes
	initializeRoutes(conn)

	// Start serving the application
	router.Run(":9999")
}
