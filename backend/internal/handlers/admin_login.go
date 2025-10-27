package handlers

import (
	"time"

	"github.com/davbauer/knock-knock-portal/internal/auth"
	"github.com/davbauer/knock-knock-portal/internal/middleware"
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// AdminLoginRequest is the admin login request
type AdminLoginRequest struct {
	AdminPassword string `json:"admin_password" binding:"required"`
}

// AdminLoginHandler handles admin login
type AdminLoginHandler struct {
	passwordVerifier *auth.PasswordVerifier
	jwtManager       *auth.JWTManager
	rateLimiter      *auth.RateLimiter
}

// NewAdminLoginHandler creates a new admin login handler
func NewAdminLoginHandler(
	passwordVerifier *auth.PasswordVerifier,
	jwtManager *auth.JWTManager,
) *AdminLoginHandler {
	return &AdminLoginHandler{
		passwordVerifier: passwordVerifier,
		jwtManager:       jwtManager,
		rateLimiter:      auth.NewRateLimiter(5, 3, 1000), // 5/min, burst 3, max 1000 IPs
	}
}

// Handle processes the admin login request
func (h *AdminLoginHandler) Handle(c *gin.Context) {
	// Get client IP
	clientIP, ok := middleware.GetClientIP(c)
	if !ok || !clientIP.IsValid() {
		c.JSON(400, models.NewErrorResponse("Could not determine client IP", "INVALID_IP"))
		return
	}

	// Rate limiting
	if !h.rateLimiter.Allow(clientIP.String()) {
		c.JSON(429, models.NewErrorResponse("Too many login attempts, please try again later", "RATE_LIMIT_EXCEEDED"))
		return
	}

	// Parse request
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.NewErrorResponse("Invalid request body", "INVALID_REQUEST"))
		return
	}

	// Verify admin password
	if err := h.passwordVerifier.VerifyAdminPassword(req.AdminPassword); err != nil {
		h.rateLimiter.RecordFailure(clientIP.String())
		c.JSON(401, models.NewErrorResponse("Invalid admin password", "INVALID_CREDENTIALS"))
		log.Warn().
			Str("client_ip", clientIP.String()).
			Msg("Failed admin login attempt")
		return
	}

	// Record successful authentication to reset rate limit backoff
	h.rateLimiter.RecordSuccess(clientIP.String())

	// Generate JWT token (24 hours)
	tokenDuration := 24 * time.Hour
	token, err := h.jwtManager.GenerateAdminToken(tokenDuration)
	if err != nil {
		c.JSON(500, models.NewErrorResponse("Failed to generate token", "INTERNAL_ERROR"))
		log.Error().Err(err).Msg("Failed to generate admin JWT token")
		return
	}

	expiresAt := time.Now().Add(tokenDuration)

	response := map[string]interface{}{
		"jwt_access_token": token,
		"token_expires_at": expiresAt,
	}

	log.Info().
		Str("client_ip", clientIP.String()).
		Msg("Admin logged in successfully")

	c.JSON(200, models.NewAPIResponse("Admin login successful", response))
}
