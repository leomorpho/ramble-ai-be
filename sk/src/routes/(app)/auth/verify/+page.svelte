<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { pb } from '$lib/pocketbase.js';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { CheckCircle2, XCircle, Loader2 } from 'lucide-svelte';

	let status: 'loading' | 'success' | 'error' = $state('loading');
	let message = $state('');

	onMount(async () => {
		const token = $page.url.searchParams.get('token');
		
		if (!token) {
			status = 'error';
			message = 'Missing verification token. Please check your email link.';
			return;
		}

		try {
			// Confirm email verification
			await pb.collection('users').confirmVerification(token);
			
			status = 'success';
			message = 'Your email has been verified successfully! You can now log in to your account.';
		} catch (error: any) {
			console.error('Email verification failed:', error);
			status = 'error';
			
			if (error?.status === 400) {
				message = 'Invalid or expired verification token. Please request a new verification email.';
			} else {
				message = 'Email verification failed. Please try again or contact support.';
			}
		}
	});

	function redirectToLogin() {
		goto('/login');
	}

	function redirectToHome() {
		goto('/');
	}
</script>

<svelte:head>
	<title>Verify Email - Ramble AI</title>
	<meta name="description" content="Verify your email address" />
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
					Verifying Email...
				{:else if status === 'success'}
					Email Verified!
				{:else if status === 'error'}
					Verification Failed
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
						Login Now
					</Button>
					<Button variant="outline" onclick={redirectToHome} class="flex-1">
						Back to Home
					</Button>
				</div>
			{:else if status === 'error'}
				<div class="flex gap-2">
					<Button onclick={redirectToHome} class="flex-1">
						Back to Home
					</Button>
					<Button variant="outline" onclick={redirectToLogin} class="flex-1">
						Try Login
					</Button>
				</div>
			{/if}
		</CardContent>
	</Card>
</div>