package main

import (
	"fmt"
	"log"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/sessions"
	"golang.org/x/oauth2"
	"go.mongodb.org/mongo-driver/bson"
)

type User struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Gender        string `json:"gender"`
}

func loginHandler(c *gin.Context) {
	state = randToken()
	session := sessions.Default(c)
	session.Set("state", state)
    session.Save()
	fmt.Println("Saved session: ", session.Get("state"))
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
	userinfo, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
    if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
        return
	}
    defer userinfo.Body.Close()
    data, _ := ioutil.ReadAll(userinfo.Body)

	// Marshal user info response from Google
	u := User{}
	if err = json.Unmarshal(data, &u); err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": "Error marshalling response. Please try agian."})
		return
	}

	// Set email as id for current session
	session.Set("user-id", u.Email)
	err = session.Save()
	if err != nil {
		log.Println(err)
		c.HTML(http.StatusBadRequest, "error.tmpl", gin.H{"message": "Error while saving session. Please try again."})
		return
	}

	// Find user in db
	filterCursor, err := con.Users.Find(context.TODO(), bson.M{"email": u.Email})
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
    c.Status(http.StatusOK)
}