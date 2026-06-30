package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
    Url         string
    Key         string
    Client      *http.Client
    IsSecure    bool
}

func CreateHandler() *Handler {
    return &Handler{
        Url:        os.Getenv("SUPABASE_URL"),
        Key:        os.Getenv("SUPABASE_ANON_KEY"),
        Client:     &http.Client{Timeout: 10 * time.Second},
        IsSecure:   gin.Mode() == gin.ReleaseMode,
    }
}

func (h *Handler) setAuthCookies(
    c *gin.Context, 
    accessToken, 
    refreshToken string,
    accessTokenMaxAge int,
) {
    c.SetSameSite(http.SameSiteLaxMode)
    c.SetCookie("access_token", accessToken, accessTokenMaxAge, "/", "", h.IsSecure, true)
    c.SetCookie("refresh_token", refreshToken, 60*60*24*30, "/", "", h.IsSecure, true)
}

func (h *Handler) clearAuthCookies(c *gin.Context) {
    c.SetSameSite(http.SameSiteLaxMode)
    c.SetCookie("access_token", "", -1, "/", "", h.IsSecure, true)
    c.SetCookie("refresh_token", "", -1, "/", "", h.IsSecure, true)
}

func (h *Handler) Login(c *gin.Context) {
    var loginReq Login
    if err := c.ShouldBindJSON(&loginReq); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }

    body, err := json.Marshal(loginReq)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "could not serialize json"})
        return
    }

    url := h.Url + "/auth/v1/token?grant_type=password"
    thirdPartyReq, err :=
        http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "auth request failed"})
        return
    }

    thirdPartyReq.Header.Set("Content-Type", "application/json")
    thirdPartyReq.Header.Set("apikey", h.Key)

    res, err := h.Client.Do(thirdPartyReq)
    if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "auth service unreachable"})
        return
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
        return
    }

    var token Token
    if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid auth response"})
        return
    }

    h.setAuthCookies(c, token.AccessToken, token.RefreshToken, token.ExpiresIn)
    c.JSON(http.StatusOK, gin.H{"message": "logged in successfully"})
}

func (h *Handler) Logout(c *gin.Context) {
    if accessToken, err := c.Cookie("access_token"); err == nil && accessToken != "" {
        url := h.Url +  "/auth/v1/logout"
        thirdPartyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, nil)
        if err == nil {
            thirdPartyReq.Header.Set("apikey", h.Key)
            thirdPartyReq.Header.Set("Authorization", "Bearer " + accessToken)

            if res, err := h.Client.Do(thirdPartyReq); err == nil {
                res.Body.Close()
            }
        }
    }

    h.clearAuthCookies(c)
    c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (h *Handler) Refresh(c *gin.Context) {
    refreshToken, err := c.Cookie("refresh_token")
    if err != nil || refreshToken == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
        return
    }

    body, err := json.Marshal(map[string]string{"refresh_token": refreshToken})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not serialize json"})
        return
    }

    url := h.Url + "/auth/v1/token?grant_type=refresh_token"
    thirdPartyReq, err :=
        http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "auth request failed"})
        return
    }

    thirdPartyReq.Header.Set("Content-Type", "application/json")
    thirdPartyReq.Header.Set("apikey", h.Key)

    res, err := h.Client.Do(thirdPartyReq)
    if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "auth service unreachable"})
        return
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        h.clearAuthCookies(c)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "could not refresh session"})
        return
    }

    var token Token
    if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid auth response"})
        return
    }

    h.setAuthCookies(c, token.AccessToken, token.RefreshToken, token.ExpiresIn)
    c.JSON(http.StatusOK, gin.H{"message": "session refreshed successfully"})
}
