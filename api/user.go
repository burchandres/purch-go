package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"purch/database"
)

func SetupUserEndpoints(r *gin.Engine) {
	r.POST("/user/register", registerUser)
	r.GET("/user/login", setUserCookie)

	protected := r.Group("/user")
	protected.Use(authMiddleware())
	{
		protected.GET("/info", getUserInfo)
		// protected.GET("/verify-auth", verifyAuth)
		protected.GET("/logout", logout)
		// protected.POST("/update", updateUser)
		// protected.GET("/link-token", getLinkToken)
		// protected.POST("/exchange-public-token", exchangePublicToken)
		// protected.DELETE("/delete", deleteUser)
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
	// hash the password before pushing to postgres
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(storeUserParams.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	storeUserParams.Password = string(hashedPassword)

	registeredUser, err := queries.StoreUser(c.Request.Context(), storeUserParams)
	if err != nil {
		slog.Error("failed to store user", "error", err)
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
		slog.Error("user with provided username does not exist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// verify provided password
	password := c.Query("password")
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		slog.Error("failed to verify password", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// create jwt token for user to set in cookie
	token, err := createToken(user.ID)
	if err != nil {
		slog.Error("failed to create token", "error", err)
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
		slog.Error("user id not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user id not found in context"})
		return
	}
	user, err := queries.GetUserById(c.Request.Context(), userID.(int64))
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
