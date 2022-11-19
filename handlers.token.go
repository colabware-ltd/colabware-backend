package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RampTokenPurchase struct {
	PurchaseID     string             `json:"purchase_id"`
	PurchaseSecret string             `json:"purchase_secret"`
	ProjectID      primitive.ObjectID `json:"project_id"`
	UserID         primitive.ObjectID `json:"user_id"`
	// Currently Crypto amount purchased == Token amount released
	CryptoAmount float64 `json:"crypto_amount"`
}

type RampTransactionResponse struct {
	Status string `json:"status"`
}

type TokenPurchaseTransactionRecord struct {
	MoneyIn         TransactionData    `json:"money_in"`
	ETHReadyUser    TransactionData    `json:"eth_ready_user"`
	ETHReadyProject TransactionData    `json:"eth_ready_project"`
	MoneyOut        TransactionData    `json:"money_out"`
	TokenOut        TransactionData    `json:"token_out"`
	UserWallet      primitive.ObjectID `json:"user_wallet" binding:"required"`
	ProjectWallet   primitive.ObjectID `json:"project_wallet" binding:"required"`
}

type TransactionData struct {
	IsDone bool
	Amount float64
	Symbol string
}

func (con Connection) purchaseToken(c *gin.Context) {
	var r RampTokenPurchase
	if err := c.BindJSON(&r); err != nil {
		log.Printf("%v", err)
		return
	}

	log.Printf("The URL is %s", fmt.Sprintf("https://api-instant-staging.supozu.com/api/host-api/purchase/%s?secret=%s", r.PurchaseID, r.PurchaseSecret))

	// Do the query
	resp, err := http.Get(fmt.Sprintf("https://api-instant-staging.supozu.com/api/host-api/purchase/%s?secret=%s", r.PurchaseID, r.PurchaseSecret))
	if err != nil {
		log.Printf("%v", err)
		return
	}

	defer resp.Body.Close()

	b, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Fatalln(err)
	}

	//Print the result of the query
	log.Printf("Transaction created on ramp with data %v", string(b))

	// // Create MongoDB Record for tracking
	// record := TokenPurchaseTransactionRecord{
	// 	MoneyIn: TransactionData{
	// 		IsDone: false,
	// 		Amount: r.CryptoAmount,
	// 		Symbol: "TEST_RAMP",
	// 	},

	// }

	// // Set off a transaction processing thread
	// go con.startTokenPurchaseWorkFlow()

	c.IndentedJSON(http.StatusOK, fmt.Sprintf("Transaction created in backend with data %v", string(b)))
}

// func (con Connection) startTokenPurchaseWorkFlow(r RampTokenPurchase) {
// 	var wg sync.WaitGroup
// 	wg.Add(1)

// 	go con.waitRampFinish(r)

// 	wg.Wait()
// 	wg.Add(2)

// 	// Supply User wallet with ETH
// 	// go supplyETHForWallet()

// 	// Supply Project wallet with ETH
// 	// go supplyETHForWallet()

// 	wg.Wait()

// 	wg.Add(2)
// 	// Release Project Tokens for the user
// 	// go transferTokenFromProjectToWallet

// 	// Send Money to Project Wallet
// 	// go transferToken

// 	wg.Wait()

// }

// func (con Connection) waitRampFinish(r RampTokenPurchase) {
// 	for {
// 		time.Sleep(5 * time.Second)

// 		resp, err := http.Get(fmt.Sprintf("https://api-instant-staging.supozu.com/api/host-api/purchase/%s?secret=%s", r.Id, r.Secret))
// 		if err != nil {
// 			log.Printf("%v", err)
// 			return
// 		}

// 		if resp.Body != nil {
// 			defer resp.Body.Close()
// 		}

// 		body, readErr := ioutil.ReadAll(resp.Body)
// 		if readErr != nil {
// 			log.Fatal(readErr)
// 		}

// 		status := RampTransactionResponse{}
// 		jsonErr := json.Unmarshal(body, &status)
// 		if jsonErr != nil {
// 			log.Fatal(jsonErr)
// 		}

// 		log.Printf("status is %v\n", status.Status)

// 		if status.Status == "RELEASED" {
// 			break
// 		}
// 	}

// 	log.Println("jumping out of the loop")
// }
