<script lang="ts">
	import { Network, Users, Shield, Download, Upload, Trash2 } from 'lucide-svelte';
	import type { Connection } from '../types';

	interface Props {
		connections: Connection[];
		onRefresh?: () => void;
		onTerminate?: (connection: Connection) => void;
	}

	let { connections, onRefresh, onTerminate }: Props = $props();

	// Helper to format bytes
	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
	}

	// Helper to format large numbers
	function formatNumber(num: number): string {
		if (num === 0) return '0';
		if (num < 1000) return num.toString();
		if (num < 1000000) return `${(num / 1000).toFixed(1)}K`;
		return `${(num / 1000000).toFixed(1)}M`;
	}

	const authenticatedCount = $derived(connections.filter((c) => c.authenticated).length);
	const anonymousCount = $derived(connections.filter((c) => !c.authenticated).length);
</script>

{#if connections.length === 0}
	<div class="border-border bg-base-100 rounded-xl border p-12 text-center">
		<Network class="text-base-muted mx-auto mb-4 h-12 w-12" />
		<h3 class="text-base-content mb-2 text-lg font-semibold">No Active Connections</h3>
		<p class="text-base-muted text-sm">
			Connections will appear here when users connect to proxy services.
		</p>
		{#if onRefresh}
			<button
				onclick={onRefresh}
				class="bg-primary hover:bg-primary-hover mt-4 rounded-lg px-4 py-2 text-sm font-semibold text-white"
			>
				Refresh
			</button>
		{/if}
	</div>
{:else}
	<div class="space-y-4">
		<!-- Stats Overview -->
		<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
			<!-- Total Connections -->
			<div class="border-border bg-base-100 rounded-xl border p-6 shadow-sm">
				<div class="flex items-center gap-4">
					<div class="bg-primary/10 flex h-12 w-12 items-center justify-center rounded-xl">
						<Network class="text-primary h-6 w-6" />
					</div>
					<div>
						<p class="text-base-muted text-sm font-medium">Total Connections</p>
						<p class="text-base-content text-2xl font-bold">{connections.length}</p>
					</div>
				</div>
			</div>

			<!-- Authenticated -->
			<div class="border-border bg-base-100 rounded-xl border p-6 shadow-sm">
				<div class="flex items-center gap-4">
					<div class="bg-success/10 flex h-12 w-12 items-center justify-center rounded-xl">
						<Users class="text-success h-6 w-6" />
					</div>
					<div>
						<p class="text-base-muted text-sm font-medium">Authenticated</p>
						<p class="text-base-content text-2xl font-bold">{authenticatedCount}</p>
					</div>
				</div>
			</div>

			<!-- Anonymous -->
			<div class="border-border bg-base-100 rounded-xl border p-6 shadow-sm">
				<div class="flex items-center gap-4">
					<div class="bg-warning/10 flex h-12 w-12 items-center justify-center rounded-xl">
						<Shield class="text-warning h-6 w-6" />
					</div>
					<div>
						<p class="text-base-muted text-sm font-medium">Anonymous</p>
						<p class="text-base-content text-2xl font-bold">{anonymousCount}</p>
					</div>
				</div>
			</div>
		</div>

		<!-- Refresh Button -->
		<div class="flex items-center justify-between">
			<p class="text-base-muted text-sm">
				{connections.length} active {connections.length === 1 ? 'connection' : 'connections'}
			</p>
			{#if onRefresh}
				<button
					onclick={onRefresh}
					class="border-border bg-base-100 text-base-content hover:bg-base-200 rounded-lg border px-3 py-1.5 text-xs font-medium"
				>
					Refresh
				</button>
			{/if}
		</div>

		<!-- Connections Table -->
		<div class="border-border bg-base-100 overflow-hidden rounded-xl border shadow-sm">
			<div class="overflow-x-auto">
				<table class="w-full">
					<thead class="bg-base-200 border-border border-b">
						<tr>
							<th
								class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
							>
								IP Address
							</th>
							<th
								class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
							>
								User
							</th>
							<th
								class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
							>
								Authentication
							</th>
							<th
								class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
							>
								Traffic Stats
							</th>
							<th
								class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
							>
								Sessions
							</th>
							<th
								class="text-base-muted px-6 py-3 text-right text-xs font-medium uppercase tracking-wider"
							>
								Actions
							</th>
						</tr>
					</thead>
					<tbody class="divide-border divide-y">
						{#each connections as conn (conn.ip)}
							<tr class="hover:bg-base-200 transition-colors">
								<td class="whitespace-nowrap px-6 py-4">
									<code class="bg-base-200 text-base-content rounded px-2 py-1 font-mono text-sm">
										{conn.ip}
									</code>
								</td>
								<td class="whitespace-nowrap px-6 py-4">
									<div class="flex items-center gap-3">
										{#if conn.authenticated}
											<div
												class="bg-success/10 flex h-10 w-10 items-center justify-center rounded-full"
											>
												<Users class="text-success h-5 w-5" />
											</div>
											<div>
												<div class="text-base-content text-sm font-medium">
													{conn.username}
												</div>
												<div class="text-success text-xs font-medium">Portal User</div>
											</div>
										{:else}
											<div
												class="bg-warning/10 flex h-10 w-10 items-center justify-center rounded-full"
											>
												<Shield class="text-warning h-5 w-5" />
											</div>
											<div>
												<div class="text-base-muted text-sm font-medium italic">
													{conn.username}
												</div>
												<div class="text-warning text-xs font-medium">Permanent Allowlist</div>
											</div>
										{/if}
									</div>
								</td>
								<td class="whitespace-nowrap px-6 py-4">
									{#if conn.authenticated}
										<span
											class="bg-success/10 text-success inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium"
										>
											<CircleCheck class="h-3.5 w-3.5" />
											Authenticated
										</span>
									{:else}
										<span
											class="bg-warning/10 text-warning inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium"
										>
											<Shield class="h-3.5 w-3.5" />
											Allowlist Entry
										</span>
									{/if}
								</td>
								<td class="whitespace-nowrap px-6 py-4">
									<div class="space-y-1.5">
										<div class="flex items-center gap-2">
											<span class="text-info text-xs font-bold">↓</span>
											<span class="text-base-content font-mono text-xs">
												{formatNumber(conn.total_packets_rx)} pkts
											</span>
											<span class="text-base-muted text-xs">
												({formatBytes(conn.total_bytes_rx)})
											</span>
										</div>
										<div class="flex items-center gap-2">
											<span class="text-success text-xs font-bold">↑</span>
											<span class="text-base-content font-mono text-xs">
												{formatNumber(conn.total_packets_tx)} pkts
											</span>
											<span class="text-base-muted text-xs">
												({formatBytes(conn.total_bytes_tx)})
											</span>
										</div>
									</div>
								</td>
								<td class="whitespace-nowrap px-6 py-4">
									<span
										class="bg-primary/10 text-primary inline-flex h-8 w-8 items-center justify-center rounded-lg text-sm font-bold"
									>
										{conn.total_sessions}
									</span>
								</td>
								<td class="whitespace-nowrap px-6 py-4 text-right">
									{#if onTerminate}
										<button
											onclick={() => onTerminate?.(conn)}
											class="bg-error/10 text-error hover:bg-error inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors hover:text-white"
											title="Terminate all connections from {conn.ip}"
										>
											<Trash2 class="h-3.5 w-3.5" />
											Terminate
										</button>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	</div>
{/if}
