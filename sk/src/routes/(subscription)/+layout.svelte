<script lang="ts">
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { subscriptionStore } from '$lib/stores/subscription.svelte.js';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		// Ensure subscription data is loaded
		if (authStore.isLoggedIn) {
			subscriptionStore.refresh();
		}
	});

	// Check auth and subscription state
	$effect(() => {
		if (browser && authStore.initialized) {
			if (!authStore.isLoggedIn) {
				console.log('🚫 Not authenticated, redirecting to login');
				goto('/login');
				return;
			}

			// Check subscription after data is loaded
			if (!subscriptionStore.isLoading && !subscriptionStore.isSubscribed) {
				console.log('🚫 No active subscription, redirecting to pricing');
				goto('/pricing');
				return;
			}
		}
	});
</script>

{#if authStore.isLoading || subscriptionStore.isLoading}
	<div class="container mx-auto px-4 py-8">
		<div class="text-center">
			<p>Loading...</p>
		</div>
	</div>
{:else if authStore.isLoggedIn && subscriptionStore.isSubscribed}
	{@render children()}
{:else if authStore.isLoggedIn && !subscriptionStore.isSubscribed}
	<div class="container mx-auto px-4 py-8">
		<div class="text-center">
			<div class="rounded-lg border-2 border-dashed border-gray-300 p-8 max-w-md mx-auto">
				<h2 class="text-xl font-semibold mb-2">Subscription Required</h2>
				<p class="text-muted-foreground mb-4">
					This feature requires an active subscription. Please choose a plan to continue.
				</p>
				<a 
					href="/pricing"
					class="inline-flex items-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
				>
					View Pricing Plans
				</a>
			</div>
		</div>
	</div>
{:else}
	<div class="container mx-auto px-4 py-8">
		<div class="text-center">
			<p>Redirecting to login...</p>
		</div>
	</div>
{/if}