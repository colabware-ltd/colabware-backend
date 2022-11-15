package main

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"

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
var config Config
var err error

type Connection struct {
	Projects        *mongo.Collection
	Requests        *mongo.Collection
	Contributions   *mongo.Collection
	Proposals       *mongo.Collection
	Users           *mongo.Collection
	Wallets         *mongo.Collection
	TokenPayments   *mongo.Collection
	TokenEventLogs  *mongo.Collection
}

func initDB() *mongo.Client {
	// Connect to the database
	credential := options.Credential{
		Username: config.DBUser,
		Password: config.DBPass,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(config.DBAddr).SetAuth(credential))
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
	log.SetReportCaller(true)

	config, err = LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// Set Gin to production mode
	gin.SetMode(gin.ReleaseMode)

	// Set the router as the default one provided by Gin
	router = gin.Default()
	router.Use(sessions.Sessions("colabware-auth", store))

	// Initialise DB
	dbClient := initDB()
	defer dbClient.Disconnect(context.Background())
	dbConn := Connection{
		Projects:         dbClient.Database("colabware").Collection("projects"),
		Requests:         dbClient.Database("colabware").Collection("requests"),
		Contributions:    dbClient.Database("colabware").Collection("contributions"),
		Proposals:        dbClient.Database("colabware").Collection("proposals"),
		Users:            dbClient.Database("colabware").Collection("users"),
		Wallets:          dbClient.Database("colabware").Collection("wallets"),
		TokenPayments:    dbClient.Database("colabware").Collection("token_payments"),
		TokenEventLogs:   dbClient.Database("colabware").Collection("token_event_logs"),
	}

	// Set API key for Stripe
	// Start payment processors
	//c := make(chan string)
	// go dbConn.tokenPaymentProcessor()

	stripe.Key = config.StripeKey

	// Initialize GitHub auth
	initAuth()

	// Initialize the routes
	initializeRoutes(dbConn)

	// Open WebSocket connection with Ethereum node
	ethClientWSS, err = ethclient.Dial(config.EthNodeWSS)
	if err != nil {
	  log.Fatal(err)
	}
	dbConn.getTokenAddresses()

	ethLogs = make(chan types.Log)
	ethSubQuery = ethereum.FilterQuery{
		Addresses: ethTokenAddresses,
	}
	ethSub, err = ethClientWSS.SubscribeFilterLogs(context.Background(), ethSubQuery, ethLogs)
	if err != nil {
  		log.Fatal(err)
	}

	// Start deployment monitor subroutine
	go dbConn.ethDeploymentMonitor()

	// Start Eth logger subrouting
	go dbConn.ethLogger()

	// Start serving the application
	err = router.Run("localhost:9998")
	if err != nil {
		log.Fatal(err)
	}
}