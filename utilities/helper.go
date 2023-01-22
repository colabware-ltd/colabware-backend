package utilities

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/params"
)

const ONE_TOKEN = 1000000000000000000

func MaxInt(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

func FloatToBigInt(val float64) *big.Int {
	bigval := new(big.Float)
	bigval.SetFloat64(val)
	// Set precision if required.
	// bigval.SetPrec(64)

	oneToken := new(big.Float)
	oneToken.SetInt(big.NewInt(ONE_TOKEN))

	bigval.Mul(bigval, oneToken)

	result := new(big.Int)
	bigval.Int(result)

	return result
}

func WeiToEther(wei *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(params.Ether))
}

func EtherToWei(eth *big.Float) *big.Int {
	truncInt, _ := eth.Int(nil)
	truncInt = new(big.Int).Mul(truncInt, big.NewInt(params.Ether))
	fracStr := strings.Split(fmt.Sprintf("%.18f", eth), ".")[1]
	fracStr += strings.Repeat("0", 18-len(fracStr))
	fracInt, _ := new(big.Int).SetString(fracStr, 10)
	wei := new(big.Int).Add(truncInt, fracInt)
	return wei
}

func BigIntToTokens(val *big.Int) *big.Int {
	if val == big.NewInt(0) {
		return big.NewInt(0)
	} else {
		return new(big.Int).Div(val, big.NewInt(ONE_TOKEN))
	}
}