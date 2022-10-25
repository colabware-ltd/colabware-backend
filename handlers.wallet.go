package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type Wallet struct {
	Owner      primitive.ObjectID `json:"owner"`
	PrivateKey string             `json:"privateKey"`
	PublicKey  string             `json:"publicKey"`
	Address    string             `json:"address"`
}

type CreateWallet struct {
	Owner primitive.ObjectID `json:"owner"`
}

type TransferRequest struct {
	// 'Wallet Name' must match an existing wallet in the database
	Wallet primitive.ObjectID `json:"walletName" binding:"required"`
	To     string             `json:"to" binding:"required"`
	Amount float64            `json:"amount" binding:"required"`
}

type ReadBalance struct {
	Wallet         string         `json:"wallet"`
	ProjectAddress common.Address `json:"projectId"`
}

func (con Connection) postWallet(c *gin.Context) {
	var w CreateWallet
	if err := c.BindJSON(&w); err != nil {
		log.Printf("%v", err)
		return
	}
	_, wallet := con.createWallet(w.Owner)
	c.IndentedJSON(http.StatusCreated, wallet)
}

func (con Connection) transfer(c *gin.Context) {
	var r TransferRequest
	if err := c.BindJSON(&r); err != nil {
		log.Printf("%v", err)
		return
	}

	wallet := con.getWallet(r.Wallet)
	if wallet == nil {
		c.IndentedJSON(http.StatusOK, "Wallet not found")
	}

	// connect to an ethereum node  hosted by infura
	client, err := ethclient.Dial("https://rinkeby.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")

	if err != nil {
		log.Printf("Unable to connect to network:%v\n", err)
		return
	}

	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(wallet.Address))
	if err != nil {
		log.Print(err)
		return
	}

	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Print(err)
		return
	}

	toAddress := common.HexToAddress(r.To)
	var data []byte
	val := new(big.Float).SetFloat64(r.Amount)
	tx := types.NewTransaction(nonce, toAddress, etherToWei(val), gasLimit, gasPrice, data)

	privateKeyByte, err := hexutil.Decode("0x" + wallet.PrivateKey)
	if err != nil {
		log.Print(err)
		return
	}

	privateKey, err := crypto.ToECDSA(privateKeyByte)
	if err != nil {
		log.Print(err)
		return
	}

	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
	if err != nil {
		log.Print(err)
		return
	}

	// TODO: Convert to interface to handle user transactions.
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("tx sent: %s", signedTx.Hash().Hex())
	c.IndentedJSON(http.StatusOK, "tx sent")
	return
}

func (con Connection) createWallet(owner primitive.ObjectID) (primitive.ObjectID, Wallet) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	// Covert private key to bytes
	privateKeyBytes := crypto.FromECDSA(privateKey)

	// Save this to the database
	privateKeyHex := hexutil.Encode(privateKeyBytes)[2:]
	log.Println(privateKeyHex)

	// Derive public key from private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	publicKeyHex := hexutil.Encode(publicKeyBytes)[4:]
	log.Println(publicKeyHex)

	// Derive Address from public key
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	log.Println(address)

	w := Wallet{
		Owner:      owner,
		PrivateKey: privateKeyHex,
		PublicKey:  publicKeyHex,
		Address:    address,
	}

	// Make 'name' the unique index that identifies the wallets
	// con.Wallets.Indexes().CreateOne(
	// 	context.Background(),
	// 	mongo.IndexModel{
	// 		Keys:    bson.D{{Key: "name", Value: 1}},
	// 		Options: options.Index().SetUnique(true),
	// 	},
	// )

	result, err := con.Wallets.InsertOne(context.TODO(), w)
	// TODO: Update user object with created project
	if err != nil {
		log.Printf("%v", err)
	}
	return result.InsertedID.(primitive.ObjectID), w
}

func weiToEther(wei *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(params.Ether))
}

func etherToWei(eth *big.Float) *big.Int {
	truncInt, _ := eth.Int(nil)
	truncInt = new(big.Int).Mul(truncInt, big.NewInt(params.Ether))
	fracStr := strings.Split(fmt.Sprintf("%.18f", eth), ".")[1]
	fracStr += strings.Repeat("0", 18-len(fracStr))
	fracInt, _ := new(big.Int).SetString(fracStr, 10)
	wei := new(big.Int).Add(truncInt, fracInt)
	return wei
}

// TODO: Search by ID instead of walletName
func (con Connection) getWallet(owner primitive.ObjectID) *Wallet {
	wallet := &Wallet{}
	result := con.Wallets.FindOne(context.TODO(), bson.M{"owner": owner})

	if result == nil {
		log.Printf("Requested wallet %s not found", owner)
		return nil
	}

	err := result.Decode(wallet)
	if err != nil {
		log.Printf("Error decoding wallet %s", owner)
		return nil
	}

	log.Printf("Wallet found with address %s", wallet.Address)
	return wallet
}

func (con Connection) getBalance(c *gin.Context) {
	project := c.Param("project")
	wallet := c.Param("wallet")
	c.IndentedJSON(http.StatusFound, readBalance(project, wallet))
}

func readBalance(project string, wallet string) *big.Int {
	// Connect to ETH node hosted by Infura
	client, err := ethclient.Dial("https://rinkeby.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")
	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
		return big.NewInt(-1)
	}

	// Create contract binding
	contract, err := contracts.NewProjectCaller(common.HexToAddress(project), client)
	
	// Get balance associated with wallet address
	b, err := contract.GetBalance(nil, common.HexToAddress(wallet))
	if err != nil {
		log.Fatalf("Unable to get token balance for specified wallet:%v\n", err)
		return big.NewInt(-1)
	}
	
	return b
}