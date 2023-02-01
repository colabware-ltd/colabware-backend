package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/account"
	"github.com/stripe/stripe-go/v72/accountlink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

var client *http.Client

type User struct {
	ID                 primitive.ObjectID   `json:"_id,omitempty" bson:"_id,omitempty"`
	Login              string               `json:"login" bson:"login,omitempty"`
	Avatar             string               `json:"avatar_url" bson:"avatar_url,omitempty"`
	WalletAddress      string               `json:"wallet_address" bson:"wallet_address,omitempty"`
	ProjectsMaintained []primitive.ObjectID `json:"projects_maintained" bson:"projects_maintained,omitempty"`
	StripeAccount      StripeAccount        `json:"stripe_account" bson:"stripe_account,omitempty"`
	Name               string               `json:"name,omitempty" bson:"name,omitempty"`
	Email              string               `json:"email,omitempty" bson:"email,omitempty"`
	Location           string               `json:"location,omitempty" bson:"location,omitempty"`
	Bio                string               `json:"bio,omitempty" bson:"bio,omitempty"`
}

type StripeAccount struct {
	AccountID string `json:"account_id" bson:"account_id,omitempty"`
	Status    string `json:"status" bson:"status,omitempty"`
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

		// Create wallet for project and get address; initial project tokens will be minted for this address.
		// _, w := con.createWallet(result.InsertedID.(primitive.ObjectID))

		// Link a wallet to a user
		// _, err = con.Users.UpdateOne(
		// 	context.TODO(),
		// 	bson.M{"_id": result.InsertedID.(primitive.ObjectID)},
		// 	bson.D{
		// 		{"$set", bson.D{{"wallet_address", w.Address}}},
		// 	},
		// )
		// if err != nil {
		// 	log.Printf("%v", err)
		// 	return
		// }
		// log.Printf("New user and wallet successfully created.")

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

func (con Connection) stripeAccountLink(c *gin.Context) {
	userId := sessions.Default(c).Get("user-id")
	var accountId string

	var user User
	userSelector := bson.M{
		"login": userId,
	}
	err := con.Users.FindOne(context.TODO(), userSelector).Decode(&user)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// Create Stripe account for user if one doesn't exist in DB
	if user.StripeAccount.AccountID == "" {
		params := &stripe.AccountParams{
			Type: stripe.String(string(stripe.AccountTypeStandard)),
		}
		result, _ := account.New(params)

		accountId = result.ID

		userUpdate := bson.M{
			"$set": bson.M{
				"stripe_account": bson.M{
					"status":     "created",
					"account_id": result.ID,
				},
			},
		}
		_, err = con.Users.UpdateOne(context.TODO(), userSelector, userUpdate)
		if err != nil {
			log.Printf("%v", err)
			c.IndentedJSON(http.StatusInternalServerError, nil)
			return
		}
	} else {
		accountId = user.StripeAccount.AccountID
	}

	params := &stripe.AccountLinkParams{
		Account:    stripe.String(accountId),
		RefreshURL: stripe.String("https://localhost:3000/"),
		ReturnURL:  stripe.String("https://localhost:3000/api/user/stripe/verify"),
		Type:       stripe.String("account_onboarding"),
	}
	result, _ := accountlink.New(params)

	c.JSON(http.StatusOK, gin.H{"url": result.URL})
}

func (con Connection) stripeVerify(c *gin.Context) {
	userId := sessions.Default(c).Get("user-id")

	userSelector := bson.M{
		"login": userId,
	}
	userUpdate := bson.M{
		"$set": bson.M{
			"stripe_account.status": "linked",
		},
	}
	_, err := con.Users.UpdateOne(context.TODO(), userSelector, userUpdate)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	c.Redirect(http.StatusFound, "/")
}

func (con Connection) getUserById(id primitive.ObjectID) (*User, error) {
	var user User
	err = con.Users.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&user)
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}
	return &user, nil
}

func (con Connection) getUserBy(field string, address string) (*User, error) {
	var user User
	err = con.Users.FindOne(context.TODO(), bson.M{field: address}).Decode(&user)
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}
	return &user, nil
}

func (con Connection) updateUserDetails(c *gin.Context) {
	userId := sessions.Default(c).Get("user-id")
	var user User
	if err := c.BindJSON(&user); err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	userUpdate := bson.M{
		"$set": bson.M{"name": user.Name, "email": user.Email, "location": user.Location, "bio": user.Bio},
	}
	_, err = con.Users.UpdateOne(context.TODO(), bson.M{"login": userId}, userUpdate)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	c.IndentedJSON(http.StatusCreated, gin.H{})
}