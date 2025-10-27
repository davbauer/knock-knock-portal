<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import { Popover } from '@ark-ui/svelte';
	import { Network, Wifi, WifiOff, AlertCircle, CheckCircle2, Shield, Lock } from 'lucide-svelte';
	import { API_BASE_URL } from '$lib/config';
	import { Dialog } from '@ark-ui/svelte';

	interface IPAllowStatus {
		current_ip: string;
		allowed: boolean;
		access_method: string;
		access_description: string;
		username?: string;
		authenticated_ips?: string[];
		session_expires_in?: number;
		services?: ServiceAccessInfo[];
		total_services?: number;
		session_active?: boolean;
		session_username?: string;
		proxy_warning?: string;
	}

	interface ServiceAccessInfo {
		service_id: string;
		service_name: string;
		description: string;
		access_granted: boolean;
		access_reasons: AccessReason[];
		access_denied_reason?: string;
	}

	interface AccessReason {
		method: string; // "permanent_ip_range" | "dynamic_dns_hostname" | "authenticated_session"
		description: string;
	}

	let status = $state<IPAllowStatus | null>(null);
	let previousIP = $state<string | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let showIPChangeDialog = $state(false);
	let isAddingIP = $state(false);
	let intervalId: number | null = null;

	async function fetchStatus() {
		// Only run in browser
		if (!browser) return;

		// Only fetch if tab is visible
		if (document.visibilityState !== 'visible') {
			return;
		}

		const token = localStorage.getItem('portal_token');

		try {
			// Try authenticated endpoint first
			if (token) {
				const response = await fetch(`${API_BASE_URL}/api/portal/session/status`, {
					headers: {
						Authorization: `Bearer ${token}`
					}
				});

				if (response.ok) {
					const data = await response.json();
					const sessionData = data.data.session;

					const newStatus: IPAllowStatus = {
						current_ip: sessionData.current_ip,
						allowed: sessionData.current_ip_allowed,
						access_method: 'authenticated_session',
						access_description: 'Access granted via authenticated session',
						username: sessionData.username,
						authenticated_ips: sessionData.authenticated_ips,
						session_expires_in: sessionData.expires_in_seconds,
						services: sessionData.services || [],
						total_services: sessionData.total_services || 0,
						session_active: true,
						session_username: sessionData.username
					};

					// Detect IP change
					if (previousIP && newStatus.current_ip !== previousIP && !newStatus.allowed) {
						showIPChangeDialog = true;
					}

					status = newStatus;
					previousIP = newStatus.current_ip;
					error = null;
					isLoading = false;
					return;
				}
			}

			// Fallback to connection-info endpoint (works without auth)
			const connectionResponse = await fetch(`${API_BASE_URL}/api/connection-info`);
			if (connectionResponse.ok) {
				const connectionData = await connectionResponse.json();
				const info = connectionData.data;

				status = {
					current_ip: info.client_ip || 'Unknown',
					allowed: info.allowed || false,
					access_method: info.access_method || 'not_allowed',
					access_description: info.access_description || '',
					proxy_warning: info.proxy_warning
				};
				error = null;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load status';
			// Don't set status to error state, keep last known state
		} finally {
			isLoading = false;
		}
	}

	async function addCurrentIP() {
		const token = localStorage.getItem('portal_token');
		if (!token) return;

		isAddingIP = true;
		try {
			const response = await fetch(`${API_BASE_URL}/api/portal/session/add-ip`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${token}`
				}
			});

			if (!response.ok) {
				throw new Error('Failed to add IP to session');
			}

			// Refresh status
			await fetchStatus();
			showIPChangeDialog = false;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to add IP';
		} finally {
			isAddingIP = false;
		}
	}

	onMount(() => {
		// Only run in browser
		if (!browser) return;

		// Initial fetch
		fetchStatus();

		// Set up interval for auto-refresh every 10 seconds
		intervalId = window.setInterval(() => {
			fetchStatus();
		}, 10000);

		// Listen for visibility changes
		const handleVisibilityChange = () => {
			if (document.visibilityState === 'visible') {
				fetchStatus();
			}
		};
		document.addEventListener('visibilitychange', handleVisibilityChange);

		return () => {
			if (intervalId) {
				clearInterval(intervalId);
			}
			document.removeEventListener('visibilitychange', handleVisibilityChange);
		};
	});

	onDestroy(() => {
		if (intervalId) {
			clearInterval(intervalId);
		}
	});

	function getStatusColor() {
		if (!status) return 'text-base-content/50';
		// Green if allowed, grey if not allowed (never red)
		return status.allowed ? 'text-success' : 'text-base-content/50';
	}

	function getStatusIcon() {
		if (!status) return WifiOff;
		if (status.allowed) return CheckCircle2;
		return AlertCircle;
	}

	function getStatusText() {
		if (!status) return 'Checking...';
		if (status.allowed) {
			if (status.access_method === 'authenticated_session' && status.username) {
				return status.username;
			}
			return 'Access Allowed';
		}
		return 'No Access';
	}
</script>

<!-- IP Change Dialog -->
{#if status?.username}
	<Dialog.Root open={showIPChangeDialog}>
		<Dialog.Backdrop class="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm" />
		<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<Dialog.Content
				class="bg-base-100 border-base-300 w-full max-w-md rounded-lg border p-6 shadow-xl"
			>
				<div class="mb-4 flex items-start gap-3">
					<div class="bg-warning/10 rounded-lg p-2">
						<Network class="text-warning h-6 w-6" />
					</div>
					<div class="flex-1">
						<Dialog.Title class="text-base-content text-lg font-semibold">
							Network Change Detected
						</Dialog.Title>
						<Dialog.Description class="text-base-content/70 mt-1 text-sm">
							Your IP address has changed. Would you like to continue on this connection?
						</Dialog.Description>
					</div>
				</div>

				{#if status}
					<div class="bg-base-200 mb-4 rounded-lg p-3">
						<div class="mb-2 flex items-center justify-between">
							<span class="text-base-content/70 text-xs font-medium">Previous IP:</span>
							<span class="text-base-content font-mono text-sm">{previousIP || 'Unknown'}</span>
						</div>
						<div class="flex items-center justify-between">
							<span class="text-base-content/70 text-xs font-medium">Current IP:</span>
							<span class="text-warning font-mono text-sm font-semibold">
								{status.current_ip}
							</span>
						</div>
					</div>
				{/if}

				<div class="flex gap-2">
					<button
						onclick={() => (showIPChangeDialog = false)}
						class="border-base-300 text-base-content hover:bg-base-200 flex-1 rounded-lg border px-4 py-2 text-sm font-medium transition-colors"
					>
						Cancel
					</button>
					<button
						onclick={addCurrentIP}
						disabled={isAddingIP}
						class="bg-primary hover:bg-primary/90 flex-1 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50"
					>
						{isAddingIP ? 'Adding...' : 'Continue'}
					</button>
				</div>

				<Dialog.CloseTrigger
					class="text-base-content/50 hover:text-base-content absolute right-4 top-4 transition-colors"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<line x1="18" y1="6" x2="6" y2="18"></line>
						<line x1="6" y1="6" x2="18" y2="18"></line>
					</svg>
				</Dialog.CloseTrigger>
			</Dialog.Content>
		</Dialog.Positioner>
	</Dialog.Root>
{/if}

<!-- Fixed Status Indicator in Top-Right Corner -->
<div class="fixed right-4 top-4 z-30">
	<Popover.Root>
		<Popover.Trigger
			class="border-base-300 bg-base-100 hover:bg-base-100/90 flex items-center gap-2 rounded-lg border px-3 py-2 shadow-lg backdrop-blur-sm transition-colors"
		>
			{#if isLoading}
				<div
					class="border-base-content/20 border-t-primary h-5 w-5 animate-spin rounded-full border-2"
				></div>
			{:else}
				{@const Icon = getStatusIcon()}
				<Icon class={`h-5 w-5 ${getStatusColor()}`} />
			{/if}
			<span class="text-base-content text-sm font-medium">{getStatusText()}</span>
		</Popover.Trigger>

		<Popover.Positioner>
			<Popover.Content
				class="bg-base-100 border-base-300 z-50 w-80 rounded-lg border p-4 shadow-xl"
			>
				<Popover.Arrow>
					<Popover.ArrowTip class="border-base-300 border-l border-t" />
				</Popover.Arrow>

				{#if error}
					<div class="bg-error/10 text-error mb-4 rounded-lg p-3 text-sm">
						{error}
					</div>
				{/if}
				{#if status}
					<div class="space-y-4">
						<!-- User Info (if authenticated) -->
						{#if status.username}
							<div>
								<div class="text-base-content/70 mb-1 text-xs font-medium uppercase tracking-wide">
									Logged in as
								</div>
								<div class="flex items-center gap-2">
									<Shield class="text-success h-4 w-4" />
									<span class="text-base-content font-semibold">{status.username}</span>
								</div>
							</div>
						{:else}
							<div>
								<div class="text-base-content/70 mb-1 text-xs font-medium uppercase tracking-wide">
									Status
								</div>
								<div class="flex items-center gap-2">
									{#if status.allowed}
										<CheckCircle2 class="text-success h-4 w-4" />
										<span class="text-success font-medium">IP Allowed</span>
									{:else}
										<Lock class="text-base-content/50 h-4 w-4" />
										<span class="text-base-content/70 font-medium">Not Authenticated</span>
									{/if}
								</div>
							</div>
						{/if}

						<!-- Current IP Status -->
						<div>
							<div class="text-base-content/70 mb-2 text-xs font-medium uppercase tracking-wide">
								Current Connection
							</div>
							
						<!-- Proxy Warning -->
						{#if status.proxy_warning}
							<div class="bg-warning/10 text-warning mb-3 rounded-lg p-3">
								<div class="mb-1 flex items-center gap-2">
									<AlertCircle class="h-4 w-4 shrink-0" />
									<span class="text-sm font-semibold">Proxy Configuration</span>
								</div>
								<div class="text-xs leading-relaxed opacity-90">
									{status.proxy_warning}
								</div>
							</div>
						{/if}							<div class="flex items-center gap-2">
								{#if status.allowed}
									<CheckCircle2 class="text-success h-4 w-4" />
									<span class="text-success text-sm font-medium">Connected</span>
								{:else}
									<AlertCircle class="text-base-content/50 h-4 w-4" />
									<span class="text-base-content/70 text-sm font-medium">Not Allowed</span>
								{/if}
							</div>
							<div class="bg-base-200 text-base-content mt-2 rounded px-2 py-1 font-mono text-sm">
								{status.current_ip}
							</div>
						</div>

						<!-- Authenticated IPs (if session exists) -->
						{#if status.authenticated_ips && status.authenticated_ips.length > 0}
							<div>
								<div class="text-base-content/70 mb-2 text-xs font-medium uppercase tracking-wide">
									Authenticated IPs ({status.authenticated_ips.length})
								</div>
								<div class="space-y-1">
									{#each status.authenticated_ips as ip}
										<div class="flex items-center gap-2">
											{#if ip === status.current_ip}
												<div class="bg-success h-2 w-2 rounded-full"></div>
											{:else}
												<div class="bg-base-content/20 h-2 w-2 rounded-full"></div>
											{/if}
											<span
												class="bg-base-200 text-base-content rounded px-2 py-0.5 font-mono text-xs"
												>{ip}</span
											>
										</div>
									{/each}
								</div>
							</div>
						{/if}

						<!-- Session Expires (if session exists) -->
						{#if status.session_expires_in !== undefined}
							<div>
								<div class="text-base-content/70 mb-1 text-xs font-medium uppercase tracking-wide">
									Session Expires
								</div>
								<div class="text-base-content text-sm">
									{Math.floor(status.session_expires_in / 60)} minutes
								</div>
							</div>
						{/if}

						<!-- Access Reason -->
						<div class="border-base-300 border-t pt-3">
							<div class="text-base-content/70 mb-1 text-xs font-medium uppercase tracking-wide">
								Access Method
							</div>
							<div class="text-base-content/70 text-xs">
								{status.access_description}
							</div>
							{#if status.access_method === 'permanent_ip_range'}
								<div class="bg-primary/10 text-primary mt-2 rounded px-2 py-1 text-xs">
									✓ Permanent IP Range
								</div>
							{:else if status.access_method === 'dynamic_dns_hostname'}
								<div class="bg-primary/10 text-primary mt-2 rounded px-2 py-1 text-xs">
									✓ Dynamic DNS Hostname
								</div>
							{:else if status.access_method === 'authenticated_session'}
								<div class="bg-success/10 text-success mt-2 rounded px-2 py-1 text-xs">
									✓ Authenticated Session
								</div>
							{/if}
						</div>

						<!-- Service Access Details -->
						{#if status.services && status.services.length > 0}
							<div class="border-base-300 border-t pt-3">
								<div class="text-base-content/70 mb-2 text-xs font-medium uppercase tracking-wide">
									Service Access ({status.total_services} services)
								</div>
								<div class="max-h-64 space-y-2 overflow-y-auto">
									{#each status.services as service}
										<div class="border-base-300 bg-base-100 rounded border p-2">
											<div class="flex items-start justify-between">
												<div class="flex-1">
													<div class="text-base-content text-sm font-medium">
														{service.service_name}
													</div>
													{#if service.description}
														<div class="text-base-content/60 text-xs">
															{service.description}
														</div>
													{/if}
												</div>
												<div class="ml-2">
													{#if service.access_granted}
														<span class="bg-success/20 text-success rounded px-2 py-0.5 text-xs">
															Allowed
														</span>
													{:else}
														<span
															class="bg-base-300 text-base-content/50 rounded px-2 py-0.5 text-xs"
														>
															Denied
														</span>
													{/if}
												</div>
											</div>

											<!-- Access Reasons -->
											{#if service.access_granted && service.access_reasons.length > 0}
												<div class="border-base-300/50 mt-2 space-y-1 border-t pt-2">
													{#each service.access_reasons as reason}
														<div class="flex items-start gap-1.5 text-xs">
															<span class="text-success mt-0.5">✓</span>
															<div class="flex-1">
																<div class="text-base-content/70 font-medium">
																	{#if reason.method === 'permanent_ip_range'}
																		Permanent IP
																	{:else if reason.method === 'dynamic_dns_hostname'}
																		Dynamic DNS
																	{:else if reason.method === 'authenticated_session'}
																		Session
																	{:else}
																		{reason.method}
																	{/if}
																</div>
																<div class="text-base-content/50">
																	{reason.description}
																</div>
															</div>
														</div>
													{/each}
												</div>
											{/if}

											<!-- Denied Reason -->
											{#if !service.access_granted && service.access_denied_reason}
												<div
													class="border-base-300/50 text-base-content/50 mt-2 border-t pt-2 text-xs"
												>
													{service.access_denied_reason}
												</div>
											{/if}
										</div>
									{/each}
								</div>
							</div>
						{/if}
					</div>
				{/if}

				<Popover.CloseTrigger />
			</Popover.Content>
		</Popover.Positioner>
	</Popover.Root>
</div>
