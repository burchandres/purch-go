package api

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"purch/database"
)

func SetupBudgetEndpoints(router *gin.Engine) {
	protected := router.Group("/budget")
	protected.Use(authMiddleware())
	{
		protected.GET("/categories", getCategories)
		protected.GET("/items", getItems)
		protected.GET("/accounts", getAccounts)  
		protected.GET("/transactions", getTransactions)
	}
}

func getCategories(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(database.User).ID
	categories, err := database.GetUserCategories(c.Request.Context(), userID)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("error pulling user categories", "error", err.Error(), "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error pulling user categories..."})
	}
	c.JSON(http.StatusOK, gin.H{"results": categories})
}

func getItems(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(database.User).ID
	items, err := database.GetUserItems(c.Request.Context(), userID)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("error pulling user items", "error", err.Error(), "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error pulling user items..."})
	}
	c.JSON(http.StatusOK, gin.H{"results": items})
}

func getAccounts(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(database.User).ID
	accounts, err := database.GetUserAccounts(c.Request.Context(), userID)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("error pulling user accounts", "error", err.Error(), "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error pulling user accounts..."})
	}
	c.JSON(http.StatusOK, gin.H{"results": accounts})
}


func getTransactions(c *gin.Context) {
	user, _ := c.Get("user")
	userID := user.(database.User).ID
	transactions, err := database.GetUserTransactions(c.Request.Context(), userID)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("error pulling user's transactions", "error", err.Error(), "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error pulling user transactions..."})
	}
	c.JSON(http.StatusOK, gin.H{"results": transactions})
}