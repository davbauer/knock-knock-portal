<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { slide } from 'svelte/transition';
	import {
		Users,
		UserX,
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
		CircleCheck,
		CircleAlert,
		TriangleAlert,
		Info
	} from 'lucide-svelte';
	import { API_BASE_URL } from '$lib/config';
	import { Tabs, Dialog, Field, Toast, Toaster } from '@ark-ui/svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import type { Session, Config } from './types';
	import ConfigUsers from './ConfigUsers.svelte';
	import ConfigProtectedServices from './ConfigProtectedServices.svelte';
	import ConfigAdvanced from './ConfigAdvanced.svelte';
	import { configStore } from './configStore.svelte';
	import { toaster } from './toastStore.svelte';

	let sessions = $state<Session[]>([]);
	let isLoadingSessions = $state(true);
	let sessionsError = $state('');

	let showImportDialog = $state(false);
	let importJsonText = $state('');
	let importError = $state('');

	let showTerminateDialog = $state(false);
	let sessionToTerminate = $state<Session | null>(null);

	// Get initial tabs from URL or defaults
	let currentMainTab = $state($page.url.searchParams.get('tab') || 'sessions');
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

		isLoadingSessions = true;
		try {
			const response = await fetch(`${API_BASE_URL}/api/admin/sessions`, {
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
				throw new Error('Failed to fetch sessions');
			}

			const data = await response.json();
			sessions = data.data.sessions || [];
			sessionsError = '';
		} catch (err) {
			sessionsError = err instanceof Error ? err.message : 'Failed to load sessions';
		} finally {
			isLoadingSessions = false;
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

	async function terminateSession(sessionId: string) {
		const token = localStorage.getItem('admin_token');
		if (!token) {
			goto('/admin');
			return;
		}

		try {
			const response = await fetch(`${API_BASE_URL}/api/admin/sessions/${sessionId}`, {
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
				throw new Error('Failed to terminate session');
			}

			toaster.success({
				title: 'Session Terminated',
				description: 'The user session has been terminated successfully'
			});

			showTerminateDialog = false;
			sessionToTerminate = null;

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

	onMount(() => {
		fetchSessions();
		fetchConfig();
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
					value="sessions"
					class="text-base-muted hover:text-base-content hover:border-border-hover data-selected:text-primary data-selected:border-primary flex items-center gap-2 border-b-2 border-transparent px-4 py-3 text-sm font-medium transition-colors"
				>
					<Users class="h-4 w-4" />
					Active Sessions
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

		<!-- Sessions Tab -->
		<Tabs.Content value="sessions">
			{#if isLoadingSessions}
				<div class="border-border bg-base-100 rounded-2xl border p-16 text-center shadow-sm">
					<div
						class="border-primary mx-auto h-12 w-12 animate-spin rounded-full border-4 border-t-transparent"
					></div>
					<p class="text-base-content mt-4 font-medium">Loading sessions...</p>
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
			{:else if sessions.length === 0}
				<div class="border-border bg-base-100 rounded-xl border p-12 text-center">
					<UserX class="text-base-muted mx-auto mb-4 h-12 w-12" />
					<h3 class="text-base-content mb-2 text-lg font-semibold">No Active Sessions</h3>
					<p class="text-base-muted text-sm">
						There are currently no authenticated users with active sessions.
					</p>
					<button
						onclick={fetchSessions}
						class="bg-primary hover:bg-primary-hover mt-4 rounded-lg px-4 py-2 text-sm font-semibold text-white"
					>
						Refresh
					</button>
				</div>
			{:else}
				<div class="space-y-4">
					<div class="flex items-center justify-between">
						<p class="text-base-muted text-sm">
							{sessions.length} active {sessions.length === 1 ? 'session' : 'sessions'}
						</p>
						<button
							onclick={fetchSessions}
							class="border-border bg-base-100 text-base-content hover:bg-base-200 rounded-lg border px-3 py-1.5 text-xs font-medium"
						>
							Refresh
						</button>
					</div>

					<div class="border-border bg-base-100 overflow-hidden rounded-xl border shadow-sm">
						<div class="overflow-x-auto">
							<table class="w-full">
								<thead class="bg-base-200 border-border border-b">
									<tr>
										<th
											class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
										>
											User
										</th>
										<th
											class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
										>
											IP Address
										</th>
										<th
											class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
										>
											Created
										</th>
										<th
											class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
										>
											Expires
										</th>
										<th
											class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
										>
											Time Left
										</th>
										<th
											class="text-base-muted px-6 py-3 text-right text-xs font-medium uppercase tracking-wider"
										>
											Actions
										</th>
									</tr>
								</thead>
								<tbody class="divide-border divide-y">
									{#each sessions as session}
										<tr class="hover:bg-base-200 transition-colors">
											<td class="whitespace-nowrap px-6 py-4">
												<div class="flex items-center gap-3">
													<div
														class="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-full"
													>
														<Users class="text-primary h-5 w-5" />
													</div>
													<div>
														<div class="text-base-content text-sm font-medium">
															{session.username}
														</div>
														<div class="text-base-muted font-mono text-xs">
															{session.user_id.slice(0, 8)}...
														</div>
													</div>
												</div>
											</td>
											<td class="whitespace-nowrap px-6 py-4">
												<div class="text-base-content space-y-1 font-mono text-sm">
													{#each session.authenticated_ips as ip}
														<div class="flex items-center gap-2">
															<span class="bg-base-300 rounded px-2 py-0.5">{ip}</span>
														</div>
													{/each}
												</div>
											</td>
											<td class="whitespace-nowrap px-6 py-4">
												<div class="text-base-content text-sm">
													{formatDate(session.created_at)}
												</div>
											</td>
											<td class="whitespace-nowrap px-6 py-4">
												<div class="text-base-content text-sm">
													{formatDate(session.expires_at)}
												</div>
											</td>
											<td class="whitespace-nowrap px-6 py-4">
												<span
													class="bg-success/10 text-success inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
												>
													{getTimeRemaining(session.expires_at)}
												</span>
											</td>
											<td class="whitespace-nowrap px-6 py-4 text-right">
												<button
													onclick={() => openTerminateDialog(session)}
													class="bg-error/10 text-error hover:bg-error/20 focus:ring-error focus:ring-offset-base-100 rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2"
												>
													Terminate
												</button>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				</div>
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

<!-- Terminate Session Dialog -->
{#if showTerminateDialog && sessionToTerminate}
	<Dialog.Root
		open={showTerminateDialog}
		onOpenChange={(details) => {
			if (!details.open) {
				showTerminateDialog = false;
				sessionToTerminate = null;
			}
		}}
	>
		<Dialog.Backdrop class="fixed inset-0 z-40 bg-black/50" />
		<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<Dialog.Content class="bg-base-100 w-full max-w-md rounded-xl p-6 shadow-xl">
				<Dialog.Title class="text-base-content mb-4 text-lg font-semibold">
					Terminate Session
				</Dialog.Title>

				<p class="text-base-content mb-6 text-sm">
					Are you sure you want to terminate the session for <strong
						>{sessionToTerminate.username}</strong
					>?
					<br />
					<span class="text-base-muted mt-2 block text-xs">
						IP: {sessionToTerminate.authenticated_ip}
					</span>
				</p>

				<div class="flex gap-3">
					<button
						onclick={() => {
							showTerminateDialog = false;
							sessionToTerminate = null;
						}}
						class="text-base-content bg-base-200 hover:bg-base-300 flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-colors"
					>
						Cancel
					</button>
					<button
						onclick={() => terminateSession(sessionToTerminate!.session_id)}
						class="bg-error hover:bg-error-hover flex-1 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors"
					>
						Terminate Session
					</button>
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
