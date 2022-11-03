package utilities

import (
	"math/big"

	log "github.com/sirupsen/logrus"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func FetchProject(address common.Address) (*big.Int, error) {
	// connect to an ethereum node hosted by infura
	blockchain, err := ethclient.Dial("https://goerli.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")

	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	// Create a new instance of the Project contract bound to a specific deployed contract
	contract, err := contracts.NewProject(address, blockchain)
	if err != nil {
		log.Fatalf("Unable to bind to deployed instance of contract:%v\n", err)
	}

	// Return total and maintainer token supply values
	return contract.GetTokenSupply(nil)
}
