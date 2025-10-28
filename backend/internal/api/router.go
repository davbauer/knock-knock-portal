package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/davbauer/knock-knock-portal/internal/auth"
	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/handlers"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/ipblocklist"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/proxy"
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
	blocklistManager *ipblocklist.Manager
	proxyManager     *proxy.Manager
	ipExtractor      *middleware.RealIPExtractor
	indexHTMLHash    string // SHA256 hash of index.html for cache busting
}

// NewRouter creates a new API router
func NewRouter(
	configLoader *config.Loader,
	jwtManager *auth.JWTManager,
	passwordVerifier *auth.PasswordVerifier,
	sessionManager *session.Manager,
	allowlistManager *ipallowlist.Manager,
	blocklistManager *ipblocklist.Manager,
	proxyManager *proxy.Manager,
) *Router {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()

	// Global middleware
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestLogger())
	engine.Use(middleware.RequestSizeLimiter(1 * 1024 * 1024)) // 1MB limit for request bodies

	// Security headers middleware
	engine.Use(func(c *gin.Context) {
		// Security headers
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()")

		// Content Security Policy - Allow inline styles and scripts for SvelteKit
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

		// Cross-Origin Policies
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")

		// Custom branding
		c.Header("X-Powered-By", "Knock-Knock Portal API")
		c.Header("X-Application", "Knock-Knock Authentication Gateway")

		// Cache control for API endpoints
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Next()
	})

	// CORS
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Configure as needed
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Real IP extractor with dynamic config reload
	cfg := configLoader.GetConfig()
	ipExtractor, _ := middleware.NewRealIPExtractor(&cfg.TrustedProxyConfig)

	// Register callback to update IP extractor and proxy manager when config reloads
	configLoader.RegisterReloadCallback(func(newCfg *config.ApplicationConfig) {
		ipExtractor.Reload(&newCfg.TrustedProxyConfig)
		allowlistManager.Reload(&newCfg.NetworkAccessControl)
		blocklistManager.Reload(&newCfg.NetworkAccessControl)

		// Reload proxy manager to apply service changes
		if err := proxyManager.Reload(); err != nil {
			// Log error but don't fail - proxy manager logs details
		}
	})

	engine.Use(ipExtractor.Middleware())

	router := &Router{
		engine:           engine,
		configLoader:     configLoader,
		jwtManager:       jwtManager,
		passwordVerifier: passwordVerifier,
		sessionManager:   sessionManager,
		allowlistManager: allowlistManager,
		blocklistManager: blocklistManager,
		proxyManager:     proxyManager,
		ipExtractor:      ipExtractor,
	}

	// Compute index.html hash for cache busting
	router.computeIndexHash()

	router.setupRoutes()

	return router
}

// setupRoutes sets up all API routes
func (r *Router) setupRoutes() {
	// API routes group
	api := r.engine.Group("/api")
	{
		// Health endpoint (service health only)
		healthHandler := handlers.NewHealthHandler("1.0.0")
		api.GET("/health", healthHandler.Handle)

		// Connection info endpoint (public, returns client IP and allowlist status)
		connectionInfoHandler := handlers.NewConnectionInfoHandler(r.allowlistManager, r.blocklistManager, r.sessionManager, r.configLoader, r.ipExtractor)
		api.GET("/connection-info", connectionInfoHandler.HandleCheck)

		// Portal API (public/authenticated)
		portal := api.Group("/portal")
		{
			// Public endpoints
			loginHandler := handlers.NewPortalLoginHandler(
				r.configLoader,
				r.passwordVerifier,
				r.jwtManager,
				r.sessionManager,
				r.allowlistManager,
				r.blocklistManager,
			)
			portal.POST("/login", loginHandler.Handle)

			usernamesHandler := handlers.NewSuggestedUsernamesHandler(r.configLoader)
			portal.GET("/suggested-usernames", usernamesHandler.Handle)

			// Authenticated endpoints (require portal JWT)
			sessionHandler := handlers.NewPortalSessionHandler(r.sessionManager, r.configLoader, r.allowlistManager)
			authenticated := portal.Group("")
			authenticated.Use(middleware.AuthMiddleware(r.jwtManager, auth.TokenTypePortal))
			{
				authenticated.GET("/session/status", sessionHandler.HandleStatus)
				authenticated.POST("/session/logout", sessionHandler.HandleLogout)
				authenticated.POST("/session/add-ip", sessionHandler.HandleAddIP)
				authenticated.POST("/session/extend", sessionHandler.HandleExtendSession)
			}
		}

		// Admin API (requires admin JWT)
		admin := api.Group("/admin")
		{
			// Login endpoint (public)
			adminLoginHandler := handlers.NewAdminLoginHandler(r.passwordVerifier, r.jwtManager, r.blocklistManager)
			admin.POST("/login", adminLoginHandler.Handle)

			// Protected admin endpoints
			protected := admin.Group("")
			protected.Use(middleware.AuthMiddleware(r.jwtManager, auth.TokenTypeAdmin))
			{
				// User/Session management (authenticated portal users only)
				sessionsHandler := handlers.NewAdminSessionsHandler(r.sessionManager, r.allowlistManager, r.proxyManager)
				protected.GET("/users", sessionsHandler.HandleList)
				protected.DELETE("/users/:session_id", sessionsHandler.HandleDelete)

				// Connection monitoring (shows ALL active connections including anonymous)
				connectionsHandler := handlers.NewAdminConnectionsHandler(r.proxyManager, r.sessionManager)
				protected.GET("/connections", connectionsHandler.HandleList)
				protected.DELETE("/connections/:ip", connectionsHandler.HandleTerminate)

				// Configuration management
				configHandler := handlers.NewAdminConfigHandler(r.configLoader)
				protected.GET("/config", configHandler.HandleGetConfig)
				protected.PUT("/config", configHandler.HandleUpdateConfig)
			}
		}
	}

	// Serve SPA static files
	r.setupSPAHandler()
}

// setupSPAHandler serves the frontend SPA
func (r *Router) setupSPAHandler() {
	// Path to the built frontend
	staticPath := filepath.Join(".", "dist_frontend")
	indexPath := filepath.Join(staticPath, "index.html")

	// Get absolute path to static directory for security checks
	staticPathAbs, err := filepath.Abs(staticPath)
	if err != nil {
		// If we can't get absolute path, disable static serving
		return
	}

	// Serve static assets from _app directory with long cache (immutable versioned assets)
	r.engine.GET("/_app/*filepath", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		// Get the filepath parameter (without the leading /_app)
		filePath := c.Param("filepath")
		c.File(filepath.Join(staticPath, "_app", filePath))
	})

	// Catch-all route for SPA: try to serve static file, fallback to index.html
	r.engine.NoRoute(func(c *gin.Context) {
		// Clean and sanitize the requested path
		requestPath := filepath.Clean(c.Request.URL.Path)

		// Build full file path
		fullPath := filepath.Join(staticPath, requestPath)

		// Get absolute path and check it's within our static directory (prevent path traversal)
		fullPathAbs, err := filepath.Abs(fullPath)
		if err != nil || !strings.HasPrefix(fullPathAbs, staticPathAbs) {
			// Path traversal attempt or invalid path - serve index.html
			r.serveIndexWithHash(c, indexPath)
			return
		}

		// Check if file exists and is not a directory
		fileInfo, err := os.Stat(fullPathAbs)
		if err == nil && !fileInfo.IsDir() {
			// Serve the static file with appropriate cache headers
			if strings.HasPrefix(requestPath, "/_app/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				// Other static files (robots.txt, favicon.png, etc.) - cache but revalidate
				c.Header("Cache-Control", "public, max-age=3600, must-revalidate")
			}
			c.File(fullPathAbs)
			return
		}

		// Fall back to index.html for client-side routing with ETag
		r.serveIndexWithHash(c, indexPath)
	})
}

// computeIndexHash computes SHA256 hash of index.html on startup
func (r *Router) computeIndexHash() {
	indexPath := filepath.Join(".", "dist_frontend", "index.html")

	file, err := os.Open(indexPath)
	if err != nil {
		// index.html doesn't exist yet (maybe frontend not built) - use timestamp
		r.indexHTMLHash = "dev-mode"
		return
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		r.indexHTMLHash = "error"
		return
	}

	// Use first 16 chars of hash for ETag
	r.indexHTMLHash = hex.EncodeToString(hash.Sum(nil))[:16]
}

// serveIndexWithHash serves index.html with ETag for cache validation
func (r *Router) serveIndexWithHash(c *gin.Context, indexPath string) {
	// Set ETag header
	etag := `"` + r.indexHTMLHash + `"`
	c.Header("ETag", etag)
	c.Header("Cache-Control", "no-cache") // Allow caching but must revalidate with ETag

	// Check if client has current version
	if c.GetHeader("If-None-Match") == etag {
		c.Status(304) // Not Modified
		return
	}

	// Serve index.html
	c.File(indexPath)
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
