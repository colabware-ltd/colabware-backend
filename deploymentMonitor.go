package main

import (
	"context"

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

					ethSubQuery = ethereum.FilterQuery{
						Addresses: ethTokenAddresses,
					}
					ethSub, err = ethClientWSS.SubscribeFilterLogs(context.Background(), ethSubQuery, ethLogs)
					if err != nil {
						log.Fatal(err)
					}

					selector = bson.M{"address": projectAddress}
					update := bson.M{
						"$set": bson.M{
							"status":        "deployed",
							"token.address": tokenAddress.Hex(),
						},
					}
					_, err = con.Projects.UpdateOne(context.TODO(), selector, update)
					if err != nil {
						log.Printf("%v", err)
						return
					}
				}
			}
		}

	}
}
