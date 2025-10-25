package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"github.com/davbauer/knock-knock-portal/internal/config"
)

type AdminConfigHandler struct {
	configLoader *config.Loader
}

func NewAdminConfigHandler(configLoader *config.Loader) *AdminConfigHandler {
	return &AdminConfigHandler{
		configLoader: configLoader,
	}
}

// HandleGetConfig returns the current configuration
func (h *AdminConfigHandler) HandleGetConfig(c *gin.Context) {
	cfg := h.configLoader.GetConfig()
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": cfg,
	})
}

// HandleUpdateConfig updates the configuration
func (h *AdminConfigHandler) HandleUpdateConfig(c *gin.Context) {
	var newConfig config.ApplicationConfig
	
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid configuration format: " + err.Error(),
		})
		return
	}
	
	// Get the existing config to compare passwords
	existingConfig := h.configLoader.GetConfig()
	
	// Hash any new or changed passwords for portal users
	for i := range newConfig.PortalUserAccounts {
		user := &newConfig.PortalUserAccounts[i]
		
		// Check if password was provided and needs hashing
		// If bcrypt_hashed_password is empty or doesn't start with $2a/$2b (bcrypt prefix), we need to hash it
		if user.BcryptHashedPassword != "" && 
		   user.BcryptHashedPassword[0] != '$' {
			// This is a plain text password, hash it
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.BcryptHashedPassword), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Failed to hash password for user " + user.Username + ": " + err.Error(),
				})
				return
			}
			user.BcryptHashedPassword = string(hashedPassword)
		} else if user.BcryptHashedPassword == "" {
			// Password was not changed, keep existing hash
			for _, existingUser := range existingConfig.PortalUserAccounts {
				if existingUser.UserID == user.UserID {
					user.BcryptHashedPassword = existingUser.BcryptHashedPassword
					break
				}
			}
			
			// If still empty, this is a new user without a password - reject
			if user.BcryptHashedPassword == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Password is required for new user: " + user.Username,
				})
				return
			}
		}
	}
	
	// Validate the configuration
	if err := config.ValidateConfig(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Configuration validation failed: " + err.Error(),
		})
		return
	}
	
	// Save the configuration
	if err := h.configLoader.SaveConfig(&newConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save configuration: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration updated successfully",
		"data": newConfig,
	})
}
