package main

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
	"golang.org/x/crypto/sha3"
)

const ONE_TOKEN = 1000000000000000000

type Wallet struct {
	Owner      primitive.ObjectID `json:"owner"`
	PrivateKey string             `json:"privateKey"`
	PublicKey  string             `json:"publicKey"`
	Address    string             `json:"address"`
}

type CreateWallet struct {
	Owner primitive.ObjectID `json:"owner"`
}

type TransferETHRequest struct {
	// 'Wallet' must be an ID that matches an existing wallet in the database
	Wallet primitive.ObjectID `json:"wallet" binding:"required"`
	To     string             `json:"to" binding:"required"`
	Amount float64            `json:"amount" binding:"required"`
}

type TransferTokenRequest struct {
	FromWallet   primitive.ObjectID `json:"from_wallet" binding:"required"`
	ToWallet     primitive.ObjectID `json:"to_wallet" binding:"required"`
	Amount       float64            `json:"amount" binding:"required"`
	TokenAddress common.Address
}

type TransferBetweenWalletsRequest struct {
	FromWallet primitive.ObjectID `json:"from_wallet" binding:"required"`
	ToWallet   primitive.ObjectID `json:"to_wallet" binding:"required"`
	Amount     float64            `json:"amount" binding:"required"`
}

type ReadBalance struct {
	Wallet         string         `json:"wallet"`
	ProjectAddress common.Address `json:"projectId"`
}

type TransferTest struct {
	ProjectWallet  string  `json:"project_wallet"`
	ToWallet       string  `json:"to_wallet"`
	Amount         float64 `json:"amount"`
	ProjectAddress string  `json:"project_address"`
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

func (con Connection) transferBetweenWallets(r TransferBetweenWalletsRequest) error {
	toWallet := con.getWalletFromID(r.ToWallet)
	if toWallet == nil {
		return fmt.Errorf("Wallet not found")
	}

	transferRequest := TransferETHRequest{
		Wallet: r.FromWallet,
		To:     toWallet.Address,
		Amount: r.Amount,
	}

	return con.transferETH(transferRequest)
}

func (con Connection) supplyETHForWallet(w primitive.ObjectID, a float64) error {
	wallet := con.getWalletFromID(w)
	if wallet == nil {
		return fmt.Errorf("Wallet not found")
	}

	// connect to an ethereum node  hosted by infura
	client, err := ethclient.Dial("https://goerli.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")
	if err != nil {
		return fmt.Errorf("Unable to connect to network:%v\n", err)
	}

	// Declan's wallet with tons of ETH (supposedly)
	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(config.EthAddr))
	if err != nil {
		return err
	}

	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	toAddress := common.HexToAddress(wallet.Address)
	var data []byte
	val := new(big.Float).SetFloat64(a)
	tx := types.NewTransaction(nonce, toAddress, etherToWei(val), gasLimit, gasPrice, data)

	privateKeyByte, err := hexutil.Decode("0x" + config.EthKey)
	if err != nil {
		return err
	}

	privateKey, err := crypto.ToECDSA(privateKeyByte)
	if err != nil {
		return err
	}

	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
	if err != nil {
		return err
	}

	// TODO: Convert to interface to handle user transactions.
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	log.Printf("tx sent: %s\n", signedTx.Hash().Hex())
	return nil

}

func (con Connection) transferETH(r TransferETHRequest) error {
	wallet := con.getWalletFromID(r.Wallet)
	if wallet == nil {
		return fmt.Errorf("Wallet not found")
	}

	// connect to an ethereum node  hosted by infura
	client, err := ethclient.Dial("https://goerli.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")

	if err != nil {
		return fmt.Errorf("Unable to connect to network:%v\n", err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(wallet.Address))
	if err != nil {
		return err
	}

	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	toAddress := common.HexToAddress(r.To)
	var data []byte
	val := new(big.Float).SetFloat64(r.Amount)
	tx := types.NewTransaction(nonce, toAddress, etherToWei(val), gasLimit, gasPrice, data)

	privateKeyByte, err := hexutil.Decode("0x" + wallet.PrivateKey)
	if err != nil {
		return err
	}

	privateKey, err := crypto.ToECDSA(privateKeyByte)
	if err != nil {
		return err
	}

	signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, privateKey)
	if err != nil {
		return err
	}

	// TODO: Convert to interface to handle user transactions.
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	log.Printf("tx sent: %s\n", signedTx.Hash().Hex())
	return nil
}

func (con Connection) transferTest(c *gin.Context) {
	var t TransferTest
	if err := c.BindJSON(&t); err != nil {
		log.Printf("%v", err)
		return
	}

	projectWallet, _ := primitive.ObjectIDFromHex(t.ProjectWallet)
	toWallet, _ := primitive.ObjectIDFromHex(t.ToWallet)
	
	con.transferTokenFromProjectToWallet(projectWallet, toWallet, t.Amount, common.HexToAddress(t.ProjectAddress))
}


func (con Connection) transferTokenFromProjectToWallet(projectWallet primitive.ObjectID, toWallet primitive.ObjectID, amount float64, projectAddr common.Address) error {
	// connect to an ethereum node  hosted by infura
	blockchain, err := ethclient.Dial("https://goerli.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")
	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	// bind to the project that created the token
	project, err := contracts.NewProject(projectAddr, blockchain)
	if err != nil {
		log.Fatalf("Unable to bind to deployed instance of contract:%v\n", err)
	}

	tokenAddr, err := project.GetTokenAddress(&bind.CallOpts{})

	transferRequest := TransferTokenRequest{
		FromWallet:   projectWallet,
		ToWallet:     toWallet,
		Amount:       amount,
		TokenAddress: tokenAddr,
	}

	err = con.transferToken(transferRequest)
	if err != nil {
		return nil
	}

	return err
}

func floatToBigInt(val float64) *big.Int {
	bigval := new(big.Float)
	bigval.SetFloat64(val)
	// Set precision if required.
	// bigval.SetPrec(64)

	oneToken := new(big.Float)
	oneToken.SetInt(big.NewInt(ONE_TOKEN))

	bigval.Mul(bigval, oneToken)

	result := new(big.Int)
	bigval.Int(result)

	return result
}

func (con Connection) transferToken(r TransferTokenRequest) error {
	fromWallet := con.getWalletFromID(r.FromWallet)
	if fromWallet == nil {
		return fmt.Errorf("Wallet not found")
	}

	toWallet := con.getWalletFromID(r.ToWallet)
	if toWallet == nil {
		return fmt.Errorf("Wallet not found")
	}

	// connect to an ethereum node  hosted by infura
	blockchain, err := ethclient.Dial("https://goerli.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")
	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	nonce, err := blockchain.PendingNonceAt(context.Background(), common.HexToAddress(fromWallet.Address))
	if err != nil {
		return err
	}

	toAddress := common.HexToAddress(toWallet.Address)

	privateKeyByte, err := hexutil.Decode("0x" + fromWallet.PrivateKey)
	if err != nil {
		return err
	}

	privateKey, err := crypto.ToECDSA(privateKeyByte)
	if err != nil {
		return err
	}

	value := big.NewInt(0) // in wei (0 eth)

	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4]

	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)

	amount := floatToBigInt(r.Amount)

	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	// gasLimit, err := blockchain.EstimateGas(context.Background(), ethereum.CallMsg{
	// 	To:   &r.TokenAddress,
	// 	Data: data,
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// 	return err
	// }
	// fmt.Println(gasLimit)
	gasLimit := uint64(84000)
	gasPrice, err := blockchain.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Token Address: %v\n", r.TokenAddress)
	tx := types.NewTransaction(nonce, r.TokenAddress, value, gasLimit, gasPrice, data)

	chainID, err := blockchain.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
		return err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
		return err
	}

	err = blockchain.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
		return err
	}

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())

	return nil
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
func (con Connection) getWalletFromOwner(owner primitive.ObjectID) *Wallet {
	wallet := &Wallet{}
	result := con.Wallets.FindOne(context.TODO(), bson.M{"owner": owner})

	if result == nil {
		log.Printf("Requested wallet for Owner: %s not found", owner)
		return nil
	}

	err := result.Decode(wallet)
	if err != nil {
		log.Printf("Error decoding wallet for Owner %s", owner)
		return nil
	}

	log.Printf("Wallet found with address %s", wallet.Address)
	return wallet
}

func (con Connection) getWalletFromID(id primitive.ObjectID) *Wallet {
	wallet := &Wallet{}
	result := con.Wallets.FindOne(context.TODO(), bson.M{"_id": id})

	if result == nil {
		log.Printf("Requested wallet for ID: %s not found", id)
		return nil
	}

	err := result.Decode(wallet)
	if err != nil {
		log.Printf("Error decoding wallet for ID: %s", id)
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
	client, err := ethclient.Dial("https://goerli.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")
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
