package utilities

import (
	"fmt"
	"log"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func Fetch() {
	// connect to an ethereum node  hosted by infura
	blockchain, err := ethclient.Dial("https://rinkeby.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")

	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	// Create a new instance of the Project contract bound to a specific deployed contract
	contract, err := contracts.NewProject(common.HexToAddress("0x9f9e9b79dfb823617d8147cdcb3fb53d4e42a589"), blockchain)
	if err != nil {
		log.Fatalf("Unable to bind to deployed instance of contract:%v\n")
	}

	tokens, err := contract.GetTokens(nil)

	fmt.Println(tokens[0].TokenAddress)
}