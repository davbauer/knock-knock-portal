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
						Authorization: `Bearer ${token}`
					}
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
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ admin_password: password })
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
		<div class="text-base-muted flex items-center gap-3">
			<svg class="h-6 w-6 animate-spin" fill="none" viewBox="0 0 24 24">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
				></circle>
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
				></path>
			</svg>
			<span class="text-sm">Checking authentication...</span>
		</div>
	</div>
{:else}
	<div class="flex min-h-[calc(100vh-16rem)] items-center justify-center">
		<div class="w-full max-w-md">
			<!-- Card -->
			<div class="border-border bg-base-100 rounded-xl border p-8 shadow-sm">
				<!-- Header -->
				<div class="mb-8 text-center">
					<div
						class="bg-primary/10 mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full"
					>
						<Shield class="text-primary h-8 w-8" />
					</div>
					<h2 class="text-base-content text-2xl font-bold">Admin Login</h2>
					<p class="text-base-muted mt-2 text-sm">
						Enter your admin credentials to access the dashboard
					</p>
				</div>

				<!-- Form -->
				<div class="space-y-6">
					<!-- Password Input -->
					<div>
						<label for="admin-password" class="text-base-content mb-2 block text-sm font-medium">
							Admin Password
						</label>

						<div class="relative">
							<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
								<Lock class="text-base-muted h-5 w-5" />
							</div>

							<input
								id="admin-password"
								type={showPassword ? 'text' : 'password'}
								bind:value={password}
								onkeydown={handleKeydown}
								placeholder="Enter your admin password"
								class="border-border bg-base-100 text-base-content placeholder:text-base-muted focus:border-primary focus:ring-primary/20 block w-full rounded-lg border py-2.5 pl-10 pr-12 text-sm focus:outline-none focus:ring-2"
							/>

							<button
								type="button"
								onclick={() => (showPassword = !showPassword)}
								class="text-base-muted hover:text-base-content absolute inset-y-0 right-0 flex items-center pr-3"
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
						<div class="bg-error/10 rounded-lg p-3">
							<p class="text-error text-sm">{error}</p>
						</div>
					{/if}

					<!-- Login Button -->
					<button
						onclick={handleLogin}
						disabled={isLoading}
						class="bg-primary hover:bg-primary-hover focus:ring-primary focus:ring-offset-base-100 w-full rounded-lg px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
					>
						{#if isLoading}
							<span class="flex items-center justify-center gap-2">
								<svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
									<circle
										class="opacity-25"
										cx="12"
										cy="12"
										r="10"
										stroke="currentColor"
										stroke-width="4"
									></circle>
									<path
										class="opacity-75"
										fill="currentColor"
										d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
									></path>
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
					<a href="/" class="text-primary hover:text-primary-hover text-sm">
						← Back to Portal Login
					</a>
				</div>
			</div>

			<!-- Info Boxes -->
			<div class="mt-6 space-y-4">
				<!-- Default Password (Collapsible) -->
				<details class="border-border bg-base-200 group overflow-hidden rounded-lg border">
					<summary
						class="hover:bg-base-300 flex cursor-pointer items-center justify-between p-4 transition-colors"
					>
						<div class="flex items-center gap-2">
							<Lock class="text-base-muted h-4 w-4" />
							<span class="text-base-content text-sm font-medium">Show Default Password</span>
						</div>
						<ChevronDown
							class="text-base-muted h-4 w-4 transition-transform group-open:rotate-180"
						/>
					</summary>
					<div class="border-border bg-base-100 border-t p-4">
						<code
							class="bg-base-200 text-base-content border-border mb-3 block rounded border px-3 py-2 font-mono text-sm"
						>
							admin123
						</code>
						<div class="flex items-start gap-2">
							<AlertTriangle class="text-error mt-0.5 h-4 w-4 shrink-0" />
							<p class="text-base-content text-xs">
								<strong class="text-error">Security Warning:</strong> Change this password
								immediately after first login by updating
								<code class="bg-base-200 border-border rounded border px-1 py-0.5 text-xs"
									>ADMIN_PASSWORD_BCRYPT_HASH</code
								>
								in your
								<code class="bg-base-200 border-border rounded border px-1 py-0.5 text-xs"
									>.env</code
								> file.
							</p>
						</div>
					</div>
				</details>

				<!-- Info Box -->
				<div class="border-border bg-base-200 rounded-lg border p-4">
					<p class="text-base-muted text-xs">
						<strong>Admin access</strong> allows you to manage active sessions, view connected users,
						and control access to protected services.
					</p>
				</div>
			</div>
		</div>
	</div>
{/if}
