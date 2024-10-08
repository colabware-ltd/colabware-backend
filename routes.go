package main

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Middleware for authorizing request
func authorizeRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		v := session.Get("user-id")
		if v == nil {
			c.IndentedJSON(http.StatusUnauthorized, nil)
			c.Abort()
		}
		c.Next()
	}
}

func initializeRoutes(db Connection) {
	// Handle the index route
	authorized := router.Group("/api/user")
	authorized.Use(authorizeRequest())
	{
		authorized.POST("/project", db.postProject)
		authorized.POST("/project/:project/request", db.postRequest)
		authorized.GET("/project/:project/request/:request/contribution", db.postContribution)
		authorized.GET("/project/branches/:owner/:repo", getProjectBranches)
		authorized.POST("/request/:request/proposal", db.postProposal)
		authorized.GET("/request/:request/proposal/:proposal/select", db.postSelectedProposal)
		authorized.GET("/request/:request/approve", db.approveRequest)
		authorized.GET("/request/list", db.getUserRequests)
		authorized.GET("/token/list", db.getUserTokens)
		authorized.GET("/logout", db.logout)
		authorized.GET("/", db.getUser)
		authorized.GET("/stripe", db.stripeAccountLink)
		authorized.GET("/stripe/verify", db.stripeVerify)
		authorized.POST("/update", db.updateUserDetails)
		// authorized.POST("/token-payment", db.createTokenPayment)
		authorized.POST("/purchase-token", db.purchaseToken)
		authorized.POST("/payment-intent", db.createPaymentIntent)
		authorized.GET("/project/:project/tokens", db.getTokenHolding)
		authorized.GET("/request/:request/proposal/:proposal/merge", db.mergeProposal)
	}
	router.GET("/api/project/:project", db.getProject)
	router.GET("/api/project/list", db.getProjects)
	router.GET("/api/project/:project/request/list", db.getProjectRequests)
	router.GET("/api/token/:token/balance/:wallet", db.getBalance)
	router.GET("/api/project/:project/balances", db.getTokenBalanceOverview)
	router.GET("/api/request/:request/proposals", db.getProposals)
	router.GET("/api/request/:request/contributions", db.getContributions)
	router.GET("/api/request/:request/approvers", db.getRequestApprovers)
	router.POST("/api/createwallet", db.postWallet)
	router.POST("/api/transfer-test", db.transferTest)
	router.GET("/api/login", loginHandler)
	router.GET("/api/auth", db.authHandler)
}
