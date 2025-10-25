<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { slide } from 'svelte/transition';
	import {
		Shield,
		LogOut,
		Clock,
		CheckCircle2,
		AlertCircle,
		Network,
		Globe,
		Server,
		Zap,
		Calendar,
		User,
		Activity
	} from 'lucide-svelte';
	import { API_BASE_URL } from '$lib/config';
	import PageHeader from '$lib/components/PageHeader.svelte';

	interface SessionInfo {
		session_id: string;
		username: string;
		user_id: string;
		authenticated_ip: string;
		created_at: string;
		last_activity_at: string;
		expires_at: string;
		expires_in_seconds: number;
		auto_extend_enabled: boolean;
		allowed_service_ids: string[];
		allowed_services: string[];
		active: boolean;
	}

	let sessionInfo = $state<SessionInfo | null>(null);
	let isLoading = $state(true);
	let error = $state('');
	let timeRemaining = $state('');
	let timeRemainingPercent = $state(100);
	let isExpiringSoon = $state(false);

	async function fetchSessionStatus() {
		const token = localStorage.getItem('portal_token');

		if (!token) {
			goto('/');
			return;
		}

		try {
			const response = await fetch(`${API_BASE_URL}/api/portal/session/status`, {
				headers: {
					Authorization: `Bearer ${token}`
				}
			});

			if (response.status === 401) {
				localStorage.removeItem('portal_token');
				localStorage.removeItem('portal_session');
				goto('/');
				return;
			}

			if (!response.ok) {
				throw new Error('Failed to fetch session status');
			}

			const data = await response.json();
			sessionInfo = data.data.session;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load session';
		} finally {
			isLoading = false;
		}
	}

	async function handleLogout() {
		const token = localStorage.getItem('portal_token');

		if (!token) {
			goto('/');
			return;
		}

		try {
			await fetch(`${API_BASE_URL}/api/portal/session/logout`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${token}`
				}
			});
		} catch (err) {
			console.error('Logout error:', err);
		} finally {
			localStorage.removeItem('portal_token');
			localStorage.removeItem('portal_session');
			goto('/');
		}
	}

	function updateTimeRemaining() {
		if (!sessionInfo) return;

		const now = new Date().getTime();
		const expiry = new Date(sessionInfo.expires_at).getTime();
		const diff = expiry - now;

		if (diff <= 0) {
			timeRemaining = 'Expired';
			timeRemainingPercent = 0;
			setTimeout(() => {
				handleLogout();
			}, 1000);
			return;
		}

		const hours = Math.floor(diff / (1000 * 60 * 60));
		const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
		const seconds = Math.floor((diff % (1000 * 60)) / 1000);

		if (hours > 0) {
			timeRemaining = `${hours}h ${minutes}m ${seconds}s`;
		} else if (minutes > 0) {
			timeRemaining = `${minutes}m ${seconds}s`;
		} else {
			timeRemaining = `${seconds}s`;
		}

		// Calculate percentage (assuming max session is 24 hours)
		const maxDuration = 24 * 60 * 60 * 1000;
		timeRemainingPercent = Math.min(100, (diff / maxDuration) * 100);

		// Check if expiring soon (less than 5 minutes)
		isExpiringSoon = diff < 5 * 60 * 1000;
	}

	function formatDateTime(dateString: string): string {
		return new Date(dateString).toLocaleString('en-US', {
			weekday: 'short',
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	onMount(() => {
		// Try to load from localStorage first for instant display
		const stored = localStorage.getItem('portal_session');
		if (stored) {
			try {
				sessionInfo = JSON.parse(stored);
			} catch (e) {
				console.error('Failed to parse stored session:', e);
			}
		}

		fetchSessionStatus();

		// Update time remaining every second
		const interval = setInterval(updateTimeRemaining, 1000);
		updateTimeRemaining();

		return () => clearInterval(interval);
	});
</script>

<div class="mx-auto max-w-7xl px-4 py-8">
	<!-- Header -->
	<PageHeader
		title="Portal Dashboard"
		subtitle={sessionInfo ? `Welcome, ${sessionInfo.username}` : 'Loading...'}
		icon={Shield}
	>
		<button
			onclick={handleLogout}
			class="bg-error hover:bg-error-hover focus:ring-error focus:ring-offset-base-100 flex items-center justify-center gap-2 rounded-lg px-5 py-2.5 text-sm font-semibold text-white shadow-sm transition-all hover:shadow-md focus:outline-none focus:ring-2 focus:ring-offset-2"
		>
			<LogOut class="h-4 w-4" />
			Logout
		</button>
	</PageHeader>

	{#if isLoading}
		<div class="border-border bg-base-100 rounded-2xl border p-16 text-center shadow-sm">
			<div
				class="border-primary mx-auto h-12 w-12 animate-spin rounded-full border-4 border-t-transparent"
			></div>
			<p class="text-base-content mt-4 font-medium">Loading session...</p>
		</div>
	{:else if error}
		<div
			transition:slide={{ duration: 300 }}
			class="border-error/30 bg-error/5 flex items-start gap-3 rounded-lg border p-6"
		>
			<AlertCircle class="text-error mt-0.5 h-6 w-6 shrink-0" />
			<div>
				<h3 class="text-error font-semibold">Error Loading Session</h3>
				<p class="text-error mt-1 text-sm">{error}</p>
				<button
					onclick={fetchSessionStatus}
					class="bg-error hover:bg-error-hover mt-4 rounded-lg px-4 py-2 text-sm font-semibold text-white"
				>
					Retry
				</button>
			</div>
		</div>
	{:else if sessionInfo}
		<div class="space-y-6">
			<!-- Hero Status Banner -->
			<div
				class="border-success/30 bg-linear-to-br from-success/10 via-success/5 overflow-hidden rounded-2xl border-2 to-transparent shadow-lg"
			>
				<div class="p-8">
					<div class="flex items-start justify-between">
						<div class="flex items-start gap-4">
							<div class="bg-success/20 rounded-xl p-3">
								<CheckCircle2 class="text-success h-8 w-8" />
							</div>
							<div>
								<h2 class="text-base-content mb-1 text-2xl font-bold">Session Active</h2>
								<p class="text-base-muted text-sm">
									Your authentication is valid and services are accessible
								</p>
							</div>
						</div>
						<div class="bg-success/20 rounded-lg px-3 py-1">
							<div class="flex items-center gap-2">
								<Activity class="text-success h-4 w-4 animate-pulse" />
								<span class="text-success text-xs font-bold uppercase">Live</span>
							</div>
						</div>
					</div>

					<!-- Time Remaining Progress -->
					<div class="mt-6">
						<div class="mb-3 flex items-end justify-between">
							<div>
								<p class="text-base-muted mb-1 text-xs font-medium uppercase tracking-wide">
									Time Remaining
								</p>
								<p
									class="font-mono text-4xl font-bold {isExpiringSoon
										? 'text-error animate-pulse'
										: 'text-base-content'}"
								>
									{timeRemaining}
								</p>
							</div>
							{#if sessionInfo.auto_extend_enabled}
								<div class="bg-primary/10 flex items-center gap-2 rounded-lg px-3 py-2">
									<Zap class="text-primary h-5 w-5" />
									<span class="text-primary text-sm font-semibold">Auto-Extend Enabled</span>
								</div>
							{/if}
						</div>
						<div class="bg-base-200 h-3 overflow-hidden rounded-full">
							<div
								class="h-full transition-all duration-1000 {isExpiringSoon
									? 'bg-error'
									: 'bg-success'}"
								style="width: {timeRemainingPercent}%"
							></div>
						</div>
						{#if isExpiringSoon}
							<p class="text-error mt-2 text-xs font-medium">
								⚠️ Your session is expiring soon. Save your work!
							</p>
						{/if}
					</div>
				</div>
			</div>

			<!-- Session Details Grid -->
			<div class="grid gap-6 lg:grid-cols-2">
				<!-- Session Information Card -->
				<div class="border-border bg-base-100 overflow-hidden rounded-2xl border shadow-sm">
					<div class="border-border border-b px-6 py-4">
						<h3 class="text-base-content flex items-center gap-2 font-semibold">
							<Shield class="h-5 w-5" />
							Session Information
						</h3>
					</div>
					<div class="divide-border divide-y p-6">
						<!-- Username -->
						<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
							<div class="flex items-center gap-3">
								<div class="bg-primary/10 rounded-lg p-2">
									<User class="text-primary h-5 w-5" />
								</div>
								<div>
									<p class="text-base-muted text-xs font-medium uppercase tracking-wide">
										Username
									</p>
									<p class="text-base-content mt-0.5 font-semibold">
										{sessionInfo.username}
									</p>
								</div>
							</div>
						</div>

						<!-- Session ID -->
						<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
							<div class="flex items-center gap-3">
								<div class="bg-primary/10 rounded-lg p-2">
									<Shield class="text-primary h-5 w-5" />
								</div>
								<div>
									<p class="text-base-muted text-xs font-medium uppercase tracking-wide">
										Session ID
									</p>
									<p class="text-base-content mt-0.5 font-mono text-xs font-medium">
										{sessionInfo.session_id.split('-')[0]}...
									</p>
								</div>
							</div>
						</div>

						<!-- Authenticated IP -->
						<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
							<div class="flex items-center gap-3">
								<div class="bg-primary/10 rounded-lg p-2">
									<Globe class="text-primary h-5 w-5" />
								</div>
								<div>
									<p class="text-base-muted text-xs font-medium uppercase tracking-wide">
										Authenticated IP
									</p>
									<p class="text-base-content mt-0.5 font-mono text-sm font-semibold">
										{sessionInfo.authenticated_ip}
									</p>
								</div>
							</div>
						</div>

						<!-- Created At -->
						<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
							<div class="flex items-center gap-3">
								<div class="bg-primary/10 rounded-lg p-2">
									<Calendar class="text-primary h-5 w-5" />
								</div>
								<div>
									<p class="text-base-muted text-xs font-medium uppercase tracking-wide">
										Session Created
									</p>
									<p class="text-base-content mt-0.5 text-sm font-medium">
										{formatDateTime(sessionInfo.created_at)}
									</p>
								</div>
							</div>
						</div>

						<!-- Last Activity -->
						<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
							<div class="flex items-center gap-3">
								<div class="bg-primary/10 rounded-lg p-2">
									<Activity class="text-primary h-5 w-5" />
								</div>
								<div>
									<p class="text-base-muted text-xs font-medium uppercase tracking-wide">
										Last Activity
									</p>
									<p class="text-base-content mt-0.5 text-sm font-medium">
										{formatDateTime(sessionInfo.last_activity_at)}
									</p>
								</div>
							</div>
						</div>

						<!-- Session Expires -->
						<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
							<div class="flex items-center gap-3">
								<div class="bg-primary/10 rounded-lg p-2">
									<Clock class="text-primary h-5 w-5" />
								</div>
								<div>
									<p class="text-base-muted text-xs font-medium uppercase tracking-wide">
										Session Expires
									</p>
									<p class="text-base-content mt-0.5 text-sm font-medium">
										{formatDateTime(sessionInfo.expires_at)}
									</p>
								</div>
							</div>
						</div>

						<!-- Auto-Extend Status -->
						<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
							<div class="flex items-center gap-3">
								<div class="bg-primary/10 rounded-lg p-2">
									<Zap class="text-primary h-5 w-5" />
								</div>
								<div>
									<p class="text-base-muted text-xs font-medium uppercase tracking-wide">
										Auto-Extend on Connection
									</p>
									<p class="text-base-content mt-0.5 text-sm font-semibold">
										{#if sessionInfo.auto_extend_enabled}
											<span class="text-success flex items-center gap-1">
												<CheckCircle2 class="h-4 w-4" />
												Enabled
											</span>
										{:else}
											<span class="text-base-muted flex items-center gap-1">
												<AlertCircle class="h-4 w-4" />
												Disabled
											</span>
										{/if}
									</p>
								</div>
							</div>
						</div>
					</div>
				</div>

				<!-- Allowed Services Card -->
				<div class="border-border bg-base-100 overflow-hidden rounded-2xl border shadow-sm">
					<div class="border-border border-b px-6 py-4">
						<h3 class="text-base-content flex items-center gap-2 font-semibold">
							<Network class="h-5 w-5" />
							Accessible Services
						</h3>
					</div>

					<div class="p-6">
						{#if sessionInfo.allowed_services.length === 0}
							<div class="bg-primary/5 rounded-xl p-6 text-center">
								<div
									class="bg-primary/10 mx-auto mb-3 flex h-16 w-16 items-center justify-center rounded-xl"
								>
									<Server class="text-primary h-8 w-8" />
								</div>
								<h4 class="text-base-content mb-1 font-semibold">All Services Available</h4>
								<p class="text-base-muted text-sm">
									You have unrestricted access to all protected services
								</p>
							</div>
						{:else}
							<div class="space-y-2">
								{#each sessionInfo.allowed_services as service}
									<div
										class="border-border bg-base-200/50 hover:border-primary/50 hover:bg-primary/5 group flex items-center gap-3 rounded-lg border p-3 transition-all hover:shadow-md"
									>
										<div
											class="bg-primary/10 group-hover:bg-primary/20 rounded-lg p-2 transition-colors"
										>
											<Server class="text-primary h-5 w-5" />
										</div>
										<div class="flex-1">
											<p class="text-base-content font-medium">{service}</p>
										</div>
										<div class="bg-success/10 rounded-full px-2.5 py-1">
											<span class="text-success text-xs font-semibold">Active</span>
										</div>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
