package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

type Payment struct {
	Type   string  `json:"type"` // Is the payment a bounty or token purchase
	Id     string  `json:"id"` // Id of Token if token purchase. Check that stored price matches price from client.
	Amount float32 `json:"amount"` // Amount from client
}

func calculateOrderAmount(payment Payment) int64 {
	// Replace this constant with a calculation of the order's amount
	// TODO: Calculate the order total on the server to prevent
	// people from directly manipulating the amount on the client (e.g. token price)
	return int64(payment.Amount) * 100
  }

func createPaymentIntent(c *gin.Context) {
	var p Payment
	if err := c.BindJSON(&p); err != nil {
		log.Printf("%v", err)
		return
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(calculateOrderAmount(p)),
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