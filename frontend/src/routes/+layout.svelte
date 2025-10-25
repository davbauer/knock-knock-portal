<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { Sun, Moon, Monitor, Shield } from 'lucide-svelte';
	import { themeStore } from '$lib/stores/theme.svelte';

	let { children } = $props();

	// React to theme changes
	$effect(() => {
		// This will run whenever themeStore.current changes
		themeStore.applyTheme();
	});

	function getThemeIcon() {
		if (themeStore.current === 'light') return Sun;
		if (themeStore.current === 'dark') return Moon;
		return Monitor;
	}

	function getThemeLabel() {
		if (themeStore.current === 'light') return 'Light';
		if (themeStore.current === 'dark') return 'Dark';
		return 'System';
	}
</script>

<div class="bg-base-200 flex min-h-screen flex-col">
	<!-- Header -->
	<header class="border-border bg-base-100 border-b">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="flex h-16 items-center justify-between">
				<a href="/" class="flex items-center gap-2 transition-opacity hover:opacity-80">
					<Shield class="text-primary h-6 w-6" />
					<h1 class="text-base-content text-xl font-bold">Knock-Knock Portal</h1>
				</a>
				<div class="flex items-center gap-4">
					<nav class="hidden items-center gap-6 md:flex">
						<a href="/" class="text-base-muted hover:text-base-content text-sm font-medium">
							Portal
						</a>
						<a href="/admin" class="text-base-muted hover:text-base-content text-sm font-medium">
							Admin
						</a>
					</nav>
					<!-- Theme Switcher -->
					<button
						onclick={() => themeStore.toggle()}
						class="border-border hover:bg-base-200 flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-colors"
						title="Toggle theme: {getThemeLabel()}"
					>
						{#if themeStore.current === 'light'}
							<Sun class="text-base-content h-4 w-4" />
						{:else if themeStore.current === 'dark'}
							<Moon class="text-base-content h-4 w-4" />
						{:else}
							<Monitor class="text-base-content h-4 w-4" />
						{/if}
						<span class="text-base-content hidden sm:inline">{getThemeLabel()}</span>
					</button>
				</div>
			</div>
		</div>
	</header>

	<!-- Main Content -->
	<main class="flex-1">
		<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
			{@render children?.()}
		</div>
	</main>

	<!-- Footer -->
	<footer class="border-border bg-base-100 border-t">
		<div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
			<div class="flex flex-col items-center justify-between gap-4 sm:flex-row">
				<p class="text-base-muted text-sm">
					Knock-Knock Portal. Authentication-based port access gateway.
				</p>
				<div class="flex gap-6">
					<a href="/health" class="text-base-muted hover:text-base-content text-sm"> Health </a>
					<a
						href="https://github.com/davbauer/knock-knock-portal"
						class="text-base-muted hover:text-base-content text-sm"
						target="_blank"
						rel="noopener noreferrer"
					>
						GitHub
					</a>
				</div>
			</div>
		</div>
	</footer>
</div>
