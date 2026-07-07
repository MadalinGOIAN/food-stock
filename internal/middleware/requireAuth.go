package middleware

import (
	"net/http"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ContextUserId    = "userId"
	ContextUserEmail = "userEmail"
)

func RequireAuth(jwksUrl, issuer string) (gin.HandlerFunc, error) {
	k, err := keyfunc.NewDefault([]string{jwksUrl})
	if err != nil {
		return nil, err
	}

	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"ES256"}),
		jwt.WithIssuer(issuer),
		jwt.WithAudience("authenticated"),
		jwt.WithExpirationRequired(),
	}

	return func(c *gin.Context) {
		rawToken, err := c.Cookie("access_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing access token"})
			return
		}

		token, err := jwt.Parse(rawToken, k.Keyfunc, parserOptions...)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		subject, err := token.Claims.GetSubject()
		if err != nil || subject == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid subject"})
			return
		}

		claims, _ := token.Claims.(jwt.MapClaims)
		email, _ := claims["email"].(string)

		c.Set(ContextUserId, subject)
		c.Set(ContextUserEmail, email)
		c.Next()
	}, nil
}

func UserId(c *gin.Context) (string, bool) {
	v, ok := c.Get(ContextUserId)
	if !ok {
		return "", false
	}

	id, ok := v.(string)
	return id, ok
}

func UserEmail(c *gin.Context) (string, bool) {
	v, ok := c.Get(ContextUserEmail)
	if !ok {
		return "", false
	}

	email, ok := v.(string)
	return email, ok
}
