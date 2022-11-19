package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
)

type RampTokenPurchase struct {
	Id     string `json:"id"`
	Secret string `json:"secret"`
}

func (con Connection) purchaseToken(c *gin.Context) {
	var r RampTokenPurchase
	if err := c.BindJSON(&r); err != nil {
		log.Printf("%v", err)
		return
	}

	log.Printf("The URL is %s", fmt.Sprintf("https://api-instant-staging.supozu.com/api/host-api/purchase/%s?secret=%s", r.Id, r.Secret))

	// Do the query
	resp, err := http.Get(fmt.Sprintf("https://api-instant-staging.supozu.com/api/host-api/purchase/%s?secret=%s", r.Id, r.Secret))
	if err != nil {
		log.Printf("%v", err)
		return
	}

	defer resp.Body.Close()

	b, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Fatalln(err)
	}

	//Print the result of the query
	log.Printf("Transaction created on ramp with data %v", string(b))

	c.IndentedJSON(http.StatusOK, fmt.Sprintf("Transaction created on ramp with data %v", string(b)))
}
