<script lang="ts">
	import { onMount } from 'svelte';
	import { Activity, CheckCircle, XCircle, Loader, Server, Clock, Code } from 'lucide-svelte';
	import { API_BASE_URL } from '$lib/config';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let healthData = $state<any>(null);
	let isLoading = $state(true);
	let error = $state('');
	let lastChecked = $state<Date>(new Date());

	async function fetchHealth() {
		isLoading = true;
		error = '';

		try {
			const response = await fetch(`${API_BASE_URL}/api/health`);

			if (!response.ok) {
				throw new Error(`Health check failed: ${response.status}`);
			}

			healthData = await response.json();
			lastChecked = new Date();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to fetch health status';
		} finally {
			isLoading = false;
		}
	}

	function formatUptime(seconds: number): string {
		if (seconds < 60) return `${seconds}s`;
		if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
		const hours = Math.floor(seconds / 3600);
		const mins = Math.floor((seconds % 3600) / 60);
		return `${hours}h ${mins}m`;
	}

	onMount(() => {
		fetchHealth();
		// Refresh every 5 seconds
		const interval = setInterval(fetchHealth, 5000);
		return () => clearInterval(interval);
	});
</script>

<div class="mx-auto max-w-7xl px-4 py-8">
	<!-- Header -->
	<PageHeader
		title="System Health"
		subtitle="Real-time monitoring of backend services and API status"
		icon={Activity}
	/>

	{#if isLoading && !healthData}
		<!-- Loading State -->
		<div class="border-border bg-base-100 rounded-2xl border p-16 text-center shadow-sm">
			<Loader class="text-primary mx-auto h-12 w-12 animate-spin" />
			<p class="text-base-content mt-4 font-medium">Connecting to backend...</p>
			<p class="text-base-muted mt-1 text-sm">Please wait</p>
		</div>
	{:else if error}
		<!-- Error State -->
		<div class="border-error/30 bg-error/5 rounded-2xl border-2 p-8 shadow-sm">
			<div class="flex items-start gap-4">
				<div class="bg-error/10 rounded-full p-3">
					<XCircle class="text-error h-8 w-8" />
				</div>
				<div class="flex-1">
					<h3 class="text-error text-xl font-bold">Connection Failed</h3>
					<p class="text-error/90 mt-2">{error}</p>
					<p class="text-error/80 mt-2 text-sm">
						Make sure the backend server is running on <code
							class="bg-error/10 rounded px-1.5 py-0.5 font-mono text-xs"
							>http://127.0.0.1:8000</code
						>
					</p>
				</div>
			</div>
			<button
				onclick={fetchHealth}
				class="bg-error hover:bg-error-hover focus:ring-error focus:ring-offset-base-100 mt-6 rounded-lg px-6 py-2.5 text-sm font-semibold text-white transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2"
			>
				Retry Connection
			</button>
		</div>
	{:else if healthData}
		<div class="space-y-6">
			<!-- Main Status Card -->
			<div
				class="border-border bg-linear-to-br from-base-100 to-base-200 rounded-2xl border p-8 shadow-sm"
			>
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-4">
						<div class="bg-success/10 rounded-full p-4">
							<CheckCircle class="text-success h-10 w-10" />
						</div>
						<div>
							<div class="flex items-center gap-2">
								<h2 class="text-base-content text-3xl font-bold">
									{healthData.data.status === 'healthy'
										? 'All Systems Operational'
										: healthData.data.status}
								</h2>
								<div class="bg-success h-3 w-3 animate-pulse rounded-full"></div>
							</div>
							<p class="text-base-muted mt-1">Backend services are running smoothly</p>
						</div>
					</div>
					<div class="hidden text-right sm:block">
						<div class="text-base-muted text-sm font-medium">API Version</div>
						<div class="bg-primary/10 mt-1 inline-flex items-center gap-2 rounded-lg px-4 py-2">
							<Code class="text-primary h-4 w-4" />
							<span class="text-primary font-mono text-lg font-bold"
								>v{healthData.data.version}</span
							>
						</div>
					</div>
				</div>
			</div>

			<!-- Metrics Grid -->
			<div class="grid gap-6 sm:grid-cols-3">
				<!-- Status Badge -->
				<div class="border-border bg-base-100 rounded-xl border p-6 shadow-sm">
					<div class="mb-3 flex items-center gap-3">
						<Server class="text-base-muted h-5 w-5" />
						<div class="text-base-muted text-sm font-medium">Service Status</div>
					</div>
					<div class="flex items-center gap-2">
						<div class="bg-success h-2.5 w-2.5 rounded-full"></div>
						<div class="text-base-content text-2xl font-bold capitalize">
							{healthData.data.status}
						</div>
					</div>
				</div>

				<!-- Uptime -->
				<div class="border-border bg-base-100 rounded-xl border p-6 shadow-sm">
					<div class="mb-3 flex items-center gap-3">
						<Activity class="text-base-muted h-5 w-5" />
						<div class="text-base-muted text-sm font-medium">Uptime</div>
					</div>
					<div class="text-success text-2xl font-bold">
						{formatUptime(healthData.data.uptime_seconds)}
					</div>
					<div class="text-base-muted mt-1 text-xs">Since last restart</div>
				</div>

				<!-- Last Checked -->
				<div class="border-border bg-base-100 rounded-xl border p-6 shadow-sm">
					<div class="mb-3 flex items-center gap-3">
						<Clock class="text-base-muted h-5 w-5" />
						<div class="text-base-muted text-sm font-medium">Last Checked</div>
					</div>
					<div class="text-base-content text-2xl font-bold">
						{lastChecked.toLocaleTimeString()}
					</div>
					<div class="text-base-muted mt-1 text-xs">
						{lastChecked.toLocaleDateString()}
					</div>
				</div>
			</div>

			<!-- Response Details -->
			<details class="border-border bg-base-100 group rounded-xl border shadow-sm">
				<summary
					class="text-base-content hover:bg-base-200 flex cursor-pointer items-center justify-between p-6 font-medium transition-colors"
				>
					<span class="flex items-center gap-2">
						<Code class="text-base-muted h-5 w-5" />
						Raw API Response
					</span>
					<span class="text-base-muted text-xs group-open:hidden">Click to expand</span>
					<span class="text-base-muted hidden text-xs group-open:block">Click to collapse</span>
				</summary>
				<div class="border-border border-t p-6">
					<pre
						class="bg-base-200 text-base-content overflow-auto rounded-lg p-4 text-sm">{JSON.stringify(
							healthData,
							null,
							2
						)}</pre>
				</div>
			</details>

			<!-- Auto-refresh Footer -->
			<div class="bg-base-200 flex items-center justify-center gap-2 rounded-lg px-4 py-3">
				<div class="bg-primary h-2 w-2 animate-pulse rounded-full"></div>
				<p class="text-base-muted text-xs font-medium">Auto-refreshing every 5 seconds</p>
			</div>
		</div>
	{/if}
</div>
