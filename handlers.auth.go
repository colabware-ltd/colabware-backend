package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

var client *http.Client

type User struct {
	Login              string               `json:"login"`
	Avatar             string               `json:"avatar_url"`
	WalletAddress      string               `json:"wallet_address"`
	ProjectsMaintained []primitive.ObjectID `json:"projects_maintained"`
}

func loginHandler(c *gin.Context) {
	state = randToken()
	session := sessions.Default(c)
	session.Set("state", state)
    session.Save()
	fmt.Println("Saved session: ", session.Get("state"))
	c.JSON(http.StatusOK, gin.H{"url": getLoginURL(state)})
}

func (con Connection) getUser(c *gin.Context) {
	session := sessions.Default(c)
	userId := session.Get("user-id")
	log.Printf(fmt.Sprint(userId))
	filterCursor, err := con.Users.Find(context.TODO(), bson.M{"login": userId})
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	var usersFiltered []bson.M
	err = filterCursor.All(context.TODO(), &usersFiltered)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	if len(usersFiltered) == 0 {
		log.Printf("no user found")
		c.IndentedJSON(http.StatusOK, "no user found")
		return
	}
	log.Printf("%v", usersFiltered[0])
	c.JSON(http.StatusFound, usersFiltered[0])
	// c.IndentedJSON(http.StatusFound, usersFiltered[0])
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
	client = conf.Client(oauth2.NoContext, tok)

	// Ger information about the user
	userInfo, err := client.Get("https://api.github.com/user")
    if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
        return
	}
    defer userInfo.Body.Close()
    data, _ := ioutil.ReadAll(userInfo.Body)

	// Marshal user info response
	u := User{}
	if err = json.Unmarshal(data, &u); err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": "Error marshalling response. Please try agian."})
		return
	}
	// Set login as id for current session
	session.Set("user-id", u.Login)
	err = session.Save()
	if err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": "Error while saving session. Please try again."})
		return
	}

	// Find user in db
	filterCursor, err := con.Users.Find(context.TODO(), bson.M{"login": u.Login})
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	var usersFiltered []bson.M
	err = filterCursor.All(context.TODO(), &usersFiltered)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	if len(usersFiltered) == 0 {
		// Create new user in db
		log.Printf("Existing user not found. Create new entry in db.")

		_, err := con.Users.InsertOne(context.TODO(), u)
		if err != nil {
			log.Printf("%v", err)
			return
		}

	}
    c.Redirect(http.StatusFound, "/")
	
}

func (con Connection) logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user-id")
	err := session.Save()
	if err != nil {
		log.Println(err)
		return
	}
}