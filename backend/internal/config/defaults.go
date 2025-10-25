package config

// GetDefaultConfig returns default configuration values
func GetDefaultConfig() *ApplicationConfig {
	defaultMaxDuration := 86400 // 24 hours

	return &ApplicationConfig{
		SessionConfig: SessionConfiguration{
			DefaultSessionDurationSeconds: 3600,
			AutoExtendSessionOnConnection: true,
			MaximumSessionDurationSeconds: &defaultMaxDuration,
			SessionCleanupIntervalSeconds: 60,
		},
		NetworkAccessControl: NetworkAccessControlConfig{
			AllowedDynamicDNSHostnames: []string{},
			PermanentlyAllowedIPRanges: []string{},
			DNSRefreshIntervalSeconds:  300,
		},
		ProxyServerConfig: ProxyServerConfiguration{
			ListenAddress:            "0.0.0.0",
			AdminAPIPort:             8000,
			ConnectionTimeoutSeconds: 30,
			MaxConnectionsPerService: 1000,
			TCPBufferSizeBytes:       32768,
			UDPBufferSizeBytes:       65507,
			UDPSessionTimeoutSeconds: 300,
		},
		TrustedProxyConfig: TrustedProxyConfiguration{
			Enabled:                false,
			TrustedProxyIPRanges:   []string{},
			ClientIPHeaderPriority: []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"},
		},
		PortalUserAccounts: []PortalUserAccount{},
		ProtectedServices:  []ProtectedServiceConfig{},
	}
}
