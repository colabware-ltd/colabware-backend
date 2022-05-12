package main

import (
	"context"
	"log"
	"math/big"
	"strings"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const key = `{"address":"1bc22202dc39a523ebe114724a6e6f428031edd4","crypto":{"cipher":"aes-128-ctr","ciphertext":"f66e3e8398ab89f4ce51828f17d5e026d3c17ebbf86bdd8fd3f4cd3cd8bfccb6","cipherparams":{"iv":"7324b9b6cbdbb9f069e526d06d730e31"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":262144,"p":1,"r":8,"salt":"7d34704545a545a7157ae4ce9e6d757d8ad9812530b900a71c97207ae080f1de"},"mac":"b66f97f02751e01578a5eb5e2beca856b2d8bdc4bd14a39b455d7e32b9d259dd"},"id":"ee0f9732-05ab-49eb-b67e-1fc1870c3a7a","version":3}`

func main() {
	// connect to an ethereum node  hosted by infura
	blockchain, err := ethclient.Dial("https://rinkeby.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")

	if err != nil {
		log.Fatalf("Unable to connect to network:%v\n", err)
	}

	// Get credentials for the account to charge for contract deployments
	auth, err := bind.NewTransactor(strings.NewReader(key), "")

	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
	}

	gasPrice, err := blockchain.SuggestGasPrice(context.Background())

	// auth.Signer = types.LatestSignerForChainID(big.NewInt(int64(1)))

	contract, err := contracts.NewInbox(common.HexToAddress("0x907c3136f9689923710d2ee1983033136af390e4"), blockchain)
	if err != nil {
		log.Fatalf("Unable to bind to deployed instance of contract:%v\n")
	}

	_, err = contract.SetMessage(&bind.TransactOpts{
		From:      auth.From,
		Signer:    auth.Signer,
		Value:     nil,
		Nonce:     big.NewInt(int64(2)),
		GasFeeCap: nil,
		GasPrice:  gasPrice,
	}, "Hello From Earth")

	if err != nil {
		log.Fatalf("Failed to run transaction: %v", err)
	}

}
