package main

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//sample token payment request to put in MongoDB
/*
{
	"_id": ObjectID(),
	"status": "NEW",
	"buyer_wallet": "634ae544b28984de38951ce5", // treenhan wallet
	"buyer_amount": 0.005, // money sent out
	"seller_wallet": "63630cd5d76cddfc9e14f20f", // NTH3 project wallet
	"seller_amount": 0.01, // money sent back in
	"project_addr": "0x789D6aa9f93eA9F4F6148391BB55235Ae0F64b06", // NTH2 project address
}
*/
type TokenPaymentRequest struct {
	ID           primitive.ObjectID `bson:"_id, omitempty"`
	Status       string             `bson:"status"`
	BuyerWallet  primitive.ObjectID `bson:"buyer_wallet"`
	BuyerAmount  float64            `bson:"buyer_amount,string"`
	SellerWallet primitive.ObjectID `bson:"seller_wallet"`
	SellerAmount float64            `bson:"seller_amount,string"`
	ProjectAddr  string             `bson:"project_addr"`
}

type TokenPaymentResponse struct {
	success   bool
	paymentId primitive.ObjectID
}

func (con Connection) tokenPaymentProcessor() {
	for {
		time.Sleep(5 * time.Second)
		// Get request out of database to process
		var request TokenPaymentRequest
		selector := bson.M{"status": "NEW"}
		options := options.FindOne()
		err := con.TokenPayments.FindOne(context.TODO(), selector, options).Decode(&request)

		if err != nil {
			// ErrNoDocuments means that the filter did not match any documents in
			// the collection.
			if err == mongo.ErrNoDocuments {
				continue
			}

			log.Printf("Failed to find new requests: %v", err)
			continue
		}

		// Process the request
		err = con.processPayment(request)

		if err != nil {
			log.Printf("Failed to process token payment: %v\n", err)
			// Update the record to failed
			con.updatePaymentRecordToFailed(request.ID)
		} else {
			// Update the record to success
			con.updatePaymentRecordToSucceeded(request.ID)
		}

	}
}

func (con Connection) updatePaymentRecordToFailed(id primitive.ObjectID) error {
	var result bson.M
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
	}
	selector := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{"status": "FAILED"},
	}

	err := con.TokenPayments.FindOneAndUpdate(context.TODO(), selector, update, &opt).Decode(&result)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if err == mongo.ErrNoDocuments {
			log.Println("Can't find the payment request to update after processing")
		}
	} else {
		fmt.Printf("token payment order failed to process: %v\n", result)
	}

	return nil
}

func (con Connection) updatePaymentRecordToSucceeded(id primitive.ObjectID) error {
	var result bson.M
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
	}
	selector := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{"status": "SUCCESS"},
	}

	err := con.TokenPayments.FindOneAndUpdate(context.TODO(), selector, update, &opt).Decode(&result)
	if err != nil {
		// ErrNoDocuments means that the filter did not match any documents in
		// the collection.
		if err == mongo.ErrNoDocuments {
			log.Println("Can't find the payment request to update after processing")
		}
	} else {
		fmt.Printf("token payment order is successfully processed: %v\n", result)
	}

	return nil
}

func (con Connection) processPayment(r TokenPaymentRequest) error {
	// Using RAMP TEST token as main currency instead of USDC for testing
	// https://goerli.etherscan.io/token/0x5248dddc7857987a2efd81522afba1fcb017a4b7.
	sendMoneyToProject := TransferTokenRequest{
		FromWallet:   r.BuyerWallet,
		ToWallet:     r.SellerWallet,
		Amount:       r.BuyerAmount,
		TokenAddress: common.HexToAddress("0x5248dDdC7857987A2EfD81522AFBA1fCb017A4b7"),
	}
	err := con.transferToken(sendMoneyToProject)
	if err != nil {
		return err
	}

	err = con.transferTokenFromProjectToWallet(r.SellerWallet, r.BuyerWallet, r.SellerAmount, common.HexToAddress(r.ProjectAddr))
	if err != nil {
		return err
	}

	return nil

	// forwardTransferRequest = TransferRequest{
	// 	Wallet: r.FromWallet,
	// 	To: r.To,
	// 	Amount: r.ToAmount
	// }
	// con.transfer()
	// blockchain, err := ethclient.Dial("https://goerli.infura.io/v3/f3f2d6ceb53143cfbba9d2326bf5617f")
	// if err != nil {
	// 	return nil, fmt.Errorf("Unable to connect to network:%v\n", err)
	// }

	// toWallet := con.getWallet(r.ToWallet)
	// if toWallet == nil {
	// 	return nil, fmt.Errorf("Wallet not found")
	// }

	// fromWallet := con.getWallet(r.FromWallet)
	// if fromWallet == nil {
	// 	return nil, fmt.Errorf("Wallet not found")
	// }

	// // Send Token back
	// toPrivateKeyByte, err := hexutil.Decode("0x" + toWallet.PrivateKey)
	// if err != nil {
	// 	return nil, err
	// }

	// toPrivateKey, err := crypto.ToECDSA(toPrivateKeyByte)
	// if err != nil {
	// 	return nil, err
	// }

	// toAddress := common.HexToAddress(toWallet.Address)

	// // Sent ETH out

	// // ---- Sample implementation

	// // Fetch private key in
	// privateKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// publicKey := privateKey.Public()
	// publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	// if !ok {
	// 	return nil, fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	// }
	// fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// value := big.NewInt(0) // in wei (0 eth)
	// gasPrice, err := client.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	// tokenAddress := common.HexToAddress("0x28b149020d2152179873ec60bed6bf7cd705775d")

}
