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
		reason: string; // "session" | "ip_allowlist" | "not_allowed"
		username?: string; // Only present if authenticated via session
		authenticated_ips?: string[]; // Only present if session exists
		session_expires_in?: number; // Only present if session exists
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
						reason: 'session',
						username: sessionData.username,
						authenticated_ips: sessionData.authenticated_ips,
						session_expires_in: sessionData.expires_in_seconds
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
					reason: info.reason || 'not_allowed'
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
			if (status.reason === 'session') return status.username || 'Authenticated';
			return 'Access Allowed';
		}
		return 'No Access';
	}
</script>

<!-- IP Change Dialog -->
{#if status?.username}
<Dialog.Root open={showIPChangeDialog}>
	<Dialog.Backdrop class="fixed inset-0 bg-black/50 backdrop-blur-sm z-40" />
	<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<Dialog.Content
			class="bg-base-100 w-full max-w-md rounded-lg border border-base-300 p-6 shadow-xl"
		>
			<div class="mb-4 flex items-start gap-3">
				<div class="bg-warning/10 rounded-lg p-2">
					<Network class="text-warning h-6 w-6" />
				</div>
				<div class="flex-1">
					<Dialog.Title class="text-lg font-semibold text-base-content">
						Network Change Detected
					</Dialog.Title>
					<Dialog.Description class="mt-1 text-sm text-base-content/70">
						Your IP address has changed. Would you like to continue on this connection?
					</Dialog.Description>
				</div>
			</div>

			{#if status}
				<div class="mb-4 rounded-lg bg-base-200 p-3">
					<div class="mb-2 flex items-center justify-between">
						<span class="text-xs font-medium text-base-content/70">Previous IP:</span>
						<span class="font-mono text-sm text-base-content">{previousIP || 'Unknown'}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-xs font-medium text-base-content/70">Current IP:</span>
						<span class="font-mono text-sm font-semibold text-warning">
							{status.current_ip}
						</span>
					</div>
				</div>
			{/if}

			<div class="flex gap-2">
				<button
					onclick={() => (showIPChangeDialog = false)}
					class="flex-1 rounded-lg border border-base-300 px-4 py-2 text-sm font-medium text-base-content transition-colors hover:bg-base-200"
				>
					Cancel
				</button>
				<button
					onclick={addCurrentIP}
					disabled={isAddingIP}
					class="flex-1 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary/90 disabled:opacity-50"
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
			class="flex items-center gap-2 rounded-lg border border-base-300 bg-base-100 px-3 py-2 shadow-lg backdrop-blur-sm transition-colors hover:bg-base-100/90"
		>
			{#if isLoading}
				<div class="h-5 w-5 animate-spin rounded-full border-2 border-base-content/20 border-t-primary"></div>
			{:else}
				{@const Icon = getStatusIcon()}
				<Icon class={`h-5 w-5 ${getStatusColor()}`} />
			{/if}
			<span class="text-sm font-medium text-base-content">{getStatusText()}</span>
		</Popover.Trigger>

		<Popover.Positioner>
			<Popover.Content
				class="bg-base-100 z-50 w-80 rounded-lg border border-base-300 p-4 shadow-xl"
			>
				<Popover.Arrow>
					<Popover.ArrowTip class="border-base-300 border-l border-t" />
				</Popover.Arrow>

			{#if error}
				<div class="mb-4 rounded-lg bg-error/10 p-3 text-sm text-error">
					{error}
				</div>
			{/if}				{#if status}
					<div class="space-y-4">
					<!-- User Info (if authenticated) -->
					{#if status.username}
						<div>
							<div class="mb-1 text-xs font-medium uppercase tracking-wide text-base-content/70">
								Logged in as
							</div>
							<div class="flex items-center gap-2">
								<Shield class="h-4 w-4 text-success" />
								<span class="font-semibold text-base-content">{status.username}</span>
							</div>
						</div>
					{:else}
						<div>
							<div class="mb-1 text-xs font-medium uppercase tracking-wide text-base-content/70">
								Status
							</div>
							<div class="flex items-center gap-2">
								{#if status.allowed}
									<CheckCircle2 class="h-4 w-4 text-success" />
									<span class="font-medium text-success">IP Allowed</span>
								{:else}
									<Lock class="h-4 w-4 text-base-content/50" />
									<span class="font-medium text-base-content/70">Not Authenticated</span>
								{/if}
							</div>
						</div>
					{/if}

					<!-- Current IP Status -->
					<div>
						<div class="mb-2 text-xs font-medium uppercase tracking-wide text-base-content/70">
							Current Connection
						</div>
							<div class="flex items-center gap-2">
								{#if status.allowed}
									<CheckCircle2 class="h-4 w-4 text-success" />
									<span class="text-sm font-medium text-success">Connected</span>
								{:else}
									<AlertCircle class="h-4 w-4 text-base-content/50" />
									<span class="text-sm font-medium text-base-content/70">Not Allowed</span>
								{/if}
							</div>
						<div class="mt-2 rounded bg-base-200 px-2 py-1 font-mono text-sm text-base-content">
							{status.current_ip}
						</div>
					</div>

					<!-- Authenticated IPs (if session exists) -->
					{#if status.authenticated_ips && status.authenticated_ips.length > 0}
						<div>
							<div class="mb-2 text-xs font-medium uppercase tracking-wide text-base-content/70">
								Authenticated IPs ({status.authenticated_ips.length})
							</div>
							<div class="space-y-1">
								{#each status.authenticated_ips as ip}
									<div class="flex items-center gap-2">
										{#if ip === status.current_ip}
											<div class="h-2 w-2 rounded-full bg-success"></div>
										{:else}
											<div class="h-2 w-2 rounded-full bg-base-content/20"></div>
										{/if}
										<span class="rounded bg-base-200 px-2 py-0.5 font-mono text-xs text-base-content">{ip}</span>
									</div>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Session Expires (if session exists) -->
					{#if status.session_expires_in !== undefined}
						<div>
							<div class="mb-1 text-xs font-medium uppercase tracking-wide text-base-content/70">
								Session Expires
							</div>
							<div class="text-sm text-base-content">
								{Math.floor(status.session_expires_in / 60)} minutes
								</div>
							</div>
						{/if}

						<div class="border-t border-base-300 pt-3">
							<div class="text-xs text-base-content/70">
								{#if status.allowed}
									{#if status.reason === 'session'}
										Access granted via authenticated session
									{:else if status.reason === 'ip_allowlist' || status.reason === 'permanent_ip' || status.reason === 'dynamic_dns'}
										Access granted via IP allowlist
									{:else}
										Access granted
									{/if}
								{:else}
									Your IP is not in the allowlist. Login to gain access.
								{/if}
							</div>
						</div>
					</div>
				{/if}

				<Popover.CloseTrigger />
			</Popover.Content>
		</Popover.Positioner>
	</Popover.Root>
</div>
