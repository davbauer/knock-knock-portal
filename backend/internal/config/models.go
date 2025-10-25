package config

// ApplicationConfig is the root configuration structure
type ApplicationConfig struct {
	SessionConfig        SessionConfiguration       `yaml:"session_config"`
	NetworkAccessControl NetworkAccessControlConfig `yaml:"network_access_control"`
	ProxyServerConfig    ProxyServerConfiguration   `yaml:"proxy_server_config"`
	TrustedProxyConfig   TrustedProxyConfiguration  `yaml:"trusted_proxy_config"`
	PortalUserAccounts   []PortalUserAccount        `yaml:"portal_user_accounts"`
	ProtectedServices    []ProtectedServiceConfig   `yaml:"protected_services"`
}

// SessionConfiguration defines session behavior
type SessionConfiguration struct {
	DefaultSessionDurationSeconds int  `yaml:"default_session_duration_seconds"`
	AutoExtendSessionOnConnection bool `yaml:"auto_extend_session_on_connection"`
	MaximumSessionDurationSeconds *int `yaml:"maximum_session_duration_seconds"` // nil = unlimited
	SessionCleanupIntervalSeconds int  `yaml:"session_cleanup_interval_seconds"`
}

// NetworkAccessControlConfig defines IP allowlist settings
type NetworkAccessControlConfig struct {
	AllowedDynamicDNSHostnames   []string `yaml:"allowed_dynamic_dns_hostnames"`
	PermanentlyAllowedIPRanges   []string `yaml:"permanently_allowed_ip_ranges"`
	DNSRefreshIntervalSeconds    int      `yaml:"dns_refresh_interval_seconds"`
}

// ProxyServerConfiguration defines proxy server settings
type ProxyServerConfiguration struct {
	ListenAddress            string `yaml:"listen_address"`
	AdminAPIPort             int    `yaml:"admin_api_port"`
	ConnectionTimeoutSeconds int    `yaml:"connection_timeout_seconds"`
	MaxConnectionsPerService int    `yaml:"max_connections_per_service"`
	TCPBufferSizeBytes       int    `yaml:"tcp_buffer_size_bytes"`
	UDPBufferSizeBytes       int    `yaml:"udp_buffer_size_bytes"`
	UDPSessionTimeoutSeconds int    `yaml:"udp_session_timeout_seconds"`
}

// TrustedProxyConfiguration defines trusted proxy settings for real IP extraction
type TrustedProxyConfiguration struct {
	Enabled                bool     `yaml:"enabled"`
	TrustedProxyIPRanges   []string `yaml:"trusted_proxy_ip_ranges"`
	ClientIPHeaderPriority []string `yaml:"client_ip_header_priority"`
}

// PortalUserAccount defines a user who can login to the portal
type PortalUserAccount struct {
	UserID                               string   `yaml:"user_id"`
	Username                             string   `yaml:"username"`
	DisplayUsernameInPublicSuggestions   bool     `yaml:"display_username_in_public_login_suggestions"`
	BcryptHashedPassword                 string   `yaml:"bcrypt_hashed_password"`
	AllowedServiceIDs                    []string `yaml:"allowed_service_ids"` // Empty = all
	Notes                                string   `yaml:"notes"`
}

// ProtectedServiceConfig defines a service that requires authentication
type ProtectedServiceConfig struct {
	ServiceID             string              `yaml:"service_id"`
	ServiceName           string              `yaml:"service_name"`
	ProxyListenPortStart  int                 `yaml:"proxy_listen_port_start"`
	ProxyListenPortEnd    int                 `yaml:"proxy_listen_port_end"`
	BackendTargetHost     string              `yaml:"backend_target_host"`
	BackendTargetPortStart int                `yaml:"backend_target_port_start"`
	BackendTargetPortEnd  int                 `yaml:"backend_target_port_end"`
	TransportProtocol     string              `yaml:"transport_protocol"` // tcp | udp | both
	IsHTTPProtocol        bool                `yaml:"is_http_protocol"`
	Enabled               bool                `yaml:"enabled"`
	Description           string              `yaml:"description"`
	HTTPConfig            *HTTPProtocolConfig `yaml:"http_config,omitempty"`
}

// HTTPProtocolConfig defines HTTP-specific configuration
type HTTPProtocolConfig struct {
	InjectHTTPRequestHeaders   map[string]string `yaml:"inject_http_request_headers"`
	OverrideHTTPRequestHeaders map[string]string `yaml:"override_http_request_headers"`
	RemoveHTTPRequestHeaders   []string          `yaml:"remove_http_request_headers"`
	InjectHTTPResponseHeaders  map[string]string `yaml:"inject_http_response_headers"`
}
