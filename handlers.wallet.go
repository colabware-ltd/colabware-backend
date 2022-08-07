package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type Wallet struct {
	Name       string `json:"name"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	Address    string `json:"address"`
}

type CreateWalletReuest struct {
	Name string `json:"name" binding:"required"`
}

type TransferReuest struct {
	// 'Wallet Name' must match an existing wallet in the database
	WalletName string  `json:"walletName" binding:"required"`
	To         string  `json:"to" binding:"required"`
	Amount     float64 `json:"amount" binding:"required"`
}

func (con Connection) postWallet(c *gin.Context) {
	var r CreateWalletReuest
	if err := c.BindJSON(&r); err != nil {
		log.Printf("%v", err)
		return
	}
	w := con.createWallet(r.Name)
	c.IndentedJSON(http.StatusCreated, w)
}

func (con Connection) transfer(c *gin.Context) {
	var r TransferReuest
	if err := c.BindJSON(&r); err != nil {
		log.Printf("%v", err)
		return
	}

	wallet := con.getWallet(r.WalletName)
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

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("tx sent: %s", signedTx.Hash().Hex())
	c.IndentedJSON(http.StatusOK, "tx sent")
	return
}

func (con Connection) createWallet(name string) Wallet {
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
		Name:       name,
		PrivateKey: privateKeyHex,
		PublicKey:  publicKeyHex,
		Address:    address,
	}

	// Make 'name' the unique index that identifies the wallets
	con.Wallets.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	)

	_, err = con.Wallets.InsertOne(context.TODO(), w)
	// TODO: Update user object with created project
	if err != nil {
		log.Printf("%v", err)
	}
	return w
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

func (con Connection) getWallet(walletName string) *Wallet {
	wallet := &Wallet{}
	result := con.Wallets.FindOne(context.TODO(), bson.M{"name": walletName})

	if result == nil {
		log.Printf("Requested wallet %s not found", walletName)
		return nil
	}

	err := result.Decode(wallet)
	if err != nil {
		log.Printf("Error decoding wallet %s", walletName)
		return nil
	}

	log.Printf("Wallet found with address %s", wallet.Address)
	return wallet
}

func readAccountBalance() {
	// TODO
	// connect to an ethereum node hosted by infura
	client, err := ethclient.Dial("https://rinkeby.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")

	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
		return
	}

	header, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(header.Number.String()) // 5671744

}
