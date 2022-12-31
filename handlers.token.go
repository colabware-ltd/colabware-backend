package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-contrib/sessions"
	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RampTokenPurchaseRequest struct {
	PurchaseID      string             `json:"purchase_id"`
	PurchaseSecret  string             `json:"purchase_secret"`
	ProjectWalletID primitive.ObjectID `json:"project_wallet_id" bson:"project_wallet_id"`
	UserWalletAddr  string             `json:"user_wallet_addr"`
	// Currently Crypto amount purchased == Token amount released
	CryptoAmount float64 `json:"crypto_amount"`
}

type RampTransactionResponse struct {
	Status string `json:"status"`
}

type TokenPurchaseTransactionRecord struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	PurchaseID      string             `json:"purchase_id" bson:"purchase_id"`
	PurchaseSecret  string             `json:"purchase_secret" bson:"purchase_secret"`
	MoneyIn         TransactionData    `json:"money_in" bson:"money_in"`
	ETHReadyUser    TransactionData    `json:"eth_ready_user" bson:"eth_ready_user"`
	ETHReadyProject TransactionData    `json:"eth_ready_project" bson:"eth_ready_project"`
	MoneyOut        TransactionData    `json:"money_out" bson:"money_out"`
	TokenOut        TransactionData    `json:"token_out" bson:"token_out"`
	UserWalletID    primitive.ObjectID `json:"user_wallet_id" bson:"user_wallet_id" binding:"required"`
	ProjectWalletID primitive.ObjectID `json:"project_wallet_id" bson:"project_wallet_id" binding:"required"`
}

type TransactionData struct {
	IsDone bool    `json:"is_done" bson:"is_done"`
	Amount float64 `json:"amount" bson:"amount"`
	Symbol string  `json:"symbol" bson:"symbol"`
}

/*
Get RampTokenPurchaseRequest from FrontEnd, then kickstart a workflow thread to process it

Args:
	RampTokenPurchaseRequest: got from the request body, which should have all information needed to create a transaction record.

Returns:
	Nothing, but you get a database record, and a workflow (thread) running to fulfill the request

*/
func (con Connection) purchaseToken(c *gin.Context) {
	var r RampTokenPurchaseRequest
	if err := c.BindJSON(&r); err != nil {
		log.Printf("%v", err)
		return
	}

	// Get User Wallet ID from Wallet Address
	userWalletID, err := con.getWalletIDFromAddr(r.UserWalletAddr)
	if err != nil {
		log.Printf("%v\n", err)
		return
	}

	// Get Project
	project, err := con.getProjectByWalletID(r.ProjectWalletID)
	if err != nil {
		log.Printf("%v\n", err)
		return
	}

	// Convert the request to the internal transaction record for saving
	record := TokenPurchaseTransactionRecord{
		MoneyIn: TransactionData{
			IsDone: false,
			Amount: r.CryptoAmount,
			Symbol: "MATIC_TEST",
		},
		ETHReadyUser: TransactionData{
			IsDone: false,
			// TODO: Estimate gas amount to transfer
			Amount: 0.0009,
			Symbol: "MATIC",
		},
		ETHReadyProject: TransactionData{
			IsDone: false,
			// TODO: Estimate gas amount to transfer
			Amount: 0.0009,
			Symbol: "MATIC",
		},
		// TODO: Subtract percentage as transaction commission fee
		MoneyOut: TransactionData{
			IsDone: false,
			Amount: r.CryptoAmount,
			Symbol: "MATIC_TEST",
		},
		TokenOut: TransactionData{
			IsDone: false,
			Amount: con.calculateTokens(r.CryptoAmount, project),
			Symbol: project.Token.Symbol,
		},
		ProjectWalletID: r.ProjectWalletID,
		UserWalletID:    *userWalletID,
		PurchaseID:      r.PurchaseID,
		PurchaseSecret:  r.PurchaseSecret,
		ID:              primitive.NewObjectID(),
	}

	result, err := con.TokenPayments.InsertOne(context.TODO(), record)
	if err != nil {
		log.Printf("%v", err)
	}

	if result == nil {
		log.Printf("why nil here")
		return
	}

	// Start the processing flow
	go con.startTokenPurchaseWorkFlow(record)

	c.IndentedJSON(http.StatusOK, fmt.Sprintf("Workflow initiated in backend with ID: %v", result.InsertedID))

	log.Printf("Token payment record inserted: %v", record)
}

func (con Connection) startTokenPurchaseWorkFlow(r TokenPurchaseTransactionRecord) {
	// connect to an ethereum node  hosted by infura
	client, err := ethclient.Dial(colabwareConf.EthNode)
	if err != nil {
		log.Printf("Unable to connect to network:%v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go con.waitRampFinish(r, &wg)
	wg.Wait()

	// Cannot do these in parallel (yet) because you are connecting to the same Infura, which cause the later transaction to overwrite the first.
	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(colabwareConf.EthAddr))
	if err != nil {
		log.Printf("Unable to get nonce:%v\n", err)
		return
	}

	wg.Add(2)

	// Supply User wallet with ETH
	go con.waitSupplyETHForWallet(r.UserWalletID, r.ETHReadyUser.Amount, r.ID, "eth_ready_user", &wg, nonce)

	// Supply Project wallet with ETH
	go con.waitSupplyETHForWallet(r.ProjectWalletID, r.ETHReadyProject.Amount, r.ID, "eth_ready_project", &wg, nonce+1)

	wg.Wait()

	wg.Add(2)

	// // Send Money to Project Wallet
	// TODO: Investigate why transfer isn't being processed
	go con.waitTransferMoneyToProjectWallet(r, &wg, 0)

	// // Release Project Tokens for the user
	go con.waitTransferTokenToUserWallet(r, &wg, 0)

	wg.Wait()

	log.Debugf("Token Purchase Flow completed, transcation ID for Mongo Lookup: %v", r.ID)
}

func (con Connection) waitTransferMoneyToProjectWallet(r TokenPurchaseTransactionRecord, wg *sync.WaitGroup, n uint64) {
	transferRequest := TransferTokenRequest{
		FromWallet: r.UserWalletID,
		ToWallet:   r.ProjectWalletID,
		Amount:     r.MoneyOut.Amount,
		// MATIC_TEST RAMP test token address
		TokenAddress: common.HexToAddress(colabwareConf.MaticTestAddr),
	}

	hash, err := con.transferToken(transferRequest, n)
	if err != nil {
		log.Printf("Error transfer money to project wallet. Trans ID: %v. Err: %v\n", r.ID, err)
		return
	}

	tx := waitForTransaction(hash)
	log.Debugf("Transaction %v completed, waitTransferMoneyToProjectWallet thread finished\n", tx)

	err = con.flipRecordFlag(r.ID, "money_out")
	if err != nil {
		log.Println(err)
	}

	wg.Done()
}

func (con Connection) waitTransferTokenToUserWallet(r TokenPurchaseTransactionRecord, wg *sync.WaitGroup, n uint64) {
	p, err := con.getProjectByWalletID(r.ProjectWalletID)
	if err != nil {
		log.Printf("Error find project from Wallet ID: %v", err)
		return
	}

	hash, err := con.transferTokenFromProjectToWallet(r.ProjectWalletID, r.UserWalletID, r.TokenOut.Amount, common.HexToAddress(p.Address), n)
	if err != nil {
		log.Printf("Error transfering token to user wallet: %v", err)
		return
	}

	tx := waitForTransaction(hash)
	log.Debugf("Transaction %v completed, waitSupplyETHForWallet thread finished\n", tx)

	err = con.flipRecordFlag(r.ID, "token_out")
	if err != nil {
		log.Println(err)
	}

	wg.Done()
}

func (con Connection) waitSupplyETHForWallet(w primitive.ObjectID, a float64, record primitive.ObjectID, flag string, wg *sync.WaitGroup, n uint64) {
	log.Debugf("waitSupplyETHForWallet started for wallet: %v\n", w)
	hash, err := con.supplyETHForWallet(w, a, n)
	if err != nil {
		log.Printf("Failed to supply ETH for user wallet: %v\n", err)
	}

	tx := waitForTransaction(hash)
	log.Tracef("Transaction %v completed, waitSupplyETHForWallet thread finished\n", tx)

	err = con.flipRecordFlag(record, flag)
	if err != nil {
		log.Println(err)
	}

	wg.Done()
}

func waitForTransaction(h common.Hash) *types.Transaction {
	c, err := ethclient.Dial("https://goerli.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")
	if err != nil {
		log.Errorf("Unable to connect to network:%v\n", err)
		return nil
	}

	for {
		time.Sleep(5 * time.Second)

		tx, pending, err := c.TransactionByHash(context.Background(), h)
		if err != nil {
			log.Errorf("Unable to wait for transaction: %v\n", err)
			return nil
		}

		log.Debugf("Is transaction %v pending: %v", h, pending)
		if !pending {
			log.Tracef("Transaction %v completed\n", tx)
			return tx
		}

	}

}

func (con Connection) waitRampFinish(r TokenPurchaseTransactionRecord, wg *sync.WaitGroup) {
	for {
		time.Sleep(5 * time.Second)

		log.Debugf("The URL to query RAMP is %s", fmt.Sprintf("https://api-instant-staging.supozu.com/api/host-api/purchase/%s?secret=%s", r.PurchaseID, r.PurchaseSecret))

		resp, err := http.Get(fmt.Sprintf("https://api-instant-staging.supozu.com/api/host-api/purchase/%s?secret=%s", r.PurchaseID, r.PurchaseSecret))
		if err != nil {
			log.Printf("%v", err)
			return
		}

		if resp.Body != nil {
			defer resp.Body.Close()
		}

		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			log.Fatal(readErr)
		}

		status := RampTransactionResponse{}
		jsonErr := json.Unmarshal(body, &status)
		if jsonErr != nil {
			log.Fatal(jsonErr)
		}

		log.Debugf("status is %v\n", status.Status)

		if status.Status == "RELEASED" {
			break
		}
	}

	// Update the database record
	err := con.flipRecordFlag(r.ID, "money_in")
	if err != nil {
		log.Println(err)
	}

	log.Debug("waitRampFinish thread finished")
	wg.Done()
}

func (con Connection) flipRecordFlag(id primitive.ObjectID, flag string) error {
	log.Trace("Flipping %s on record _id: %s", flag, id)

	selector := bson.M{
		"_id": id,
	}

	update := bson.M{
		"$set": bson.M{
			fmt.Sprintf("%s.is_done", flag): true,
		},
	}
	_, err := con.TokenPayments.UpdateOne(context.TODO(), selector, update)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func (con Connection) calculateTokens(cryptoAmount float64, project *Project) float64 {
	usdcReceived := cryptoAmount / ONE_TOKEN
	return usdcReceived / project.Token.Price
}

func (con Connection) getUserTokens(c *gin.Context) {
	userId := sessions.Default(c).Get("user-id")
	var user User

	// Get wallet address of current user
	err = con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// Get token holdings for user
	filterCursor, err := con.TokenHoldings.Find(context.TODO(), bson.M{"wallet_address": user.WalletAddress})
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	var tokenHoldings []TokenHolding
	err = filterCursor.All(context.TODO(), &tokenHoldings)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	c.IndentedJSON(http.StatusFound, gin.H{"results": tokenHoldings})
}
