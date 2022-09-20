package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Payment struct {
	Type    string  `json:"type"` // Is the payment a bounty or token purchase.
	Project string  `json:"project"` // Id of the project associated with the current purchase.
	Amount  float32 `json:"amount"` // Amount from client.
	Tokens  uint64  `json:"tokens"` // Number of tokens 
}

func (con Connection) calculateOrderAmount(payment Payment) int64 {
	// Replace this constant with a calculation of the order's amount
	// TODO: Calculate the order total on the server to prevent
	// people from directly manipulating the amount on the client (e.g. token price)
	var amount int64
	projectId,_ := primitive.ObjectIDFromHex(payment.Project)
	if payment.Project != "" && payment.Type == "token" {
		var project Project
		options := options.FindOne()
		options.SetProjection(bson.M{"token": 1})
		selector := bson.M{"_id": projectId}
		err := con.Projects.FindOne(context.TODO(), selector, options).Decode(&project)
		if err != nil {
			log.Printf("%v", err)
			return -1
		}
		log.Println(project)
		amount = int64(project.Token.Price) * int64(payment.Tokens) * 100
	} else {
		amount = int64(payment.Amount) * 100
	}
	log.Println(amount)
	return amount 
  }

func (con Connection) createPaymentIntent(c *gin.Context) {
	var p Payment
	if err := c.BindJSON(&p); err != nil {
		log.Printf("%v", err)
		return
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(con.calculateOrderAmount(p)),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
		  Enabled: stripe.Bool(true),
		},
	  }
	  pi, err := paymentintent.New(params)
	  log.Printf("pi.New: %v", pi)
	
	  if err != nil {
		log.Printf("pi.New: %v", err)
		return
	  }
	  
	  // Return client secret
	  c.JSON(http.StatusOK, gin.H{"clientSecret": pi.ClientSecret})
}