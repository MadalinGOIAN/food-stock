package auth

import "github.com/gin-gonic/gin"

func Routes(rg *gin.RouterGroup, h *Handler) {
	rg.POST("/login", h.Login)
	rg.POST("/logout", h.Logout)
	rg.POST("/refresh", h.Refresh)
}
