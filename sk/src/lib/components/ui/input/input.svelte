<script lang="ts">
	import { cn } from '$lib/utils.js';
	import type { HTMLInputAttributes } from 'svelte/elements';

	let {
		class: className,
		value = $bindable(),
		...restProps
	}: HTMLInputAttributes & {
		value?: string;
	} = $props();
</script>

<input
	class={cn(
		'border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-10 w-full rounded-md border px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50',
		className
	)}
	bind:value
	{...restProps}
/>

<style>
	/* Override browser autocomplete styling to match theme */
	input:-webkit-autofill,
	input:-webkit-autofill:hover,
	input:-webkit-autofill:focus,
	input:-webkit-autofill:active {
		-webkit-box-shadow: 0 0 0 30px hsl(var(--background)) inset !important;
		-webkit-text-fill-color: hsl(var(--foreground)) !important;
		border-color: hsl(var(--border)) !important;
	}
	
	/* For non-webkit browsers */
	input:autofill {
		background-color: hsl(var(--background)) !important;
		color: hsl(var(--foreground)) !important;
		border-color: hsl(var(--border)) !important;
	}
</style>
