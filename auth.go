package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/oauth2"
)

// Credentials which stores google ids.
type Credentials struct {
    Cid string `json:"cid"`
    Csecret string `json:"csecret"`
}

var cred Credentials
var conf *oauth2.Config
var state string

func initAuth() {
    file, err := ioutil.ReadFile("./creds.json")
    if err != nil {
        fmt.Printf("File error: %v\n", err)
        os.Exit(1)
    }
    json.Unmarshal(file, &cred)
	
    conf = &oauth2.Config{
        ClientID:     cred.Cid,
        ClientSecret: cred.Csecret,
        // Update RedirectURL
        RedirectURL:  "http://localhost:3000/api/auth",
        Scopes: []string{
            "https://github.com/login/oauth/authorize?client_id=eaf969af655329a70640&scope=read:user%20user:email",
        },
        Endpoint: oauth2.Endpoint{
            AuthURL:  "https://github.com/login/oauth/authorize",
	        TokenURL: "https://github.com/login/oauth/access_token",
        },
    }
}

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func getLoginURL(state string) string {
    // State can be some kind of random generated hash string.
    // See relevant RFC: http://tools.ietf.org/html/rfc6749#section-10.12
    return conf.AuthCodeURL(state)
}