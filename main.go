package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
    if gin.IsDebugging() {
        if err := godotenv.Load(); err != nil {
            log.Fatal("Error loading .env file")
        }
    }
    
    port := ":" + os.Getenv("PORT")

	r := gin.Default()

    r.GET("/", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "message": "hello",
        })
    })

	if err := r.Run(port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
