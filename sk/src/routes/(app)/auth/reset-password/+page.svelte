<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { pb } from '$lib/pocketbase.js';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { CheckCircle2, XCircle, Loader2, KeyRound } from 'lucide-svelte';

	let status: 'form' | 'loading' | 'success' | 'error' = $state('form');
	let message = $state('');
	let token = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let passwordError = $state('');

	onMount(() => {
		const urlToken = $page.url.searchParams.get('token');
		
		if (!urlToken) {
			status = 'error';
			message = 'Missing reset token. Please check your email link.';
			return;
		}

		token = urlToken;
	});

	function validatePasswords(): boolean {
		if (password.length < 8) {
			passwordError = 'Password must be at least 8 characters long.';
			return false;
		}

		if (password !== confirmPassword) {
			passwordError = 'Passwords do not match.';
			return false;
		}

		passwordError = '';
		return true;
	}

	async function handleSubmit() {
		if (!validatePasswords()) return;

		status = 'loading';
		message = '';

		try {
			// Confirm password reset
			await pb.collection('users').confirmPasswordReset(
				token,
				password,
				confirmPassword
			);
			
			status = 'success';
			message = 'Your password has been reset successfully! You can now log in with your new password.';
		} catch (error: any) {
			console.error('Password reset failed:', error);
			status = 'error';
			
			if (error?.status === 400) {
				message = 'Invalid or expired reset token. Please request a new password reset.';
			} else {
				message = 'Password reset failed. Please try again or contact support.';
			}
		}
	}

	function redirectToLogin() {
		goto('/login');
	}

	function redirectToForgotPassword() {
		goto('/forgot-password');
	}
</script>

<svelte:head>
	<title>Reset Password - Ramble AI</title>
	<meta name="description" content="Reset your password" />
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
				{:else}
					<KeyRound class="h-8 w-8 text-primary" />
				{/if}
			</div>
			<CardTitle>
				{#if status === 'form'}
					Reset Password
				{:else if status === 'loading'}
					Resetting Password...
				{:else if status === 'success'}
					Password Reset!
				{:else if status === 'error'}
					Reset Failed
				{/if}
			</CardTitle>
			<CardDescription>
				{#if status === 'form'}
					Enter your new password below.
				{:else}
					{message}
				{/if}
			</CardDescription>
		</CardHeader>
		<CardContent class="space-y-4">
			{#if status === 'form'}
				<form onsubmit={handleSubmit} class="space-y-4">
					<div class="space-y-2">
						<Label for="password">New Password</Label>
						<Input
							id="password"
							type="password"
							bind:value={password}
							placeholder="Enter new password"
							required
							disabled={status === 'loading'}
						/>
					</div>
					<div class="space-y-2">
						<Label for="confirm-password">Confirm New Password</Label>
						<Input
							id="confirm-password"
							type="password"
							bind:value={confirmPassword}
							placeholder="Confirm new password"
							required
							disabled={status === 'loading'}
						/>
					</div>
					{#if passwordError}
						<p class="text-sm text-destructive">{passwordError}</p>
					{/if}
					<Button 
						type="submit" 
						class="w-full"
						disabled={status === 'loading' || !password || !confirmPassword}
					>
						{#if status === 'loading'}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						Reset Password
					</Button>
				</form>
			{:else if status === 'success'}
				<div class="flex gap-2">
					<Button onclick={redirectToLogin} class="flex-1">
						Login Now
					</Button>
				</div>
			{:else if status === 'error'}
				<div class="flex gap-2">
					<Button onclick={redirectToForgotPassword} class="flex-1">
						Request New Reset
					</Button>
					<Button variant="outline" onclick={redirectToLogin} class="flex-1">
						Back to Login
					</Button>
				</div>
			{/if}
		</CardContent>
	</Card>
</div>