<script lang="ts">
	import { Label } from '$lib/components/ui/label/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { authStore } from '$lib/stores/authClient.svelte';
	import { pb } from '$lib/pocketbase';
	import { goto } from '$app/navigation';
	import { Mail, ArrowLeft } from 'lucide-svelte';
	import { page } from '$app/stores';

	let email = $state('');
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	let success = $state(false);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		
		if (!email) {
			error = 'Please enter your email address';
			return;
		}

		isLoading = true;
		error = null;

		try {
			await authStore.pb.collection('users').requestPasswordReset(email);
			success = true;
		} catch (err: any) {
			error = err?.message || 'Failed to send password reset email';
		} finally {
			isLoading = false;
		}
	}

	function goBack() {
		goto('/login');
	}
</script>

<svelte:head>
	<title>Forgot Password - Pulse</title>
	<meta name="description" content="Reset your Pulse account password" />
</svelte:head>

<div class="container mx-auto flex h-screen w-screen flex-col items-center justify-center">
	<div class="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[400px]">
		<div class="flex flex-col space-y-2 text-center">
			<h1 class="text-2xl font-semibold tracking-tight">Forgot your password?</h1>
			<p class="text-sm text-muted-foreground">
				Enter your email address and we'll send you a link to reset your password.
			</p>
		</div>

		{#if success}
			<Card>
				<CardHeader>
					<div class="flex items-center space-x-2">
						<Mail class="h-5 w-5 text-green-500" />
						<CardTitle>Check your email</CardTitle>
					</div>
				</CardHeader>
				<CardContent>
					<p class="text-sm text-muted-foreground">
						We've sent a password reset link to <strong>{email}</strong>. 
						Check your inbox and click the link to reset your password.
					</p>
				</CardContent>
				<CardFooter>
					<Button variant="outline" onclick={goBack} class="w-full">
						<ArrowLeft class="mr-2 h-4 w-4" />
						Back to login
					</Button>
				</CardFooter>
			</Card>
		{:else}
			<form onsubmit={handleSubmit} class="space-y-4">
				<Card>
					<CardHeader>
						<CardTitle>Reset Password</CardTitle>
						<CardDescription>
							Enter the email address associated with your account.
						</CardDescription>
					</CardHeader>
					<CardContent class="space-y-4">
						{#if error}
							<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-600">
								{error}
							</div>
						{/if}

						<div class="space-y-2">
							<Label for="email">Email address</Label>
							<Input
								id="email"
								type="email"
								placeholder="Enter your email address"
								bind:value={email}
								disabled={isLoading}
								required
								autocomplete="email"
							/>
						</div>
					</CardContent>
					<CardFooter class="flex flex-col space-y-2">
						<Button type="submit" disabled={isLoading || !email} class="w-full">
							{#if isLoading}
								<div class="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"></div>
								Sending reset link...
							{:else}
								Send reset link
							{/if}
						</Button>
						
						<Button variant="ghost" onclick={goBack} type="button" class="w-full">
							<ArrowLeft class="mr-2 h-4 w-4" />
							Back to login
						</Button>
					</CardFooter>
				</Card>
			</form>
		{/if}
	</div>
</div>