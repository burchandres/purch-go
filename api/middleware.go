package api

import (
	"fmt"
	"net/http"
	"os"
	"time"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Secret key for signing tokens (in production, load from environment)
var secretKey = []byte(os.Getenv("SECRET_KEY"))

// Claims represents the JWT claims
type Claims struct {
	UserID               string       `json:"user_id"`
	jwt.RegisteredClaims
}

// CreateToken generates a JWT token and returns it as a string
func createToken(userID string) (string, error) {
	issuedAt := time.Now()
	expiresAt := time.Now().Add(30 * time.Minute)

	claims := &Claims{
		UserID:   userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			Issuer:    "Purch",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseToken validates and parses a JWT token string
func parseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
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
			slog.Error("Missing purch token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing purch token"})
			c.Abort()
			return
		}

		// Parse and validate token
		claims, err := parseToken(tokenString)
		if err != nil {
			slog.Error("Invalid purch token", "error", err)
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
		
		// Store userID in context for use in handlers
		c.Set("userID", claims.UserID)
		slog.Info("user authenticated and user_id context set", "userID", claims.UserID)
		
		c.Next()
	}
}