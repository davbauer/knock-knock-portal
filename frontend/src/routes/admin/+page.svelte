<script lang="ts">
	import { onMount } from 'svelte';
	import { Eye, EyeOff, Lock, Shield, AlertTriangle, ChevronDown } from 'lucide-svelte';
	import { API_BASE_URL } from '$lib/config';
	import { goto } from '$app/navigation';
	
	let password = $state('');
	let showPassword = $state(false);
	let isLoading = $state(false);
	let error = $state('');
	let showDefaultPassword = $state(false);
	let isCheckingAuth = $state(true);

	onMount(async () => {
		// Check if user is already logged in
		const token = localStorage.getItem('admin_token');
		if (token) {
			// Verify token is still valid
			try {
				const response = await fetch(`${API_BASE_URL}/api/admin/sessions`, {
					headers: {
						'Authorization': `Bearer ${token}`,
					},
				});

				if (response.ok) {
					// Token is valid, redirect to dashboard
					goto('/admin/dashboard');
					return;
				} else {
					// Token is invalid, remove it
					localStorage.removeItem('admin_token');
				}
			} catch (err) {
				// Network error or token invalid
				localStorage.removeItem('admin_token');
			}
		}
		isCheckingAuth = false;
	});

	async function handleLogin() {
		if (!password) {
			error = 'Please enter your admin password';
			return;
		}

		isLoading = true;
		error = '';

		try {
			const response = await fetch(`${API_BASE_URL}/api/admin/login`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({ admin_password: password }),
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error || 'Login failed');
			}

			// Store JWT token
			localStorage.setItem('admin_token', data.data.jwt_access_token);
			
			// Redirect to admin dashboard
			window.location.href = '/admin/dashboard';
		} catch (err) {
			error = err instanceof Error ? err.message : 'An error occurred';
		} finally {
			isLoading = false;
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			handleLogin();
		}
	}
</script>

{#if isCheckingAuth}
	<div class="flex min-h-[calc(100vh-16rem)] items-center justify-center">
		<div class="flex items-center gap-3 text-base-muted">
			<svg class="h-6 w-6 animate-spin" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
			</svg>
			<span class="text-sm">Checking authentication...</span>
		</div>
	</div>
{:else}
<div class="flex min-h-[calc(100vh-16rem)] items-center justify-center">
	<div class="w-full max-w-md">
		<!-- Card -->
		<div class="rounded-xl border border-border bg-base-100 p-8 shadow-sm">
			<!-- Header -->
			<div class="mb-8 text-center">
				<div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
					<Shield class="h-8 w-8 text-primary" />
				</div>
				<h2 class="text-2xl font-bold text-base-content">Admin Login</h2>
				<p class="mt-2 text-sm text-base-muted">
					Enter your admin credentials to access the dashboard
				</p>
			</div>

			<!-- Form -->
			<div class="space-y-6">
				<!-- Password Input -->
				<div>
					<label for="admin-password" class="mb-2 block text-sm font-medium text-base-content">
						Admin Password
					</label>
					
					<div class="relative">
						<div class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
							<Lock class="h-5 w-5 text-base-muted" />
						</div>
						
						<input
							id="admin-password"
							type={showPassword ? 'text' : 'password'}
							bind:value={password}
							onkeydown={handleKeydown}
							placeholder="Enter your admin password"
							class="block w-full rounded-lg border border-border bg-base-100 py-2.5 pl-10 pr-12 text-sm text-base-content placeholder:text-base-muted focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
						/>

						<button
							type="button"
							onclick={() => showPassword = !showPassword}
							class="absolute inset-y-0 right-0 flex items-center pr-3 text-base-muted hover:text-base-content"
						>
							{#if showPassword}
								<EyeOff class="h-5 w-5" />
							{:else}
								<Eye class="h-5 w-5" />
							{/if}
						</button>
					</div>
				</div>

				<!-- Error Message -->
				{#if error}
					<div class="rounded-lg bg-error/10 p-3">
						<p class="text-sm text-error">{error}</p>
					</div>
				{/if}

				<!-- Login Button -->
				<button
					onclick={handleLogin}
					disabled={isLoading}
					class="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-primary-hover focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-base-100 disabled:cursor-not-allowed disabled:opacity-50"
				>
					{#if isLoading}
						<span class="flex items-center justify-center gap-2">
							<svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
							</svg>
							Logging in...
						</span>
					{:else}
						Sign In
					{/if}
				</button>
			</div>

			<!-- Footer -->
			<div class="mt-6 text-center">
				<a href="/" class="text-sm text-primary hover:text-primary-hover">
					← Back to Portal Login
				</a>
			</div>
		</div>

		<!-- Info Boxes -->
		<div class="mt-6 space-y-4">
			<!-- Default Password (Collapsible) -->
			<details class="group rounded-lg border border-border bg-base-200 overflow-hidden">
				<summary class="flex items-center justify-between p-4 cursor-pointer hover:bg-base-300 transition-colors">
					<div class="flex items-center gap-2">
						<Lock class="h-4 w-4 text-base-muted" />
						<span class="text-sm font-medium text-base-content">Show Default Password</span>
					</div>
					<ChevronDown class="h-4 w-4 text-base-muted transition-transform group-open:rotate-180" />
				</summary>
				<div class="border-t border-border p-4 bg-base-100">
					<code class="block rounded bg-base-200 px-3 py-2 text-sm font-mono text-base-content border border-border mb-3">
						admin123
					</code>
					<div class="flex items-start gap-2">
						<AlertTriangle class="h-4 w-4 text-error shrink-0 mt-0.5" />
						<p class="text-xs text-base-content">
							<strong class="text-error">Security Warning:</strong> Change this password immediately after first login by updating <code class="text-xs bg-base-200 px-1 py-0.5 rounded border border-border">ADMIN_PASSWORD_BCRYPT_HASH</code> in your <code class="text-xs bg-base-200 px-1 py-0.5 rounded border border-border">.env</code> file.
						</p>
					</div>
				</div>
			</details>

			<!-- Info Box -->
			<div class="rounded-lg border border-border bg-base-200 p-4">
				<p class="text-xs text-base-muted">
					<strong>Admin access</strong> allows you to manage active sessions, view connected users, and control access to protected services.
				</p>
			</div>
		</div>
	</div>
</div>
{/if}
