package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v40/plaid"
	"golang.org/x/crypto/bcrypt"

	"purch/database"
	"purch/tasks"
	"purch/utils"
)

func SetupUserEndpoints(r *gin.Engine) {
	r.POST("/user/register", registerUser)
	r.GET("/user/login", setUserCookie)

	protected := r.Group("/user")
	protected.Use(authMiddleware())
	{
		protected.GET("/info", getUserInfo)
		protected.GET("/logout", logout)
		protected.POST("/update", updateUser)
		protected.GET("/link-token", getLinkToken)
		protected.POST("/exchange-public-token", exchangePublicToken)
		protected.DELETE("/delete", deleteUser)
	}
}

func registerUser(c *gin.Context) {
	db := database.GetPool()
	queries := database.New(db)
	// Implement user registration logic here
	var storeUserParams database.StoreUserParams

	if err := c.BindJSON(&storeUserParams); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// verify user doesn't already exist
	_, err := queries.GetUserByUsername(c.Request.Context(), storeUserParams.Username)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "user with this username already exists"})
		return
	}
	// hash the password before pushing to postgres
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(storeUserParams.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "error", err, "endpoint", "/api/user/register")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	storeUserParams.Password = string(hashedPassword)

	registeredUser, err := queries.StoreUser(c.Request.Context(), storeUserParams)
	if err != nil {
		slog.Error("failed to store user", "error", err, "endpoint", "/api/user/register")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, registeredUser)
}

func setUserCookie(c *gin.Context) {
	db := database.GetPool()
	queries := database.New(db)
	// get user from db with provided username and password
	username := c.Query("username")
	slog.Info("setting cookie for user", "username", username)
	user, err := queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		slog.Error("user with provided username does not exist", "error", err, "endpoint", "/api/user/set")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// verify provided password
	password := c.Query("password")
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		slog.Error("failed to verify password", "error", err, "endpoint", "/api/user/set")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// create jwt token for user to set in cookie
	token, err := createToken(user.ID)
	if err != nil {
		slog.Error("failed to create token", "error", err, "endpoint", "/api/user/set")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// set cookie for user
	c.SetCookie(
		"purch_token",
		token,
		1800,
		"/",
		"localhost",
		false,
		true,
	)
	c.JSON(http.StatusOK, gin.H{"message": "user cookie set"})
}

func getUserInfo(c *gin.Context) {
	db := database.GetPool()
	queries := database.New(db)
	// get user information from db
	userID, exists := c.Get("user_id")
	if !exists {
		slog.Error("user id not found in context", "endpoint", "/api/user/get")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user id not found in context"})
		return
	}
	user, err := queries.GetUserById(c.Request.Context(), userID.(string))
	if err != nil {
		slog.Error("failed to get user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func logout(c *gin.Context) {
	// delete cookie for user
	c.SetCookie(
		"purch_token",
		"",
		-1,
		"/",
		"localhost",
		false,
		true,
	)
	c.JSON(http.StatusOK, gin.H{"message": "logout successful, cookie cleared"})
}

func deleteUser(c *gin.Context) {
	db := database.GetPool()
	queries := database.New(db)
	// get user id from context
	userID, exists := c.Get("user_id")
	if !exists {
		slog.Error("user id not found in context", "endpoint", "/api/user/delete")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user id not found in context"})
		return
	}
	// TODO: call `/item/remove` to cancel associated access tokens to prevent unnecessary billing
	// delete user from db
	if err := queries.DeleteUser(c.Request.Context(), userID.(string)); err != nil {
		slog.Error("failed to delete user", "error", err, "endpoint", "/api/user/delete")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

func updateUser(c *gin.Context) {
	db := database.GetPool()
	queries := database.New(db)
	userID, exists := c.Get("user_id")
	if !exists {
		slog.Error("user id not found in context", "endpoint", "/api/user/update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user id not found in context"})
		return
	}
	user, err := queries.GetUserById(c.Request.Context(), userID.(string))
	if err != nil {
		slog.Error("failed to get user", "error", err, "endpoint", "/api/user/update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// get update params
	var updateParams database.UpdateUserParams
	// get update params from request body
	if err := c.BindJSON(&updateParams); err != nil {
		slog.Error("failed to bind update params", "error", err, "endpoint", "/api/user/update")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// hash password if changed
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(updateParams.Password)); err != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updateParams.Password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("failed to generate password hash for new password", "error", err, "endpoint", "/api/user/update")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		updateParams.Password = string(hashedPassword)
	}
	// update user
	user, err = queries.UpdateUser(c.Request.Context(), updateParams)
	if err != nil {
		slog.Error("failed to update user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user updated", "user": user})
}

func getLinkToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		slog.Error("user id not found in context", "endpoint", "/api/user/link-token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user id not found in context"})
		return
	}
	plaidClient := utils.GetPlaidClient()
	config := utils.GetConfig()
	
	user := plaid.NewLinkTokenCreateRequestUser(userID.(string))
	
	request := plaid.NewLinkTokenCreateRequest(
		"Purch",
		"en",
		config.GetPlaidCountryCodes(),
	)
	
	request.SetUser(*user)
	request.SetProducts(config.GetPlaidProducts())
	request.SetRedirectUri(config.PlaidRedirectUri)
	
	resp, _, err := plaidClient.PlaidApi.LinkTokenCreate(c.Request.Context()).LinkTokenCreateRequest(*request).Execute()
	
	if err != nil {
		slog.Error("failed to create link token", "error", err, "endpoint", "/api/user/link-token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, 
		gin.H{
			"link_token": resp.GetLinkToken(), 
			"expires_at": resp.GetExpiration(),
		},
	)
}

func exchangePublicToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		slog.Error("user not authenticated", "endpoint", "/api/user/exchange-public-token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user id not found in context"})
		return
	}
	plaidClient := utils.GetPlaidClient()
	
	publicToken := c.Query("public_token")
	if publicToken == "" {
		slog.Error("public token not found in query params", "endpoint", "/api/user/exchange-public-token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "public token not provided in query params as public_token"})
		return
	}
	
	request := plaid.NewItemPublicTokenExchangeRequest(publicToken)
	
	resp, _, err := plaidClient.PlaidApi.ItemPublicTokenExchange(c.Request.Context()).ItemPublicTokenExchangeRequest(*request).Execute()
	
	if err != nil {
		slog.Error("failed to exchange public token for access token", "error", err, "endpoint", "/api/user/exchange-public-token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	accessToken, ok := resp.GetAccessTokenOk()
	if !ok {
		slog.Error("no access token present after exchanging with public token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no access token provided"})
		return
	}

	itemID, ok := resp.GetItemIdOk()
	if !ok {
		slog.Error("no item ID present after getting access token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no item ID present after getting access token"})
		return
	}
	
	// in separate goroutine run item->accounts->transactions pipeline
	// and log whether it was successful or not
	go func() {
		if err := tasks.SyncItemAccountsTransactionsPipeline(
			c.Request.Context(),
			userID.(string),
			*itemID,
			*accessToken,
		); err != nil {
			slog.Error("error with item->accounts->transactions initial sync pipeline", "error", err.Error())
		} else {
			slog.Info("successfully synced all accounts and transactions", "itemID", *itemID, "userID", userID)
		}
	}()
	
	c.JSON(http.StatusOK, gin.H{"message": "Bank linked with Purch, syncing information..."})
}
