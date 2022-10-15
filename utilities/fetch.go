package utilities

import (
	"log"
	"math/big"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func FetchProject(address common.Address, ethNode string) (*big.Int, error) {
	// connect to an ethereum node hosted by infura
	blockchain, err := ethclient.Dial(ethNode)

	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	// Create a new instance of the Project contract bound to a specific deployed contract
	contract, err := contracts.NewProject(address, blockchain)
	if err != nil {
		log.Fatalf("Unable to bind to deployed instance of contract:%v\n")
	}

	// Return total and maintainer token supply values
	return contract.GetTokenSupply(nil);
}