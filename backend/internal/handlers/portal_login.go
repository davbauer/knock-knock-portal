package handlers

import (
	"time"

	"github.com/davbauer/knock-knock-portal/internal/auth"
	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/ipblocklist"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/davbauer/knock-knock-portal/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// PortalLoginRequest is the login request body
type PortalLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// PortalLoginHandler handles portal user login
type PortalLoginHandler struct {
	configLoader     *config.Loader
	passwordVerifier *auth.PasswordVerifier
	jwtManager       *auth.JWTManager
	sessionManager   *session.Manager
	allowlistManager *ipallowlist.Manager
	blocklistManager *ipblocklist.Manager
	rateLimiter      *auth.RateLimiter
}

// NewPortalLoginHandler creates a new portal login handler
func NewPortalLoginHandler(
	configLoader *config.Loader,
	passwordVerifier *auth.PasswordVerifier,
	jwtManager *auth.JWTManager,
	sessionManager *session.Manager,
	allowlistManager *ipallowlist.Manager,
	blocklistManager *ipblocklist.Manager,
) *PortalLoginHandler {
	return &PortalLoginHandler{
		configLoader:     configLoader,
		passwordVerifier: passwordVerifier,
		jwtManager:       jwtManager,
		sessionManager:   sessionManager,
		allowlistManager: allowlistManager,
		blocklistManager: blocklistManager,
		rateLimiter:      auth.NewRateLimiter(10, 5, 5000), // 10/min, burst 5, max 5000 IPs
	}
}

// Handle processes the login request
func (h *PortalLoginHandler) Handle(c *gin.Context) {
	// Get client IP
	clientIP, ok := middleware.GetClientIP(c)
	if !ok || !clientIP.IsValid() {
		c.JSON(400, models.NewErrorResponse("Could not determine client IP", "INVALID_IP"))
		return
	}

	// HIGHEST PRIORITY: Check IP blocklist FIRST - blocked IPs cannot login
	if blocked, blockReason := h.blocklistManager.IsIPBlocked(clientIP.AsSlice()); blocked {
		c.JSON(403, models.NewErrorResponse("Access denied", "IP_BLOCKED"))
		log.Warn().
			Str("client_ip", clientIP.String()).
			Str("reason", blockReason).
			Msg("Login attempt from blocked IP denied")
		return
	}

	// Rate limiting
	if !h.rateLimiter.Allow(clientIP.String()) {
		c.JSON(429, models.NewErrorResponse("Too many login attempts, please try again later", "RATE_LIMIT_EXCEEDED"))
		return
	}

	// Parse request
	var req PortalLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.NewErrorResponse("Invalid request body", "INVALID_REQUEST"))
		return
	}

	// Find user in config
	cfg := h.configLoader.GetConfig()
	var user *config.PortalUserAccount
	for i := range cfg.PortalUserAccounts {
		if cfg.PortalUserAccounts[i].Username == req.Username {
			user = &cfg.PortalUserAccounts[i]
			break
		}
	}

	// Always verify password to maintain constant time
	// even if user doesn't exist (prevents username enumeration via timing)
	var passwordHash string
	if user != nil {
		passwordHash = user.BcryptHashedPassword
	} else {
		// Use a dummy hash with same computational cost as real bcrypt
		passwordHash = "$2a$10$AAAAAAAAAAAAAAAAAAAAAO1234567890123456789012345678"
	}

	// Verify password (always performed)
	passwordErr := h.passwordVerifier.VerifyUserPassword(req.Password, passwordHash)

	// Check if user exists and password is valid
	if user == nil || passwordErr != nil {
		h.rateLimiter.RecordFailure(clientIP.String())
		c.JSON(401, models.NewErrorResponse("Invalid username or password", "INVALID_CREDENTIALS"))
		log.Warn().
			Str("username", req.Username).
			Str("client_ip", clientIP.String()).
			Bool("user_found", user != nil).
			Msg("Login attempt failed")
		return
	}

	// Record successful authentication to reset rate limit backoff
	h.rateLimiter.RecordSuccess(clientIP.String())

	// Create session
	sess, err := h.sessionManager.CreateSession(
		user.UserID,
		user.Username,
		clientIP,
		user.AllowedServiceIDs,
	)
	if err != nil {
		c.JSON(500, models.NewErrorResponse("Failed to create session", "INTERNAL_ERROR"))
		log.Error().Err(err).Msg("Failed to create session")
		return
	}

	// Add IP to allowlist
	h.allowlistManager.AddSessionIP(sess.SessionID, clientIP, sess.ExpiresAt)

	// Generate JWT token
	tokenDuration := time.Until(sess.ExpiresAt)
	token, err := h.jwtManager.GeneratePortalToken(user.UserID, sess.SessionID, tokenDuration)
	if err != nil {
		c.JSON(500, models.NewErrorResponse("Failed to generate token", "INTERNAL_ERROR"))
		log.Error().Err(err).Msg("Failed to generate JWT token")
		return
	}

	// Get allowed service names
	allowedServices := utils.GetServiceNames(cfg, user.AllowedServiceIDs)

	// Build response
	response := map[string]interface{}{
		"session_id":       sess.SessionID,
		"jwt_access_token": token,
		"token_expires_at": sess.ExpiresAt,
		"session_info": map[string]interface{}{
			"username":            user.Username,
			"authenticated_ip":    clientIP.String(),
			"expires_at":          sess.ExpiresAt,
			"auto_extend_enabled": sess.AutoExtendEnabled,
			"allowed_services":    allowedServices,
		},
	}

	log.Info().
		Str("username", user.Username).
		Str("user_id", user.UserID).
		Str("client_ip", clientIP.String()).
		Msg("User logged in successfully")

	c.JSON(200, models.NewAPIResponse("Login successful", response))
}
