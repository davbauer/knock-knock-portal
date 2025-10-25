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
		<div class="rounded-2xl border border-border bg-base-100 p-16 text-center shadow-sm">
			<Loader class="mx-auto h-12 w-12 animate-spin text-primary" />
			<p class="mt-4 font-medium text-base-content">Connecting to backend...</p>
			<p class="mt-1 text-sm text-base-muted">Please wait</p>
		</div>
	{:else if error}
		<!-- Error State -->
		<div class="rounded-2xl border-2 border-error/30 bg-error/5 p-8 shadow-sm">
			<div class="flex items-start gap-4">
				<div class="rounded-full bg-error/10 p-3">
					<XCircle class="h-8 w-8 text-error" />
				</div>
				<div class="flex-1">
					<h3 class="text-xl font-bold text-error">Connection Failed</h3>
					<p class="mt-2 text-error/90">{error}</p>
					<p class="mt-2 text-sm text-error/80">
						Make sure the backend server is running on <code class="rounded bg-error/10 px-1.5 py-0.5 font-mono text-xs">http://127.0.0.1:8000</code>
					</p>
				</div>
			</div>
			<button
				onclick={fetchHealth}
				class="mt-6 rounded-lg bg-error px-6 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-error-hover focus:outline-none focus:ring-2 focus:ring-error focus:ring-offset-2 focus:ring-offset-base-100"
			>
				Retry Connection
			</button>
		</div>
	{:else if healthData}
		<div class="space-y-6">
			<!-- Main Status Card -->
			<div class="rounded-2xl border border-border bg-linear-to-br from-base-100 to-base-200 p-8 shadow-sm">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-4">
						<div class="rounded-full bg-success/10 p-4">
							<CheckCircle class="h-10 w-10 text-success" />
						</div>
						<div>
							<div class="flex items-center gap-2">
								<h2 class="text-3xl font-bold text-base-content">
									{healthData.data.status === 'healthy' ? 'All Systems Operational' : healthData.data.status}
								</h2>
								<div class="h-3 w-3 animate-pulse rounded-full bg-success"></div>
							</div>
							<p class="mt-1 text-base-muted">
								Backend services are running smoothly
							</p>
						</div>
					</div>
					<div class="hidden sm:block text-right">
						<div class="text-sm font-medium text-base-muted">API Version</div>
						<div class="mt-1 inline-flex items-center gap-2 rounded-lg bg-primary/10 px-4 py-2">
							<Code class="h-4 w-4 text-primary" />
							<span class="font-mono text-lg font-bold text-primary">v{healthData.data.version}</span>
						</div>
					</div>
				</div>
			</div>

			<!-- Metrics Grid -->
			<div class="grid gap-6 sm:grid-cols-3">
				<!-- Status Badge -->
				<div class="rounded-xl border border-border bg-base-100 p-6 shadow-sm">
					<div class="flex items-center gap-3 mb-3">
						<Server class="h-5 w-5 text-base-muted" />
						<div class="text-sm font-medium text-base-muted">Service Status</div>
					</div>
					<div class="flex items-center gap-2">
						<div class="h-2.5 w-2.5 rounded-full bg-success"></div>
						<div class="text-2xl font-bold text-base-content capitalize">
							{healthData.data.status}
						</div>
					</div>
				</div>

				<!-- Uptime -->
				<div class="rounded-xl border border-border bg-base-100 p-6 shadow-sm">
					<div class="flex items-center gap-3 mb-3">
						<Activity class="h-5 w-5 text-base-muted" />
						<div class="text-sm font-medium text-base-muted">Uptime</div>
					</div>
					<div class="text-2xl font-bold text-success">
						{formatUptime(healthData.data.uptime_seconds)}
					</div>
					<div class="mt-1 text-xs text-base-muted">
						Since last restart
					</div>
				</div>

				<!-- Last Checked -->
				<div class="rounded-xl border border-border bg-base-100 p-6 shadow-sm">
					<div class="flex items-center gap-3 mb-3">
						<Clock class="h-5 w-5 text-base-muted" />
						<div class="text-sm font-medium text-base-muted">Last Checked</div>
					</div>
					<div class="text-2xl font-bold text-base-content">
						{lastChecked.toLocaleTimeString()}
					</div>
					<div class="mt-1 text-xs text-base-muted">
						{lastChecked.toLocaleDateString()}
					</div>
				</div>
			</div>

			<!-- Response Details -->
			<details class="group rounded-xl border border-border bg-base-100 shadow-sm">
				<summary class="flex cursor-pointer items-center justify-between p-6 font-medium text-base-content transition-colors hover:bg-base-200">
					<span class="flex items-center gap-2">
						<Code class="h-5 w-5 text-base-muted" />
						Raw API Response
					</span>
					<span class="text-xs text-base-muted group-open:hidden">Click to expand</span>
					<span class="text-xs text-base-muted hidden group-open:block">Click to collapse</span>
				</summary>
				<div class="border-t border-border p-6">
					<pre class="overflow-auto rounded-lg bg-base-200 p-4 text-sm text-base-content">{JSON.stringify(healthData, null, 2)}</pre>
				</div>
			</details>

			<!-- Auto-refresh Footer -->
			<div class="flex items-center justify-center gap-2 rounded-lg bg-base-200 px-4 py-3">
				<div class="h-2 w-2 animate-pulse rounded-full bg-primary"></div>
				<p class="text-xs font-medium text-base-muted">
					Auto-refreshing every 5 seconds
				</p>
			</div>
		</div>
	{/if}
</div>
