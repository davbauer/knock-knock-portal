<script lang="ts">
	import type { Config, PortalUser } from './types';
	import { Dialog, Field, Switch, Checkbox } from '@ark-ui/svelte';
	import { X, Plus, Check } from 'lucide-svelte';
	import { configStore } from './configStore.svelte';
	import { toaster } from './toastStore.svelte';

	function generateUUID(): string {
		return crypto.randomUUID();
	}

	interface Props {
		config: Config;
	}

	let { config }: Props = $props();

	let showAddDialog = $state(false);
	let showDeleteDialog = $state(false);
	let editingUser = $state<PortalUser | null>(null);
	let userToDelete = $state<PortalUser | null>(null);
	
	// Form state
	let formUserId = $state('');
	let formUsername = $state('');
	let formPassword = $state('');
	let formNotes = $state('');
	let formDisplayInSuggestions = $state(true);
	let formSelectedServices = $state<string[]>([]);

	function openAddDialog() {
		editingUser = null;
		formUserId = generateUUID();
		formUsername = '';
		formPassword = '';
		formNotes = '';
		formDisplayInSuggestions = true;
		formSelectedServices = [];
		showAddDialog = true;
	}

	function openEditDialog(user: PortalUser) {
		editingUser = user;
		formUserId = user.user_id;
		formUsername = user.username;
		formPassword = ''; // Don't show existing password
		formNotes = user.notes;
		formDisplayInSuggestions = user.display_username_in_public_login_suggestions;
		formSelectedServices = [...user.allowed_service_ids];
		showAddDialog = true;
	}

	function closeDialog() {
		showAddDialog = false;
		editingUser = null;
	}

	function handleSubmit() {
		// Validation
		if (!formUsername.trim()) {
			toaster.error({
				title: 'Validation Error',
				description: 'Username is required'
			});
			return;
		}

		// Password required for new users
		if (!editingUser && !formPassword.trim()) {
			toaster.error({
				title: 'Validation Error',
				description: 'Password is required for new users'
			});
			return;
		}

		// Create user object
		const newUser: PortalUser = {
			user_id: formUserId.trim(),
			username: formUsername.trim(),
			display_username_in_public_login_suggestions: formDisplayInSuggestions,
			// Send plain password if provided, backend will hash it
			// If password is empty (editing user), send empty string and backend will keep existing hash
			bcrypt_hashed_password: formPassword.trim() || (editingUser?.bcrypt_hashed_password || ''),
			allowed_service_ids: formSelectedServices,
			notes: formNotes.trim(),
		};

		if (editingUser) {
			// Update existing user
			const index = config.portal_user_accounts.findIndex(u => u.user_id === editingUser!.user_id);
			if (index !== -1) {
				configStore.updateConfig((cfg) => {
					cfg.portal_user_accounts[index] = newUser;
				});
			}
		} else {
			// Add new user
			configStore.updateConfig((cfg) => {
				if (!cfg.portal_user_accounts) {
					cfg.portal_user_accounts = [];
				}
				cfg.portal_user_accounts.push(newUser);
			});
		}

		closeDialog();
	}

	function openDeleteDialog(user: PortalUser) {
		userToDelete = user;
		showDeleteDialog = true;
	}

	function confirmDelete() {
		if (!userToDelete) return;
		configStore.updateConfig((cfg) => {
			cfg.portal_user_accounts = cfg.portal_user_accounts.filter(u => u.user_id !== userToDelete!.user_id);
		});
		showDeleteDialog = false;
		userToDelete = null;
	}

	function toggleService(serviceId: string) {
		if (formSelectedServices.includes(serviceId)) {
			formSelectedServices = formSelectedServices.filter(id => id !== serviceId);
		} else {
			formSelectedServices = [...formSelectedServices, serviceId];
		}
	}

	function abbreviateId(id: string): string {
		// If it looks like a UUID (has dashes), show first 8 chars
		if (id.includes('-') && id.length > 12) {
			return id.substring(0, 8) + '...';
		}
		// If it's long without dashes, show first 8 chars
		if (id.length > 12) {
			return id.substring(0, 8) + '...';
		}
		return id;
	}

	function formatProtocol(protocol: string): string {
		if (protocol === 'both') return 'TCP+UDP';
		return protocol.toUpperCase();
	}
</script>

<div class="rounded-xl border border-border bg-base-100 p-6">
	<div class="flex items-center justify-between mb-4">
		<h3 class="text-lg font-semibold text-base-content">Portal Users</h3>
		<button
			onclick={openAddDialog}
			class="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary-hover"
		>
			<Plus class="w-4 h-4" />
			Add User
		</button>
	</div>

	{#if !config.portal_user_accounts || config.portal_user_accounts.length === 0}
		<div class="text-center py-12 text-base-muted">
			<svg class="w-12 h-12 mx-auto mb-3 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
			</svg>
			<p class="text-sm">No portal users configured</p>
			<p class="text-xs mt-1">Add users to grant access to protected services</p>
		</div>
	{:else}
		<div class="overflow-x-auto">
			<table class="w-full">
				<thead>
					<tr class="border-b border-border">
						<th class="text-left py-3 px-4 text-sm font-medium text-base-muted">User</th>
						<th class="text-left py-3 px-4 text-sm font-medium text-base-muted">Service Access</th>
						<th class="text-left py-3 px-4 text-sm font-medium text-base-muted">Settings</th>
						<th class="text-right py-3 px-4 text-sm font-medium text-base-muted">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each config.portal_user_accounts as user}
						<tr class="border-b border-border hover:bg-base-200">
							<td class="py-3 px-4">
								<div class="flex flex-col gap-1">
									<div class="flex items-center gap-2">
										<span class="text-sm font-medium text-base-content">{user.username}</span>
										<span class="text-xs font-mono text-base-muted">
											{user.user_id.split('-')[0]}
										</span>
									</div>
									{#if user.notes}
										<p class="text-xs italic text-base-muted">{user.notes}</p>
									{/if}
								</div>
							</td>
							<td class="py-3 px-4">
								{#if !user.allowed_service_ids || user.allowed_service_ids.length === 0}
									<span class="inline-flex items-center gap-1 px-2 py-1 rounded bg-gray-500/10 text-base-muted text-xs font-medium">
										No Services
									</span>
								{:else}
									<div class="flex flex-col gap-1.5">
										{#each user.allowed_service_ids as serviceId}
											{@const service = config.protected_services.find((s) => s.service_id === serviceId)}
											{#if service}
												<div class="flex items-center gap-2">
													<span class="text-sm font-medium text-base-content">{service.service_name}</span>
													<span class="text-xs font-mono text-base-muted">
														{service.service_id.split('-')[0]}
													</span>
													<span class="text-xs text-base-muted">•</span>
													<span class="text-xs text-base-muted">
														{#if service.proxy_listen_port_start === service.proxy_listen_port_end}
															:{service.proxy_listen_port_start}
														{:else}
															:{service.proxy_listen_port_start}-{service.proxy_listen_port_end}
														{/if}
													</span>
													<span class="text-xs text-base-muted">•</span>
													<span class="rounded bg-blue-500/10 px-1.5 py-0.5 text-xs font-mono uppercase text-blue-600">
														{service.transport_protocol === 'both' ? 'TCP+UDP' : service.transport_protocol.toUpperCase()}
													</span>
													{#if service.is_http_protocol}
														<span class="rounded bg-purple-500/10 px-1.5 py-0.5 text-xs font-mono uppercase text-purple-600">
															HTTP
														</span>
													{/if}
												</div>
											{:else}
												<span class="text-xs text-error">Unknown: {serviceId.split('-')[0]}</span>
											{/if}
										{/each}
									</div>
								{/if}
							</td>
							<td class="py-3 px-4">
								<div class="flex items-center gap-2">
									{#if user.display_username_in_public_login_suggestions}
										<span class="rounded bg-green-500/10 px-2 py-1 text-xs font-medium text-green-600">
											Visible
										</span>
									{:else}
										<span class="rounded bg-gray-500/10 px-2 py-1 text-xs font-medium text-base-muted">
											Hidden
										</span>
									{/if}
								</div>
							</td>
							<td class="py-3 px-4 text-right">
								<div class="flex items-center justify-end gap-2">
									<button
										onclick={() => openEditDialog(user)}
										class="p-1 text-base-muted hover:text-primary transition-colors"
										title="Edit user"
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
										</svg>
									</button>
									<button
										onclick={() => openDeleteDialog(user)}
										class="p-1 text-base-muted hover:text-error transition-colors"
										title="Delete user"
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
										</svg>
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Add/Edit User Dialog -->
{#if showAddDialog}
	<Dialog.Root open={showAddDialog} onOpenChange={(details) => { if (!details.open) closeDialog(); }}>
		<Dialog.Backdrop class="fixed inset-0 bg-black/50 z-40" />
		<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<Dialog.Content class="bg-base-100 rounded-xl shadow-xl max-w-md w-full max-h-[90vh] overflow-y-auto">
				<div class="p-6">
					<div class="flex items-center justify-between mb-6">
						<Dialog.Title class="text-xl font-semibold text-base-content">
							{editingUser ? 'Edit User' : 'Add New User'}
						</Dialog.Title>
						<Dialog.CloseTrigger
							onclick={closeDialog}
							class="text-base-muted hover:text-base-content transition-colors"
						>
							<X class="w-5 h-5" />
						</Dialog.CloseTrigger>
					</div>

					<div class="space-y-4">
						<div class="grid grid-cols-2 gap-4">
							<!-- User ID (auto-generated, read-only) -->
							<Field.Root>
								<Field.Label class="text-base-content mb-2 text-sm font-medium">
									User ID
									<span class="text-base-muted ml-1 text-xs font-normal">(auto-generated)</span>
								</Field.Label>
								<Field.Input
									type="text"
									value={formUserId}
									readonly
									class="border-border bg-base-200 text-base-muted w-full cursor-not-allowed rounded-lg border px-3 py-2 font-mono text-sm"
								/>
							</Field.Root>

							<!-- Username -->
							<Field.Root>
								<Field.Label class="text-base-content mb-2 text-sm font-medium"
									>Username</Field.Label
								>
								<Field.Input
									type="text"
									bind:value={formUsername}
									placeholder="e.g., john_doe"
									autocomplete="off"
									data-form-type="other"
									data-lpignore="true"
									class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
								/>
							</Field.Root>
						</div>

						<!-- Password -->
						<Field.Root>
							<Field.Label class="text-sm font-medium text-base-content mb-2">
								Password {editingUser ? '(leave empty to keep current)' : ''}
							</Field.Label>
							<Field.Input
								type="password"
								bind:value={formPassword}
								placeholder={editingUser ? 'Leave empty to keep current' : 'Enter password'}
								autocomplete="new-password"
								data-form-type="other"
								data-lpignore="true"
								data-1p-ignore="true"
								class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary"
							/>
							<Field.HelperText class="text-xs text-base-muted mt-1">
								Password will be securely hashed before storage
							</Field.HelperText>
						</Field.Root>						<!-- Display in suggestions -->
						<Checkbox.Root bind:checked={formDisplayInSuggestions} class="flex items-center gap-3">
							<Checkbox.Control class="w-5 h-5 rounded border-2 border-border bg-base-100 flex items-center justify-center data-[state=checked]:bg-primary data-[state=checked]:border-primary transition-colors">
								<Checkbox.Indicator>
									<Check class="w-3 h-3 text-white" />
								</Checkbox.Indicator>
							</Checkbox.Control>
							<Checkbox.Label class="text-sm text-base-content cursor-pointer">
								Display username in public login suggestions
							</Checkbox.Label>
							<Checkbox.HiddenInput />
						</Checkbox.Root>						<!-- Allowed Services -->
						<Field.Root>
							<Field.Label class="text-sm font-medium text-base-content mb-2">Allowed Services</Field.Label>
							<div class="space-y-2 max-h-60 overflow-y-auto border border-border rounded-lg p-3">
								{#if config.protected_services && config.protected_services.length > 0}
									{#each config.protected_services as service, idx}
										<label class="flex items-start gap-3 p-2 rounded hover:bg-base-200 cursor-pointer transition-colors">
											<input
												type="checkbox"
												checked={formSelectedServices.includes(service.service_id)}
												onchange={() => toggleService(service.service_id)}
												class="mt-1 w-4 h-4 rounded border-2 border-border text-primary focus:ring-2 focus:ring-primary bg-base-100"
											/>
											<div class="flex-1 min-w-0">
												<div class="font-medium text-sm text-base-content">
													{service.service_name}
												</div>
												<div class="flex flex-wrap items-center gap-2 mt-1">
													<span class="text-xs px-1.5 py-0.5 rounded bg-base-300 text-base-muted font-mono">
														{abbreviateId(service.service_id)}
													</span>
													<span class="text-xs text-base-muted">
														{#if service.proxy_listen_port_start === service.proxy_listen_port_end}
															:{service.proxy_listen_port_start}
														{:else}
															:{service.proxy_listen_port_start}-{service.proxy_listen_port_end}
														{/if}
													</span>
													<span class="text-xs px-1.5 py-0.5 rounded bg-primary/10 text-primary font-medium">
														{formatProtocol(service.transport_protocol)}
													</span>
													{#if service.is_http_protocol}
														<span class="text-xs px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-500 font-medium">
															HTTP
														</span>
													{/if}
												</div>
											</div>
										</label>
									{/each}
								{:else}
									<p class="text-xs text-base-muted">No services configured yet</p>
								{/if}
							</div>
						</Field.Root>

						<!-- Notes -->
						<Field.Root>
							<Field.Label class="text-sm font-medium text-base-content mb-2">Notes (optional)</Field.Label>
							<Field.Textarea
								bind:value={formNotes}
								rows={3}
								placeholder="Add any notes about this user..."
								class="w-full rounded-lg border border-border bg-base-100 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary"
							/>
						</Field.Root>
					</div>

					<div class="flex gap-3 mt-6">
						<button
							onclick={closeDialog}
							class="flex-1 px-4 py-2 text-sm font-medium text-base-content bg-base-200 hover:bg-base-300 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							onclick={handleSubmit}
							class="flex-1 px-4 py-2 text-sm font-medium text-white bg-primary hover:bg-primary-hover rounded-lg transition-colors"
						>
							{editingUser ? 'Save Changes' : 'Add User'}
						</button>
					</div>
				</div>
			</Dialog.Content>
		</Dialog.Positioner>
	</Dialog.Root>
{/if}

<!-- Delete Confirmation Dialog -->
{#if showDeleteDialog && userToDelete}
	<Dialog.Root open={showDeleteDialog} onOpenChange={(details) => { if (!details.open) { showDeleteDialog = false; userToDelete = null; } }}>
		<Dialog.Backdrop class="fixed inset-0 bg-black/50 z-40" />
		<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<Dialog.Content class="bg-base-100 rounded-xl shadow-xl max-w-md w-full">
				<div class="p-6">
					<div class="flex items-start gap-4 mb-6">
						<div class="shrink-0 w-12 h-12 rounded-full bg-error/10 flex items-center justify-center">
							<svg class="w-6 h-6 text-error" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
							</svg>
						</div>
						<div class="flex-1">
							<Dialog.Title class="text-lg font-semibold text-base-content mb-2">
								Delete User
							</Dialog.Title>
							<Dialog.Description class="text-sm text-base-muted">
								Are you sure you want to delete <strong class="text-base-content">{userToDelete.username}</strong>? This action cannot be undone.
							</Dialog.Description>
						</div>
					</div>

					<div class="flex gap-3">
						<button
							onclick={() => { showDeleteDialog = false; userToDelete = null; }}
							class="flex-1 px-4 py-2 text-sm font-medium text-base-content bg-base-200 hover:bg-base-300 rounded-lg transition-colors"
						>
							Cancel
						</button>
						<button
							onclick={confirmDelete}
							class="flex-1 px-4 py-2 text-sm font-medium text-white bg-error hover:bg-error/90 rounded-lg transition-colors"
						>
							Delete User
						</button>
					</div>
				</div>
			</Dialog.Content>
		</Dialog.Positioner>
	</Dialog.Root>
{/if}
