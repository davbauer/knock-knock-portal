package api

import (
	"github.com/davbauer/knock-knock-portal/internal/auth"
	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/handlers"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Router handles API routing
type Router struct {
	engine           *gin.Engine
	configLoader     *config.Loader
	jwtManager       *auth.JWTManager
	passwordVerifier *auth.PasswordVerifier
	sessionManager   *session.Manager
	allowlistManager *ipallowlist.Manager
}

// NewRouter creates a new API router
func NewRouter(
	configLoader *config.Loader,
	jwtManager *auth.JWTManager,
	passwordVerifier *auth.PasswordVerifier,
	sessionManager *session.Manager,
	allowlistManager *ipallowlist.Manager,
) *Router {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// Global middleware
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestLogger())

	// CORS
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Configure as needed
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Real IP extractor
	cfg := configLoader.GetConfig()
	ipExtractor, _ := middleware.NewRealIPExtractor(&cfg.TrustedProxyConfig)
	engine.Use(ipExtractor.Middleware())

	router := &Router{
		engine:           engine,
		configLoader:     configLoader,
		jwtManager:       jwtManager,
		passwordVerifier: passwordVerifier,
		sessionManager:   sessionManager,
		allowlistManager: allowlistManager,
	}

	router.setupRoutes()

	return router
}

// setupRoutes sets up all API routes
func (r *Router) setupRoutes() {
	// Health endpoint
	healthHandler := handlers.NewHealthHandler("1.0.0")
	r.engine.GET("/health", healthHandler.Handle)

	// Portal API (public/authenticated)
	portal := r.engine.Group("/api/portal")
	{
		// Public endpoints
		loginHandler := handlers.NewPortalLoginHandler(
			r.configLoader,
			r.passwordVerifier,
			r.jwtManager,
			r.sessionManager,
			r.allowlistManager,
		)
		portal.POST("/login", loginHandler.Handle)

		usernamesHandler := handlers.NewSuggestedUsernamesHandler(r.configLoader)
		portal.GET("/suggested-usernames", usernamesHandler.Handle)

		// Authenticated endpoints (require portal JWT)
		sessionHandler := handlers.NewPortalSessionHandler(r.sessionManager)
		authenticated := portal.Group("")
		authenticated.Use(middleware.AuthMiddleware(r.jwtManager, auth.TokenTypePortal))
		{
			authenticated.GET("/session/status", sessionHandler.HandleStatus)
			authenticated.POST("/session/logout", sessionHandler.HandleLogout)
		}
	}

	// Admin API (requires admin JWT)
	admin := r.engine.Group("/api/admin")
	{
		// Login endpoint (public)
		adminLoginHandler := handlers.NewAdminLoginHandler(r.passwordVerifier, r.jwtManager)
		admin.POST("/login", adminLoginHandler.Handle)

		// Protected admin endpoints
		protected := admin.Group("")
		protected.Use(middleware.AuthMiddleware(r.jwtManager, auth.TokenTypeAdmin))
		{
			// Session management
			sessionsHandler := handlers.NewAdminSessionsHandler(r.sessionManager)
			protected.GET("/sessions", sessionsHandler.HandleList)
			protected.DELETE("/sessions/:session_id", sessionsHandler.HandleDelete)
		}
	}
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
