<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { slide, fade } from 'svelte/transition';
	import { Shield, Lock, User, Eye, EyeOff, ArrowRight, AlertCircle } from 'lucide-svelte';
	import { API_BASE_URL } from '$lib/config';
	import { Field } from '@ark-ui/svelte';

	let username = $state('');
	let password = $state('');
	let showPassword = $state(false);
	let isLoading = $state(false);
	let error = $state('');
	let suggestedUsernames = $state<string[]>([]);
	let showSuggestions = $state(false);
	let loadingSuggestions = $state(true);

	// Fetch suggested usernames
	async function fetchSuggestedUsernames() {
		try {
			const response = await fetch(`${API_BASE_URL}/api/portal/suggested-usernames`);
			if (response.ok) {
				const data = await response.json();
				suggestedUsernames = data.data.usernames || [];
			}
		} catch (err) {
			console.error('Failed to fetch suggestions:', err);
		} finally {
			loadingSuggestions = false;
		}
	}

	async function handleLogin(e: Event) {
		e.preventDefault();
		error = '';
		isLoading = true;

		try {
			const response = await fetch(`${API_BASE_URL}/api/portal/login`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ username, password })
			});

			const data = await response.json();

			if (!response.ok) {
				error = data.error || 'Login failed. Please check your credentials.';
				isLoading = false;
				return;
			}

		// Store the token and session info
		localStorage.setItem('portal_token', data.data.jwt_access_token);
		localStorage.setItem('portal_session', JSON.stringify(data.data.session_info));

		// Dispatch custom event to trigger connection info refresh
		window.dispatchEvent(new CustomEvent('portal-login-success'));

		// Redirect to portal dashboard
		goto('/portal/dashboard');
		} catch (err) {
			error = 'Network error. Please check your connection and try again.';
			isLoading = false;
		}
	}

	function selectUsername(selectedUsername: string) {
		username = selectedUsername;
		showSuggestions = false;
		// Focus password field
		setTimeout(() => {
			document.getElementById('password-input')?.focus();
		}, 100);
	}

	function toggleSuggestions() {
		if (suggestedUsernames.length > 0) {
			showSuggestions = !showSuggestions;
		}
	}

	onMount(() => {
		// Check if already logged in
		const token = localStorage.getItem('portal_token');
		if (token) {
			goto('/portal/dashboard');
			return;
		}

		fetchSuggestedUsernames();
	});
</script>

<div class="flex min-h-[calc(100vh-16rem)] items-center justify-center px-4 py-12">
	<div class="w-full max-w-md">
		<!-- Logo & Title -->
		<div class="mb-8 text-center">
			<div
				class="bg-primary/10 mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl"
			>
				<Shield class="text-primary h-8 w-8" />
			</div>
			<h1 class="text-base-content text-3xl font-bold">Portal Access</h1>
			<p class="text-base-muted mt-2 text-sm">Sign in to access your protected services</p>
		</div>

		<!-- Login Card -->
		<div class="border-border bg-base-100 rounded-2xl border shadow-xl">
			<form onsubmit={handleLogin} class="p-8">
				<!-- Error Message -->
				{#if error}
					<div
						transition:slide={{ duration: 300 }}
						class="border-error/30 bg-error/5 mb-6 flex items-start gap-3 rounded-lg border p-4"
					>
						<AlertCircle class="text-error mt-0.5 h-5 w-5 shrink-0" />
						<p class="text-error text-sm">{error}</p>
					</div>
				{/if}

				<!-- Username Field -->
				<Field.Root class="mb-6">
					<Field.Label class="text-base-content mb-2 block text-sm font-medium">
						Username
					</Field.Label>
					<div class="relative">
						<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
							<User class="text-base-muted h-5 w-5" />
						</div>
						<Field.Input
							bind:value={username}
							type="text"
							autocomplete="username"
							required
							placeholder="Enter your username"
							onfocus={() => {
								if (suggestedUsernames.length > 0 && !username) {
									showSuggestions = true;
								}
							}}
							oninput={() => {
								error = '';
							}}
							class="border-border bg-base-100 text-base-content placeholder:text-base-muted focus:border-primary focus:ring-primary w-full rounded-lg border py-3 pl-10 pr-4 transition-colors focus:outline-none focus:ring-2"
						/>
					</div>

					<!-- Username Suggestions -->
					{#if suggestedUsernames.length > 0}
						<button
							type="button"
							onclick={toggleSuggestions}
							class="text-primary hover:text-primary-hover mt-2 text-xs font-medium transition-colors"
						>
							{showSuggestions ? 'Hide' : 'Show'} suggested usernames ({suggestedUsernames.length})
						</button>

						{#if showSuggestions}
							<div
								transition:slide={{ duration: 300 }}
								class="border-border bg-base-200/50 mt-3 space-y-1 rounded-lg border p-2"
							>
								{#each suggestedUsernames as suggestedUsername}
									<button
										type="button"
										onclick={() => selectUsername(suggestedUsername)}
										class="hover:bg-primary/10 hover:border-primary/30 text-base-content w-full rounded-md border border-transparent px-3 py-2 text-left text-sm transition-all hover:shadow-sm"
									>
										<div class="flex items-center gap-2">
											<User class="text-base-muted h-4 w-4" />
											<span class="font-medium">{suggestedUsername}</span>
										</div>
									</button>
								{/each}
							</div>
						{/if}
					{/if}

					{#if loadingSuggestions}
						<p class="text-base-muted mt-2 text-xs">Loading suggestions...</p>
					{/if}
				</Field.Root>

				<!-- Password Field -->
				<Field.Root class="mb-6">
					<Field.Label class="text-base-content mb-2 block text-sm font-medium">
						Password
					</Field.Label>
					<div class="relative">
						<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
							<Lock class="text-base-muted h-5 w-5" />
						</div>
						<Field.Input
							id="password-input"
							bind:value={password}
							type={showPassword ? 'text' : 'password'}
							autocomplete="current-password"
							required
							placeholder="Enter your password"
							oninput={() => {
								error = '';
							}}
							class="border-border bg-base-100 text-base-content placeholder:text-base-muted focus:border-primary focus:ring-primary w-full rounded-lg border py-3 pl-10 pr-12 transition-colors focus:outline-none focus:ring-2"
						/>
						<button
							type="button"
							onclick={() => (showPassword = !showPassword)}
							class="text-base-muted hover:text-base-content absolute inset-y-0 right-0 flex items-center pr-3 transition-colors"
						>
							{#if showPassword}
								<EyeOff class="h-5 w-5" />
							{:else}
								<Eye class="h-5 w-5" />
							{/if}
						</button>
					</div>
				</Field.Root>

				<!-- Submit Button -->
				<button
					type="submit"
					disabled={isLoading || !username || !password}
					class="bg-primary hover:bg-primary-hover focus:ring-primary focus:ring-offset-base-100 flex w-full items-center justify-center gap-2 rounded-lg px-6 py-3 font-semibold text-white shadow-lg transition-all hover:shadow-xl focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:shadow-lg"
				>
					{#if isLoading}
						<div
							class="h-5 w-5 animate-spin rounded-full border-2 border-white border-t-transparent"
						></div>
						<span>Signing in...</span>
					{:else}
						<span>Sign In</span>
						<ArrowRight class="h-5 w-5" />
					{/if}
				</button>
			</form>

			<!-- Admin Link -->
			<div class="border-border border-t px-8 py-6">
				<p class="text-base-muted text-center text-sm">
					Administrator?
					<a
						href="/admin"
						class="text-primary hover:text-primary-hover ml-1 font-medium transition-colors"
					>
						Sign in here
					</a>
				</p>
			</div>
		</div>

		<!-- Security Notice -->
		<div class="mt-6 text-center">
			<p class="text-base-muted flex items-center justify-center gap-2 text-xs">
				<Shield class="h-4 w-4" />
				<span>Your session is encrypted and protected</span>
			</p>
		</div>
	</div>
</div>
