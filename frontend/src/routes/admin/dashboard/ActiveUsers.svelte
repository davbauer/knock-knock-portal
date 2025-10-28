<script lang="ts">
	import { Users, UserX, Trash2 } from 'lucide-svelte';
	import { Dialog } from '@ark-ui/svelte';
	import type { Session } from './types';

	interface Props {
		sessions: Session[];
		onRefresh?: () => void;
		onTerminate?: (session: Session) => void;
	}

	let { sessions, onRefresh, onTerminate }: Props = $props();

	let showTerminateDialog = $state(false);
	let sessionToTerminate = $state<Session | null>(null);

	function openTerminateDialog(session: Session) {
		sessionToTerminate = session;
		showTerminateDialog = true;
	}

	function confirmTerminate() {
		if (sessionToTerminate && onTerminate) {
			onTerminate(sessionToTerminate);
		}
		showTerminateDialog = false;
		sessionToTerminate = null;
	}

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

	// Helper to format relative time
	function formatExpiresIn(expiresAt: string): string {
		const now = new Date();
		const expires = new Date(expiresAt);
		const diff = expires.getTime() - now.getTime();

		if (diff < 0) return 'Expired';

		const hours = Math.floor(diff / (1000 * 60 * 60));
		const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

		if (hours > 0) {
			return `${hours}h ${minutes}m`;
		}
		return `${minutes}m`;
	}
</script>

{#if sessions.length === 0}
	<div class="border-border bg-base-100 rounded-xl border p-12 text-center">
		<UserX class="text-base-muted mx-auto mb-4 h-12 w-12" />
		<h3 class="text-base-content mb-2 text-lg font-semibold">No Active Users</h3>
		<p class="text-base-muted text-sm">
			There are currently no authenticated users with active portal sessions.
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
		<div class="flex items-center justify-between">
			<p class="text-base-muted text-sm">
				{sessions.length} active {sessions.length === 1 ? 'user' : 'users'}
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
								Traffic Stats
							</th>
							<th
								class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
							>
								Sessions
							</th>
							<th
								class="text-base-muted px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
							>
								Expires
							</th>
							<th
								class="text-base-muted px-6 py-3 text-right text-xs font-medium uppercase tracking-wider"
							>
								Actions
							</th>
						</tr>
					</thead>
					<tbody class="divide-border divide-y">
						{#each sessions as session (session.session_id)}
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
												<span class="bg-base-200 rounded px-2 py-0.5 text-xs">{ip}</span>
											</div>
										{/each}
									</div>
								</td>
								<td class="whitespace-nowrap px-6 py-4">
									<div class="space-y-1.5">
										<div class="flex items-center gap-2">
											<span class="text-info text-xs font-bold">↓</span>
											<span class="text-base-content font-mono text-xs">
												{formatNumber(session.total_packets_rx)} pkts
											</span>
											<span class="text-base-muted text-xs">
												({formatBytes(session.total_bytes_rx)})
											</span>
										</div>
										<div class="flex items-center gap-2">
											<span class="text-success text-xs font-bold">↑</span>
											<span class="text-base-content font-mono text-xs">
												{formatNumber(session.total_packets_tx)} pkts
											</span>
											<span class="text-base-muted text-xs">
												({formatBytes(session.total_bytes_tx)})
											</span>
										</div>
									</div>
								</td>
								<td class="whitespace-nowrap px-6 py-4">
									<span
										class="bg-primary/10 text-primary inline-flex h-8 w-8 items-center justify-center rounded-lg text-sm font-bold"
									>
										{session.total_sessions}
									</span>
								</td>
								<td class="whitespace-nowrap px-6 py-4">
									<div class="text-base-content text-sm">
										{formatExpiresIn(session.expires_at)}
									</div>
								</td>
								<td class="whitespace-nowrap px-6 py-4 text-right">
									<button
										onclick={() => openTerminateDialog(session)}
										class="border-error/30 bg-error/10 text-error hover:bg-error/20 inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors"
									>
										<Trash2 class="h-3.5 w-3.5" />
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

<!-- Terminate Confirmation Dialog -->
<Dialog.Root open={showTerminateDialog} onOpenChange={(e) => (showTerminateDialog = e.open)}>
	<Dialog.Backdrop class="fixed inset-0 bg-black/50 backdrop-blur-sm" />
	<Dialog.Positioner class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<Dialog.Content
			class="border-border bg-base-100 w-full max-w-md rounded-2xl border p-6 shadow-2xl"
		>
			<Dialog.Title class="text-base-content mb-4 text-xl font-bold">
				Terminate Session?
			</Dialog.Title>
			<Dialog.Description class="text-base-muted mb-6 text-sm">
				Are you sure you want to terminate the session for
				<strong class="text-base-content">{sessionToTerminate?.username}</strong>? This will
				immediately disconnect all their active connections and remove their IP from the allowlist.
			</Dialog.Description>

			{#if sessionToTerminate}
				<div class="bg-base-200 mb-6 rounded-lg p-4">
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-base-muted">Username:</span>
							<span class="text-base-content font-medium">{sessionToTerminate.username}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-base-muted">IP Addresses:</span>
							<span class="text-base-content font-mono text-xs">
								{sessionToTerminate.authenticated_ips.join(', ')}
							</span>
						</div>
					</div>
				</div>
			{/if}

			<div class="flex gap-3">
				<Dialog.CloseTrigger
					class="border-border bg-base-100 text-base-content hover:bg-base-200 flex-1 rounded-lg border px-4 py-2.5 text-sm font-semibold transition-colors"
				>
					Cancel
				</Dialog.CloseTrigger>
				<button
					onclick={confirmTerminate}
					class="bg-error hover:bg-error-hover flex-1 rounded-lg px-4 py-2.5 text-sm font-semibold text-white transition-colors"
				>
					Terminate Session
				</button>
			</div>
		</Dialog.Content>
	</Dialog.Positioner>
</Dialog.Root>
