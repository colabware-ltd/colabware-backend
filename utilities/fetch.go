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

	// Create a new instance of the Inbox contract bound to a specific deployed contract
	contract, err := contracts.NewInbox(common.HexToAddress("0x907c3136f9689923710d2ee1983033136af390e4"), blockchain)
	if err != nil {
		log.Fatalf("Unable to bind to deployed instance of contract:%v\n")
	}

	fmt.Println(contract.Message(nil))

}
