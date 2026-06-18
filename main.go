package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

    r.GET("/", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "message": "hello",
        })
    })

	if err := r.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
