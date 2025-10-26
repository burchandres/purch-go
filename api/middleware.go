package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"purch/database"
	"purch/utils"
)

// Claims represents the JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// CreateToken generates a JWT token and returns it as a string
func createToken(userID string) (string, error) {
	config := utils.GetConfig()
	issuedAt := time.Now()
	expiresAt := time.Now().Add(30 * time.Minute)

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			Issuer:    "Purch",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseToken validates and parses a JWT token string
func parseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	config := utils.GetConfig()

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// authMiddleware is a Gin middleware that validates JWT tokens
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from cookie
		tokenString, err := c.Cookie("purch_token")
		if err != nil {
			slog.Error("missing purch token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing purch token"})
			c.Abort()
			return
		}

		// Parse and validate token
		claims, err := parseToken(tokenString)
		if err != nil {
			slog.Error("invalid purch token", "error", err.Error(), "endpoint", c.Request.URL)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid purch token"})
			c.Abort()
			return
		}

		// Check if token expires soon (e.g., within 10 minutes) and refresh it automatically
		if time.Until(claims.ExpiresAt.Time) < 10*time.Minute {
			newToken, err := createToken(claims.UserID)
			if err == nil {
				c.SetCookie("purch_token", newToken, 3600, "/", "localhost", false, true)
				c.Header("X-Token-Refreshed", "true") // Optional: signal to frontend
			}
		}

		// Store user in context for use in handlers
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			slog.Error("error parsing user id from purch_token.", "error", err.Error(), "userID", claims.UserID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id format in purch_token"})
			c.Abort()
			return

		}
		user, err := database.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			slog.Error("error getting user from db.", "error", err.Error(), "userID", userID, "endpoint", c.Request.URL)
		}
		c.Set("user", user)
		slog.Debug("user authenticated and userID context set.", "userID", userID, "endpoint", c.Request.URL)

		c.Next()
	}
}
