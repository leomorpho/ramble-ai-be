<script lang="ts">
	interface Props {
		size?: 'sm' | 'md' | 'lg';
		showText?: boolean;
		href?: string;
		class?: string;
	}

	let { size = 'md', showText = true, href = '/', class: className = '' }: Props = $props();

	const sizeClasses = {
		sm: 'w-6 h-6',
		md: 'w-8 h-8',
		lg: 'w-12 h-12'
	};

	const textSizeClasses = {
		sm: 'text-lg',
		md: 'text-xl',
		lg: 'text-2xl'
	};
</script>

<a
	{href}
	class="flex items-center space-x-3 hover:opacity-80 transition-opacity cursor-pointer {className}"
>
	<img 
		src="/logo-128.png" 
		alt="Ramble logo" 
		class="{sizeClasses[size]} rounded-lg"
	/>
	{#if showText}
		<span class="font-bold tracking-tight {textSizeClasses[size]}">
			<span class="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">RAMBLE</span>
		</span>
	{/if}
</a>