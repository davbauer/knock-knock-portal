<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { slide } from 'svelte/transition';
	import {
		Users,
		Settings,
		LogOut,
		Shield,
		Save,
		X,
		Plus,
		Trash2,
		Check,
		Download,
		Upload,
		Copy,
		Network,
		CircleCheck,
		CircleAlert,
		TriangleAlert,
		Info
	} from 'lucide-svelte';
	import { API_BASE_URL } from '$lib/config';
	import { Tabs, Dialog, Field, Toast, Toaster } from '@ark-ui/svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import type { Session, Config, Connection } from './types';
	import ConfigUsers from './ConfigUsers.svelte';
	import ConfigProtectedServices from './ConfigProtectedServices.svelte';
	import ConfigAdvanced from './ConfigAdvanced.svelte';
	import ActiveConnections from './ActiveConnections.svelte';
	import ActiveUsers from './ActiveUsers.svelte';
	import { configStore } from './configStore.svelte';
	import { toaster } from './toastStore.svelte';

	let refreshInterval: number | null = null;

	let sessions = $state<Session[]>([]);
	let isLoadingSessions = $state(true);
	let sessionsError = $state('');

	let connections = $state<Connection[]>([]);
	let isLoadingConnections = $state(true);
	let connectionsError = $state('');

	let showImportDialog = $state(false);
	let importJsonText = $state('');
	let importError = $state('');

	let sessionToTerminate = $state<Session | null>(null);
	let showTerminateDialog = $state(false);

	let connectionToTerminate = $state<Connection | null>(null);
	let showTerminateConnectionDialog = $state(false);

	// Get initial tabs from URL or defaults
	let currentMainTab = $state($page.url.searchParams.get('tab') || 'connections');
	let currentConfigTab = $state($page.url.searchParams.get('config_tab') || 'users');

	// Update URL when tabs change
	function updateMainTab(tab: string) {
		currentMainTab = tab;
		const url = new URL(window.location.href);
		url.searchParams.set('tab', tab);
		if (tab !== 'configuration') {
			url.searchParams.delete('config_tab');
		} else {
			url.searchParams.set('config_tab', currentConfigTab);
		}
		goto(url.toString(), { replaceState: true, noScroll: true });

		// Fetch data when switching tabs and setup auto-refresh
		setupAutoRefresh(tab);
	}

	function setupAutoRefresh(tab: string) {
		// Clear existing interval
		if (refreshInterval) {
			clearInterval(refreshInterval);
			refreshInterval = null;
		}

		if (tab === 'connections') {
			fetchConnections();
			// Auto-refresh every 5 seconds for connections
			refreshInterval = window.setInterval(() => {
				fetchConnections();
			}, 5000);
		} else if (tab === 'users') {
			fetchSessions();
			// Auto-refresh every 5 seconds for users
			refreshInterval = window.setInterval(() => {
				fetchSessions();
			}, 5000);
		}
	}

	function updateConfigTab(tab: string) {
		currentConfigTab = tab;
		const url = new URL(window.location.href);
		url.searchParams.set('config_tab', tab);
		goto(url.toString(), { replaceState: true, noScroll: true });
	}

	async function fetchSessions() {
		const token = localStorage.getItem('admin_token');

		if (!token) {
			goto('/admin');
			return;
		}

		// Only show loading state on first fetch
		if (sessions.length === 0) {
			isLoadingSessions = true;
		}

		try {
			const response = await fetch(`${API_BASE_URL}/api/admin/users`, {
				headers: {
					Authorization: `Bearer ${token}`
				}
			});

			if (response.status === 401) {
				localStorage.removeItem('admin_token');
				goto('/admin');
				return;
			}

			if (!response.ok) {
				throw new Error('Failed to fetch active users');
			}

			const data = await response.json();
			const newSessions = data.data.sessions || [];

			// Update existing sessions in place to avoid re-rendering
			if (sessions.length > 0) {
				// Create a map of new sessions by session_id
				const newSessionMap = new Map(newSessions.map((s: Session) => [s.session_id, s]));

				// Update existing sessions
				sessions = sessions
					.map((sess) => {
						const updated = newSessionMap.get(sess.session_id);
						if (updated) {
							newSessionMap.delete(sess.session_id);
							return updated;
						}
						return null;
					})
					.filter((s): s is Session => s !== null)
					.concat(Array.from(newSessionMap.values()) as Session[]);
			} else {
				sessions = newSessions;
			}

			sessionsError = '';
		} catch (err) {
			sessionsError = err instanceof Error ? err.message : 'Failed to load active users';
		} finally {
			isLoadingSessions = false;
		}
	}

	async function fetchConnections() {
		const token = localStorage.getItem('admin_token');

		if (!token) {
			goto('/admin');
			return;
		}

		// Only show loading state on first fetch
		if (connections.length === 0) {
			isLoadingConnections = true;
		}

		try {
			const response = await fetch(`${API_BASE_URL}/api/admin/connections`, {
				headers: {
					Authorization: `Bearer ${token}`
				}
			});

			if (response.status === 401) {
				localStorage.removeItem('admin_token');
				goto('/admin');
				return;
			}

			if (!response.ok) {
				throw new Error('Failed to fetch connections');
			}

			const data = await response.json();
			const newConnections = data.data.connections || [];

			// Update existing connections in place to avoid re-rendering
			if (connections.length > 0) {
				// Create a map of new connections by IP
				const newConnMap = new Map(newConnections.map((c: Connection) => [c.ip, c]));

				// Update existing connections
				connections = connections
					.map((conn) => {
						const updated = newConnMap.get(conn.ip);
						if (updated) {
							newConnMap.delete(conn.ip);
							return updated;
						}
						return null;
					})
					.filter((c): c is Connection => c !== null)
					.concat(Array.from(newConnMap.values()) as Connection[]);
			} else {
				connections = newConnections;
			}

			connectionsError = '';
		} catch (err) {
			connectionsError = err instanceof Error ? err.message : 'Failed to load connections';
		} finally {
			isLoadingConnections = false;
		}
	}

	async function fetchConfig() {
		const token = localStorage.getItem('admin_token');

		if (!token) {
			goto('/admin');
			return;
		}

		configStore.setLoading(true);
		try {
			const response = await fetch(`${API_BASE_URL}/api/admin/config`, {
				headers: {
					Authorization: `Bearer ${token}`
				}
			});

			if (response.status === 401) {
				localStorage.removeItem('admin_token');
				goto('/admin');
				return;
			}

			if (!response.ok) {
				throw new Error('Failed to fetch configuration');
			}

			const data = await response.json();
			configStore.setConfig(data.data);
			configStore.setError('');
		} catch (err) {
			configStore.setError(err instanceof Error ? err.message : 'Failed to load configuration');
		} finally {
			configStore.setLoading(false);
		}
	}

	async function saveConfig() {
		if (!configStore.config) return;

		const token = localStorage.getItem('admin_token');

		if (!token) {
			goto('/admin');
			return;
		}

		configStore.setSaving(true);
		configStore.setSaveSuccess(false);
		try {
			const response = await fetch(`${API_BASE_URL}/api/admin/config`, {
				method: 'PUT',
				headers: {
					Authorization: `Bearer ${token}`,
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(configStore.config)
			});

			if (response.status === 401) {
				localStorage.removeItem('admin_token');
				goto('/admin');
				return;
			}

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to save configuration');
			}

			const data = await response.json();
			configStore.setConfig(data.data);
			configStore.setSaveSuccess(true);
			configStore.setError('');

			toaster.success({
				title: 'Configuration Saved',
				description: 'All changes have been saved successfully'
			});
		} catch (err) {
			configStore.setError(err instanceof Error ? err.message : 'Failed to save configuration');
		} finally {
			configStore.setSaving(false);
		}
	}

	function cancelConfigChanges() {
		configStore.cancel();
	}

	async function copyConfigToClipboard() {
		const result = await configStore.copyToClipboard();
		if (result.success) {
			toaster.success({
				title: 'Copied!',
				description: 'Configuration copied to clipboard'
			});
		} else {
			toaster.error({
				title: 'Copy Failed',
				description: result.error || 'Failed to copy configuration'
			});
		}
	}

	function openImportDialog() {
		importJsonText = '';
		importError = '';
		showImportDialog = true;
	}

	function closeImportDialog() {
		showImportDialog = false;
		importJsonText = '';
		importError = '';
	}

	function handleImportConfig() {
		const result = configStore.importFromJSON(importJsonText);
		if (result.success) {
			closeImportDialog();
		} else {
			importError = result.error || 'Failed to import configuration';
		}
	}

	async function terminateSession(session: Session) {
		const token = localStorage.getItem('admin_token');
		if (!token) {
			goto('/admin');
			return;
		}

		try {
			const response = await fetch(`${API_BASE_URL}/api/admin/users/${session.session_id}`, {
				method: 'DELETE',
				headers: {
					Authorization: `Bearer ${token}`
				}
			});

			if (response.status === 401) {
				localStorage.removeItem('admin_token');
				goto('/admin');
				return;
			}

			if (!response.ok) {
				throw new Error('Failed to terminate user session');
			}

			toaster.success({
				title: 'User Session Terminated',
				description: `${session.username}'s session has been terminated successfully`
			});

			// Refresh the list
			await fetchSessions();
		} catch (err) {
			toaster.error({
				title: 'Termination Failed',
				description: err instanceof Error ? err.message : 'Failed to terminate session'
			});
		}
	}

	function openTerminateDialog(session: Session) {
		sessionToTerminate = session;
		showTerminateDialog = true;
	}

	async function terminateConnection(connection: Connection) {
		const token = localStorage.getItem('admin_token');
		if (!token) {
			goto('/admin');
			return;
		}

		try {
			const response = await fetch(
				`${API_BASE_URL}/api/admin/connections/${encodeURIComponent(connection.ip)}`,
				{
					method: 'DELETE',
					headers: {
						Authorization: `Bearer ${token}`
					}
				}
			);

			if (response.status === 401) {
				localStorage.removeItem('admin_token');
				goto('/admin');
				return;
			}

			if (!response.ok) {
				throw new Error('Failed to terminate connections');
			}

			toaster.success({
				title: 'Connections Terminated',
				description: `All connections from ${connection.ip} have been terminated`
			});

			// Refresh the list
			await fetchConnections();
		} catch (err) {
			toaster.error({
				title: 'Termination Failed',
				description: err instanceof Error ? err.message : 'Failed to terminate connections'
			});
		}
	}

	function openTerminateConnectionDialog(connection: Connection) {
		connectionToTerminate = connection;
		showTerminateConnectionDialog = true;
	}

	function handleLogout() {
		localStorage.removeItem('admin_token');
		goto('/admin');
	}

	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleString();
	}

	function getTimeRemaining(expiresAt: string): string {
		const now = new Date().getTime();
		const expiry = new Date(expiresAt).getTime();
		const diff = expiry - now;

		if (diff <= 0) return 'Expired';

		const hours = Math.floor(diff / (1000 * 60 * 60));
		const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

		if (hours > 0) {
			return `${hours}h ${minutes}m`;
		}
		return `${minutes}m`;
	}

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
	}

	function formatNumber(num: number): string {
		if (num >= 1000000) {
			return (num / 1000000).toFixed(1) + 'M';
		} else if (num >= 1000) {
			return (num / 1000).toFixed(1) + 'K';
		}
		return num.toString();
	}

	onMount(() => {
		fetchConfig();

		// Setup auto-refresh for the active tab
		setupAutoRefresh(currentMainTab);
	});

	onDestroy(() => {
		// Clean up interval on component destroy
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});
</script>

<div class="mx-auto max-w-7xl px-4 py-8">
	<!-- Header -->
	<PageHeader
		title="Admin Dashboard"
		subtitle="Manage sessions and system configuration"
		icon={Shield}
	>
		<button
			onclick={handleLogout}
			class="bg-error hover:bg-error-hover focus:ring-error focus:ring-offset-base-100 flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2"
		>
			<LogOut class="h-4 w-4" />
			Logout
		</button>
	</PageHeader>

	<!-- Tabs -->
	<Tabs.Root value={currentMainTab} onValueChange={(e) => updateMainTab(e.value)} class="w-full">
		<Tabs.List class="border-border mb-6 border-b">
			<div class="flex gap-2">
				<Tabs.Trigger
					value="connections"
					class="text-base-muted hover:text-base-content hover:border-border-hover data-selected:text-primary data-selected:border-primary flex items-center gap-2 border-b-2 border-transparent px-4 py-3 text-sm font-medium transition-colors"
				>
					<Network class="h-4 w-4" />
					All Connections
				</Tabs.Trigger>
				<Tabs.Trigger
					value="users"
					class="text-base-muted hover:text-base-content hover:border-border-hover data-selected:text-primary data-selected:border-primary flex items-center gap-2 border-b-2 border-transparent px-4 py-3 text-sm font-medium transition-colors"
				>
					<Users class="h-4 w-4" />
					Active Users
				</Tabs.Trigger>
				<Tabs.Trigger
					value="configuration"
					class="text-base-muted hover:text-base-content hover:border-border-hover data-selected:text-primary data-selected:border-primary flex items-center gap-2 border-b-2 border-transparent px-4 py-3 text-sm font-medium transition-colors"
				>
					<Settings class="h-4 w-4" />
					Configuration
				</Tabs.Trigger>
			</div>
		</Tabs.List>

		<!-- All Connections Tab (First) -->
		<Tabs.Content value="connections">
			{#if isLoadingConnections}
				<div class="border-border bg-base-100 rounded-2xl border p-16 text-center shadow-sm">
					<div
						class="border-primary mx-auto h-12 w-12 animate-spin rounded-full border-4 border-t-transparent"
					></div>
					<p class="text-base-content mt-4 font-medium">Loading connections...</p>
				</div>
			{:else if connectionsError}
				<div class="border-error/30 bg-error/5 rounded-2xl border p-8">
					<p class="text-error">{connectionsError}</p>
					<button
						onclick={fetchConnections}
						class="bg-error hover:bg-error-hover mt-4 rounded-lg px-4 py-2 text-sm font-semibold text-white"
					>
						Retry
					</button>
				</div>
			{:else}
				<ActiveConnections
					{connections}
					onRefresh={fetchConnections}
					onTerminate={openTerminateConnectionDialog}
				/>
			{/if}
		</Tabs.Content>

		<!-- Active Users Tab (Second) -->
		<Tabs.Content value="users">
			{#if isLoadingSessions}
				<div class="border-border bg-base-100 rounded-2xl border p-16 text-center shadow-sm">
					<div
						class="border-primary mx-auto h-12 w-12 animate-spin rounded-full border-4 border-t-transparent"
					></div>
					<p class="text-base-content mt-4 font-medium">Loading active users...</p>
				</div>
			{:else if sessionsError}
				<div class="border-error/30 bg-error/5 rounded-2xl border p-8">
					<p class="text-error">{sessionsError}</p>
					<button
						onclick={fetchSessions}
						class="bg-error hover:bg-error-hover mt-4 rounded-lg px-4 py-2 text-sm font-semibold text-white"
					>
						Retry
					</button>
				</div>
			{:else}
				<ActiveUsers {sessions} onRefresh={fetchSessions} onTerminate={terminateSession} />
			{/if}
		</Tabs.Content>

		<!-- Configuration Tab -->
		<Tabs.Content value="configuration">
			{#if configStore.isLoading}
				<div class="border-border bg-base-100 rounded-2xl border p-16 text-center shadow-sm">
					<div
						class="border-primary mx-auto h-12 w-12 animate-spin rounded-full border-4 border-t-transparent"
					></div>
					<p class="text-base-content mt-4 font-medium">Loading configuration...</p>
				</div>
			{:else if configStore.error}
				<div class="border-error/30 bg-error/5 rounded-2xl border p-8">
					<p class="text-error">{configStore.error}</p>
					<button
						onclick={fetchConfig}
						class="bg-error hover:bg-error-hover mt-4 rounded-lg px-4 py-2 text-sm font-semibold text-white"
					>
						Retry
					</button>
				</div>
			{:else if configStore.config}
				<div class="space-y-6">
					<!-- Import/Export Buttons -->
					<div class="flex gap-2">
						<button
							onclick={copyConfigToClipboard}
							class="border-border bg-base-100 text-base-content hover:bg-base-200 flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors"
						>
							<Copy class="h-4 w-4" />
							Copy Config JSON
						</button>
						<button
							onclick={openImportDialog}
							class="border-border bg-base-100 text-base-content hover:bg-base-200 flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium transition-colors"
						>
							<Upload class="h-4 w-4" />
							Import Config JSON
						</button>
					</div>

					<!-- Save/Cancel Actions - Animated Floating Banner -->
					<div class="relative">
						{#if configStore.hasChanges}
							<div
								class="border-primary bg-primary animate-in slide-in-from-top-5 fade-in sticky top-4 z-10 flex items-center justify-between rounded-lg border-2 p-4 shadow-2xl duration-300"
								style="animation: slideDown 0.3s ease-out;"
							>
								<div class="flex items-center gap-2">
									<svg
										class="h-5 w-5 animate-pulse text-white"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
										/>
									</svg>
									<p class="text-sm font-semibold text-white">
										You have unsaved changes ({configStore.dirtyFieldCount} field{configStore.dirtyFieldCount ===
										1
											? ''
											: 's'})
									</p>
								</div>
								<div class="flex gap-2">
									<button
										onclick={cancelConfigChanges}
										class="flex items-center gap-2 rounded-lg bg-white/20 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-white/30"
									>
										<X class="h-4 w-4" />
										Cancel
									</button>
									<button
										onclick={saveConfig}
										disabled={configStore.isSaving}
										class="text-primary flex items-center gap-2 rounded-lg bg-white px-4 py-2 text-sm font-semibold transition-colors hover:bg-gray-100 disabled:opacity-50"
									>
										{#if configStore.isSaving}
											<div
												class="border-primary h-4 w-4 animate-spin rounded-full border-2 border-t-transparent"
											></div>
										{:else}
											<Save class="h-4 w-4" />
										{/if}
										Save Configuration
									</button>
								</div>
							</div>
						{/if}
					</div>
					{#if configStore.error}
						<div class="border-error/30 bg-error/5 rounded-lg border p-3">
							<p class="text-error text-sm">{configStore.error}</p>
						</div>
					{/if}

					<!-- Nested Tabs for Configuration Sections -->
					<Tabs.Root
						value={currentConfigTab}
						onValueChange={(e) => updateConfigTab(e.value)}
						class="w-full"
					>
						<Tabs.List class="border-border mb-6 flex gap-1 border-b">
							<Tabs.Trigger
								value="users"
								class="text-base-muted hover:text-base-content data-selected:text-primary data-selected:border-b-2 data-selected:border-primary px-4 py-2 text-sm font-medium transition-colors"
							>
								Users
							</Tabs.Trigger>
							<Tabs.Trigger
								value="protected_services"
								class="text-base-muted hover:text-base-content data-selected:text-primary data-selected:border-b-2 data-selected:border-primary px-4 py-2 text-sm font-medium transition-colors"
							>
								Protected Services
							</Tabs.Trigger>
							<Tabs.Trigger
								value="advanced"
								class="text-base-muted hover:text-base-content data-selected:text-primary data-selected:border-b-2 data-selected:border-primary px-4 py-2 text-sm font-medium transition-colors"
							>
								Advanced
							</Tabs.Trigger>
							<Tabs.Indicator class="bg-primary h-0.5" />
						</Tabs.List>

						<Tabs.Content value="users">
							<ConfigUsers config={configStore.config} />
						</Tabs.Content>

						<Tabs.Content value="protected_services">
							<ConfigProtectedServices config={configStore.config} />
						</Tabs.Content>

						<Tabs.Content value="advanced">
							<ConfigAdvanced config={configStore.config} />
						</Tabs.Content>
					</Tabs.Root>
				</div>
			{/if}
		</Tabs.Content>
	</Tabs.Root>
</div>

<!-- Import Config Dialog -->
{#if showImportDialog}
	<Dialog.Root
		open={showImportDialog}
		onOpenChange={(details) => {
			if (!details.open) closeImportDialog();
		}}
	>
		<Dialog.Backdrop class="fixed inset-0 z-40 bg-black/50" />
		<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<Dialog.Content
				class="bg-base-100 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl shadow-xl"
			>
				<div class="p-6">
					<div class="mb-6 flex items-center justify-between">
						<Dialog.Title class="text-base-content text-xl font-semibold">
							Import Configuration
						</Dialog.Title>
						<Dialog.CloseTrigger
							onclick={closeImportDialog}
							class="text-base-muted hover:text-base-content transition-colors"
						>
							<X class="h-5 w-5" />
						</Dialog.CloseTrigger>
					</div>

					<div class="space-y-4">
						<Field.Root>
							<Field.Label class="text-base-content mb-2 text-sm font-medium">
								Paste Configuration JSON
							</Field.Label>
							<Field.Textarea
								bind:value={importJsonText}
								rows={15}
								placeholder="Paste complete configuration JSON here..."
								class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
							/>
							<Field.HelperText class="text-base-muted mt-1 text-xs">
								Paste the complete configuration JSON. This will replace your current configuration.
							</Field.HelperText>
						</Field.Root>

						{#if importError}
							<div class="border-error/30 bg-error/5 rounded-lg border p-3">
								<p class="text-error text-sm">{importError}</p>
							</div>
						{/if}
					</div>

					<div class="mt-6 flex gap-3">
						<button
							onclick={closeImportDialog}
							class="text-base-content bg-base-200 hover:bg-base-300 flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-colors"
						>
							Cancel
						</button>
						<button
							onclick={handleImportConfig}
							disabled={!importJsonText.trim()}
							class="bg-primary hover:bg-primary-hover flex-1 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50"
						>
							Import Configuration
						</button>
					</div>
				</div>
			</Dialog.Content>
		</Dialog.Positioner>
	</Dialog.Root>
{/if}

<!-- Terminate Connection Dialog -->
{#if showTerminateConnectionDialog && connectionToTerminate}
	<Dialog.Root
		open={showTerminateConnectionDialog}
		onOpenChange={(details) => {
			if (!details.open) {
				showTerminateConnectionDialog = false;
				connectionToTerminate = null;
			}
		}}
	>
		<Dialog.Backdrop class="fixed inset-0 z-40 bg-black/50" />
		<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<Dialog.Content class="bg-base-100 w-full max-w-md rounded-xl shadow-xl">
				<div class="p-6">
					<div class="mb-6 flex items-center gap-4">
						<div class="bg-error/10 flex h-12 w-12 items-center justify-center rounded-full">
							<Trash2 class="text-error h-6 w-6" />
						</div>
						<div class="flex-1">
							<Dialog.Title class="text-base-content text-lg font-semibold">
								Terminate Connections
							</Dialog.Title>
							<Dialog.Description class="text-base-muted text-sm">
								This action cannot be undone
							</Dialog.Description>
						</div>
					</div>

					<div class="border-border bg-base-200/50 rounded-lg border p-4">
						<p class="text-base-content text-sm">
							Are you sure you want to terminate all connections from:
						</p>
						<code
							class="bg-base-100 text-base-content mt-2 block rounded px-2 py-1 font-mono text-sm"
						>
							{connectionToTerminate.ip}
						</code>
						{#if connectionToTerminate.authenticated}
							<p class="text-base-muted mt-2 text-xs">
								User: <span class="font-semibold">{connectionToTerminate.username}</span>
							</p>
						{:else}
							<p class="text-warning mt-2 text-xs font-medium">
								This is a permanent allowlist entry
							</p>
						{/if}
						<p class="text-base-muted mt-2 text-xs">
							Active sessions: <span class="font-semibold"
								>{connectionToTerminate.total_sessions}</span
							>
						</p>
					</div>

					<div class="mt-6 flex gap-3">
						<button
							onclick={() => {
								showTerminateConnectionDialog = false;
								connectionToTerminate = null;
							}}
							class="text-base-content bg-base-200 hover:bg-base-300 flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-colors"
						>
							Cancel
						</button>
						<button
							onclick={() => {
								if (connectionToTerminate) {
									terminateConnection(connectionToTerminate);
									showTerminateConnectionDialog = false;
									connectionToTerminate = null;
								}
							}}
							class="bg-error hover:bg-error-hover flex-1 rounded-lg px-4 py-2 text-sm font-semibold text-white transition-colors"
						>
							Terminate Connections
						</button>
					</div>
				</div>
			</Dialog.Content>
		</Dialog.Positioner>
	</Dialog.Root>
{/if}

<!-- Toast Notifications -->
<Toaster {toaster}>
	{#snippet children(toast)}
		<Toast.Root
			class="bg-base-100 border-border min-w-[320px] rounded-lg border-2 p-4 shadow-lg {toast()
				.type === 'error'
				? 'border-error'
				: toast().type === 'success'
					? 'border-success'
					: toast().type === 'warning'
						? 'border-warning'
						: 'border-info'}"
		>
			<div class="flex items-start gap-3">
				{#if toast().type === 'success'}
					<CircleCheck class="text-success h-5 w-5 shrink-0" />
				{:else if toast().type === 'error'}
					<CircleAlert class="text-error h-5 w-5 shrink-0" />
				{:else if toast().type === 'warning'}
					<TriangleAlert class="text-warning h-5 w-5 shrink-0" />
				{:else}
					<Info class="text-info h-5 w-5 shrink-0" />
				{/if}
				<div class="flex-1">
					<Toast.Title class="text-base-content text-sm font-semibold">
						{toast().title}
					</Toast.Title>
					{#if toast().description}
						<Toast.Description class="text-base-muted mt-1 text-xs">
							{toast().description}
						</Toast.Description>
					{/if}
				</div>
				<Toast.CloseTrigger class="text-base-muted hover:text-base-content transition-colors">
					<X class="h-4 w-4" />
				</Toast.CloseTrigger>
			</div>
		</Toast.Root>
	{/snippet}
</Toaster>
