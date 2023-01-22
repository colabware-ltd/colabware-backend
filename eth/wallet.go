package eth

import (
	"fmt"
	"math/big"

	"github.com/colabware-ltd/colabware-backend/contracts"
	"github.com/colabware-ltd/colabware-backend/utilities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

func FetchBalance(wallet string, token string, ethNode string, ethChainId int64) (*big.Int, error) {
	client, err := ethclient.Dial(ethNode)
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}

	contract, err := contracts.NewERC20Caller(common.HexToAddress(token), client)
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}

	balance, err := contract.BalanceOf(nil, common.HexToAddress(wallet))
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}

	return utilities.BigIntToTokens(balance), nil
}

