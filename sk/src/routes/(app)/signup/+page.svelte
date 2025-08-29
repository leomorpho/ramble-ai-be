<script lang="ts">
	import SignupForm from '$lib/components/SignupForm.svelte';
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	// Redirect if already logged in
	onMount(() => {
		if (authStore.isLoggedIn) {
			goto('/dashboard');
		}
	});
</script>

<svelte:head>
	<title>Sign Up - App Name</title>
	<meta name="description" content="Create your account" />
</svelte:head>

<div class="flex min-h-[calc(100vh-4rem)] flex-col items-center justify-center p-6 md:p-10">
	<div class="w-full max-w-sm">
		<SignupForm />
	</div>
</div>
