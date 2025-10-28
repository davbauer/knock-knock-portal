package config

// ApplicationConfig is the root configuration structure
type ApplicationConfig struct {
	SessionConfig        SessionConfiguration       `yaml:"session_config" json:"session_config"`
	NetworkAccessControl NetworkAccessControlConfig `yaml:"network_access_control" json:"network_access_control"`
	ProxyServerConfig    ProxyServerConfiguration   `yaml:"proxy_server_config" json:"proxy_server_config"`
	TrustedProxyConfig   TrustedProxyConfiguration  `yaml:"trusted_proxy_config" json:"trusted_proxy_config"`
	PortalUserAccounts   []PortalUserAccount        `yaml:"portal_user_accounts" json:"portal_user_accounts"`
	ProtectedServices    []ProtectedServiceConfig   `yaml:"protected_services" json:"protected_services"`
}

// SessionConfiguration defines session behavior
type SessionConfiguration struct {
	DefaultSessionDurationSeconds int  `yaml:"default_session_duration_seconds" json:"default_session_duration_seconds"`
	AutoExtendSessionOnConnection bool `yaml:"auto_extend_session_on_connection" json:"auto_extend_session_on_connection"`
	MaximumSessionDurationSeconds *int `yaml:"maximum_session_duration_seconds" json:"maximum_session_duration_seconds"` // nil = unlimited
	SessionCleanupIntervalSeconds int  `yaml:"session_cleanup_interval_seconds" json:"session_cleanup_interval_seconds"`
	MaxConcurrentSessions         int  `yaml:"max_concurrent_sessions" json:"max_concurrent_sessions"` // 0 = unlimited
}

// NetworkAccessControlConfig defines IP allowlist settings
type NetworkAccessControlConfig struct {
	BlockedIPAddresses         []string `yaml:"blocked_ip_addresses" json:"blocked_ip_addresses"`                       // Highest priority - blocks IPs and CIDR ranges
	AllowedDynamicDNSHostnames []string `yaml:"allowed_dynamic_dns_hostnames" json:"allowed_dynamic_dns_hostnames"`
	PermanentlyAllowedIPRanges []string `yaml:"permanently_allowed_ip_ranges" json:"permanently_allowed_ip_ranges"`
	DNSRefreshIntervalSeconds  int      `yaml:"dns_refresh_interval_seconds" json:"dns_refresh_interval_seconds"`
}

// ProxyServerConfiguration defines proxy server settings
type ProxyServerConfiguration struct {
	ListenAddress            string `yaml:"listen_address" json:"listen_address"`
	AdminAPIPort             int    `yaml:"admin_api_port" json:"admin_api_port"`
	ConnectionTimeoutSeconds int    `yaml:"connection_timeout_seconds" json:"connection_timeout_seconds"`
	MaxConnectionsPerService int    `yaml:"max_connections_per_service" json:"max_connections_per_service"`
	TCPBufferSizeBytes       int    `yaml:"tcp_buffer_size_bytes" json:"tcp_buffer_size_bytes"`
	UDPBufferSizeBytes       int    `yaml:"udp_buffer_size_bytes" json:"udp_buffer_size_bytes"`
	UDPSessionTimeoutSeconds int    `yaml:"udp_session_timeout_seconds" json:"udp_session_timeout_seconds"`
}

// TrustedProxyConfiguration defines trusted proxy settings for real IP extraction
type TrustedProxyConfiguration struct {
	Enabled                bool     `yaml:"enabled" json:"enabled"`
	TrustedProxyIPRanges   []string `yaml:"trusted_proxy_ip_ranges" json:"trusted_proxy_ip_ranges"`
	ClientIPHeaderPriority []string `yaml:"client_ip_header_priority" json:"client_ip_header_priority"`
}

// PortalUserAccount defines a user who can login to the portal
type PortalUserAccount struct {
	UserID                             string   `yaml:"user_id" json:"user_id"`
	Username                           string   `yaml:"username" json:"username"`
	DisplayUsernameInPublicSuggestions bool     `yaml:"display_username_in_public_login_suggestions" json:"display_username_in_public_login_suggestions"`
	BcryptHashedPassword               string   `yaml:"bcrypt_hashed_password" json:"bcrypt_hashed_password"`
	AllowedServiceIDs                  []string `yaml:"allowed_service_ids" json:"allowed_service_ids"` // Empty = all
	Notes                              string   `yaml:"notes" json:"notes"`
}

// ProtectedServiceConfig defines a service that requires authentication
type ProtectedServiceConfig struct {
	ServiceID            string              `yaml:"service_id" json:"service_id"`
	ServiceName          string              `yaml:"service_name" json:"service_name"`
	ProxyListenPortStart int                 `yaml:"proxy_listen_port_start" json:"proxy_listen_port_start"`
	ProxyListenPortEnd   int                 `yaml:"proxy_listen_port_end" json:"proxy_listen_port_end"`
	BackendTargetHost    string              `yaml:"backend_target_host" json:"backend_target_host"`
	BackendTargetPort    int                 `yaml:"backend_target_port" json:"backend_target_port"`
	TransportProtocol    string              `yaml:"transport_protocol" json:"transport_protocol"` // tcp | udp | both
	IsHTTPProtocol       bool                `yaml:"is_http_protocol" json:"is_http_protocol"`
	Enabled              bool                `yaml:"enabled" json:"enabled"`
	Description          string              `yaml:"description" json:"description"`
	HTTPConfig           *HTTPProtocolConfig `yaml:"http_config,omitempty" json:"http_config,omitempty"`
}

// HTTPProtocolConfig defines HTTP-specific configuration
type HTTPProtocolConfig struct {
	InjectHTTPRequestHeaders   map[string]string `yaml:"inject_http_request_headers" json:"inject_http_request_headers"`
	OverrideHTTPRequestHeaders map[string]string `yaml:"override_http_request_headers" json:"override_http_request_headers"`
	RemoveHTTPRequestHeaders   []string          `yaml:"remove_http_request_headers" json:"remove_http_request_headers"`
	InjectHTTPResponseHeaders  map[string]string `yaml:"inject_http_response_headers" json:"inject_http_response_headers"`
}
