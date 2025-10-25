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
	<div class="rounded-xl border border-border bg-base-100 p-6">
		<h3 class="text-lg font-semibold text-base-content mb-4">Session Settings</h3>
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			<Field.Root>
				<Field.Label class="text-sm font-medium text-base-content mb-2">Default Session Duration (seconds)</Field.Label>
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
					<NumberInput.Input class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary" />
				</NumberInput.Root>
				<Field.HelperText class="text-xs text-base-muted mt-1">How long sessions last (60-86400 seconds)</Field.HelperText>
			</Field.Root>

			<Field.Root>
				<Field.Label class="text-sm font-medium text-base-content mb-2">Session Cleanup Interval (seconds)</Field.Label>
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
					<NumberInput.Input class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary" />
				</NumberInput.Root>
				<Field.HelperText class="text-xs text-base-muted mt-1">How often to check for expired sessions</Field.HelperText>
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
				<Switch.Control class="w-11 h-6 rounded-full bg-border data-[state=checked]:bg-primary transition-colors relative">
					<Switch.Thumb class="absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white transition-transform data-[state=checked]:translate-x-5" />
				</Switch.Control>
				<Switch.Label class="text-sm font-medium text-base-content cursor-pointer">Auto-extend session on connection</Switch.Label>
				<Switch.HiddenInput />
			</Switch.Root>
			<p class="text-xs text-base-muted mt-1 ml-14">Automatically extend session duration when user makes a connection</p>
		</div>
		</div>
	</div>

	<!-- Proxy Server Configuration -->
	<div class="rounded-xl border border-border bg-base-100 p-6">
		<h3 class="text-lg font-semibold text-base-content mb-4">Proxy Server Settings</h3>
		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
		<Field.Root>
			<Field.Label class="text-sm font-medium text-base-content mb-2">Listen Address</Field.Label>
			<input
				type="text"
				value={config.proxy_server_config.listen_address}
				oninput={(e) => {
					configStore.updateConfig((cfg) => {
						cfg.proxy_server_config.listen_address = e.currentTarget.value;
					});
				}}
				class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary"
			/>
			<Field.HelperText class="text-xs text-base-muted mt-1">IP address to bind proxy server (e.g., 0.0.0.0)</Field.HelperText>
		</Field.Root>

		<Field.Root>
			<Field.Label class="text-sm font-medium text-base-content mb-2">Admin API Port</Field.Label>
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
				<NumberInput.Input class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary" />
			</NumberInput.Root>
			<Field.HelperText class="text-xs text-base-muted mt-1">Port for admin API (1024-65535)</Field.HelperText>
		</Field.Root>
		
		<Field.Root>
			<Field.Label class="text-sm font-medium text-base-content mb-2">Connection Timeout (seconds)</Field.Label>
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
				<NumberInput.Input class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary" />
			</NumberInput.Root>
			<Field.HelperText class="text-xs text-base-muted mt-1">Timeout for establishing connections</Field.HelperText>
		</Field.Root>

		<Field.Root>
			<Field.Label class="text-sm font-medium text-base-content mb-2">Max Connections Per Service</Field.Label>
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
					<NumberInput.Input class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary" />
				</NumberInput.Root>
				<Field.HelperText class="text-xs text-base-muted mt-1">Maximum concurrent connections per service</Field.HelperText>
			</Field.Root>
		</div>
	</div>

	<!-- Network Access Control -->
	<div class="rounded-xl border border-border bg-base-100 p-6">
		<h3 class="text-lg font-semibold text-base-content mb-4">Network Access Control</h3>
		<div class="space-y-4">
		<Field.Root>
			<Field.Label class="text-sm font-medium text-base-content mb-2">Permanently Allowed IP Ranges</Field.Label>
			<textarea
				value={config.network_access_control.permanently_allowed_ip_ranges.join('\n')}
				rows={3}
				placeholder="e.g., 10.0.0.0/8, 192.168.1.0/24"
				class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary font-mono"
				oninput={(e) => {
					configStore.updateConfig((cfg) => {
						const value = e.currentTarget.value;
						cfg.network_access_control.permanently_allowed_ip_ranges = value.split('\n').filter(s => s.trim());
					});
				}}
			></textarea>
			<Field.HelperText class="text-xs text-base-muted mt-1">One IP range per line (CIDR notation)</Field.HelperText>
		</Field.Root>

		<Field.Root>
			<Field.Label class="text-sm font-medium text-base-content mb-2">Allowed Dynamic DNS Hostnames</Field.Label>
			<textarea
				value={config.network_access_control.allowed_dynamic_dns_hostnames.join('\n')}
				rows={3}
				placeholder="e.g., home.example.com, vpn.example.net"
				class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary font-mono"
				oninput={(e) => {
					configStore.updateConfig((cfg) => {
						const value = e.currentTarget.value;
						cfg.network_access_control.allowed_dynamic_dns_hostnames = value.split('\n').filter(s => s.trim());
					});
				}}
			></textarea>
			<Field.HelperText class="text-xs text-base-muted mt-1">One hostname per line</Field.HelperText>
		</Field.Root>			<Field.Root>
				<Field.Label class="text-sm font-medium text-base-content mb-2">DNS Refresh Interval (seconds)</Field.Label>
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
					<NumberInput.Input class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary" />
				</NumberInput.Root>
				<Field.HelperText class="text-xs text-base-muted mt-1">How often to resolve dynamic DNS hostnames</Field.HelperText>
			</Field.Root>
		</div>
	</div>

	<!-- Trusted Proxy Configuration -->
	<div class="rounded-xl border border-border bg-base-100 p-6">
		<h3 class="text-lg font-semibold text-base-content mb-4">Trusted Proxy Settings</h3>
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
				<Switch.Control class="w-11 h-6 rounded-full bg-border data-[state=checked]:bg-primary transition-colors relative">
					<Switch.Thumb class="absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white transition-transform data-[state=checked]:translate-x-5" />
				</Switch.Control>
				<Switch.Label class="text-sm font-medium text-base-content cursor-pointer">Enable Trusted Proxy</Switch.Label>
				<Switch.HiddenInput />
			</Switch.Root>
			<p class="text-xs text-base-muted mt-1 ml-14">Trust X-Forwarded-For headers from specified proxies</p>
		</div>			{#if config.trusted_proxy_config.enabled}
				<Field.Root>
				<Field.Label class="text-sm font-medium text-base-content mb-2">Trusted Proxy IP Ranges</Field.Label>
				<textarea
					value={config.trusted_proxy_config.trusted_proxy_ip_ranges.join('\n')}
					rows={2}
					placeholder="e.g., 10.0.0.1/32"
					class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary font-mono"
					oninput={(e) => {
						configStore.updateConfig((cfg) => {
							const value = e.currentTarget.value;
							cfg.trusted_proxy_config.trusted_proxy_ip_ranges = value.split('\n').filter(s => s.trim());
						});
					}}
				></textarea>
					<Field.HelperText class="text-xs text-base-muted mt-1">One IP range per line (CIDR notation)</Field.HelperText>
				</Field.Root>
			{/if}
		</div>
	</div>
</div>
