package auth

import "github.com/gin-gonic/gin"

func Routes(rg *gin.RouterGroup, h *Handler, requireAuth gin.HandlerFunc) {
	rg.POST("/signup", h.Signup)
	rg.POST("/login", h.Login)
	rg.POST("/refresh", h.Refresh)
	rg.POST("/logout", requireAuth, h.Logout)
	rg.DELETE("/account", requireAuth, h.DeleteAccount)
}
