package main

import (
	"context"
	"fmt"
	"math/big"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/colabware-ltd/colabware-backend/utilities"
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
	"golang.org/x/crypto/sha3"
)

type Wallet struct {
	ID         primitive.ObjectID `bson:"_id" json:"_id"`
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



func (con Connection) supplyETHForWallet(w primitive.ObjectID, a float64, n uint64) (common.Hash, error) {
	wallet := con.getWalletFromID(w)
	if wallet == nil {
		return *new(common.Hash), fmt.Errorf("Wallet not found")
	}

	// Connect to an ethereum node hosted by infura
	client, err := ethclient.Dial(colabwareConf.EthNode)
	if err != nil {
		return *new(common.Hash), fmt.Errorf("Unable to connect to network:%v\n", err)
	}

	privateKey, err := crypto.HexToECDSA(colabwareConf.EthKey)
	if err != nil {
		return *new(common.Hash), err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return *new(common.Hash), fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return *new(common.Hash), err
	}
	nonce = utilities.MaxInt(nonce, n)

	chainId := big.NewInt(colabwareConf.EthChainId) // Goerli Chain ID

	gasLimit := uint64(21000)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return *new(common.Hash), err
	}

	toAddress := common.HexToAddress(wallet.Address)
	var data []byte
	val := new(big.Float).SetFloat64(a)
	
	tx := types.NewTransaction(nonce, toAddress, utilities.EtherToWei(val), gasLimit, gasPrice, data)
	
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainId), privateKey)
	if err != nil {
		return *new(common.Hash), err
	}

	// TODO: Convert to interface to handle user transactions.
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return *new(common.Hash), err
	}

	log.Debugf("tx sent: %s\n", signedTx.Hash())
	return signedTx.Hash(), nil

}

func (con Connection) transferETH(r TransferETHRequest) error {
	wallet := con.getWalletFromID(r.Wallet)
	if wallet == nil {
		return fmt.Errorf("Wallet not found")
	}

	// connect to an ethereum node  hosted by infura
	client, err := ethclient.Dial(colabwareConf.EthNode)

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
	tx := types.NewTransaction(nonce, toAddress, utilities.EtherToWei(val), gasLimit, gasPrice, data)

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

	log.Debugf("tx sent: %s\n", signedTx.Hash().Hex())
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

	con.transferTokenFromProjectToWallet(projectWallet, toWallet, t.Amount, common.HexToAddress(t.ProjectAddress), 0)
}

func (con Connection) transferTokenFromProjectToWallet(projectWallet primitive.ObjectID, toWallet primitive.ObjectID, amount float64, projectAddr common.Address, n uint64) (common.Hash, error) {
	// connect to an ethereum node  hosted by infura
	blockchain, err := ethclient.Dial(colabwareConf.EthNode)
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

	tx, err := con.transferToken(transferRequest, n)
	if err != nil {
		return *new(common.Hash), nil
	}

	return tx, err
}

func (con Connection) transferTokenFromWalletToProject(projectWallet primitive.ObjectID, fromWallet primitive.ObjectID, amount float64, projectAddr common.Address) (common.Hash, error) {
	// connect to an ethereum node  hosted by infura
	blockchain, err := ethclient.Dial(colabwareConf.EthNode)
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
		FromWallet:   fromWallet,
		ToWallet:     projectWallet,
		Amount:       amount,
		TokenAddress: tokenAddr,
	}

	hash, err := con.transferToken(transferRequest, 0)
	if err != nil {
		return *new(common.Hash), nil
	}

	return hash, err
}



func (con Connection) transferToken(r TransferTokenRequest, n uint64) (common.Hash, error) {
	fromWallet := con.getWalletFromID(r.FromWallet)
	if fromWallet == nil {
		return *new(common.Hash), fmt.Errorf("Wallet not found")
	}

	toWallet := con.getWalletFromID(r.ToWallet)
	if toWallet == nil {
		return *new(common.Hash), fmt.Errorf("Wallet not found")
	}

	// connect to an ethereum node  hosted by infura
	blockchain, err := ethclient.Dial(colabwareConf.EthNode)
	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	nonce, err := blockchain.PendingNonceAt(context.Background(), common.HexToAddress(fromWallet.Address))
	if err != nil {
		return *new(common.Hash), err
	}
	nonce = utilities.MaxInt(nonce, n)

	toAddress := common.HexToAddress(toWallet.Address)

	privateKeyByte, err := hexutil.Decode("0x" + fromWallet.PrivateKey)
	if err != nil {
		return *new(common.Hash), err
	}

	privateKey, err := crypto.ToECDSA(privateKeyByte)
	if err != nil {
		return *new(common.Hash), err
	}

	value := big.NewInt(0) // in wei (0 eth)

	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4]

	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)

	amount := utilities.FloatToBigInt(r.Amount)

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
	gasLimit := uint64(200000)
	gasPrice, err := blockchain.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Token Address: %v Amount: %v\n", r.TokenAddress, amount)
	tx := types.NewTransaction(nonce, r.TokenAddress, value, gasLimit, gasPrice, data)

	chainID, err := blockchain.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
		return *new(common.Hash), err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
		return *new(common.Hash), err
	}

	err = blockchain.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
		return *new(common.Hash), err
	}

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())

	return signedTx.Hash(), nil
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
		ID:         primitive.NewObjectID(),
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

func (con Connection) getWalletIDFromAddr(addr string) (*primitive.ObjectID, error) {
	var wallet *Wallet
	result := con.Wallets.FindOne(context.TODO(), bson.M{"address": addr})

	if result == nil {
		return nil, fmt.Errorf("Requested wallet for Address: %s not found", addr)
	}

	err := result.Decode(&wallet)
	if err != nil {
		return nil, fmt.Errorf("Error decoding wallet for Address: %s due to %v", addr, err)
	}

	log.Printf("Wallet ID found: %s", wallet.ID)
	return &wallet.ID, nil
}