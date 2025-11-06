package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v40/plaid"
	"golang.org/x/crypto/bcrypt"

	"purch/internal/database"
	"purch/internal/tasks"
	"purch/internal/utils"
	"purch/internal/config"
)

func SetupUserEndpoints(r *gin.Engine) {
	r.POST("/user/register", registerUser)
	r.POST("/user/login", setUserCookie)

	protected := r.Group("/user")
	protected.Use(authMiddleware())
	{
		protected.GET("/info", getUserInfo)
		protected.POST("/logout", logout)
		protected.PUT("/update", updateUser)
		protected.DELETE("/delete", deleteUser)
		protected.GET("/link-token", getLinkToken)
		protected.POST("/exchange-public-token", exchangePublicToken)
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
		slog.Error("failed to hash password.", "error", err.Error(), "endpoint", "/api/user/register")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	newUser.Password = string(hashedPassword)

	// store user
	err = database.StoreUser(c.Request.Context(), newUser)
	if err != nil {
		slog.Error("failed to store user.", "error", err, "endpoint", "/api/user/register")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store user"})
		return
	}
	c.JSON(http.StatusCreated, newUser)
}

func setUserCookie(c *gin.Context) {
	// pull provided user credentials for verifying login
	var credentials struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.BindJSON(&credentials); err != nil {
		slog.Error("error pulling user credentials from body", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "ill structured user credentials provided"})
		return
	}
	// get user from db with provided username and password
	user, err := database.GetUserByUsername(c.Request.Context(), credentials.Username)
	if err != nil {
		slog.Error("user with provided username does not exist.", "error", err.Error(), "username", credentials.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user with provided username does not exist"})
		return
	}
	userID := user.ID.String()
	// verify provided password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		slog.Error("incorrect password provided.", "error", err.Error(), "userID", userID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password provided."})
		return
	}
	// create jwt token for user to set in cookie
	slog.Debug("setting cookie for user.", "username", credentials.Username)
	token, err := createToken(userID)
	if err != nil {
		slog.Error("failed to create token.", "error", err.Error(), "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token."})
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
	slog.Debug("user logged in successfully", "userID", userID)
	c.JSON(http.StatusOK, gin.H{"message": "user cookie set"})
}

func getUserInfo(c *gin.Context) {
	// get user information from db
	user, _ := c.Get("user")
	c.JSON(http.StatusOK, user)
}

func logout(c *gin.Context) {
	// delete cookie for user
	deleteCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logout successful, cookie cleared"})
}

func deleteCookie(c *gin.Context) {
	c.SetCookie(
		"purch_token",
		"",
		-1,
		"/",
		"localhost",
		false,
		true,
	)
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
	deleteCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "user deleted and cookie session cleared"})
}

func updateUser(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(database.User).ID
	// get update params
	var updateParams database.UpdateUserParams
	// get update params from request body
	if err := c.BindJSON(&updateParams); err != nil {
		slog.Error("error binding update params.", "error", err.Error(), "userID", userID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "error reading update params"})
		return
	}
	if err := database.UpdateUser(c.Request.Context(), userID, updateParams); err != nil {
		slog.Error("error updating user.", "error", err.Error(), "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error updating user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user updated successfully"})
}

func getLinkToken(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(database.User).ID.String()
	plaidClient := utils.GetPlaidClient()
	config := config.GetConfig()

	requestUser := plaid.NewLinkTokenCreateRequestUser(userID)

	request := plaid.NewLinkTokenCreateRequest(
		"Purch",
		"en",
		config.GetPlaidCountryCodes(),
	)

	request.SetUser(*requestUser)
	request.SetProducts(config.GetPlaidProducts())
	request.SetRedirectUri(config.PlaidRedirectUri)
	// request.SetWebhook()

	resp, _, err := plaidClient.PlaidApi.LinkTokenCreate(c.Request.Context()).LinkTokenCreateRequest(*request).Execute()

	if err != nil {
		slog.Error("failed to create link token.", "error", err, "endpoint", "/api/user/link-token")
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
	user, _ := c.Get("user")
	userID := user.(database.User).ID
	plaidClient := utils.GetPlaidClient()

	publicToken := c.Query("public_token")
	if publicToken == "" {
		slog.Error("public token not found in query params.", "userID", userID)
		c.JSON(http.StatusBadRequest, "public token not provided in query params as public_token")
		return
	}

	request := plaid.NewItemPublicTokenExchangeRequest(publicToken)

	resp, _, err := plaidClient.PlaidApi.ItemPublicTokenExchange(c.Request.Context()).ItemPublicTokenExchangeRequest(*request).Execute()

	if err != nil {
		slog.Error("failed to exchange public token for access token.", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange public token for access token"})
		return
	}

	accessToken, ok := resp.GetAccessTokenOk()
	if !ok {
		slog.Error("no access token present after exchanging with public token.", "userID", userID)
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
			userID,
			*itemID,
			*accessToken,
		); err != nil {
			slog.Error("error with item->accounts->transactions initial sync pipeline.", "error", err.Error(), "userID", userID)
		} else {
			slog.Info("successfully synced all accounts and transactions.", "itemID", *itemID, "userID", userID)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Bank linked with Purch, syncing information..."})
}
