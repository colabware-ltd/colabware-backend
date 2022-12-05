package main

import (
	"context"
	"fmt"
	"math/big"
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
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ethTokenAddresses []common.Address
var ethSub ethereum.Subscription
var ethClientWSS *ethclient.Client
var ethLogs chan types.Log
var ethSubQuery ethereum.FilterQuery

type TokenHolding struct {
	WalletAddress string `json:"wallet_address" bson:"wallet_address"`
	TokenAddress  string `json:"token_address" bson:"token_address"`
	Balance       uint64 `json:"balance" bson:"balance"`
} 

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
	ethSubQuery = ethereum.FilterQuery{
		Addresses: ethTokenAddresses,
	}
	ethSub, err = ethClientWSS.SubscribeFilterLogs(context.Background(), ethSubQuery, ethLogs)
	if err != nil {
  		log.Fatal(err)
	}
}

func (con Connection) updateTokenHoldings(tokenAddress string, fromAddress string, toAddress string, amount int64) {
	opts := options.FindOneAndUpdate().SetUpsert(true)

	toSelector := bson.M{
		"wallet_address": toAddress,
		"token_address": tokenAddress,
	}
	toUpdate := bson.M{ 
		"$inc": bson.M{
			"balance": amount,
		},
	}
	con.TokenHoldings.FindOneAndUpdate(context.TODO(), toSelector, toUpdate, opts)

	// TODO: Check tokenAddress is being passed correctly
	if (fromAddress != NULL_ADDRESS) {
		fromSelector := bson.M{
			"wallet_address": fromAddress,
			"token_address": tokenAddress,
		}
		fromUpdate := bson.M{ 
			"$inc": bson.M{
				"balance": -amount,
			},
		}
		con.TokenHoldings.FindOneAndUpdate(context.TODO(), fromSelector, fromUpdate)
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
				log.Printf("%v", err)
				// Restart EthLogger
				con.ethLogger()
				return
			case vLog := <-ethLogs:
				fmt.Printf("Log Block Number: %d\n", vLog.BlockNumber)
				fmt.Printf("Log Index: %d\n", vLog.Index)

				switch vLog.Topics[0].Hex() {
				case logTransferSigHash.Hex():
					fmt.Printf("Log Name: Transfer\n")

					var transferEvent contracts.ERC20Transfer
					
					err = contractAbi.UnpackIntoInterface(&transferEvent, "Transfer", vLog.Data)
					if err != nil {
						log.Printf("%v", err)
						continue
					}
					
					transferEvent.From = common.HexToAddress(vLog.Topics[1].Hex())
					transferEvent.To = common.HexToAddress(vLog.Topics[2].Hex())
					
					tokens := new(big.Int).Div(transferEvent.Value, big.NewInt(ONE_TOKEN)).Int64()

					_, err := con.TokenEventLogs.InsertOne(context.TODO(), bson.M{
						"from": transferEvent.From.Hex(),
						"to": transferEvent.To.Hex(),
						"tokens": tokens,
					})
					if err != nil {
						log.Printf("%v", err)
					}

					// TODO: Update balance information in DB
					con.updateTokenHoldings(
						vLog.Address.Hex(),
						transferEvent.From.Hex(),
						transferEvent.To.Hex(),
						tokens,
					)
					
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

					// _, err := con.TokenEventLogs.InsertOne(context.TODO(), bson.M{
					// 	"owner": approvalEvent.Owner.Hex(),
					// 	"spender": approvalEvent.Spender.Hex(),
					// 	"tokens": approvalEvent.Value.String(),
					// })
					// if err != nil {
					// 	log.Printf("%v", err)
					// }
					
					fmt.Printf("Token Owner: %s\n", approvalEvent.Owner.Hex())
					fmt.Printf("Spender: %s\n", approvalEvent.Spender.Hex())
					fmt.Printf("Tokens: %s\n", approvalEvent.Value.String())
				}
		}
	}
}