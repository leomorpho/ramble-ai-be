<script lang="ts">
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';

	let { children } = $props();

	// Check auth state and redirect if not logged in
	$effect(() => {
		if (browser && authStore.initialized && !authStore.isLoggedIn) {
			console.log('🚫 Not authenticated, redirecting to login');
			goto('/login');
		}
	});
</script>

{#if authStore.isLoading}
	<div class="container mx-auto px-4 py-8">
		<div class="text-center">
			<p>Loading...</p>
		</div>
	</div>
{:else if authStore.isLoggedIn}
	{@render children()}
{:else}
	<div class="container mx-auto px-4 py-8">
		<div class="text-center">
			<p>Redirecting to login...</p>
		</div>
	</div>
{/if}