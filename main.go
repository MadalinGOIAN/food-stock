package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/MadalinGOIAN/food-stock/internal/db"
	"github.com/MadalinGOIAN/food-stock/internal/api/auth"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
    if gin.IsDebugging() {
        if err := godotenv.Load(); err != nil {
            log.Fatal("Error loading .env file")
        }
    }

    ctx := context.Background()
    pool, err := db.CreatePool(ctx)
    if err != nil {
        log.Fatalf("Database connection failed: %v", err)
    }
    defer pool.Close()
    
    port := ":" + os.Getenv("PORT")

	r := gin.Default()

    r.GET("/", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "message": "hello",
        })
    })

    auth.Routes(r.Group("/auth"))

	if err := r.Run(port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
