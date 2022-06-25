package main

import (
	"fmt"
	"log"
	"io/ioutil"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
	"golang.org/x/oauth2"
)

func loginHandler(c *gin.Context) {
	state = randToken()
	session := sessions.Default(c)
	session.Set("state", state)
    session.Save()
	fmt.Println(session.Get("state"))
	c.Writer.Write([]byte("<html><title>Golang Google</title> <body> <a href='" + getLoginURL(state) + "'><button>Login with Google!</button> </a> </body></html>"))
	// Send login URL with state to client in JSON data
	// jsonData := []byte(`{"msg":"Authenticated!"}`)
	// c.Data(http.StatusOK, "application/json", jsonData)
}

func (con Connection) authHandler(c *gin.Context) {
    session := sessions.Default(c)
	retrievedState := session.Get("state")
    if retrievedState != c.Query("state") {
        c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid session state: %s", retrievedState))
        return
    }
	// Handle the exchange code to initiate a transport.
	tok, err := conf.Exchange(oauth2.NoContext, c.Query("code"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
        return
	}
	client := conf.Client(oauth2.NoContext, tok)
	// Ger information about the user
	email, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
    if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
        return
	}
    defer email.Body.Close()
    data, _ := ioutil.ReadAll(email.Body)
    log.Println("Email body: ", string(data))
    c.Status(http.StatusOK)
}