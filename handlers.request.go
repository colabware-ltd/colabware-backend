package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/procyon-projects/chrono"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/refund"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Request struct {
	Created           primitive.DateTime   `json:"created" bson:"created,omitempty"`
	CreatorId         primitive.ObjectID   `json:"creator_id" bson:"creator_id,omitempty"`
	CreatorName       string               `json:"creator_name" bson:"creator_name,omitempty"`
	Project           primitive.ObjectID   `json:"project" bson:"project,omitempty"`
	Name              string               `json:"name" bson:"name,omitempty"`
	Description       string               `json:"description" bson:"description,omitempty"`
	Expiry            string               `json:"expiry" bson:"expiry,omitempty"`
	Categories        []string             `json:"categories" bson:"categories,omitempty"`
	Contributions     []primitive.ObjectID `json:"contributions" bson:"contributions,omitempty"`
	ContributionTotal float32              `json:"contribution_total" bson:"contribution_total,omitempty"`
	ProjectVotes      []ProjectVote        `json:"project_votes" bson:"project_votes,omitempty"`
	Proposals         []primitive.ObjectID `json:"proposals" bson:"proposals,omitempty"`
	ProposalMerged    primitive.ObjectID   `json:"proposal_merged" bson:"proposal_merged,omitempty"`
	GithubIssue       uint64               `json:"github_issue" bson:"github_issue,omitempty"`
	Status            string               `json:"status" bson:"status,omitempty"`
}

type ProjectVote struct {
	CreatorId   primitive.ObjectID `json:"creator_id" bson:"creator_id,omitempty"`
	CreatorName string             `json:"creator_name" bson:"creator_name,omitempty"`
	TokensHeld  int64              `json:"tokens" bson:"tokens,omitempty"`
}

type Issue struct {
	Title string `json:"title" bson:"title,omitempty"`
	Body  string `json:"body" bson:"body,omitempty"`
}

func (con Connection) postRequest(c *gin.Context) {
	projectId, err := primitive.ObjectIDFromHex(c.Param("project"))
	var r Request
	if err := c.BindJSON(&r); err != nil {
		log.Printf("%v", err)
		return
	}

	userId := sessions.Default(c).Get("user-id")
	var user struct {
		ID    primitive.ObjectID `bson:"_id, omitempty"`
		Login string             `bson:"login"`
	}
	err = con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	log.Println(user.Login)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	r.CreatorId = user.ID
	r.CreatorName = user.Login
	r.Project = projectId
	r.Created = primitive.NewDateTimeFromTime(time.Now())

	// TODO: Create issue with GitHub API
	var project Project
	projectSelector := bson.M{"_id": projectId}
	options := options.FindOne().SetProjection(bson.M{"github": 1})
	err = con.Projects.FindOne(context.TODO(), projectSelector, options).Decode(&project)
	if err != nil {
		log.Fatal(err)
	}

	f := Issue{
		Title: r.Name,
		Body:  "**[" + r.Categories[0] + "]** " + r.Description + "\n___\n**This request was created with Colabware.** For more information on claiming or contributing to the funds allocated for its development, view the original request [here]().",
	}
	data, err := json.Marshal(f)
	if err != nil {
		log.Fatal(err)
	}
	reader := bytes.NewReader(data)
	log.Println(reader)

	var resTarget struct {
		Number uint64 `bson:"number"`
	}
	res, err := client.Post("https://api.github.com/repos/"+project.GitHub.RepoOwner+"/"+project.GitHub.RepoName+"/issues", "application/vnd.github+json", reader)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&resTarget)

	r.GithubIssue = resTarget.Number
	log.Println(resTarget.Number)

	result, err := con.Requests.InsertOne(context.TODO(), r)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	projectUpdate := bson.M{
		"$push": bson.M{"requests": result.InsertedID},
	}
	_, err = con.Projects.UpdateOne(context.TODO(), projectSelector, projectUpdate)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	c.IndentedJSON(http.StatusCreated, gin.H{"_id": result.InsertedID})
	con.handleExpiry(r.Expiry, result.InsertedID.(primitive.ObjectID))
}

func (con Connection) handleExpiry(expiry string, requestId primitive.ObjectID) {
	// Handle actions on expiry
	taskScheduler := chrono.NewDefaultTaskScheduler()
	layout := "2006-01-02T15:04:05.000Z"
	t, err := time.Parse(layout, expiry)
	if err != nil {
		log.Println(err)
	}

	_, err = taskScheduler.Schedule(func(ctx context.Context) {
		var request Request
		var requestUpdate bson.M
		requestSelector := bson.M{
			"_id": requestId,
		}

		// Get request using request ID
		err = con.Requests.FindOne(context.TODO(), requestSelector).Decode(&request)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		// If no proposals submitted, refund contributors
		if len(request.Proposals) == 0 || request.Proposals == nil {

			// Find all contributions for request to refund
			filterCursor, err := con.Contributions.Find(context.TODO(), bson.M{"request_id": requestId})
			if err != nil {
				log.Printf("%v", err)
				return
			}
			var contributionsFiltered []Contribution
			err = filterCursor.All(context.TODO(), &contributionsFiltered)
			if err != nil {
				log.Printf("%v", err)
				return
			}

			// TODO: Add transaction collection in DB
			for _, contribution := range contributionsFiltered {
				for _, transaction := range contribution.Transactions {
					params := &stripe.RefundParams{
						PaymentIntent: &transaction,
					}
					_, err := refund.New(params)
					if err != nil {
						log.Printf("%v", err)
						return
					}
				}
			}
			requestUpdate = bson.M{
				"$set": bson.M{
					"status": "expired",
				},
			}
			_, err = con.Requests.UpdateOne(context.TODO(), bson.M{"_id": requestId}, requestUpdate)
			if err != nil {
				log.Printf("%v", err)
				return
			}
		}
	}, chrono.WithTime(t))
	if err != nil {
		log.Printf("%v", err)
		return
	}
}

func (con Connection) getRequests(c *gin.Context) {
	id,_ := primitive.ObjectIDFromHex(c.Param("project"))
	status := c.DefaultQuery("status", "open")
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
	selector := bson.M{"project": id, "status": status}
	options := options.Find()
	// options.SetProjection(bson.M{"name": 1, "categories": 1, "description": 1, "bounty": 1, "created": 1, "_id": 0})
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
	c.IndentedJSON(http.StatusFound, gin.H{"total": total, "results": requestsFiltered})
}
