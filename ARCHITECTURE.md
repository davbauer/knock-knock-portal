# Knock-Knock Portal - Backend Architecture

## Overview

A high-performance Go-based reverse proxy with dynamic IP-based access control. Users authenticate via web portal to temporarily whitelist their IP addresses for specific services/ports.

## Core Concept

**Problem Solved**: Protect services (game servers, APIs, databases) from bots and unauthorized access while providing frictionless access to authorized users without VPN overhead.

**Solution**: Dynamic IP whitelisting through web authentication. Users login → their IP gets temporary access → can connect to protected services.

## Architecture Components

### 1. Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                          # Application entry point, bootstraps all components
│
├── internal/
│   ├── config/
│   │   ├── models.go                        # All configuration struct definitions
│   │   ├── loader.go                        # Load config.yml + .env, merge with defaults
│   │   ├── validator.go                     # Validate config (ports, IPs, bcrypt, conflicts)
│   │   ├── watcher.go                       # File watcher for hot-reload (fsnotify)
│   │   └── defaults.go                      # Default configuration values
│   │
│   ├── auth/
│   │   ├── jwt_manager.go                   # JWT creation, parsing, validation
│   │   ├── password_verifier.go             # Bcrypt password comparison
│   │   ├── rate_limiter.go                  # Rate limiting for login endpoints
│   │   └── middleware.go                    # HTTP middleware for JWT validation
│   │
│   ├── session/
│   │   ├── session_manager.go               # Core session CRUD operations
│   │   ├── session_store.go                 # In-memory storage with sync.Map
│   │   ├── session_cleanup.go               # Background goroutine for expiry cleanup
│   │   └── models.go                        # Session data structures
│   │
│   ├── ipallowlist/
│   │   ├── allowlist_manager.go             # IP allowlist operations, orchestration
│   │   ├── allowlist_store.go               # Concurrent-safe storage (sync.Map + slice)
│   │   ├── dns_resolver.go                  # Resolve dynamic DNS hostnames to IPs
│   │   ├── ip_matcher.go                    # CIDR/range matching logic
│   │   └── models.go                        # IP allowlist entry structures
│   │
│   ├── proxy/
│   │   ├── listener_manager.go              # Manages all proxy listeners lifecycle
│   │   ├── tcp_proxy.go                     # TCP connection proxying implementation
│   │   ├── udp_proxy.go                     # UDP packet proxying implementation
│   │   ├── http_reverse_proxy.go            # HTTP reverse proxy with header manipulation
│   │   ├── connection_filter.go             # IP-based connection filtering
│   │   ├── connection_tracker.go            # Track active connections, stats
│   │   └── models.go                        # Connection/proxy data structures
│   │
│   ├── api/
│   │   ├── router.go                        # HTTP router setup, route registration
│   │   ├── server.go                        # HTTP server initialization
│   │   │
│   │   ├── handlers/
│   │   │   ├── portal_login.go              # POST /api/portal/login
│   │   │   ├── portal_session.go            # GET/POST /api/portal/session/*
│   │   │   ├── portal_usernames.go          # GET /api/portal/suggested-usernames
│   │   │   ├── admin_login.go               # POST /api/admin/login
│   │   │   ├── admin_config.go              # GET/PUT/PATCH /api/admin/config/*
│   │   │   ├── admin_sessions.go            # GET/DELETE /api/admin/sessions/*
│   │   │   ├── admin_services.go            # GET /api/admin/services/*
│   │   │   ├── admin_allowlist.go           # GET /api/admin/allowlist/*
│   │   │   ├── admin_users.go               # GET/PUT /api/admin/users/*
│   │   │   ├── health.go                    # GET /health
│   │   │   └── metrics.go                   # GET /metrics (future)
│   │   │
│   │   └── middleware/
│   │       ├── real_ip_extractor.go         # Extract real IP from trusted proxies
│   │       ├── rate_limiter.go              # HTTP rate limiting middleware
│   │       ├── cors.go                      # CORS headers for frontend
│   │       ├── request_logger.go            # Structured request/response logging
│   │       └── error_handler.go             # Centralized error response formatting
│   │
│   └── models/
│       ├── config.go                        # Configuration-related models
│       ├── session.go                       # Session-related models
│       ├── api_request.go                   # API request body structures
│       ├── api_response.go                  # API response body structures
│       └── errors.go                        # Custom error types
│
├── pkg/
│   └── utils/
│       ├── ip_parser.go                     # IP address parsing and normalization
│       ├── ip_validator.go                  # IP/CIDR validation utilities
│       ├── dns_lookup.go                    # DNS resolution helpers
│       ├── bcrypt_hasher.go                 # Bcrypt hashing utility (for CLI)
│       └── uid_generator.go                 # UUID v4 generation
│
├── scripts/
│   ├── generate_admin_hash.go               # CLI tool: hash admin password
│   ├── generate_user_hash.go                # CLI tool: hash user password
│   └── validate_config.go                   # CLI tool: validate config.yml
│
├── config.example.yml                        # Example configuration file
├── .env.example                             # Example environment variables
├── config.yml                               # Actual configuration (gitignored)
├── .env                                     # Actual environment (gitignored)
├── go.mod                                   # Go module definition
├── go.sum                                   # Go module checksums
├── Dockerfile                               # Multi-stage Docker build
├── docker-compose.yml                       # Docker Compose example
├── .dockerignore                            # Docker build exclusions
├── .gitignore                               # Git exclusions
└── README.md                                # Project documentation
```

### 2. Configuration Schema

#### `.env` File
```env
# Admin Authentication
# This password grants access to /api/admin/* endpoints for configuration management
ADMIN_PASSWORD_BCRYPT_HASH=$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# JWT Token Security
# Secret key for signing/verifying JWT tokens (generate with: openssl rand -base64 32)
JWT_SIGNING_SECRET_KEY=your-random-secret-key-here-change-me-in-production

# Application Configuration
CONFIG_FILE_PATH=./config.yml           # Path to YAML config file
HTTP_SERVER_PORT=8000                   # Port for API endpoints (overrides config.yml if set)
LOG_LEVEL=info                          # debug | info | warn | error
LOG_FORMAT=json                         # json | text

# Optional: Override config.yml trusted proxy settings via environment
TRUSTED_PROXY_ENABLED=false
TRUSTED_PROXY_IP_RANGES=172.17.0.0/16,10.0.0.1
```

#### `config.yml` File
```yaml
# Session Management Configuration
session_config:
  default_session_duration_seconds: 3600        # How long a user session lasts by default (1 hour)
  auto_extend_session_on_connection: true       # If true, each proxied connection resets expiration timer
  maximum_session_duration_seconds: 86400       # Hard limit for session extension (24 hours). Use null for no limit
  session_cleanup_interval_seconds: 60          # How often to purge expired sessions from memory

# Network Access Control Lists
network_access_control:
  # Hostnames that resolve to allowed IPs (checked every 5 minutes)
  # Useful for dynamic DNS - all resolved IPs (A/AAAA records) are automatically allowed
  allowed_dynamic_dns_hostnames:
    - "dyn.example.com"
    - "home.mydomain.net"
  
  # Static IP addresses and CIDR ranges that bypass authentication
  # These IPs are ALWAYS allowed access to all services
  permanently_allowed_ip_ranges:
    - "192.168.1.0/24"           # Local network
    - "2001:db8::/32"            # IPv6 range
    - "10.0.5.100"               # Single IP (equivalent to /32)
  
  # How often to re-resolve DNS hostnames (in seconds)
  dns_refresh_interval_seconds: 300

# Proxy Server Configuration
proxy_server_config:
  listen_address: "0.0.0.0"                     # Interface to bind all proxy listeners to
  admin_api_port: 8000                          # Port for admin/portal API endpoints
  connection_timeout_seconds: 30                # Idle connection timeout
  max_connections_per_service: 1000             # Per-service connection limit (0 = unlimited)
  tcp_buffer_size_bytes: 32768                  # Buffer size for TCP proxying (32KB)
  udp_buffer_size_bytes: 65507                  # Buffer size for UDP packets (max UDP payload)
  udp_session_timeout_seconds: 300              # How long to keep UDP client mappings alive

# Real IP Extraction for Proxied Deployments
trusted_proxy_config:
  enabled: false                                # Set to true if behind reverse proxy (Nginx, Caddy, etc)
  trusted_proxy_ip_ranges:                      # Only trust these proxies for X-Forwarded-For
    - "172.17.0.0/16"          # Docker network
    - "10.0.0.1"               # Specific proxy IP
  client_ip_header_priority:                    # Check headers in this order
    - "CF-Connecting-IP"       # Cloudflare
    - "X-Real-IP"              # Nginx
    - "X-Forwarded-For"        # Standard (uses first IP)

# Portal User Accounts
# These users can login at /api/portal/login to authorize their IP address
portal_user_accounts:
  - user_id: "550e8400-e29b-41d4-a716-446655440000"
    username: "kevin"
    display_username_in_public_login_suggestions: true   # If true, username appears in login dropdown
    bcrypt_hashed_password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
    allowed_service_ids:                        # Which services this user can access (empty = all)
      - "7c9e6679-7425-40de-944b-e07fc1f90ae7" # Minecraft server only
      - "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d" # Web services
    notes: "Kevin's personal account - Minecraft + web access"

  - user_id: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
    username: "steve"
    display_username_in_public_login_suggestions: false  # Hidden from dropdown
    bcrypt_hashed_password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
    allowed_service_ids: []                     # Empty array = access to ALL services
    notes: "Admin user - full service access"

# Protected Services Definitions
# Each service represents a port/port-range that requires authentication
protected_services:
  - service_id: "7c9e6679-7425-40de-944b-e07fc1f90ae7"
    service_name: "Minecraft Server"            # Human-readable name shown in portal
    proxy_listen_port_start: 25565              # First port to proxy
    proxy_listen_port_end: 25565                # Last port to proxy (same = single port)
    backend_target_host: "127.0.0.1"            # Where to proxy connections to
    backend_target_port_start: 25565            # Backend port (if range, maps 1:1)
    backend_target_port_end: 25565
    transport_protocol: "tcp"                   # tcp | udp | both
    is_http_protocol: false                     # If true, enables HTTP reverse proxy features
    enabled: true                               # Can disable without deleting config
    description: "Main Minecraft game server"

  - service_id: "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d"
    service_name: "Web Services"
    proxy_listen_port_start: 8080
    proxy_listen_port_end: 8090                 # Proxies ports 8080-8090 (11 ports)
    backend_target_host: "192.168.1.100"
    backend_target_port_start: 8080             # Maps to backend 8080-8090
    backend_target_port_end: 8090
    transport_protocol: "tcp"
    is_http_protocol: true                      # Enables HTTP header manipulation
    enabled: true
    description: "Internal web applications"
    
    # HTTP-specific configuration (only used if is_http_protocol: true)
    http_config:
      # Headers to add (preserves existing if present)
      inject_http_request_headers:
        X-Forwarded-User: "${authenticated_username}"  # Template variable
        X-Session-ID: "${session_id}"
        X-Authenticated: "true"
      
      # Headers to set/overwrite (replaces existing)
      override_http_request_headers:
        X-Real-IP: "${client_ip_address}"
        X-Forwarded-Proto: "https"
      
      # Remove these headers before proxying
      remove_http_request_headers:
        - "X-Custom-Auth"
        - "Authorization"  # Prevent auth bypass
      
      # Modify response headers
      inject_http_response_headers:
        X-Proxied-By: "knock-knock-portal"
        
  - service_id: "b2c3d4e5-f6a7-4b5c-8d9e-0f1a2b3c4d5e"
    service_name: "Game Server API"
    proxy_listen_port_start: 9000
    proxy_listen_port_end: 9000
    backend_target_host: "gameserver.internal.lan"  # DNS names supported
    backend_target_port_start: 9000
    backend_target_port_end: 9000
    transport_protocol: "both"                   # Proxies both TCP and UDP on same port
    is_http_protocol: false
    enabled: true
    description: "Game server query API (TCP) and voice chat (UDP)"
```

### 3. Core Components Design

#### A. Configuration Management

**Design Goals**:
- Type-safe configuration with validation
- Hot-reload capability without service disruption
- Environment variable overrides for sensitive data
- Clear error messages for misconfigurations

**Implementation**:
```go
package config

import (
    "fmt"
    "gopkg.in/yaml.v3"
    "github.com/joho/godotenv"
    "github.com/fsnotify/fsnotify"
)

// Root configuration structure
type ApplicationConfig struct {
    SessionConfig             SessionConfiguration         `yaml:"session_config"`
    NetworkAccessControl      NetworkAccessControlConfig   `yaml:"network_access_control"`
    ProxyServerConfig         ProxyServerConfiguration     `yaml:"proxy_server_config"`
    TrustedProxyConfig        TrustedProxyConfiguration    `yaml:"trusted_proxy_config"`
    PortalUserAccounts        []PortalUserAccount          `yaml:"portal_user_accounts"`
    ProtectedServices         []ProtectedServiceConfig     `yaml:"protected_services"`
}

type SessionConfiguration struct {
    DefaultSessionDurationSeconds      int  `yaml:"default_session_duration_seconds"`
    AutoExtendSessionOnConnection      bool `yaml:"auto_extend_session_on_connection"`
    MaximumSessionDurationSeconds      *int `yaml:"maximum_session_duration_seconds"` // nil = unlimited
    SessionCleanupIntervalSeconds      int  `yaml:"session_cleanup_interval_seconds"`
}

type NetworkAccessControlConfig struct {
    AllowedDynamicDNSHostnames       []string `yaml:"allowed_dynamic_dns_hostnames"`
    PermanentlyAllowedIPRanges       []string `yaml:"permanently_allowed_ip_ranges"`
    DNSRefreshIntervalSeconds        int      `yaml:"dns_refresh_interval_seconds"`
}

type ProxyServerConfiguration struct {
    ListenAddress                    string `yaml:"listen_address"`
    AdminAPIPort                     int    `yaml:"admin_api_port"`
    ConnectionTimeoutSeconds         int    `yaml:"connection_timeout_seconds"`
    MaxConnectionsPerService         int    `yaml:"max_connections_per_service"`
    TCPBufferSizeBytes              int    `yaml:"tcp_buffer_size_bytes"`
    UDPBufferSizeBytes              int    `yaml:"udp_buffer_size_bytes"`
    UDPSessionTimeoutSeconds        int    `yaml:"udp_session_timeout_seconds"`
}

type TrustedProxyConfiguration struct {
    Enabled                          bool     `yaml:"enabled"`
    TrustedProxyIPRanges            []string `yaml:"trusted_proxy_ip_ranges"`
    ClientIPHeaderPriority          []string `yaml:"client_ip_header_priority"`
}

type PortalUserAccount struct {
    UserID                           string   `yaml:"user_id"`
    Username                         string   `yaml:"username"`
    DisplayUsernameInPublicSuggestions bool   `yaml:"display_username_in_public_login_suggestions"`
    BcryptHashedPassword            string   `yaml:"bcrypt_hashed_password"`
    AllowedServiceIDs               []string `yaml:"allowed_service_ids"` // Empty = all
    Notes                           string   `yaml:"notes"`               // Optional documentation
}

type ProtectedServiceConfig struct {
    ServiceID                        string                `yaml:"service_id"`
    ServiceName                      string                `yaml:"service_name"`
    ProxyListenPortStart            int                   `yaml:"proxy_listen_port_start"`
    ProxyListenPortEnd              int                   `yaml:"proxy_listen_port_end"`
    BackendTargetHost               string                `yaml:"backend_target_host"`
    BackendTargetPortStart          int                   `yaml:"backend_target_port_start"`
    BackendTargetPortEnd            int                   `yaml:"backend_target_port_end"`
    TransportProtocol               string                `yaml:"transport_protocol"` // tcp | udp | both
    IsHTTPProtocol                  bool                  `yaml:"is_http_protocol"`
    Enabled                         bool                  `yaml:"enabled"`
    Description                     string                `yaml:"description"`
    HTTPConfig                      *HTTPProtocolConfig   `yaml:"http_config,omitempty"`
}

type HTTPProtocolConfig struct {
    InjectHTTPRequestHeaders        map[string]string `yaml:"inject_http_request_headers"`
    OverrideHTTPRequestHeaders      map[string]string `yaml:"override_http_request_headers"`
    RemoveHTTPRequestHeaders        []string          `yaml:"remove_http_request_headers"`
    InjectHTTPResponseHeaders       map[string]string `yaml:"inject_http_response_headers"`
}

// Configuration loader with environment variable override
type ConfigLoader struct {
    configFilePath    string
    config            *ApplicationConfig
    configMutex       sync.RWMutex
    fileWatcher       *fsnotify.Watcher
    reloadCallbacks   []func(*ApplicationConfig)
}

func NewConfigLoader(configPath string) (*ConfigLoader, error) {
    loader := &ConfigLoader{
        configFilePath: configPath,
    }
    
    // Load .env file
    godotenv.Load()
    
    // Initial load
    if err := loader.loadConfigFromFile(); err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    
    // Start file watcher
    if err := loader.startFileWatcher(); err != nil {
        return nil, fmt.Errorf("failed to start config watcher: %w", err)
    }
    
    return loader, nil
}

func (cl *ConfigLoader) loadConfigFromFile() error {
    data, err := os.ReadFile(cl.configFilePath)
    if err != nil {
        return err
    }
    
    var config ApplicationConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return fmt.Errorf("YAML parse error: %w", err)
    }
    
    // Apply environment variable overrides
    cl.applyEnvironmentOverrides(&config)
    
    // Validate configuration
    if err := cl.validateConfig(&config); err != nil {
        return fmt.Errorf("config validation failed: %w", err)
    }
    
    // Store config
    cl.configMutex.Lock()
    cl.config = &config
    cl.configMutex.Unlock()
    
    return nil
}

func (cl *ConfigLoader) GetConfig() *ApplicationConfig {
    cl.configMutex.RLock()
    defer cl.configMutex.RUnlock()
    return cl.config
}

func (cl *ConfigLoader) RegisterReloadCallback(callback func(*ApplicationConfig)) {
    cl.reloadCallbacks = append(cl.reloadCallbacks, callback)
}

// Hot reload on file change
func (cl *ConfigLoader) startFileWatcher() error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    
    cl.fileWatcher = watcher
    
    go func() {
        for {
            select {
            case event := <-watcher.Events:
                if event.Op&fsnotify.Write == fsnotify.Write {
                    log.Info().Str("file", cl.configFilePath).Msg("Config file changed, reloading...")
                    
                    if err := cl.loadConfigFromFile(); err != nil {
                        log.Error().Err(err).Msg("Failed to reload config")
                        continue
                    }
                    
                    // Notify all components
                    for _, callback := range cl.reloadCallbacks {
                        callback(cl.GetConfig())
                    }
                    
                    log.Info().Msg("Config reloaded successfully")
                }
            case err := <-watcher.Errors:
                log.Error().Err(err).Msg("Config watcher error")
            }
        }
    }()
    
    return watcher.Add(cl.configFilePath)
}
```

**Validation Functions**:
```go
func (cl *ConfigLoader) validateConfig(config *ApplicationConfig) error {
    // Validate port ranges
    for _, service := range config.ProtectedServices {
        if service.ProxyListenPortStart < 1 || service.ProxyListenPortStart > 65535 {
            return fmt.Errorf("service %s: invalid proxy_listen_port_start", service.ServiceID)
        }
        
        if service.ProxyListenPortEnd < service.ProxyListenPortStart {
            return fmt.Errorf("service %s: proxy_listen_port_end must be >= proxy_listen_port_start", service.ServiceID)
        }
        
        portRangeSize := service.ProxyListenPortEnd - service.ProxyListenPortStart + 1
        backendRangeSize := service.BackendTargetPortEnd - service.BackendTargetPortStart + 1
        if portRangeSize != backendRangeSize {
            return fmt.Errorf("service %s: proxy and backend port range sizes must match", service.ServiceID)
        }
    }
    
    // Check for port conflicts
    if err := cl.checkPortConflicts(config.ProtectedServices); err != nil {
        return err
    }
    
    // Validate bcrypt passwords
    for _, user := range config.PortalUserAccounts {
        if !strings.HasPrefix(user.BcryptHashedPassword, "$2a$") && !strings.HasPrefix(user.BcryptHashedPassword, "$2b$") {
            return fmt.Errorf("user %s: bcrypt_hashed_password does not appear to be valid bcrypt hash", user.Username)
        }
    }
    
    // Validate IP ranges
    for _, ipRange := range config.NetworkAccessControl.PermanentlyAllowedIPRanges {
        if _, err := netip.ParsePrefix(ipRange); err != nil {
            // Try parsing as single IP
            if _, err := netip.ParseAddr(ipRange); err != nil {
                return fmt.Errorf("invalid IP range: %s", ipRange)
            }
        }
    }
    
    return nil
}
```

#### B. Session Management
**Session Data Model**:
```go
type AuthenticatedUserSession struct {
    SessionID                    string              // UUID v4
    UserID                       string              // From portal_user_accounts[].user_id
    Username                     string              // From portal_user_accounts[].username
    AuthenticatedClientIPAddress netip.Addr          // The IP that authenticated
    AllowedServiceIDs            []string            // Service UUIDs user can access (nil = all)
    SessionCreatedAt             time.Time
    SessionLastActivityAt        time.Time
    SessionExpiresAt             time.Time
    SessionAutoExtendEnabled     bool                // From config: auto_extend_session_on_connection
    SessionMaximumDuration       *time.Duration      // Maximum total session lifetime
}

type SessionManager struct {
    activeSessions          sync.Map                // map[sessionID]AuthenticatedUserSession
    sessionsByUserID        sync.Map                // map[userID][]sessionID - for multi-device tracking
    sessionsByIPAddress     sync.Map                // map[ipAddress][]sessionID - for IP-based lookups
    configReference         *config.Config
    cleanupTicker           *time.Ticker
    shutdownChannel         chan struct{}
}
```

**Core Features**:
- **Concurrent access**: `sync.Map` for lock-free reads (sessions checked on every proxied connection)
- **Multiple indices**: Quick lookup by session ID, user ID, or IP address
- **Automatic cleanup**: Background goroutine removes expired sessions every 60s
- **Activity tracking**: Updates `SessionLastActivityAt` on each proxied connection if `auto_extend_session_on_connection: true`
- **Hard limits**: `SessionMaximumDuration` prevents indefinite extension

**Key Methods**:
```go
func (m *SessionManager) CreateSession(userID, username string, clientIP netip.Addr, allowedServices []string) (*AuthenticatedUserSession, error)
func (m *SessionManager) GetSessionByID(sessionID string) (*AuthenticatedUserSession, error)
func (m *SessionManager) GetSessionByIPAddress(ip netip.Addr) (*AuthenticatedUserSession, bool)
func (m *SessionManager) ValidateAndRefreshSession(sessionID string) (*AuthenticatedUserSession, error)
func (m *SessionManager) RecordSessionActivity(sessionID string) error
func (m *SessionManager) TerminateSession(sessionID string) error
func (m *SessionManager) GetAllActiveSessions() []AuthenticatedUserSession
func (m *SessionManager) CleanupExpiredSessions() int // Returns number of sessions removed
```

**Session Lifecycle**:
```
1. User authenticates -> CreateSession()
2. Each proxy connection -> GetSessionByIPAddress() -> check AllowedServiceIDs
3. If auto_extend enabled -> RecordSessionActivity() -> update SessionLastActivityAt, extend SessionExpiresAt
4. Check SessionMaximumDuration -> if exceeded, prevent extension
5. User logs out OR session expires -> TerminateSession()
6. Background cleanup goroutine -> CleanupExpiredSessions() every 60s
```

#### C. IP Allowlist Manager
**Three-Tier Allowlist System**:
1. **Permanent IPs**: From `permanently_allowed_ip_ranges` config - always allowed, no auth required
2. **DNS-Resolved IPs**: Periodically resolved from `allowed_dynamic_dns_hostnames` (every 5min) - always allowed
3. **Authenticated Session IPs**: Users who logged in via portal - temporary access based on session

**Data Structure**:
```go
type IPAllowlistEntry struct {
    IPAddress           netip.Addr           // Normalized IP (v4 or v6)
    IPPrefix            netip.Prefix         // For CIDR ranges
    SourceType          string               // "permanent" | "dns_resolved" | "authenticated_session"
    AssociatedSessionID string               // Only for authenticated_session type
    AddedAt             time.Time
    ExpiresAt           *time.Time           // nil for permanent/DNS IPs
    LastVerifiedAt      time.Time            // For DNS entries
    OriginalHostname    string               // For DNS entries, which hostname resolved to this
}

type IPAllowlistManager struct {
    exactIPAllowlist    sync.Map             // map[string]*IPAllowlistEntry - key: IP string
    cidrRangeAllowlist  []*IPAllowlistEntry  // Slice of CIDR entries (requires iteration)
    cidrRangeMutex      sync.RWMutex         // Protects CIDR slice
    dnsResolver         *DNSResolver
    configReference     *config.Config
}
```

**Implementation Details**:
- **Fast path**: Exact IP lookup in `sync.Map` - O(1)
- **Slow path**: CIDR range matching - O(n) but typically <10 ranges
- **Normalization**: All IPs normalized to canonical form (IPv4-mapped IPv6 removed)
- **Thread-safe**: Concurrent reads, mutex-protected writes

**Methods**:
```go
func (m *IPAllowlistManager) IsIPAllowed(ip netip.Addr) (allowed bool, reason string)
func (m *IPAllowlistManager) IsIPAllowedForService(ip netip.Addr, serviceID string) (allowed bool, reason string)
func (m *IPAllowlistManager) AddSessionIP(sessionID string, ip netip.Addr, expiresAt time.Time)
func (m *IPAllowlistManager) RemoveSessionIP(sessionID string)
func (m *IPAllowlistManager) RefreshDNSEntries(ctx context.Context) error
func (m *IPAllowlistManager) GetAllowlistStats() AllowlistStatistics
```

#### D. Proxy Architecture (Simplified)

**TCP Proxy**:
```
For each service with transport_protocol: "tcp" or "both":
1. Listen on proxy_listen_port_start:proxy_listen_port_end
2. On connection accept:
   - Extract client IP
   - Check if IP allowed (permanent/DNS/session)
   - If session-based: verify user has access to this service_id
   - If denied: close connection
   - If allowed: dial backend, io.Copy bidirectional (client ↔ backend)
   - Update session activity if auto_extend enabled
```

**UDP Proxy**:
```
For each service with transport_protocol: "udp" or "both":
1. Listen on proxy_listen_port
2. Maintain map of client→backend connections
3. On packet receive:
   - Extract client IP
   - Check if IP allowed
   - If new client: create backend connection
   - Forward packet to backend
   - Forward backend responses back to client
   - Cleanup idle connections after timeout
```

**HTTP Reverse Proxy**:
```
For services with is_http_protocol: true:
1. Use httputil.ReverseProxy
2. Before proxying:
   - Extract client IP (trusted proxy aware)
   - Check IP allowlist
   - Inject/override/remove headers based on http_config
   - Replace template variables: ${authenticated_username}, ${client_ip_address}
3. Proxy to backend
4. Update session activity
```

#### E. Authentication Flow

**Portal User Login** (`POST /api/portal/login`):
```go
// Standard API Response Wrapper
type APIResponse struct {
    Message      string      `json:"message"`
    Data         interface{} `json:"data"`
    TotalResults *int        `json:"total_results,omitempty"`
}

// Login Flow
1. Extract client IP (respects trusted_proxy_config)
2. Check rate limit (10/min per IP)
3. Find user by username in config
4. Verify password with bcrypt
5. Create session with SessionManager
6. Add IP to allowlist
7. Generate JWT token (user_id, session_id, type="portal")
8. Return APIResponse{
    Message: "Login successful",
    Data: {
        session_id, jwt_access_token, session_info
    }
}
```

**Admin Login** (`POST /api/admin/login`):
```go
// Flow
1. Extract client IP (for logging)
2. Check rate limit (5/min per IP)
3. Verify password against ADMIN_PASSWORD_BCRYPT_HASH
4. Generate JWT token (user_id="admin", type="admin", 24h expiry)
5. Log admin login event
6. Return APIResponse{
    Message: "Admin login successful",
    Data: {
        jwt_access_token, token_expires_at
    }
}
```

**JWT Token Claims**:
```go
type JWTTokenClaims struct {
    UserID    string `json:"user_id"`     // User UUID or "admin"
    SessionID string `json:"session_id"`  // Session UUID (empty for admin)
    TokenType string `json:"token_type"`  // "portal" or "admin"
    jwt.RegisteredClaims
}
```

**JWT Middleware** (simplified):
```go
func ValidateJWT(requiredType string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Extract and parse token from "Authorization: Bearer <token>"
        // 2. Validate signature and expiration
        // 3. Check token_type matches (portal/admin)
        // 4. For portal: verify session still exists
        // 5. Store claims in context for handlers
        c.Next()
    }
}
```

#### F. Real IP Extraction (Trusted Proxy Support)

**Purpose**: Get real client IP when behind reverse proxy (Nginx, Cloudflare, etc.)

**Logic**:
```
1. Get connection IP from r.RemoteAddr
2. If trusted_proxy_config.enabled == false:
   → Return connection IP (done)
3. Check if connection IP is in trusted_proxy_ip_ranges
4. If NOT trusted proxy:
   → Return connection IP (ignore headers - prevent spoofing)
5. If trusted proxy:
   → Check headers in client_ip_header_priority order
   → Parse first valid IP found
   → For X-Forwarded-For: take FIRST IP (original client)
6. Return extracted IP or fallback to connection IP
```

**Security**: Only trust headers from configured trusted proxies. Prevents IP spoofing.

### 4. API Endpoints

#### Standard API Response Format
**All API endpoints return this consistent structure**:
```json
{
  "message": "Human-readable status message",
  "data": { /* actual response payload */ },
  "total_results": 10  // Optional: for list/paginated responses
}
```

**Error Response Format**:
```json
{
  "message": "Error description",
  "data": null,
  "error_code": "INVALID_CREDENTIALS"  // Optional: machine-readable error code
}
```

#### Portal Endpoints (Public - No Authentication Required for Login)
```
POST   /api/portal/login
    Request: { "username": "kevin", "password": "secret123" }
    Response: {
        "message": "Login successful",
        "data": {
            "session_id": "uuid",
            "jwt_access_token": "eyJ...",
            "token_expires_at": "2025-10-25T11:30:00Z",
            "session_info": {
                "username": "kevin",
                "authenticated_ip": "203.0.113.45",
                "expires_at": "2025-10-25T11:30:00Z",
                "auto_extend_enabled": true,
                "allowed_services": ["Minecraft Server", "Web Services"]
            }
        }
    }
    Rate Limit: 10/min per IP

GET    /api/portal/suggested-usernames
    Response: {
        "message": "Suggested usernames retrieved",
        "data": {
            "usernames": ["kevin", "alice"]
        },
        "total_results": 2
    }

GET    /api/portal/session/status
    Auth: Bearer token required
    Response: {
        "message": "Session active",
        "data": {
            "active": true,
            "expires_in_seconds": 3245,
            "authenticated_ip": "203.0.113.45",
            "allowed_services": ["Minecraft Server"],
            "auto_extend_enabled": true
        }
    }

POST   /api/portal/session/logout
    Auth: Bearer token required
    Response: {
        "message": "Session terminated successfully",
        "data": null
    }
```

#### Admin Endpoints (Protected - Require Admin JWT)
```
POST   /api/admin/login
    Request: { "admin_password": "secret" }
    Response: {
        "message": "Admin login successful",
        "data": {
            "jwt_access_token": "eyJ...",
            "token_expires_at": "2025-10-26T10:30:00Z"
        }
    }
    Rate Limit: 5/min per IP

GET    /api/admin/sessions
    Auth: Admin token required
    Response: {
        "message": "Active sessions retrieved",
        "data": {
            "sessions": [
                {
                    "session_id": "uuid",
                    "username": "kevin",
                    "authenticated_ip": "203.0.113.45",
                    "created_at": "2025-10-25T10:30:00Z",
                    "expires_at": "2025-10-25T11:30:00Z",
                    "allowed_services": ["service_id_1"]
                }
            ]
        },
        "total_results": 1
    }

DELETE /api/admin/sessions/{session_id}
    Auth: Admin token required
    Response: {
        "message": "Session terminated",
        "data": null
    }

GET    /api/admin/services/status
    Auth: Admin token required
    Response: {
        "message": "Service status retrieved",
        "data": {
            "services": [
                {
                    "service_id": "uuid",
                    "service_name": "Minecraft Server",
                    "enabled": true,
                    "active_connections": 5,
                    "listener_status": "running"
                }
            ]
        },
        "total_results": 1
    }

GET    /api/admin/config
    Auth: Admin token required
    Response: {
        "message": "Configuration retrieved",
        "data": {
            "session_config": {...},
            "protected_services": [...],
            "portal_user_accounts": [...]
        }
    }
    Note: Passwords are sanitized (replaced with "***")
```

#### Health Endpoint (Public)
```
GET    /health
    Response: {
        "message": "Service healthy",
        "data": {
            "status": "healthy",
            "version": "1.0.0",
            "uptime_seconds": 86400
        }
    }
```

### 5. Performance & Concurrency

**Key Patterns**:
- **Goroutine per connection**: Each TCP/UDP connection runs in own goroutine
- **sync.Map for sessions**: Lock-free reads (hot path)
- **sync.Pool for buffers**: Reuse 32KB buffers for TCP proxying
- **Atomic counters**: Track stats without locks
- **Channel semaphores**: Limit max connections per service

**IP Matching**:
```
1. Check exact IP in sync.Map → O(1) fast path
2. If not found, iterate CIDR ranges → O(n) but n is small (~10)
```

**Targets**:
- 10k+ concurrent connections on 1GB RAM
- <100ns session lookup
- <1μs IP check
- <10KB memory per connection

### 6. Security

**Critical Points**:
1. **Bcrypt passwords** (cost 10) - timing-safe comparison
2. **JWT tokens** signed with secret key from env
3. **Rate limiting**: 10/min portal login, 5/min admin login
4. **IP spoofing prevention**: Only trust headers from configured proxies
5. **Session IP binding**: Can't hijack session from different IP
6. **Per-user service restrictions**: `allowed_service_ids` enforced
7. **Config validation**: Port conflicts, bcrypt format, CIDR syntax
8. **Deploy behind HTTPS proxy** (Nginx/Caddy with TLS)

**Audit Logging**:
- Auth attempts (success/failure) with IP
- Session creation/termination
- Admin config changes
- Access denials

### 7. Logging

**Use structured JSON logging** (zerolog):
- **DEBUG**: Detailed flow (dev only)
- **INFO**: Auth events, config reload, session lifecycle
- **WARN**: Rate limit hits, DNS failures
- **ERROR**: Connection errors, config validation failures

**Future**: Prometheus metrics endpoint for monitoring

### 8. Deployment

#### Dockerfile
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o knock-knock ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/knock-knock .
COPY config.yml .
EXPOSE 80 443
CMD ["./knock-knock"]
```

**Docker Compose Example**:
```yaml
version: '3.8'
services:
  knock-knock:
    build: ./backend
    ports:
      - "80:80"
      - "443:443"
      - "25565:25565"  # Minecraft
    volumes:
      - ./config.yml:/app/config.yml
      - ./data:/app/data
    environment:
      - ADMIN_BCRYPT=${ADMIN_BCRYPT}
      - JWT_SECRET=${JWT_SECRET}
    restart: unless-stopped
```

### 9. Dependencies (Key Go Packages)

```go
// Core Web Framework
github.com/gin-gonic/gin                      // High-performance HTTP router
github.com/gin-contrib/cors                   // CORS middleware for Gin

// Authentication & Security
github.com/golang-jwt/jwt/v5                  // JWT token generation/validation
golang.org/x/crypto/bcrypt                    // Password hashing
golang.org/x/time/rate                        // Rate limiting

// Unique Identifiers
github.com/google/uuid                        // UUID v4 generation for sessions/services

// Configuration Management
gopkg.in/yaml.v3                              // YAML parsing for config.yml
github.com/joho/godotenv                      // .env file loading
github.com/fsnotify/fsnotify                  // File system watcher for hot-reload

// Logging
github.com/rs/zerolog                         // Structured, zero-allocation JSON logging
// Alternative: go.uber.org/zap               // Higher performance, more complex

// Network Utilities
net/netip                                     // Standard library: IP/CIDR parsing (Go 1.18+)
net/http/httputil                             // Standard library: ReverseProxy implementation

// Utilities (Optional)
github.com/patrickmn/go-cache                 // In-memory cache with expiration (for rate limiting)

// Testing
github.com/stretchr/testify                   // Assertion library for tests
github.com/stretchr/testify/mock             // Mock generation for interfaces
```

**Justifications**:
- **Gin**: Battle-tested, fastest Go router, excellent middleware ecosystem
- **zerolog**: Zero-allocation logging, crucial for high-performance proxy
- **net/netip**: Standard library (Go 1.18+), faster and safer than older net.IP
- **jwt/v5**: Most popular Go JWT library, actively maintained
- **fsnotify**: Cross-platform file watching for config hot-reload

### 10. Implementation Phases

**Phase 1: Foundation**
- Config loading (YAML + .env)
- API response wrapper
- Basic models

**Phase 2: Auth & Sessions**
- JWT manager
- Session manager (create, validate, cleanup)
- Portal/admin login endpoints

**Phase 3: IP Allowlist**
- IP matching (exact + CIDR)
- DNS resolver
- Real IP extraction

**Phase 4: Proxy Core**
- TCP proxy with IP filtering
- UDP proxy
- HTTP reverse proxy with headers

**Phase 5: Admin API**
- List/terminate sessions
- Service status
- Config retrieval

**Phase 6: Production**
- Docker setup
- Testing
- Documentation

## Technical Decisions

### Why In-Memory Sessions?
- **Performance**: Sub-microsecond lookups
- **Simplicity**: No external dependencies
- **Acceptable**: Sessions are ephemeral by design
- **Future**: Can add Redis persistence if needed

### Why Not iptables/nftables?
- **Portability**: Works on any OS, not just Linux
- **Simplicity**: No root/CAP_NET_ADMIN required
- **Flexibility**: Application-level filtering more dynamic
- **Docker-friendly**: No host network manipulation

### Protocol Choice
- **REST API**: Simple, well-understood, easy to debug
- **JWT**: Stateless admin auth, stateful portal sessions
- **YAML Config**: Human-readable, easy manual editing

## Performance Targets

- **Session lookup**: <100ns (in-memory map)
- **IP check**: <1μs (optimized matching)
- **Proxy overhead**: <5% latency increase
- **Concurrent connections**: 10k+ per service
- **Memory**: <100MB for 1000 active sessions

## Future Enhancements

- [ ] Redis session storage (optional)
- [ ] Prometheus metrics endpoint
- [ ] WebSocket support for real-time status
- [ ] GeoIP blocking
- [ ] 2FA support for portal logins
- [ ] Audit log for admin actions
- [ ] Rate limiting per user
- [ ] IPv6 NAT64/DNS64 handling
![alt text](image.png)