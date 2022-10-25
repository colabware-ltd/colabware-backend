package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Contribution struct {
	CreatorId        primitive.ObjectID `json:"creator_id" bson:"creator_id,omitempty"`
	CreatorName      string             `json:"creator_name" bson:"creator_name,omitempty"`
	RequestId        primitive.ObjectID `json:"request_id" bson:"request_id,omitempty"`
	AmountReceived   float32            `json:"amount_received" bson:"amount_received,omitempty"`
	Transactions     []string           `json:"transactions" bson:"transactions,omitempty"`
	SelectedProposal primitive.ObjectID `json:"selected_proposal" bson:"selected_proposal,omitempty"`
}

// TODO: Update existing contribution entry for same user and request
func (con Connection) postContribution(c *gin.Context) {
	projectId := c.Param("project")
	requestId,_ := primitive.ObjectIDFromHex(c.Param("request"))
	paymentIntent := c.Query("payment_intent")

	var contribution Contribution
	var request Request
	var result *mongo.InsertOneResult
	var requestUpdate bson.M

	// Verify transaction
	pi,_ := paymentintent.Get(paymentIntent, nil)
	if (pi.Status != "succeeded") {
		return
	}
	
	userId := sessions.Default(c).Get("user-id")
	var user struct {
		ID    primitive.ObjectID `bson:"_id, omitempty"`
		Login string             `bson:"login, omitempty"`

	}
	err := con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if err != nil { 
		log.Printf("%v", err)
		return
	}

	// Get total contribution information from exiting request
	selector := bson.M{"_id": requestId}
	err = con.Requests.FindOne(context.TODO(), selector).Decode(&request)
	if err != nil { 
		log.Printf("%v", err)
		return
	}

	// Find existing contributions
	contributionSelector := bson.M{
		"request_id": requestId,
		"creator_name": userId,
	}
	err = con.Contributions.FindOne(context.TODO(), contributionSelector).Decode(&contribution)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create new contribution record
			contribution.CreatorId = user.ID
			contribution.CreatorName = user.Login
			contribution.AmountReceived = float32(pi.AmountReceived)/100
			contribution.RequestId = requestId
			contribution.Transactions = []string{paymentIntent}

			result, err = con.Contributions.InsertOne(context.TODO(), contribution)
			if err != nil { 
				log.Printf("%v", err)
				return
			}

			requestUpdate = bson.M{
				"$push": bson.M{"contributions": result.InsertedID},
				"$set" : bson.M{
					"contribution_total": request.ContributionTotal + float32(pi.AmountReceived)/100,
				},
			}
		} else {
			log.Printf("%v", err)
			return
		}
	} else {
		log.Printf("Updating existing contribution record")
		// Update existing contribution record
		contributionUpdate := bson.M{
			"$push": bson.M{"transactions": paymentIntent},
			"$set": bson.M{
				"amount_received": contribution.AmountReceived + float32(pi.AmountReceived)/100,
			},
		}
		_, err = con.Contributions.UpdateOne(context.TODO(), contributionSelector, contributionUpdate)
		if err != nil { 
			log.Printf("%v", err)
			return
		}

		requestUpdate = bson.M{
			"$set" : bson.M{
				"contribution_total": request.ContributionTotal + float32(pi.AmountReceived)/100,
			},
		}
	}
	// Update request
	_, err = con.Requests.UpdateOne(context.TODO(), selector, requestUpdate)
	if err != nil { 
		log.Printf("%v", err)
		return
	}

	c.Redirect(http.StatusFound, "/project/" + projectId)
}

func (con Connection) getContributions(c *gin.Context) {
	requestId, err := primitive.ObjectIDFromHex(c.Param("request"))
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	contributionSelector := bson.M{"request_id": requestId}
	filterCursor, err := con.Contributions.Find(context.TODO(), contributionSelector)
	var contributionsFiltered []bson.M
	err = filterCursor.All(context.TODO(), &contributionsFiltered)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	c.IndentedJSON(http.StatusFound, contributionsFiltered)
}