package auth

import "github.com/gin-gonic/gin"

func Routes(rg *gin.RouterGroup) {
    h := CreateHandler()

    rg.POST("/login", h.Login)
    rg.POST("/logout", h.Logout)
    rg.POST("/refresh", h.Refresh)
}
