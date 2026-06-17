package handler

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"review-view/web"
)

type Handlers struct {
	Dashboard      *DashboardHandler
	Projects       *ProjectHandler
	Models         *ModelHandler
	Settings       *SettingsHandler
	Tasks          *TaskHandler
	Webhook        *WebhookHandler
	Credentials    *CredentialHandler
	Auth           *AuthHandler
	SensitiveWords *SensitiveWordHandler
	Users          *UserHandler
}

func NewRouter(handlers *Handlers) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/static/") {
			gin.Logger()(c)
		}
	}, gin.Recovery())

	// CORS（开发环境前端 :5173 跨域）
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "http://localhost:5173" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.POST("/api/auth/login", handlers.Auth.Login)

	// 受 JWT 保护的 API
	api := router.Group("/api", JWTMiddleware())
	api.GET("/dashboard", handlers.Dashboard.API)

	// Projects
	api.GET("/projects", handlers.Projects.APIList)
	api.POST("/projects", handlers.Projects.APICreate)
	api.GET("/projects/:id", handlers.Projects.APIGet)
	api.PUT("/projects/:id", handlers.Projects.APIUpdate)
	api.PUT("/projects/:id/schedule", handlers.Projects.APIUpdateSchedule)
	api.DELETE("/projects/:id", handlers.Projects.APIDelete)
	api.GET("/projects/:id/commits", handlers.Projects.Commits)
	api.POST("/projects/:id/trigger", handlers.Projects.APITrigger)
	api.POST("/projects/:id/initialize", handlers.Projects.APIInitialize)

	// Models
	api.GET("/models", handlers.Models.APIList)
	api.POST("/models", handlers.Models.APICreate)
	api.GET("/models/:id", handlers.Models.APIGet)
	api.PUT("/models/:id", handlers.Models.APIUpdate)
	api.DELETE("/models/:id", handlers.Models.APIDelete)
	api.POST("/models/test", handlers.Models.Test)

	// Credentials
	api.GET("/credentials", handlers.Credentials.APIList)
	api.POST("/credentials", handlers.Credentials.APICreate)
	api.GET("/credentials/:id", handlers.Credentials.APIGet)
	api.PUT("/credentials/:id", handlers.Credentials.APIUpdate)
	api.DELETE("/credentials/:id", handlers.Credentials.APIDelete)

	// Tasks
	api.GET("/tasks", handlers.Tasks.APIList)
	api.GET("/tasks/:id", handlers.Tasks.APIGet)
	api.GET("/tasks/:id/stream", handlers.Tasks.Stream)
	api.POST("/tasks/:id/cancel", handlers.Tasks.Cancel)
	api.POST("/tasks/:id/retry", handlers.Tasks.Retry)
	api.DELETE("/tasks/:id", handlers.Tasks.APIDelete)

	// Settings
	api.GET("/settings", handlers.Settings.APIGet)
	api.PUT("/settings", handlers.Settings.APIUpdate)
	api.POST("/settings/test-email", handlers.Settings.APITestEmail)

	// SensitiveWords
	api.GET("/sensitive-words", handlers.SensitiveWords.APIList)
	api.POST("/sensitive-words", handlers.SensitiveWords.APICreate)
	api.PUT("/sensitive-words/:id", handlers.SensitiveWords.APIUpdate)
	api.DELETE("/sensitive-words/:id", handlers.SensitiveWords.APIDelete)

	// Users - own profile (any authenticated user)
	api.GET("/users/me", handlers.Users.APIGetMe)
	api.PUT("/users/me", handlers.Users.APIUpdateMe)
	api.POST("/users/me/test-email", handlers.Users.APITestMyEmail)

	// Users - admin only
	adminAPI := api.Group("", AdminRequired())
	adminAPI.GET("/users", handlers.Users.APIList)
	adminAPI.POST("/users", handlers.Users.APICreate)
	adminAPI.GET("/users/:id", handlers.Users.APIGet)
	adminAPI.PUT("/users/:id", handlers.Users.APIUpdate)
	adminAPI.DELETE("/users/:id", handlers.Users.APIDelete)

	router.POST("/webhook/:projectId", handlers.Webhook.Trigger)

	// Vue3 SPA fallback
	router.NoRoute(mustSPAHandler())

	return router
}

func mustSPAHandler() gin.HandlerFunc {
	distFS, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		panic(err)
	}
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(distFS))
	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path != "" {
			f, err := distFS.Open(path)
			if err == nil {
				stat, _ := f.Stat()
				f.Close()
				if !stat.IsDir() {
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}
}
