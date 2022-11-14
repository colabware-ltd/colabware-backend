package utilities

// Run this by
// go test -test.run=TestGetAddress

// public address: 0xe43aE80873919c38bb21D43d12963A55a38A6EB2

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestGetAddress(t *testing.T) {
	privateKeyByte, err := hexutil.Decode("0x" + "5e07055cc82d4df284f65da9296e5ec46010fc2b0061f68d4439a01f09ecbb95")
	if err != nil {
		fmt.Printf("%v", err)
	}

	privateKey, err := crypto.ToECDSA(privateKeyByte)
	if err != nil {
		fmt.Printf("%v", err)
	}

	// Derive public key from private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	publicKeyHex := hexutil.Encode(publicKeyBytes)[4:]
	fmt.Println(publicKeyHex)

	// Derive Address from public key
	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	fmt.Printf("public address: %s\n", address)
}
