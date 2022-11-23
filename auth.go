package main

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/oauth2"
)

var conf *oauth2.Config
var state string

func initAuth() {
	conf = &oauth2.Config{
		ClientID:     colabwareConf.GitHubCID,
		ClientSecret: colabwareConf.GitHubCSecret,
		// Update RedirectURL
		RedirectURL: "http://localhost:3000/api/auth",
		Scopes: []string{
			"public_repo",
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
