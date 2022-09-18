package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var router *gin.Engine
var store = cookie.NewStore([]byte("secret"))

type Connection struct {
	Projects *mongo.Collection
	Requests *mongo.Collection
	Users    *mongo.Collection
	Wallets  *mongo.Collection
}

func initDB(dbUser, dbPass, dbAddr string) *mongo.Client {
	// Connect to the database
	credential := options.Credential{
		Username: "colabware",
		Password: "zfbj3c7oEFgsuSrTx6",
	}
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017").SetAuth(credential))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
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
	config, err := LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
	// Set Gin to production mode
	gin.SetMode(gin.ReleaseMode)

	// Set the router as the default one provided by Gin
	router = gin.Default()
	router.Use(sessions.Sessions("colabware-auth", store))

	client := initDB(config.DBUser, config.DBPass, config.DBAddr)
	defer client.Disconnect(context.Background())
	conn := Connection{
		Projects: client.Database("colabware").Collection("projects"),
		Requests: client.Database("colabware").Collection("requests"),
		Users:    client.Database("colabware").Collection("users"),
		Wallets:  client.Database("colabware").Collection("wallets"),
	}

	stripe.Key = "sk_test_51J2rxbB2yNlUi1mdGCb18x2T4nsHHkfJ17iKhrmPWlw5Rpc9Fa6pWJR5iUWovE40Q6rajMQoImpapo3EF88iGeVL003oXMIDji"

	// Initialize Google auth
	initAuth()

	// Initialize the routes
	initializeRoutes(conn)

	// Start serving the application
	err = router.Run("localhost:9998")
	if err != nil {
		log.Fatal(err)
	}
}
