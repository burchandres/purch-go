package api

import (
	// "database/sql"
	"log/slog"
	"net/http"
	// "errors"

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
		protected.PUT("/update", updateUser)
		protected.GET("/link-token", getLinkToken)
		// protected.POST("/exchange-public-token", exchangePublicToken)
		protected.DELETE("/delete", deleteUser)
	}
}

func registerUser(c *gin.Context) {
	// Implement user registration logic here
	var newUser database.User

	if err := c.BindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// verify user doesn't already exist
	_, err := database.GetUserByUsername(c.Request.Context(), newUser.Username)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user with this username already exists"})
		return
	}
	// hash the password before pushing to postgres
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password.", "error", err, "endpoint", "/api/user/register")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	newUser.Password = string(hashedPassword)

	// store user
	err = database.StoreUser(c.Request.Context(), newUser)
	if err != nil {
		slog.Error("failed to store user.", "error", err, "endpoint", "/api/user/register")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, newUser)
}

func setUserCookie(c *gin.Context) {
	// get user from db with provided username and password
	username := c.Query("username")
	slog.Info("setting cookie for user.", "username", username)
	user, err := database.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		slog.Error("user with provided username does not exist.", "error", err, "endpoint", "/api/user/set")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// verify provided password
	password := c.Query("password")
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		slog.Error("incorrect password provided.", "error", err, "endpoint", "/api/user/set")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password provided."})
		return
	}
	// create jwt token for user to set in cookie
	token, err := createToken(user.ID)
	if err != nil {
		slog.Error("failed to create token.", "error", err, "endpoint", "/api/user/set")
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
	// get user information from db
	user, _ := c.Get("user")
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

// TODO: call `/item/remove` Plaid endpoint to cancel associated access tokens to prevent unnecessary billing
func deleteUser(c *gin.Context) {
	// get user id from context
	user, _ := c.Get("user")
	// delete user from db
	if err := database.DeleteUser(c.Request.Context(), user.(database.User)); err != nil {
		slog.Error("failed to delete user.", "error", err, "userID", user.(database.User).ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

func updateUser(c *gin.Context) {
	user, _ := c.Get("user")
	// get update params
	var updateParams database.UpdateUserParams
	// get update params from request body
	if err := c.BindJSON(&updateParams); err != nil {
		slog.Error("error binding update params.", "error", err.Error(), "endpoint", "/api/user/update")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.UpdateUser(c.Request.Context(), user.(database.User).ID, updateParams); err != nil {
		slog.Error("error updating user.", "error", err.Error(), "userID")
	}
	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func getLinkToken(c *gin.Context) {
	user, _ := c.Get("user")
	plaidClient := utils.GetPlaidClient()
	config := utils.GetConfig()

	requestUser := plaid.NewLinkTokenCreateRequestUser(user.(database.User).ID)

	request := plaid.NewLinkTokenCreateRequest(
		"Purch",
		"en",
		config.GetPlaidCountryCodes(),
	)

	request.SetUser(*requestUser)
	request.SetProducts(config.GetPlaidProducts())
	request.SetRedirectUri(config.PlaidRedirectUri)

	resp, _, err := plaidClient.PlaidApi.LinkTokenCreate(c.Request.Context()).LinkTokenCreateRequest(*request).Execute()

	if err != nil {
		slog.Error("failed to create link token.", "error", err, "endpoint", "/api/user/link-token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK,
		gin.H{
			"LinkToken": resp.GetLinkToken(),
			"ExpiresAt": resp.GetExpiration(),
		},
	)
}

func exchangePublicToken(c *gin.Context) {
	user, _ := c.Get("user")
	plaidClient := utils.GetPlaidClient()

	publicToken := c.Query("public_token")
	if publicToken == "" {
		slog.Error("public token not found in query params.", "endpoint", "/api/user/exchange-public-token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "public token not provided in query params as public_token"})
		return
	}

	request := plaid.NewItemPublicTokenExchangeRequest(publicToken)

	resp, _, err := plaidClient.PlaidApi.ItemPublicTokenExchange(c.Request.Context()).ItemPublicTokenExchangeRequest(*request).Execute()

	if err != nil {
		slog.Error("failed to exchange public token for access token.", "error", err, "endpoint", "/api/user/exchange-public-token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	accessToken, ok := resp.GetAccessTokenOk()
	if !ok {
		slog.Error("no access token present after exchanging with public token.", "userID", user.(database.User).ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no access token provided"})
		return
	}

	itemID, ok := resp.GetItemIdOk()
	if !ok {
		slog.Error("no item ID present after getting access token.")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no item ID present after getting access token"})
		return
	}

	// in separate goroutine run item->accounts->transactions pipeline
	// and log whether it was successful or not
	go func() {
		if err := tasks.SyncItemAccountsTransactionsPipeline(
			c.Request.Context(),
			user.(database.User).ID,
			*itemID,
			*accessToken,
		); err != nil {
			slog.Error("error with item->accounts->transactions initial sync pipeline.", "error", err.Error())
		} else {
			slog.Info("successfully synced all accounts and transactions.", "itemID", *itemID, "userID", user.(database.User).ID)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Bank linked with Purch, syncing information..."})
}
