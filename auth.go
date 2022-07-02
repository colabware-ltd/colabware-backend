package main

import (
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "io/ioutil"
    "fmt"
    "os"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
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
        RedirectURL:  "http://127.0.0.1/auth",
        Scopes: []string{
            "https://www.googleapis.com/auth/userinfo.email",
        },
        Endpoint: google.Endpoint,
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