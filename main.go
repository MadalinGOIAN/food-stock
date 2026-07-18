package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/MadalinGOIAN/food-stock/internal/api/auth"
	"github.com/MadalinGOIAN/food-stock/internal/db"
	"github.com/MadalinGOIAN/food-stock/internal/middleware"
	"github.com/MadalinGOIAN/food-stock/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if gin.IsDebugging() {
		if err := godotenv.Load(); err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	allowedOrigins, err := utils.ParseOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if err != nil {
		log.Fatalf("CORS_ALLOWED_ORIGINS parsing failed: %v", err)
	}

	ctx := context.Background()
	pool, err := db.CreatePool(ctx)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()

	supabaseUrl := os.Getenv("SUPABASE_URL")
	jwksUrl := supabaseUrl + "/auth/v1/.well-known/jwks.json"
	issuer := supabaseUrl + "/auth/v1"

	requireAuth, err := middleware.RequireAuth(jwksUrl, issuer)
	if err != nil {
		log.Fatalf("Auth middleware init failed: %v", err)
	}

	r := gin.Default()
	r.Use(middleware.Cors(allowedOrigins))

	r.GET("/", requireAuth, func(c *gin.Context) {
		userId, ok := middleware.UserId(c)

		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "hello",
			"userId":  userId,
		})
	})

	authProvider := auth.NewSupabaseProvider(supabaseUrl, os.Getenv("SUPABASE_ANON_KEY"))
	authStorage := auth.NewPostgresStorage(pool)
	authHandler := auth.NewHandler(authProvider, authStorage, gin.Mode() == gin.ReleaseMode)

	auth.Routes(r.Group("/auth"), authHandler, requireAuth)

	port := ":" + os.Getenv("PORT")
	if err := r.Run(port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
