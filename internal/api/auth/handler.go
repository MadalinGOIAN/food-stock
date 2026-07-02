package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	provider Provider
	isSecure bool
}

func NewHandler(provider Provider, isSecure bool) *Handler {
	return &Handler{
		provider: provider,
		isSecure: isSecure,
	}
}

func (h *Handler) Login(c *gin.Context) {
	var login Login
	if err := c.ShouldBindJSON(&login); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	token, err := h.provider.Login(c.Request.Context(), login)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}

		writeProviderError(c, err)
		return
	}

	h.setAuthCookies(c, token.AccessToken, token.RefreshToken, token.ExpiresIn)
	c.JSON(http.StatusOK, gin.H{"message": "logged in successfully"})
}

func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}

	token, err := h.provider.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			h.clearAuthCookies(c)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "could not refresh session"})
			return
		}

		writeProviderError(c, err)
		return
	}

	h.setAuthCookies(c, token.AccessToken, token.RefreshToken, token.ExpiresIn)
	c.JSON(http.StatusOK, gin.H{"message": "session refreshed successfully"})
}

func (h *Handler) Logout(c *gin.Context) {
	if accessToken, err := c.Cookie("access_token"); err == nil && accessToken != "" {
		_ = h.provider.Logout(c.Request.Context(), accessToken)
	}

	h.clearAuthCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (h *Handler) setAuthCookies(
	c *gin.Context,
	accessToken,
	refreshToken string,
	accessTokenMaxAge int,
) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", accessToken, accessTokenMaxAge, "/", "", h.isSecure, true)
	c.SetCookie("refresh_token", refreshToken, 60*60*24*30, "/", "", h.isSecure, true)
}

func (h *Handler) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", "", h.isSecure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", h.isSecure, true)
}

func writeProviderError(c *gin.Context, err error) {
	if errors.Is(err, ErrUnreachable) {
		c.JSON(http.StatusBadGateway, gin.H{"error": "auth service unreachable"})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": "auth request failed"})
}
