<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Download } from 'lucide-svelte';
	import { config } from '$lib/config.js';

	interface Props {
		text: string;
		size?: 'sm' | 'md' | 'lg';
		variant?: 'default' | 'outline';
		showIcon?: boolean;
		class?: string;
	}

	let { 
		text, 
		size = 'lg', 
		variant = 'default',
		showIcon = true,
		class: className = ''
	}: Props = $props();

	const handleClick = () => {
		window.open(config.download.url, '_blank');
	};

	// Icon size based on button size
	const iconSize = size === 'sm' ? 'w-3 h-3' : size === 'md' ? 'w-4 h-4' : 'w-4 h-4';
</script>

<Button 
	{size} 
	{variant}
	class="font-semibold {className}"
	onclick={handleClick}
>
	{#if showIcon}
		<Download class="{iconSize} mr-2" />
	{/if}
	{text}
</Button>