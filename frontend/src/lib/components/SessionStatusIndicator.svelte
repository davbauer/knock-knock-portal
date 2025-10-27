<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Popover } from '@ark-ui/svelte';
	import { Network, Wifi, WifiOff, AlertCircle, CheckCircle2 } from 'lucide-svelte';
	import { API_BASE_URL } from '$lib/config';
	import { Dialog } from '@ark-ui/svelte';

	interface SessionStatusResponse {
		session_id: string;
		username: string;
		authenticated_ips: string[];
		current_ip: string;
		current_ip_allowed: boolean;
		expires_in_seconds: number;
		active: boolean;
	}

	let sessionStatus = $state<SessionStatusResponse | null>(null);
	let previousIP = $state<string | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let showIPChangeDialog = $state(false);
	let isAddingIP = $state(false);
	let intervalId: number | null = null;

	async function fetchStatus() {
		// Only fetch if tab is visible
		if (document.visibilityState !== 'visible') {
			return;
		}

		const token = localStorage.getItem('portal_token');
		if (!token) return;

		try {
			const response = await fetch(`${API_BASE_URL}/api/portal/session/status`, {
				headers: {
					Authorization: `Bearer ${token}`
				}
			});

			if (!response.ok) {
				throw new Error('Failed to fetch session status');
			}

			const data = await response.json();
			const newStatus = data.data.session as SessionStatusResponse;

			// Detect IP change
			if (previousIP && newStatus.current_ip !== previousIP && !newStatus.current_ip_allowed) {
				showIPChangeDialog = true;
			}

			sessionStatus = newStatus;
			previousIP = newStatus.current_ip;
			error = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load status';
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
		if (!sessionStatus) return 'text-base-content/50';
		return sessionStatus.current_ip_allowed ? 'text-success' : 'text-warning';
	}

	function getStatusIcon() {
		if (!sessionStatus) return WifiOff;
		return sessionStatus.current_ip_allowed ? CheckCircle2 : AlertCircle;
	}
</script>

<!-- IP Change Dialog -->
<Dialog.Root open={showIPChangeDialog}>
	<Dialog.Backdrop class="fixed inset-0 bg-black/50 backdrop-blur-sm" />
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

			{#if sessionStatus}
				<div class="bg-base-200 mb-4 rounded-lg p-3">
					<div class="mb-2 flex items-center justify-between">
						<span class="text-base-content/70 text-xs font-medium">Previous IP:</span>
						<span class="text-base-content font-mono text-sm">{previousIP || 'Unknown'}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-base-content/70 text-xs font-medium">Current IP:</span>
						<span class="text-warning font-mono text-sm font-semibold">
							{sessionStatus.current_ip}
						</span>
					</div>
				</div>
			{/if}

			<div class="flex gap-2">
				<button
					onclick={() => (showIPChangeDialog = false)}
					class="hover:bg-base-200 border-base-300 flex-1 rounded-lg border px-4 py-2 text-sm font-medium transition-colors"
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

<!-- Status Indicator Popover -->
<Popover.Root>
	<Popover.Trigger
		class="hover:bg-base-200 flex items-center gap-2 rounded-lg px-3 py-2 transition-colors"
	>
		{#if isLoading}
			<div
				class="border-base-content/20 border-t-primary h-5 w-5 animate-spin rounded-full border-2"
			></div>
		{:else}
			{@const Icon = getStatusIcon()}
			<Icon class={`h-5 w-5 ${getStatusColor()}`} />
		{/if}
		{#if sessionStatus}
			<span class="text-base-content text-sm font-medium">{sessionStatus.username}</span>
		{/if}
	</Popover.Trigger>

	<Popover.Positioner>
		<Popover.Content class="bg-base-100 border-base-300 z-50 w-80 rounded-lg border p-4 shadow-xl">
			<Popover.Arrow>
				<Popover.ArrowTip class="border-base-300 border-l border-t" />
			</Popover.Arrow>

			{#if error}
				<div class="bg-error/10 text-error mb-4 rounded-lg p-3 text-sm">
					{error}
				</div>
			{/if}

			{#if sessionStatus}
				<div class="space-y-4">
					<!-- User Info -->
					<div>
						<div class="text-base-content/70 mb-1 text-xs font-medium uppercase tracking-wide">
							Logged in as
						</div>
						<div class="text-base-content font-semibold">{sessionStatus.username}</div>
					</div>

					<!-- Current IP Status -->
					<div>
						<div class="text-base-content/70 mb-2 text-xs font-medium uppercase tracking-wide">
							Current Connection
						</div>
						<div class="flex items-center gap-2">
							{#if sessionStatus.current_ip_allowed}
								<CheckCircle2 class="text-success h-4 w-4" />
								<span class="text-success text-sm font-medium">Connected</span>
							{:else}
								<AlertCircle class="text-warning h-4 w-4" />
								<span class="text-warning text-sm font-medium">New IP Detected</span>
							{/if}
						</div>
						<div class="bg-base-200 mt-2 rounded px-2 py-1 font-mono text-sm">
							{sessionStatus.current_ip}
						</div>
					</div>

					<!-- Authenticated IPs -->
					<div>
						<div class="text-base-content/70 mb-2 text-xs font-medium uppercase tracking-wide">
							Authenticated IPs ({sessionStatus.authenticated_ips.length})
						</div>
						<div class="space-y-1">
							{#each sessionStatus.authenticated_ips as ip}
								<div class="flex items-center gap-2">
									{#if ip === sessionStatus.current_ip}
										<div class="bg-success h-2 w-2 rounded-full"></div>
									{:else}
										<div class="bg-base-content/20 h-2 w-2 rounded-full"></div>
									{/if}
									<span class="bg-base-200 rounded px-2 py-0.5 font-mono text-xs">{ip}</span>
								</div>
							{/each}
						</div>
					</div>

					<!-- Session Expires -->
					<div>
						<div class="text-base-content/70 mb-1 text-xs font-medium uppercase tracking-wide">
							Session Expires
						</div>
						<div class="text-base-content text-sm">
							{Math.floor(sessionStatus.expires_in_seconds / 60)} minutes
						</div>
					</div>
				</div>
			{/if}

			<Popover.CloseTrigger />
		</Popover.Content>
	</Popover.Positioner>
</Popover.Root>
