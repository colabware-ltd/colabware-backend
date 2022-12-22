package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/transfer"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Proposal struct {
	CreatorId         primitive.ObjectID `json:"creator_id" bson:"creator_id,omitempty"`
	CreatorName       string             `json:"creator_name" bson:"creator_name,omitempty"`
	RequestId         primitive.ObjectID `json:"request_id" bson:"request_id,omitempty"`
	Repository        string             `json:"repository" bson:"repository,omitempty"`
	ContributionTotal float32            `json:"contribution_total" bson:"contribution_total"`
	PaymentStatus     string             `json:"payment_status" bson:"payment_status,omitempty"`
	Transactions      []string           `json:"transactions" bson:"transactions,omitempty"`
	PullRequest       PullRequest        `json:"pull_request" bson:"pull_request,omitempty"`
	PullRequestNumber uint64             `json:"pull_request_number" bson:"pull_request_number,omitempty"`
	PullRequestRepo   string             `json:"pull_request_repo" bson:"pull_request_repo,omitempty"`
}

type ProposalVote struct {
	CreatorId   primitive.ObjectID `json:"creator_id" bson:"creator_id,omitempty"`
	CreatorName string             `json:"creator_name" bson:"creator_name,omitempty"`
	TokensHeld  int64              `json:"tokens" bson:"tokens,omitempty"`
}

type PullRequest struct {
	Title string `json:"title" bson:"title,omitempty"`
	Body  string `json:"body" bson:"body,omitempty"`
	Head  string `json:"head" bson:"head,omitempty"`
	Base  string `json:"base" bson:"base,omitempty"`
}

type GitHubError struct {
	Resource string `json:"resource" bson:"resource,omitempty"`
	Code     string `json:"code" bson:"code,omitempty"`
	Message  string `json:"message" bson:"message,omitempty"`
}

type GitHubResMergePR struct {
	Number  uint64        `json:"number" bson:"number,omitempty"`
	Message string        `json:"message" bson:"message,omitempty"`
	Errors  []GitHubError `json:"errors" bson:"errors,omitempty"`
}

func (con Connection) checkRequestApproved(id primitive.ObjectID) (bool, error) {
	request, err := con.getRequestById(id)
	if err != nil {
		log.Printf("%v", err)
		return false, fmt.Errorf("%v", err)
	}
	return request.Approved, nil
}

func (con Connection) postProposal(c *gin.Context) {
	requestId, _ := primitive.ObjectIDFromHex(c.Param("request"))

	// Check if request has been approved by token holders
	request, err := con.getRequestById(requestId)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// Stop creation of proposal if request not approved
	if !request.Approved {
		c.IndentedJSON(http.StatusForbidden, nil)
		return
	}

	var proposal Proposal
	if err := c.BindJSON(&proposal); err != nil {
		log.Printf("%v", err)
		return
	}

	// Retrieve User ID
	userId := sessions.Default(c).Get("user-id")
	var user struct {
		ID    primitive.ObjectID `bson:"_id, omitempty"`
		Login string             `bson:"login"`
	}
	err = con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	proposal.CreatorId = user.ID
	proposal.CreatorName = user.Login
	proposal.RequestId = requestId
	proposal.ContributionTotal = 0

	data, err := json.Marshal(proposal.PullRequest)
	if err != nil {
		log.Fatal(err)
	}
	reader := bytes.NewReader(data)

	var resTarget struct {
		Number  uint64        `json:"number" bson:"number,omitempty"`
		Message string        `json:"message" bson:"message,omitempty"`
		Errors  []GitHubError `json:"errors" bson:"errors,omitempty"`
	}

	res, err := client.Post("https://api.github.com/repos/"+proposal.Repository+"/pulls", "application/vnd.github+json", reader)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&resTarget)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	// Create proposal in DB if pull request successfully created
	if res.StatusCode == 201 {
		proposal.PullRequestNumber = resTarget.Number
		result, err := con.Proposals.InsertOne(context.TODO(), proposal)
		if err != nil {
			log.Printf("%v", err)
			return
		}

		// Create response in DB once PR is created
		requestSelector := bson.M{"_id": requestId}
		requestUpdate := bson.M{
			"$push": bson.M{"proposals": result.InsertedID},
			"$set": bson.M{
				"status": "pending",
			},
		}
		_, err = con.Requests.UpdateOne(context.TODO(), requestSelector, requestUpdate)
		if err != nil {
			log.Printf("%v", err)
			return
		}
		c.IndentedJSON(http.StatusCreated, gin.H{})

	} else {
		c.IndentedJSON(http.StatusUnprocessableEntity, resTarget)
	}

}

func (con Connection) getProposals(c *gin.Context) {
	requestId, _ := primitive.ObjectIDFromHex(c.Param("request"))
	proposalSelector := bson.M{"request_id": requestId}
	filterCursor, err := con.Proposals.Find(context.TODO(), proposalSelector)
	var proposalsFiltered []bson.M
	err = filterCursor.All(context.TODO(), &proposalsFiltered)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	c.IndentedJSON(http.StatusFound, proposalsFiltered)
}

func (con Connection) postSelectedProposal(c *gin.Context) {
	proposalId, _ := primitive.ObjectIDFromHex(c.Param("proposal"))
	requestId, _ := primitive.ObjectIDFromHex(c.Param("request"))
	userId := sessions.Default(c).Get("user-id")

	var contribution Contribution

	contributionSelector := bson.M{
		"request_id":   requestId,
		"creator_name": userId,
	}
	err := con.Contributions.FindOne(context.TODO(), contributionSelector).Decode(&contribution)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// Remove contribution allocated to previously selected proposal
	prevProposalId := contribution.SelectedProposal
	proposalSelector := bson.M{
		"_id": prevProposalId,
	}
	proposalUpdate := bson.M{
		"$inc": bson.M{
			"contribution_total": -contribution.AmountReceived,
		},
	}
	_, err = con.Proposals.UpdateOne(context.TODO(), proposalSelector, proposalUpdate)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	// Add contribution allocated to newly selected proposal
	proposalSelector = bson.M{
		"_id": proposalId,
	}
	proposalUpdate = bson.M{
		"$inc": bson.M{
			"contribution_total": contribution.AmountReceived,
		},
	}
	_, err = con.Proposals.UpdateOne(context.TODO(), proposalSelector, proposalUpdate)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	// Update contribution with selected proposal
	contributionUpdate := bson.M{
		"$set": bson.M{
			"selected_proposal": proposalId,
		},
	}
	_, err = con.Contributions.UpdateMany(context.TODO(), contributionSelector, contributionUpdate)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	c.IndentedJSON(http.StatusCreated, bson.M{})
}

func (con Connection) getProposalById(id primitive.ObjectID) (*Proposal, error) {
	var proposal Proposal
	err = con.Proposals.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&proposal)
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}
	return &proposal, nil
}

// TODO: Complete request; pay out submission amounts.
func (con Connection) mergeProposal(c *gin.Context) {
	requestId, _ := primitive.ObjectIDFromHex(c.Param("request"))
	proposalId, _ := primitive.ObjectIDFromHex(c.Param("proposal"))
	userId := sessions.Default(c).Get("user-id")

	var user User
	err = con.Users.FindOne(context.TODO(), bson.M{"login": userId}).Decode(&user)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	request, err := con.getRequestById(requestId)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// Check if user is maintainer
	maintainer, _, err := con.isMaintainer(user.ID, request.Project)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	if !maintainer {
		c.IndentedJSON(http.StatusUnauthorized, nil)
		return
	}

	proposal, err := con.getProposalById(proposalId)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	// TODO: Handle incomplete PR from GitHub (other than 201 response)
	_, status, err := mergePullRequest(proposal)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}
	if status != 201 {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	requestUpdate := bson.M{
		"$set": bson.M{
			"proposal_merged": proposalId,
			"status":          "closed",
		},
	}
	_, err = con.Requests.UpdateOne(context.TODO(), bson.M{"_id": requestId}, requestUpdate)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	// Process disbursement of allocated request funds
	err = con.processDisbursement(request.Proposals)
	if err != nil {
		log.Printf("%v", err)
		c.IndentedJSON(http.StatusInternalServerError, nil)
		return
	}

	c.IndentedJSON(http.StatusCreated, bson.M{})
}

func (con Connection) processDisbursement(proposals []primitive.ObjectID) (error) {
	// TODO: Disburse unassigned funds to winning proposal
	for _, p := range proposals {
		proposal, err := con.getProposalById(p)
		if err != nil {
			log.Printf("%v", err)
			continue
		}

		// Check that proposal creator has connected their Stripe account
		user, err := con.getUserById(proposal.CreatorId)
		if err != nil {
			log.Printf("%v", err)
			continue
		}

		params := &stripe.TransferParams{
			Amount:      stripe.Int64(int64(proposal.ContributionTotal) * 100),
			Currency:    stripe.String(string(stripe.CurrencyUSD)),
			Destination: stripe.String(user.StripeAccount.AccountID),
		}

		tf, err := transfer.New(params)
		if err != nil {
			log.Printf("%v", err)
			continue
		} else {
			proposalUpdate := bson.M{
				"$push": bson.M{"transactions": tf.ID},
				"$set": bson.M{
					"payment_status": "complete",
				},
			}
			_, err = con.Proposals.UpdateOne(context.TODO(), bson.M{"_id": p}, proposalUpdate)
			if err != nil {
				log.Printf("%v", err)
				return fmt.Errorf("%v", err)
			}
		}
	}
	return nil
}

func mergePullRequest(proposal *Proposal) (*GitHubResMergePR, int, error) {
	payload, err := json.Marshal(map[string]interface{}{})
	if err != nil {
		log.Printf("%v", err)
		return nil, -1, fmt.Errorf("%v", err)
	}

	var resBody GitHubResMergePR

	url := "https://api.github.com/repos/" + proposal.Repository + "/pulls/" + strconv.FormatUint(proposal.PullRequestNumber, 10) + "/merge"

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("%v", err)
		return nil, -1, fmt.Errorf("%v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	res, err := client.Do(req)
	if err != nil {
		log.Printf("%v", err)
		return nil, -1, fmt.Errorf("%v", err)
	}

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&resBody)
	if err != nil {
		log.Printf("%v", err)
		return nil, -1, fmt.Errorf("%v", err)
	}

	return &resBody, res.StatusCode, nil
}
