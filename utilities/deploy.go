package utilities

import (
	"context"
	"crypto/ecdsa"
	"log"
	"math/big"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func DeployProject(tokenName string, tokenSymbol string, totalSupply int64, maintainerSupply int64, walletAddress string, ethNode string, key string) common.Address {
	// Connect to an ethereum node hosted by Infura
	client, err := ethclient.Dial(ethNode)
	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	privateKey, err := crypto.HexToECDSA(key)
    if err != nil {
        log.Fatal(err)
    }

	publicKey := privateKey.Public()
    publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
    if !ok {
        log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
    }

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
    nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
    if err != nil {
        log.Fatal(err)
    }

	gasPrice, err := client.SuggestGasPrice(context.Background())
    if err != nil {
        log.Fatal(err)
    }

	chainId := big.NewInt(5) // Goerli Chain ID
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
        log.Fatal(err)
    }
    auth.Nonce = big.NewInt(int64(nonce))
    auth.Value = big.NewInt(0)     // in wei
    // auth.GasLimit = uint64(300000) // in units
    auth.GasPrice = gasPrice
	log.Printf(gasPrice.String())

	address, _, _, err := contracts.DeployProject(
		auth,
		client,
		tokenName,
		tokenSymbol,
		big.NewInt(totalSupply),
		big.NewInt(maintainerSupply),
		common.HexToAddress(walletAddress),
	)
	if err != nil {
		log.Fatalf("Unable to deploy: %v\n", err)
	}

	return address
}
