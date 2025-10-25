package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading and hot-reload
type Loader struct {
	configFilePath  string
	config          *ApplicationConfig
	configMutex     sync.RWMutex
	fileWatcher     *fsnotify.Watcher
	reloadCallbacks []func(*ApplicationConfig)
	stopChan        chan struct{}
}

// NewLoader creates a new configuration loader
func NewLoader(configPath string) (*Loader, error) {
	loader := &Loader{
		configFilePath:  configPath,
		reloadCallbacks: []func(*ApplicationConfig){},
		stopChan:        make(chan struct{}),
	}

	// Load .env file (optional)
	_ = godotenv.Load()

	// Initial load
	if err := loader.reload(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Start file watcher
	if err := loader.startFileWatcher(); err != nil {
		log.Warn().Err(err).Msg("Failed to start config file watcher, hot-reload disabled")
	}

	return loader, nil
}

// reload loads configuration from file
func (l *Loader) reload() error {
	// Start with defaults
	cfg := GetDefaultConfig()

	// Load from YAML file if it exists
	data, err := os.ReadFile(l.configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn().Str("path", l.configFilePath).Msg("Config file not found, using defaults")
		} else {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	}

	// Apply environment variable overrides
	l.applyEnvironmentOverrides(cfg)

	// Validate configuration
	if err := ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Store config
	l.configMutex.Lock()
	l.config = cfg
	l.configMutex.Unlock()

	log.Info().Msg("Configuration loaded successfully")
	return nil
}

// applyEnvironmentOverrides applies environment variable overrides
func (l *Loader) applyEnvironmentOverrides(cfg *ApplicationConfig) {
	// HTTP_SERVER_PORT override
	if port := os.Getenv("HTTP_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.ProxyServerConfig.AdminAPIPort = p
		}
	}

	// TRUSTED_PROXY_ENABLED override
	if enabled := os.Getenv("TRUSTED_PROXY_ENABLED"); enabled != "" {
		cfg.TrustedProxyConfig.Enabled = strings.ToLower(enabled) == "true"
	}

	// TRUSTED_PROXY_IP_RANGES override
	if ranges := os.Getenv("TRUSTED_PROXY_IP_RANGES"); ranges != "" {
		cfg.TrustedProxyConfig.TrustedProxyIPRanges = strings.Split(ranges, ",")
	}
}

// GetConfig returns the current configuration (thread-safe)
func (l *Loader) GetConfig() *ApplicationConfig {
	l.configMutex.RLock()
	defer l.configMutex.RUnlock()
	return l.config
}

// RegisterReloadCallback registers a callback to be called on config reload
func (l *Loader) RegisterReloadCallback(callback func(*ApplicationConfig)) {
	l.reloadCallbacks = append(l.reloadCallbacks, callback)
}

// startFileWatcher starts watching the config file for changes
func (l *Loader) startFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	l.fileWatcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Info().Str("file", l.configFilePath).Msg("Config file changed, reloading...")

					if err := l.reload(); err != nil {
						log.Error().Err(err).Msg("Failed to reload config")
						continue
					}

					// Notify all registered callbacks
					cfg := l.GetConfig()
					for _, callback := range l.reloadCallbacks {
						callback(cfg)
					}

					log.Info().Msg("Config reloaded successfully")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error().Err(err).Msg("Config watcher error")
			case <-l.stopChan:
				return
			}
		}
	}()

	return watcher.Add(l.configFilePath)
}

// SaveConfig saves the configuration to file
func (l *Loader) SaveConfig(cfg *ApplicationConfig) error {
	// Validate before saving
	if err := ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(l.configFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Update in-memory config
	l.configMutex.Lock()
	l.config = cfg
	l.configMutex.Unlock()

	log.Info().Msg("Configuration saved successfully")
	return nil
}

// Close closes the config loader and file watcher
func (l *Loader) Close() error {
	close(l.stopChan)
	if l.fileWatcher != nil {
		return l.fileWatcher.Close()
	}
	return nil
}
