<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { pb } from '$lib/pocketbase.js';
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { CheckCircle2, XCircle, Loader2, Mail } from 'lucide-svelte';

	let status: 'loading' | 'success' | 'error' = $state('loading');
	let message = $state('');

	onMount(async () => {
		const token = $page.url.searchParams.get('token');
		const password = $page.url.searchParams.get('password'); // Some implementations require password
		
		if (!token) {
			status = 'error';
			message = 'Missing confirmation token. Please check your email link.';
			return;
		}

		try {
			// Confirm email change
			await pb.collection('users').confirmEmailChange(token, password || '');
			
			status = 'success';
			message = 'Your email address has been updated successfully! Please log in again with your new email.';
			
			// Clear current auth since email changed
			pb.authStore.clear();
		} catch (error: any) {
			console.error('Email change confirmation failed:', error);
			status = 'error';
			
			if (error?.status === 400) {
				message = 'Invalid or expired confirmation token. Please try changing your email again.';
			} else if (error?.status === 403) {
				message = 'Password required for email change confirmation. Please try the link from your email again.';
			} else {
				message = 'Email change confirmation failed. Please try again or contact support.';
			}
		}
	});

	function redirectToLogin() {
		goto('/login');
	}

	function redirectToDashboard() {
		if (authStore.isLoggedIn) {
			goto('/dashboard');
		} else {
			goto('/login');
		}
	}

	function redirectToHome() {
		goto('/');
	}
</script>

<svelte:head>
	<title>Confirm Email Change - Ramble AI</title>
	<meta name="description" content="Confirm your new email address" />
</svelte:head>

<div class="flex min-h-[calc(100vh-4rem)] flex-col items-center justify-center p-6">
	<Card class="w-full max-w-md">
		<CardHeader class="text-center">
			<div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center">
				{#if status === 'loading'}
					<Loader2 class="h-8 w-8 animate-spin text-primary" />
				{:else if status === 'success'}
					<CheckCircle2 class="h-8 w-8 text-green-500" />
				{:else if status === 'error'}
					<XCircle class="h-8 w-8 text-destructive" />
				{/if}
			</div>
			<CardTitle>
				{#if status === 'loading'}
					Confirming Email Change...
				{:else if status === 'success'}
					Email Updated!
				{:else if status === 'error'}
					Confirmation Failed
				{/if}
			</CardTitle>
			<CardDescription>
				{message}
			</CardDescription>
		</CardHeader>
		<CardContent class="space-y-4">
			{#if status === 'success'}
				<div class="flex gap-2">
					<Button onclick={redirectToLogin} class="flex-1">
						Login with New Email
					</Button>
					<Button variant="outline" onclick={redirectToHome} class="flex-1">
						Back to Home
					</Button>
				</div>
			{:else if status === 'error'}
				<div class="flex gap-2">
					<Button onclick={redirectToDashboard} class="flex-1">
						{authStore.isLoggedIn ? 'Back to Dashboard' : 'Login'}
					</Button>
					<Button variant="outline" onclick={redirectToHome} class="flex-1">
						Back to Home
					</Button>
				</div>
			{/if}
		</CardContent>
	</Card>
</div>