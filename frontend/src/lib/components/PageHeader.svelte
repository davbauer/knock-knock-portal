<script lang="ts">
	import type { ComponentType } from 'svelte';

	interface Props {
		title: string;
		subtitle?: string;
		icon: ComponentType;
		iconBgClass?: string;
		iconTextClass?: string;
		children?: import('svelte').Snippet;
	}

	let {
		title,
		subtitle,
		icon: Icon,
		iconBgClass = 'bg-primary/10',
		iconTextClass = 'text-primary',
		children
	}: Props = $props();
</script>

<div class="mb-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
	<div class="flex items-center gap-3">
		<div class="rounded-xl p-3 {iconBgClass}">
			<Icon class="h-8 w-8 {iconTextClass}" />
		</div>
		<div>
			<h1 class="text-base-content text-3xl font-bold">{title}</h1>
			{#if subtitle}
				<p class="text-base-muted mt-1 text-sm">{subtitle}</p>
			{/if}
		</div>
	</div>

	{#if children}
		<div class="flex items-center gap-2">
			{@render children()}
		</div>
	{/if}
</div>
