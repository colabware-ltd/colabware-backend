package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

var ethTokenAddresses []common.Address
var ethSub ethereum.Subscription
var ethClientWSS *ethclient.Client
var ethLogs chan types.Log
var ethSubQuery ethereum.FilterQuery

func (con Connection) getTokenAddresses() {
	// Add random address for subscription query filter
	ethTokenAddresses = append(ethTokenAddresses, common.HexToAddress("0x4702c2881bccf4e3c1bebcadc863678518bbe85720878ed835fbc940815373ae"))
	tokens, err := con.Projects.Distinct(context.TODO(), "token.address", bson.M{})
	if err != nil {
		log.Printf("Failed to find token addresses: %v", err)
		return
	}

	for _, t := range tokens {
		// Append addresses as common.Address
		address, ok := t.(string)
		if ok {
			ethTokenAddresses = append(ethTokenAddresses, common.HexToAddress(address))
		}
	}
}

func (con Connection) ethLogger() {
	contractAbi, err := abi.JSON(strings.NewReader(string(contracts.ERC20ABI)))
	if err != nil {
		log.Fatal(err)
	}
	logTransferSig := []byte("Transfer(address,address,uint256)")
	LogApprovalSig := []byte("Approval(address,address,uint256)")
	logTransferSigHash := crypto.Keccak256Hash(logTransferSig)
	logApprovalSigHash := crypto.Keccak256Hash(LogApprovalSig)

	for {
		select {
			case err := <-ethSub.Err():
				log.Fatal(err)
			case vLog := <-ethLogs:
				fmt.Printf("Log Block Number: %d\n", vLog.BlockNumber)
				fmt.Printf("Log Index: %d\n", vLog.Index)

				switch vLog.Topics[0].Hex() {
				case logTransferSigHash.Hex():
					fmt.Printf("Log Name: Transfer\n")

					var transferEvent contracts.ERC20Transfer
					
					err = contractAbi.UnpackIntoInterface(&transferEvent, "Transfer", vLog.Data)
					if err != nil {
					  log.Fatal(err)
					}
					
					transferEvent.From = common.HexToAddress(vLog.Topics[1].Hex())
					transferEvent.To = common.HexToAddress(vLog.Topics[2].Hex())
					
					fmt.Printf("From: %s\n", transferEvent.From.Hex())
					fmt.Printf("To: %s\n", transferEvent.To.Hex())
					fmt.Printf("Tokens: %s\n", transferEvent.Value.String())
				case logApprovalSigHash.Hex():
					fmt.Printf("Log Name: Approval\n")

					var approvalEvent contracts.ERC20Approval
					
					err = contractAbi.UnpackIntoInterface(&approvalEvent, "Approval", vLog.Data)
					if err != nil {
					  log.Fatal(err)
					}
					
					approvalEvent.Owner = common.HexToAddress(vLog.Topics[1].Hex())
					approvalEvent.Spender = common.HexToAddress(vLog.Topics[2].Hex())
					
					fmt.Printf("Token Owner: %s\n", approvalEvent.Owner.Hex())
					fmt.Printf("Spender: %s\n", approvalEvent.Spender.Hex())
					fmt.Printf("Tokens: %s\n", approvalEvent.Value.String())
				}
		}
	}
}