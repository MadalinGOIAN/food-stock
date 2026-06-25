package auth

import "github.com/gin-gonic/gin"

func Routes(rg *gin.RouterGroup) {
    rg.POST("/login", login)
    rg.POST("/logout", logout)
}
