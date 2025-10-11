package api

import (
	"net/http"
	"log/slog"
	
	"github.com/gin-gonic/gin"
	
	"purch/database"
)

func SetupUserEndpoints(r *gin.Engine) {
	r.POST("/user/register", registerUser)
	// r.GET("/user/login", getCookie)
	
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
	if err := c.ShouldBind(&storeUserParams); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	registeredUser, err := queries.StoreUser(c.Request.Context(), storeUserParams)
	if err != nil {
		slog.Error("failed to store user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, registeredUser)
}
