<script lang="ts">
	import type { Config, ProtectedService } from './types';
	import { Dialog, Switch, Field, Checkbox } from '@ark-ui/svelte';
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
	let editingService = $state<ProtectedService | null>(null);
	let serviceToDelete = $state<ProtectedService | null>(null);

	// Form state
	let formServiceId = $state('');
	let formServiceName = $state('');
	let formDescription = $state('');
	let formProxyPortStart = $state<number>(10000);
	let formProxyPortEnd = $state<number>(10000);
	let formBackendHost = $state('');
	let formBackendPort = $state<number>(8080);
	let formTransportProtocol = $state('tcp');
	let formIsHttp = $state(false);
	let formEnabled = $state(true);

	// HTTP headers state
	let formInjectRequestHeaders = $state('');
	let formOverrideRequestHeaders = $state('');
	let formRemoveRequestHeaders = $state('');
	let formInjectResponseHeaders = $state('');

	// Mirror proxy port start to end and backend port when creating new service
	function handleProxyPortStartChange(value: number) {
		formProxyPortStart = value;
		if (!editingService) {
			formProxyPortEnd = value;
			formBackendPort = value;
		}
	}

	function openAddDialog() {
		editingService = null;
		formServiceId = generateUUID();
		formServiceName = '';
		formDescription = '';
		formProxyPortStart = 10000;
		formProxyPortEnd = 10000;
		formBackendHost = 'localhost';
		formBackendPort = 8080;
		formTransportProtocol = 'tcp';
		formIsHttp = false;
		formEnabled = true;
		formInjectRequestHeaders = '';
		formOverrideRequestHeaders = '';
		formRemoveRequestHeaders = '';
		formInjectResponseHeaders = '';
		showAddDialog = true;
	}

	function openEditDialog(service: ProtectedService) {
		editingService = service;
		formServiceId = service.service_id;
		formServiceName = service.service_name;
		formDescription = service.description;
		formProxyPortStart = service.proxy_listen_port_start;
		formProxyPortEnd = service.proxy_listen_port_end;
		formBackendHost = service.backend_target_host;
		formBackendPort = service.backend_target_port;
		formTransportProtocol = service.transport_protocol;
		formIsHttp = service.is_http_protocol;
		formEnabled = service.enabled;

		// Load HTTP headers
		if (service.http_config) {
			formInjectRequestHeaders = service.http_config.inject_http_request_headers
				? Object.entries(service.http_config.inject_http_request_headers)
						.map(([k, v]) => `${k}: ${v}`)
						.join('\n')
				: '';
			formOverrideRequestHeaders = service.http_config.override_http_request_headers
				? Object.entries(service.http_config.override_http_request_headers)
						.map(([k, v]) => `${k}: ${v}`)
						.join('\n')
				: '';
			formRemoveRequestHeaders = service.http_config.remove_http_request_headers
				? service.http_config.remove_http_request_headers.join('\n')
				: '';
			formInjectResponseHeaders = service.http_config.inject_http_response_headers
				? Object.entries(service.http_config.inject_http_response_headers)
						.map(([k, v]) => `${k}: ${v}`)
						.join('\n')
				: '';
		} else {
			formInjectRequestHeaders = '';
			formOverrideRequestHeaders = '';
			formRemoveRequestHeaders = '';
			formInjectResponseHeaders = '';
		}

		showAddDialog = true;
	}

	function closeDialog() {
		showAddDialog = false;
		editingService = null;
	}

	function parseHeadersToMap(text: string): Record<string, string> {
		const result: Record<string, string> = {};
		text.split('\n').forEach((line) => {
			const trimmed = line.trim();
			if (trimmed && trimmed.includes(':')) {
				const [key, ...valueParts] = trimmed.split(':');
				const value = valueParts.join(':').trim();
				if (key.trim()) {
					result[key.trim()] = value;
				}
			}
		});
		return result;
	}

	function parseHeadersToArray(text: string): string[] {
		return text
			.split('\n')
			.map((line) => line.trim())
			.filter((line) => line.length > 0);
	}

	function handleSubmit() {
		// Validation
		if (!formServiceName.trim()) {
			toaster.error({
				title: 'Validation Error',
				description: 'Service Name is required'
			});
			return;
		}

		if (!formBackendHost.trim()) {
			toaster.error({
				title: 'Validation Error',
				description: 'Backend host is required'
			});
			return;
		}

		// Build HTTP config if HTTP is enabled
		let httpConfig = null;
		if (formIsHttp) {
			const injectReq = parseHeadersToMap(formInjectRequestHeaders);
			const overrideReq = parseHeadersToMap(formOverrideRequestHeaders);
			const removeReq = parseHeadersToArray(formRemoveRequestHeaders);
			const injectRes = parseHeadersToMap(formInjectResponseHeaders);

			httpConfig = {
				inject_http_request_headers: Object.keys(injectReq).length > 0 ? injectReq : undefined,
				override_http_request_headers:
					Object.keys(overrideReq).length > 0 ? overrideReq : undefined,
				remove_http_request_headers: removeReq.length > 0 ? removeReq : undefined,
				inject_http_response_headers: Object.keys(injectRes).length > 0 ? injectRes : undefined
			};
		}

		const newService: ProtectedService = {
			service_id: formServiceId.trim(),
			service_name: formServiceName.trim(),
			description: formDescription.trim(),
			proxy_listen_port_start: formProxyPortStart,
			proxy_listen_port_end: formProxyPortEnd,
			backend_target_host: formBackendHost.trim(),
			backend_target_port: formBackendPort,
			transport_protocol: formTransportProtocol,
			is_http_protocol: formIsHttp,
			enabled: formEnabled,
			http_config: httpConfig
		};

		if (editingService) {
			// Update existing service
			const index = config.protected_services.findIndex(
				(s) => s.service_id === editingService!.service_id
			);
			if (index !== -1) {
				configStore.updateConfig((cfg) => {
					cfg.protected_services[index] = newService;
				});
			}
		} else {
			// Add new service
			configStore.updateConfig((cfg) => {
				if (!cfg.protected_services) {
					cfg.protected_services = [];
				}
				cfg.protected_services.push(newService);
			});
		}

		closeDialog();
	}

	function openDeleteDialog(service: ProtectedService) {
		serviceToDelete = service;
		showDeleteDialog = true;
	}

	function confirmDelete() {
		if (!serviceToDelete) return;
		configStore.updateConfig((cfg) => {
			cfg.protected_services = cfg.protected_services.filter(
				(s) => s.service_id !== serviceToDelete!.service_id
			);
		});
		showDeleteDialog = false;
		serviceToDelete = null;
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

<div class="border-border bg-base-100 rounded-xl border p-6">
	<div class="mb-4 flex items-center justify-between">
		<h3 class="text-base-content text-lg font-semibold">Protected Services</h3>
		<button
			onclick={openAddDialog}
			class="bg-primary hover:bg-primary-hover flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold text-white"
		>
			<Plus class="h-4 w-4" />
			Add Service
		</button>
	</div>

	{#if !config.protected_services || config.protected_services.length === 0}
		<div class="text-base-muted py-12 text-center">
			<svg
				class="mx-auto mb-3 h-12 w-12 opacity-50"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
				/>
			</svg>
			<p class="text-sm">No protected services configured</p>
			<p class="mt-1 text-xs">Add services to make them available through the portal</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each config.protected_services as service}
				<div class="border-border hover:bg-base-200 rounded-lg border p-4 transition-colors">
					<div class="flex items-start justify-between">
						<div class="flex-1">
							<div class="mb-2 flex flex-wrap items-center gap-2">
								<h4 class="text-base-content font-semibold">{service.service_name}</h4>
								<span class="bg-base-300 text-base-muted rounded px-2 py-0.5 font-mono text-xs"
									>{abbreviateId(service.service_id)}</span
								>
								{#if service.enabled}
									<span class="bg-success/10 text-success rounded px-2 py-0.5 text-xs font-medium"
										>Enabled</span
									>
								{:else}
									<span
										class="bg-base-muted/10 text-base-muted rounded px-2 py-0.5 text-xs font-medium"
										>Disabled</span
									>
								{/if}
								<span class="bg-primary/10 text-primary rounded px-2 py-0.5 text-xs font-medium"
									>{formatProtocol(service.transport_protocol)}</span
								>
								{#if service.is_http_protocol}
									<span class="rounded bg-blue-500/10 px-2 py-0.5 text-xs font-medium text-blue-500"
										>HTTP</span
									>
								{/if}
							</div>
							<div class="text-base-muted space-y-1 text-sm">
								<p>
									Ports: {#if service.proxy_listen_port_start === service.proxy_listen_port_end}
										{service.proxy_listen_port_start}
									{:else}
										{service.proxy_listen_port_start}-{service.proxy_listen_port_end}
									{/if}
									→ {service.backend_target_host}:{service.backend_target_port}
								</p>
								{#if service.description}
									<p class="text-xs italic">{service.description}</p>
								{/if}
							</div>
						</div>
						<div class="flex items-center gap-2">
							<button
								onclick={() => openEditDialog(service)}
								class="text-base-muted hover:text-primary p-1 transition-colors"
								title="Edit service"
							>
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
									/>
								</svg>
							</button>
							<button
								onclick={() => openDeleteDialog(service)}
								class="text-base-muted hover:text-error p-1 transition-colors"
								title="Delete service"
							>
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
									/>
								</svg>
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Add/Edit Service Dialog -->
{#if showAddDialog}
	<Dialog.Root
		open={showAddDialog}
		onOpenChange={(details) => {
			if (!details.open) closeDialog();
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
							{editingService ? 'Edit Service' : 'Add New Service'}
						</Dialog.Title>
						<Dialog.CloseTrigger
							onclick={closeDialog}
							class="text-base-muted hover:text-base-content transition-colors"
						>
							<X class="h-5 w-5" />
						</Dialog.CloseTrigger>
					</div>

					<div class="space-y-4">
						<div class="grid grid-cols-2 gap-4">
							<!-- Service ID (auto-generated, read-only) -->
							<Field.Root>
								<Field.Label class="text-base-content mb-2 text-sm font-medium">
									Service ID
									<span class="text-base-muted ml-1 text-xs font-normal">(auto-generated)</span>
								</Field.Label>
								<Field.Input
									type="text"
									value={formServiceId}
									readonly
									class="border-border bg-base-200 text-base-muted w-full cursor-not-allowed rounded-lg border px-3 py-2 font-mono text-sm"
								/>
							</Field.Root>

							<!-- Service Name -->
							<Field.Root>
								<Field.Label class="text-base-content mb-2 text-sm font-medium"
									>Service Name</Field.Label
								>
								<Field.Input
									type="text"
									bind:value={formServiceName}
									placeholder="e.g., Web Application"
									class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
								/>
							</Field.Root>
						</div>

						<!-- Description -->
						<Field.Root>
							<Field.Label class="text-base-content mb-2 text-sm font-medium"
								>Description</Field.Label
							>
							<Field.Input
								type="text"
								bind:value={formDescription}
								placeholder="Optional description"
								class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
							/>
						</Field.Root>

						<div class="grid grid-cols-2 gap-4">
							<!-- Proxy Port Start -->
							<Field.Root>
								<Field.Label class="text-base-content mb-2 text-sm font-medium"
									>Proxy Port Start</Field.Label
								>
								<Field.Input
									type="number"
									value={formProxyPortStart}
									oninput={(e) => handleProxyPortStartChange(Number(e.currentTarget.value))}
									min="1"
									max="65535"
									class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
								/>
							</Field.Root>

							<!-- Proxy Port End -->
							<Field.Root>
								<Field.Label class="text-base-content mb-2 text-sm font-medium"
									>Proxy Port End</Field.Label
								>
								<Field.Input
									type="number"
									bind:value={formProxyPortEnd}
									min="1024"
									max="65535"
									class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
								/>
							</Field.Root>
						</div>

						<div class="grid grid-cols-3 gap-4">
							<!-- Backend Host -->
							<Field.Root class="col-span-2">
								<Field.Label class="text-base-content mb-2 text-sm font-medium"
									>Backend Host</Field.Label
								>
								<Field.Input
									type="text"
									bind:value={formBackendHost}
									placeholder="e.g., localhost or 192.168.1.100"
									class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
								/>
							</Field.Root>

							<!-- Backend Port -->
							<Field.Root>
								<Field.Label class="text-base-content mb-2 text-sm font-medium"
									>Backend Port</Field.Label
								>
								<Field.Input
									type="number"
									bind:value={formBackendPort}
									min="1"
									max="65535"
									class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
								/>
							</Field.Root>
						</div>

						<div class="grid grid-cols-2 gap-4">
							<!-- Transport Protocol -->
							<Field.Root>
								<Field.Label class="text-base-content mb-2 text-sm font-medium"
									>Transport Protocol</Field.Label
								>
								<Field.Select
									bind:value={formTransportProtocol}
									class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 text-sm focus:outline-none focus:ring-2"
								>
									<option value="tcp">TCP</option>
									<option value="udp">UDP</option>
									<option value="both">Both (TCP + UDP)</option>
								</Field.Select>
							</Field.Root>

							<!-- Enabled -->
							<div class="flex items-end pb-2">
								<Switch.Root
									checked={formEnabled}
									onCheckedChange={(e) => (formEnabled = e.checked)}
									class="flex items-center gap-3"
								>
									<Switch.Control
										class="bg-border data-[state=checked]:bg-primary relative h-6 w-11 rounded-full transition-colors"
									>
										<Switch.Thumb
											class="data-[state=checked]:translate-x-5 absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform"
										/>
									</Switch.Control>
									<Switch.Label class="text-base-content cursor-pointer text-sm font-medium"
										>Service Enabled</Switch.Label
									>
									<Switch.HiddenInput />
								</Switch.Root>
							</div>
						</div>

						<!-- HTTP Protocol -->
						<Checkbox.Root bind:checked={formIsHttp} class="flex items-center gap-3">
							<Checkbox.Control
								class="border-border bg-base-100 data-[state=checked]:bg-primary data-[state=checked]:border-primary flex h-5 w-5 items-center justify-center rounded border-2 transition-colors"
							>
								<Checkbox.Indicator>
									<Check class="h-3 w-3 text-white" />
								</Checkbox.Indicator>
							</Checkbox.Control>
							<Checkbox.Label class="text-base-content cursor-pointer text-sm">
								HTTP/HTTPS Protocol (enables header manipulation)
							</Checkbox.Label>
							<Checkbox.HiddenInput />
						</Checkbox.Root>

						<!-- HTTP Headers Configuration (only shown when HTTP is enabled) -->
						{#if formIsHttp}
							<div class="border-primary/20 bg-primary/5 space-y-4 rounded-lg border-2 p-4">
								<h4 class="text-base-content text-sm font-semibold">HTTP Header Configuration</h4>

								<!-- Inject Request Headers -->
								<Field.Root>
									<Field.Label class="text-base-content mb-2 text-sm font-medium"
										>Inject Request Headers</Field.Label
									>
									<Field.Textarea
										bind:value={formInjectRequestHeaders}
										rows={3}
										placeholder="Header-Name: value&#10;Another-Header: another value"
										class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
									/>
									<Field.HelperText class="text-base-muted mt-1 text-xs">
										Add headers to requests (one per line, format: Header-Name: value)
									</Field.HelperText>
								</Field.Root>

								<!-- Override Request Headers -->
								<Field.Root>
									<Field.Label class="text-base-content mb-2 text-sm font-medium"
										>Override Request Headers</Field.Label
									>
									<Field.Textarea
										bind:value={formOverrideRequestHeaders}
										rows={3}
										placeholder="Host: example.com&#10;X-Forwarded-For: client-ip"
										class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
									/>
									<Field.HelperText class="text-base-muted mt-1 text-xs">
										Replace existing headers (useful for Host header)
									</Field.HelperText>
								</Field.Root>

								<!-- Remove Request Headers -->
								<Field.Root>
									<Field.Label class="text-base-content mb-2 text-sm font-medium"
										>Remove Request Headers</Field.Label
									>
									<Field.Textarea
										bind:value={formRemoveRequestHeaders}
										rows={2}
										placeholder="X-Forwarded-For&#10;Cookie"
										class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
									/>
									<Field.HelperText class="text-base-muted mt-1 text-xs">
										Remove headers from requests (one header name per line)
									</Field.HelperText>
								</Field.Root>

								<!-- Inject Response Headers -->
								<Field.Root>
									<Field.Label class="text-base-content mb-2 text-sm font-medium"
										>Inject Response Headers</Field.Label
									>
									<Field.Textarea
										bind:value={formInjectResponseHeaders}
										rows={2}
										placeholder="X-Frame-Options: DENY&#10;X-Content-Type-Options: nosniff"
										class="border-border bg-base-100 text-base-content focus:ring-primary w-full rounded-lg border px-3 py-2 font-mono text-sm focus:outline-none focus:ring-2"
									/>
									<Field.HelperText class="text-base-muted mt-1 text-xs">
										Add headers to responses (security headers, etc.)
									</Field.HelperText>
								</Field.Root>
							</div>
						{/if}
					</div>

					<div class="mt-6 flex gap-3">
						<button
							onclick={closeDialog}
							class="text-base-content bg-base-200 hover:bg-base-300 flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-colors"
						>
							Cancel
						</button>
						<button
							onclick={handleSubmit}
							class="bg-primary hover:bg-primary-hover flex-1 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors"
						>
							{editingService ? 'Save Changes' : 'Add Service'}
						</button>
					</div>
				</div>
			</Dialog.Content>
		</Dialog.Positioner>
	</Dialog.Root>
{/if}

<!-- Delete Confirmation Dialog -->
{#if showDeleteDialog && serviceToDelete}
	<Dialog.Root
		open={showDeleteDialog}
		onOpenChange={(details) => {
			if (!details.open) {
				showDeleteDialog = false;
				serviceToDelete = null;
			}
		}}
	>
		<Dialog.Backdrop class="fixed inset-0 z-40 bg-black/50" />
		<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<Dialog.Content class="bg-base-100 w-full max-w-md rounded-xl shadow-xl">
				<div class="p-6">
					<div class="mb-6 flex items-start gap-4">
						<div
							class="bg-error/10 flex h-12 w-12 shrink-0 items-center justify-center rounded-full"
						>
							<svg class="text-error h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
								/>
							</svg>
						</div>
						<div class="flex-1">
							<Dialog.Title class="text-base-content mb-2 text-lg font-semibold">
								Delete Service
							</Dialog.Title>
							<Dialog.Description class="text-base-muted text-sm">
								Are you sure you want to delete <strong class="text-base-content"
									>{serviceToDelete.service_name}</strong
								>? This will prevent users from accessing this service.
							</Dialog.Description>
						</div>
					</div>

					<div class="flex gap-3">
						<button
							onclick={() => {
								showDeleteDialog = false;
								serviceToDelete = null;
							}}
							class="text-base-content bg-base-200 hover:bg-base-300 flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-colors"
						>
							Cancel
						</button>
						<button
							onclick={confirmDelete}
							class="bg-error hover:bg-error/90 flex-1 rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors"
						>
							Delete Service
						</button>
					</div>
				</div>
			</Dialog.Content>
		</Dialog.Positioner>
	</Dialog.Root>
{/if}
