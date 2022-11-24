package main

import (
	"context"
	"math/big"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

const NULL_ADDRESS = "0x0000000000000000000000000000000000000000"

func (con Connection) ethDeploymentMonitor() {
	for {
		selector := bson.M{"status": "pending"}
		projects, err := con.Projects.Distinct(context.TODO(), "address", selector)
		if err != nil {
			log.Printf("Failed to find new requests: %v", err)
			continue
		}

		for _, p := range projects {
			// Append addresses as common.Address
			projectAddress, ok := p.(string)
			if ok {
				client, err := ethclient.Dial(colabwareConf.EthNode)
				if err != nil {
					log.Fatalf("Unable to connect to network:%v\n", err)
					return
				}

				contract, err := contracts.NewProjectCaller(common.HexToAddress(projectAddress), client)
				if err != nil {
					log.Fatalf("Unable to create contract binding:%v\n", err)
					return
				}
				tokenAddress, err := contract.GetTokenAddress(&bind.CallOpts{})
				if err != nil {
					continue
				}
				if tokenAddress.Hex() != NULL_ADDRESS {
					ethTokenAddresses = append(ethTokenAddresses, tokenAddress)
					log.Printf("New token deployed: %v\n", tokenAddress.Hex())

					// Update eth log filter with new token address
					ethSubQuery = ethereum.FilterQuery{
						Addresses: ethTokenAddresses,
					}
					ethSub, err = ethClientWSS.SubscribeFilterLogs(context.Background(), ethSubQuery, ethLogs)
					if err != nil {
						log.Fatal(err)
					}

					// Update project in DB with token information
					var project Project
					selector = bson.M{ "address": projectAddress }
					update := bson.M{
						"$set": bson.M{
							"status":        "deployed",
							"token.address": tokenAddress.Hex(),
						},
					}
					_, err = con.Projects.UpdateOne(context.TODO(), selector, update)
					if err != nil {
						log.Printf("%v", err)
						continue
					}
					err = con.Projects.FindOne(context.TODO(), selector).Decode(&project)
					if err != nil {
						log.Printf("%v", err)
						continue
					}

					// Update balance information as TokenHolding in DB
					supply, err := contract.GetTokenSupply(&bind.CallOpts{})
					if err != nil {
						log.Printf("%v", err)
						continue
					}
					tokens := new(big.Int).Div(supply, big.NewInt(ONE_TOKEN)).Int64()

					con.updateTokenHoldings(
						tokenAddress.Hex(),
						NULL_ADDRESS,
						con.getWalletFromID(project.Wallet).Address,
						tokens,
					)
				}
			}
		}

	}
}
