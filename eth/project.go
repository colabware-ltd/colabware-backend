package eth

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/colabware-ltd/colabware-backend/utilities"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

func DeployProject(tokenName string, tokenSymbol string, totalSupply big.Int, maintainerSupply big.Int, walletAddress string, ethNode string, key string, ethChainId int64) common.Address {
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

	chainId := big.NewInt(ethChainId) // Goerli Chain ID
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

func ProjectTokenBalances(projectAddress string, ethNode string) (*big.Int, *big.Int, *big.Int, error) {
	client, err := ethclient.Dial(ethNode)
	if err != nil {
		log.Printf("%v", err)
		return nil, nil, nil, fmt.Errorf("%v", err)
	}

	contract, err := contracts.NewProjectCaller(common.HexToAddress(projectAddress), client)
	if err != nil {
		log.Printf("%v", err)
		return nil, nil, nil, fmt.Errorf("%v", err)
	}

	maintainerBalance, maintainerReserved, investorBalance, err := contract.ListBalances(nil)
	if err != nil {
		log.Printf("%v", err)
		return nil, nil, nil, fmt.Errorf("%v", err)
	}

	return maintainerBalance, maintainerReserved, investorBalance, nil
}

func ProjectTokenSupply(address string, ethNode string) (int64, error) {
	// Get total supply of tokens
	client, err := ethclient.Dial(ethNode)
	if err != nil {
		log.Printf("%v", err)
		return -1, fmt.Errorf("%v", err)
	}
	contract, err := contracts.NewProjectCaller(common.HexToAddress(address), client)
	if err != nil {
		log.Printf("%v", err)
		return -1, fmt.Errorf("%v", err)
	}
	supply, err := contract.GetTokenSupply(&bind.CallOpts{})
	if err != nil {
		log.Printf("%v", err)
		return -1, fmt.Errorf("%v", err)
	}
	totalSupply := utilities.BigIntToTokens(supply).Int64()

	return totalSupply, nil
}