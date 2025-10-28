<script lang="ts">
	import type { Config } from './types';
	import { Switch, Field, NumberInput } from '@ark-ui/svelte';
	import { configStore } from './configStore.svelte';

	interface Props {
		config: Config;
	}

	let { config }: Props = $props();
</script>

<div class="space-y-6">
	<!-- Session Configuration -->
	<div class="border-border bg-base-100 rounded-xl border p-6">
		<h3 class="text-base-content mb-4 text-lg font-semibold">Session Settings</h3>
		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium"
					>Default Session Duration (seconds)</Field.Label
				>
				<NumberInput.Root
					value={String(config.session_config.default_session_duration_seconds)}
					onValueChange={(details) => {
						configStore.updateConfig((cfg) => {
							cfg.session_config.default_session_duration_seconds = Number(details.value);
						});
					}}
					min={60}
					max={86400}
					class="w-full"
				>
					<NumberInput.Input
						class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
					/>
				</NumberInput.Root>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>How long sessions last (60-86400 seconds)</Field.HelperText
				>
			</Field.Root>

			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium"
					>Session Cleanup Interval (seconds)</Field.Label
				>
				<NumberInput.Root
					value={String(config.session_config.session_cleanup_interval_seconds)}
					onValueChange={(details) => {
						configStore.updateConfig((cfg) => {
							cfg.session_config.session_cleanup_interval_seconds = Number(details.value);
						});
					}}
					min={10}
					max={3600}
					class="w-full"
				>
					<NumberInput.Input
						class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
					/>
				</NumberInput.Root>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>How often to check for expired sessions</Field.HelperText
				>
			</Field.Root>

			<div class="md:col-span-2">
				<Switch.Root
					checked={config.session_config.auto_extend_session_on_connection}
					onCheckedChange={(details) => {
						configStore.updateConfig((cfg) => {
							cfg.session_config.auto_extend_session_on_connection = details.checked;
						});
					}}
					class="flex items-center gap-3"
				>
					<Switch.Control
						class="bg-border data-[state=checked]:bg-primary relative h-6 w-11 rounded-full transition-colors"
					>
						<Switch.Thumb
							class="absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform data-[state=checked]:translate-x-5"
						/>
					</Switch.Control>
					<Switch.Label class="text-base-content cursor-pointer text-sm font-medium"
						>Auto-extend session on connection</Switch.Label
					>
					<Switch.HiddenInput />
				</Switch.Root>
				<p class="text-base-muted ml-14 mt-1 text-xs">
					Automatically extend session duration when user makes a connection
				</p>
			</div>
		</div>
	</div>

	<!-- Proxy Server Configuration -->
	<div class="border-border bg-base-100 rounded-xl border p-6">
		<h3 class="text-base-content mb-4 text-lg font-semibold">Proxy Server Settings</h3>
		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium">Listen Address</Field.Label>
				<input
					type="text"
					value={config.proxy_server_config.listen_address}
					oninput={(e) => {
						configStore.updateConfig((cfg) => {
							cfg.proxy_server_config.listen_address = e.currentTarget.value;
						});
					}}
					class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
				/>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>IP address to bind proxy server (e.g., 0.0.0.0)</Field.HelperText
				>
			</Field.Root>

			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium">Admin API Port</Field.Label>
				<NumberInput.Root
					value={String(config.proxy_server_config.admin_api_port)}
					onValueChange={(details) => {
						configStore.updateConfig((cfg) => {
							cfg.proxy_server_config.admin_api_port = Number(details.value);
						});
					}}
					min={1024}
					max={65535}
					class="w-full"
				>
					<NumberInput.Input
						class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
					/>
				</NumberInput.Root>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>Port for admin API (1024-65535)</Field.HelperText
				>
			</Field.Root>

			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium"
					>Connection Timeout (seconds)</Field.Label
				>
				<NumberInput.Root
					value={String(config.proxy_server_config.connection_timeout_seconds)}
					onValueChange={(details) => {
						configStore.updateConfig((cfg) => {
							cfg.proxy_server_config.connection_timeout_seconds = Number(details.value);
						});
					}}
					min={1}
					max={300}
					class="w-full"
				>
					<NumberInput.Input
						class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
					/>
				</NumberInput.Root>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>Timeout for establishing connections</Field.HelperText
				>
			</Field.Root>

			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium"
					>Max Connections Per Service</Field.Label
				>
				<NumberInput.Root
					value={String(config.proxy_server_config.max_connections_per_service)}
					onValueChange={(details) => {
						configStore.updateConfig((cfg) => {
							cfg.proxy_server_config.max_connections_per_service = Number(details.value);
						});
					}}
					min={1}
					max={10000}
					class="w-full"
				>
					<NumberInput.Input
						class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
					/>
				</NumberInput.Root>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>Maximum concurrent connections per service</Field.HelperText
				>
			</Field.Root>
		</div>
	</div>

	<!-- Network Access Control -->
	<div class="border-border bg-base-100 rounded-xl border p-6">
		<h3 class="text-base-content mb-4 text-lg font-semibold">Network Access Control</h3>
		<div class="space-y-4">
			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium"
					>Blocked IP Addresses</Field.Label
				>
				<textarea
					value={config.network_access_control.blocked_ip_addresses?.join('\n') || ''}
					rows={3}
					placeholder="e.g., 192.0.2.100, 198.51.100.0/24"
					class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
					oninput={(e) => {
						configStore.updateConfig((cfg) => {
							const value = e.currentTarget.value;
							cfg.network_access_control.blocked_ip_addresses = value
								.split('\n')
								.filter((s) => s.trim());
						});
					}}
				></textarea>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>One IP or CIDR range per line (highest priority - blocks all access)</Field.HelperText
				>
			</Field.Root>

			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium"
					>Permanently Allowed IP Ranges</Field.Label
				>
				<textarea
					value={config.network_access_control.permanently_allowed_ip_ranges.join('\n')}
					rows={3}
					placeholder="e.g., 10.0.0.0/8, 192.168.1.0/24"
					class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
					oninput={(e) => {
						configStore.updateConfig((cfg) => {
							const value = e.currentTarget.value;
							cfg.network_access_control.permanently_allowed_ip_ranges = value
								.split('\n')
								.filter((s) => s.trim());
						});
					}}
				></textarea>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>One IP range per line (CIDR notation)</Field.HelperText
				>
			</Field.Root>

			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium"
					>Allowed Dynamic DNS Hostnames</Field.Label
				>
				<textarea
					value={config.network_access_control.allowed_dynamic_dns_hostnames.join('\n')}
					rows={3}
					placeholder="e.g., home.example.com, vpn.example.net"
					class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
					oninput={(e) => {
						configStore.updateConfig((cfg) => {
							const value = e.currentTarget.value;
							cfg.network_access_control.allowed_dynamic_dns_hostnames = value
								.split('\n')
								.filter((s) => s.trim());
						});
					}}
				></textarea>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>One hostname per line</Field.HelperText
				>
			</Field.Root>
			<Field.Root>
				<Field.Label class="text-base-content mb-2 text-sm font-medium"
					>DNS Refresh Interval (seconds)</Field.Label
				>
				<NumberInput.Root
					value={String(config.network_access_control.dns_refresh_interval_seconds)}
					onValueChange={(details) => {
						configStore.updateConfig((cfg) => {
							cfg.network_access_control.dns_refresh_interval_seconds = Number(details.value);
						});
					}}
					min={30}
					max={3600}
					class="w-full"
				>
					<NumberInput.Input
						class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
					/>
				</NumberInput.Root>
				<Field.HelperText class="text-base-muted mt-1 text-xs"
					>How often to resolve dynamic DNS hostnames</Field.HelperText
				>
			</Field.Root>
		</div>
	</div>

	<!-- Trusted Proxy Configuration -->
	<div class="border-border bg-base-100 rounded-xl border p-6">
		<h3 class="text-base-content mb-4 text-lg font-semibold">Reverse Proxy Security</h3>
		<div class="space-y-4">
			<div>
				<Switch.Root
					checked={config.trusted_proxy_config.enabled}
					onCheckedChange={(details) => {
						configStore.updateConfig((cfg) => {
							cfg.trusted_proxy_config.enabled = details.checked;
						});
					}}
					class="flex items-center gap-3"
				>
					<Switch.Control
						class="bg-border data-[state=checked]:bg-primary relative h-6 w-11 rounded-full transition-colors"
					>
						<Switch.Thumb
							class="absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform data-[state=checked]:translate-x-5"
						/>
					</Switch.Control>
					<Switch.Label class="text-base-content cursor-pointer text-sm font-medium"
						>Enable Trusted Proxy Detection</Switch.Label
					>
					<Switch.HiddenInput />
				</Switch.Root>
				<p class="text-base-muted ml-14 mt-1 text-xs">
					<strong>When ENABLED:</strong> Trusts X-Forwarded-For headers from IPs in the trusted
					range below, showing real client IPs.<br />
					<strong>When DISABLED:</strong> Ignores all proxy headers (shows proxy IP instead of client
					IP) to prevent IP spoofing.
				</p>
			</div>
			{#if config.trusted_proxy_config.enabled}
				<Field.Root>
					<Field.Label class="text-base-content mb-2 text-sm font-medium"
						>Trusted Proxy IP Ranges</Field.Label
					>
					<textarea
						value={config.trusted_proxy_config.trusted_proxy_ip_ranges.join('\n')}
						rows={3}
						placeholder="172.16.0.0/12 (Docker networks)&#10;10.0.0.0/8 (Private network)&#10;Traefik/Nginx container IP"
						class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
						oninput={(e) => {
							configStore.updateConfig((cfg) => {
								const value = e.currentTarget.value;
								cfg.trusted_proxy_config.trusted_proxy_ip_ranges = value
									.split('\n')
									.filter((s) => s.trim());
							});
						}}
					></textarea>
					<Field.HelperText class="text-base-muted mt-1 text-xs"
						>One IP/CIDR range per line. For Docker: use 172.16.0.0/12 to trust all Docker networks.
						Check connection info popup for untrusted proxy warnings.</Field.HelperText
					>
				</Field.Root>
			{/if}
		</div>
	</div>
</div>
