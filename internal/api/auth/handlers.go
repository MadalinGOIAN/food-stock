package auth

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

func login(c *gin.Context) {
    c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}

func logout(c *gin.Context) {
    c.JSON(http.StatusNotImplemented, gin.H{"message": "not implemented"})
}
