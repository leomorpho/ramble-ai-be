<script lang="ts">
	import '../app.css';
	import Navigation from '$lib/components/Navigation.svelte';
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { subscriptionStore } from '$lib/stores/subscription.svelte.js';
	import { config } from '$lib/config.js';

	let { children } = $props();

	function handleLogout() {
		console.log('🚪 Logout clicked');
		authStore.logout();
	}

	// Initialize subscription store when auth state changes
	$effect(() => {
		if (authStore.initialized) {
			subscriptionStore.initialize();
		}
	});
</script>

<div class="bg-background text-foreground min-h-screen">
	<Navigation 
		isLoggedIn={authStore.isLoggedIn}
		user={authStore.user}
		isSubscribed={subscriptionStore.isSubscribed}
		onLogout={handleLogout}
	/>

	<main class="pt-20">
		{@render children()}
	</main>

	<footer class="mt-auto border-t">
		<div class="text-muted-foreground container mx-auto px-4 py-6 text-center">
			<p>&copy; {config.getCurrentYear()} {config.app.name}. All rights reserved.</p>
		</div>
	</footer>
</div>
