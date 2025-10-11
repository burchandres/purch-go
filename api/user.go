package api

import (
	"log/slog"
	"net/http"
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"purch/database"
)

func SetupUserEndpoints(r *gin.Engine) {
	r.POST("/user/register", registerUser)
	r.GET("/user/login", setUserCookie)

	// group := r.Group("/user")
	// group.Use(AuthMiddleware)
	// group.GET("/info", getUserInfo)
	// group.GET("/verify-auth", verifyAuth)
	// group.GET("/logout", logout)
	// group.POST("/update", updateUser)
	// group.GET("/link-token", getLinkToken)
	// group.POST("/exchange-public-token", exchangePublicToken)
	// group.DELETE("/delete", deleteUser)
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
	// get user from db passed on provided username and password
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
	c.SetCookie(
		"purch_token",
		fmt.Sprintf("%d", user.ID),
		1800,
		"/",
		"localhost",
		false,
		true,
	)
	c.JSON(http.StatusOK, gin.H{"message": "user cookie set"})
}
