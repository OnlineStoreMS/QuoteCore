package router

import (
	"path/filepath"

	"quotecore/admin"
	adminmw "quotecore/admin/middleware"
	"quotecore/internal/config"
	"quotecore/internal/integrations/customercore"
	"quotecore/internal/integrations/productcore"
	jwtmgr "quotecore/internal/pkg/jwt"
	"quotecore/internal/repo"
	"quotecore/internal/service"
	"quotecore/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), corsMiddleware(cfg))

	if cfg.Storage.Driver == "local" || cfg.Storage.Driver == "" {
		uploadDir := filepath.Join(cfg.Storage.LocalPath, cfg.Storage.Prefix)
		r.Static("/uploads", uploadDir)
	}

	store, err := storage.New(&cfg.Storage)
	if err != nil {
		panic(err)
	}

	repos := repo.New(db)
	svc := service.NewQuoteService(repos)
	pc := productcore.NewClient(cfg.Integrations.ProductCoreURL)
	cc := customercore.NewClient(cfg.Integrations.CustomerCoreURL)
	h := admin.NewHandlers(svc, pc, cc)
	uploadH := admin.NewUploadHandler(store)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "quotecore"})
	})

	v1 := r.Group("/api/v1")
	jwtMgr := jwtmgr.NewManager(cfg.Auth.JWTSecret)

	adminGroup := v1.Group("/admin")
	adminGroup.Use(adminmw.AdminAuth(&cfg.Auth, jwtMgr))
	admin.RegisterRoutes(adminGroup, h)
	adminGroup.POST("/upload", uploadH.Upload)
	adminGroup.POST("/upload/from-url", uploadH.UploadFromURL)

	return r
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	origins := cfg.CORS.AllowOrigins
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin == ""
		for _, o := range origins {
			if o == origin || o == "*" {
				allowed = true
				break
			}
		}
		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
