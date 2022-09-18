package main

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Request struct {
	Creator             primitive.ObjectID   `json:"creator"`
	Project             primitive.ObjectID   `json:"project"`
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	Type                string               `json:"type"`
	Bounty              float32              `json:"bounty"`
	BountyContributions []BountyContribution `json:"bountyContributions"`
	Votes               []Vote               `json:"votes"`
	Responses           []Response           `json:"responses"`
}

// TODO: Add expiry to bounty
type BountyContribution struct {
	Creator         primitive.ObjectID `json:"creator"`
	AmountReceived  float32            `json:"amountReceived"`
	PaymentIntent   string             `json:"paymentIntent"`
}

type Vote struct {
	Creator    primitive.ObjectID `json:"creator"`
	TokensHeld int64              `json:"tokens"`
}

type Response struct {
	Creator     primitive.ObjectID `json:"creator"`
	URL         string             `json:"url"`
	Description string             `json:"description"`
}

type Issue struct {
	Title string `json:"title"`
	Body string `json:"body"`
}

func (con Connection) postRequest(c *gin.Context) {
	projectId, err := primitive.ObjectIDFromHex(c.Param("id"))
	var r Request
	if err := c.BindJSON(&r); err != nil {
		log.Printf("%v", err)
		return
	}

	userId := sessions.Default(c).Get("user-id")
	var user struct {
		ID primitive.ObjectID `bson:"_id, omitempty"`
	}
	err = con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if err != nil { 
		log.Printf("%v", err)
		return
	}
	r.Creator = user.ID
	r.Project = projectId
	if err != nil {
		log.Printf("%v", err)
		return
	}
	result, err := con.Requests.InsertOne(context.TODO(), r)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	selector := bson.M{"_id": projectId}
	update := bson.M{
		"$push": bson.M{"requests": result.InsertedID},
	}
	_, err = con.Projects.UpdateOne(context.TODO(), selector, update)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	// TODO: Create issue with GitHub API
	// var project Project
	// err = con.Projects.FindOne(context.TODO(), selector).Decode(&project)

	// f := Issue{
	// 	Title:  r.Name,
	// 	Body: "**[" + r.Type + "]** " + r.Description + "\n___\n**This request was created with Colabware.** For more information on claiming or contributing to the funds allocated for its development, view the original request [here]().",
	// }
	// data, err := json.Marshal(f)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// reader := bytes.NewReader(data)

	// res, err := client.Post("https://api.github.com/repos/colabware-ltd/test-project/issues", "application/vnd.github+json", reader)
	// res, err := client.Post("https://api.github.com/repos/" + project.GitHub.RepoOwner + "/" + project.GitHub.RepoName + "/issues", "application/vnd.github+json", reader)

	c.IndentedJSON(http.StatusCreated, gin.H{"_id": result.InsertedID})
}

func (con Connection) listRequests(c *gin.Context) {
	id,_ := primitive.ObjectIDFromHex(c.Param("name"))
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")
	limitInt, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	pageInt, err := strconv.ParseInt(page, 10, 64)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	selector := bson.M{"project": id}
	options := options.Find()
	options.SetProjection(bson.M{"name": 1, "type": 1, "description": 1, "bounty": 1, "_id": 0})
	options.SetLimit(limitInt)
	options.SetSkip(limitInt * (pageInt - 1))

	total, err := con.Requests.CountDocuments(context.TODO(), selector)
	filterCursor, err := con.Requests.Find(context.TODO(), selector, options)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	var requestsFiltered []bson.M
	err = filterCursor.All(context.TODO(), &requestsFiltered)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	c.IndentedJSON(http.StatusFound, gin.H{"total": total, "results": requestsFiltered} )
}

func (con Connection) postBounty(c *gin.Context) {
	projectId := c.Param("name")
	requestId,_ := primitive.ObjectIDFromHex(c.Param("id"))
	paymentIntent := c.Query("payment_intent")

	var b BountyContribution

	// Retrieve paymentIntent
	pi,_ := paymentintent.Get(paymentIntent, nil)
	if (pi.Status != "succeeded") {
		return
	}
	
	userId := sessions.Default(c).Get("user-id")
	var user struct {
		ID primitive.ObjectID `bson:"_id, omitempty"`
	}
	err := con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if err != nil { 
		log.Printf("%v", err)
		return
	}
	b.Creator = user.ID
	b.AmountReceived = float32(pi.AmountReceived)/100
	b.PaymentIntent = paymentIntent

	selector := bson.M{"_id": requestId}

	var r Request
	err = con.Requests.FindOne(context.TODO(), selector).Decode(&r)
	if err != nil { 
		log.Printf("%v", err)
		return
	}

	update := bson.M{
		"$push": bson.M{"bountycontributions": b},
		"$set" : bson.M{
			"bounty": r.Bounty + b.AmountReceived,
		},
	}
	_, err = con.Requests.UpdateOne(context.TODO(), selector, update)

	c.Redirect(http.StatusFound, "/project/" + projectId)
}

// func (con Connection) postRequestResponse(c *gin.Context) {
// 	project := c.Param("project")
// 	request := c.Param("request")

// 	var r Response
// 	if err := c.BindJSON(&r); err != nil {
// 		log.Printf("%v", err)
// 		return
// 	}

// 	userId := sessions.Default(c).Get("user-id")
// 	var user struct {
// 		ID primitive.ObjectID `bson:"_id, omitempty"`
// 	}
// 	e := con.Users.FindOne(context.TODO(), bson.M{"email": userId}).Decode(&user)
// 	if e != nil { 
// 		log.Printf("%v", e)
// 		return
// 	}
// 	r.Creator = user.ID

// 	selector := bson.M{"name": project}

// 	// Update 
// 	update := bson.M{
// 		"requests": bson.M{
// 			"$elemMatch": {}
// 		}

// 		"$push": bson.M{"votes": v},
// 	}

// 	_, err := con.Projects.UpdateOne(context.TODO(), selector, update)
// 	if err != nil {
// 		log.Printf("%v", err)
// 		return
// 	}
// }

// TODO: Search for request in project
// func (con Connection) postRequestVote(c *gin.Context) {
// 	project := c.Param("project")
// 	// request := c.Param("request")
// 	var v Vote

// 	userId := sessions.Default(c).Get("user-id")
// 	var user struct {
// 		ID primitive.ObjectID `bson:"_id, omitempty"`
// 	}
// 	e := con.Users.FindOne(context.TODO(), bson.M{"email": userId}).Decode(&user)
// 	if e != nil { 
// 		log.Printf("%v", e)
// 		return
// 	}
// 	v.Creator = user.ID

// 	// TODO: Lookup tokens held by voter on contract

// 	selector := bson.M{"name": project}
// 	update := bson.M{
// 		"$push": bson.M{"votes": v},
// 	}
// 	_, err := con.Projects.UpdateOne(context.TODO(), selector, update)
// 	if err != nil {
// 		log.Printf("%v", err)
// 		return
// 	}
// 	c.IndentedJSON(http.StatusCreated, v)
// }