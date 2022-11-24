package utilities

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

func DeployProject(tokenName string, tokenSymbol string, totalSupply big.Int, maintainerSupply big.Int, walletAddress string, ethNode string, key string, chain int64) common.Address {
	// Connect to an ethereum node
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
	log.Printf("Suggested gas price: %v", gasPrice.String())
	if err != nil {
		log.Fatal(err)
	}

	chainId := big.NewInt(chain) // Goerli Chain ID
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		log.Fatal(err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0) // in wei

	// For Reference
	// 3,628,241 unit burnt for contract creation and 10 tokens minted
	// 3,628,301 unit burnt for contract creation and 100 tokens minted
	// The contract creation alone with 10 token costed 635,471 units
	auth.GasLimit = uint64(4000000) // in units
	auth.GasPrice = gasPrice

	address, transaction, _, err := contracts.DeployProject(
		auth,
		client,
		tokenName,
		tokenSymbol,
		&totalSupply,
		&maintainerSupply,
		common.HexToAddress(walletAddress),
	)

	if err != nil {
		log.Fatalf("Unable to deploy: %v\nTransaction: %v", err, transaction)
	}

	return address
}
