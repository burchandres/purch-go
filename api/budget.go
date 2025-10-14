package api

import (
	"github.com/gin-gonic/gin"
)

func SetupBudgetEndpoints(router *gin.Engine) {
	protected := router.Group("/budget")
	protected.Use(authMiddleware())
	{
		// protected.GET("/transactions", getTransactions)
		// protected.GET("/categories", getCategories)
		// protected.GET("/items", getItems)
		// protected.GET("/accounts", getAccounts)
		// protected.GET("/category", getCategory)
	}
}

