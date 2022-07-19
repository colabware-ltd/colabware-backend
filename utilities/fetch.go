package main

import (
	"fmt"
	"log"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	// connect to an ethereum node  hosted by infura
	blockchain, err := ethclient.Dial("https://rinkeby.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")

	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	// Create a new instance of the Project contract bound to a specific deployed contract
	contract, err := contracts.NewProject(common.HexToAddress("0xa6ddad0fdcb3b50357352bb1dceeea5033c9d24f"), blockchain)
	if err != nil {
		log.Fatalf("Unable to bind to deployed instance of contract:%v\n")
	}

	fmt.Println(contract)
	fmt.Println("Hello!")

}
