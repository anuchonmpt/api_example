package router

import (
	"net/http"
	"time"

	"github.com/example/api-example/internal/auth"
	"github.com/example/api-example/internal/handler"
	"github.com/example/api-example/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Config struct {
	JWT             *auth.JWTManager
	AuthHandler     *handler.Auth
	DocumentHandler *handler.Document
	AllowedOrigins  []string
}

func New(cfg Config) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), middleware.RequestID())
	if len(cfg.AllowedOrigins) > 0 {
		engine.Use(cors.New(cors.Config{AllowOrigins: cfg.AllowedOrigins, AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Authorization", "Content-Type", "X-Request-ID"}, ExposeHeaders: []string{"X-Request-ID", "Content-Disposition"}, MaxAge: 12 * time.Hour}))
	}
	engine.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	if cfg.JWT == nil || cfg.AuthHandler == nil || cfg.DocumentHandler == nil {
		return engine
	}
	v1 := engine.Group("/api/v1")
	authRoutes := v1.Group("/auth")
	authRoutes.POST("/register", cfg.AuthHandler.Register)
	authRoutes.POST("/login", cfg.AuthHandler.Login)
	authRoutes.POST("/refresh", cfg.AuthHandler.Refresh)
	authRoutes.POST("/logout", cfg.AuthHandler.Logout)
	authRoutes.GET("/me", middleware.Auth(cfg.JWT), cfg.AuthHandler.Me)
	documents := v1.Group("/documents", middleware.Auth(cfg.JWT))
	documents.POST("", cfg.DocumentHandler.Create)
	documents.GET("", cfg.DocumentHandler.List)
	documents.GET("/:id", cfg.DocumentHandler.Get)
	documents.GET("/:id/download", cfg.DocumentHandler.Download)
	documents.DELETE("/:id", cfg.DocumentHandler.Delete)
	return engine
}
