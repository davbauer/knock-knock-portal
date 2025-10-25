export interface Session {
	session_id: string;
	username: string;
	user_id: string;
	authenticated_ips: string[];
	created_at: string;
	expires_at: string;
	allowed_services: string[];
}

export interface Config {
	session_config: {
		default_session_duration_seconds: number;
		auto_extend_session_on_connection: boolean;
		maximum_session_duration_seconds: number | null;
		session_cleanup_interval_seconds: number;
	};
	network_access_control: {
		allowed_dynamic_dns_hostnames: string[];
		permanently_allowed_ip_ranges: string[];
		dns_refresh_interval_seconds: number;
	};
	proxy_server_config: {
		listen_address: string;
		admin_api_port: number;
		connection_timeout_seconds: number;
		max_connections_per_service: number;
		tcp_buffer_size_bytes: number;
		udp_buffer_size_bytes: number;
		udp_session_timeout_seconds: number;
	};
	trusted_proxy_config: {
		enabled: boolean;
		trusted_proxy_ip_ranges: string[];
		client_ip_header_priority: string[];
	};
	portal_user_accounts: PortalUser[];
	protected_services: ProtectedService[];
}

export interface PortalUser {
	user_id: string;
	username: string;
	display_username_in_public_login_suggestions: boolean;
	bcrypt_hashed_password: string;
	allowed_service_ids: string[];
	notes: string;
}

export interface ProtectedService {
	service_id: string;
	service_name: string;
	proxy_listen_port_start: number;
	proxy_listen_port_end: number;
	backend_target_host: string;
	backend_target_port: number;
	transport_protocol: string;
	is_http_protocol: boolean;
	enabled: boolean;
	description: string;
	http_config: HTTPConfig | null;
}

export interface HTTPConfig {
	inject_http_request_headers?: Record<string, string>;
	override_http_request_headers?: Record<string, string>;
	remove_http_request_headers?: string[];
	inject_http_response_headers?: Record<string, string>;
}
