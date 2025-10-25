package handlers

import (
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/gin-gonic/gin"
)

// SuggestedUsernamesHandler handles getting suggested usernames
type SuggestedUsernamesHandler struct {
	configLoader *config.Loader
}

// NewSuggestedUsernamesHandler creates a new handler
func NewSuggestedUsernamesHandler(configLoader *config.Loader) *SuggestedUsernamesHandler {
	return &SuggestedUsernamesHandler{
		configLoader: configLoader,
	}
}

// Handle processes the request
func (h *SuggestedUsernamesHandler) Handle(c *gin.Context) {
	cfg := h.configLoader.GetConfig()
	
	usernames := []string{}
	for _, user := range cfg.PortalUserAccounts {
		if user.DisplayUsernameInPublicSuggestions {
			usernames = append(usernames, user.Username)
		}
	}

	response := map[string]interface{}{
		"usernames": usernames,
	}

	c.JSON(200, models.NewAPIResponseWithCount("Suggested usernames retrieved", response, len(usernames)))
}
