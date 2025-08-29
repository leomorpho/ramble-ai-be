<script lang="ts">
	import LoginForm from '$lib/components/LoginForm.svelte';
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';

	// Get the initial tab from URL parameters reactively
	let initialTab = $derived($page.url.searchParams.get('tab') === 'signup' ? 'register' : 'login');

	// Redirect if already logged in
	onMount(() => {
		if (authStore.isLoggedIn) {
			goto('/dashboard');
		}
	});
</script>

<svelte:head>
	<title>Login - App Name</title>
	<meta name="description" content="Login to your account" />
</svelte:head>

<div class="flex min-h-[calc(100vh-4rem)] flex-col items-center justify-center p-6 md:p-10">
	<div class="w-full max-w-sm">
		<LoginForm {initialTab} />
	</div>
</div>
