<script lang="ts">
	import { Label } from '$lib/components/ui/label/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	let {
		email = $bindable(''),
		onContinue,
		isLoading = false,
		error = null
	}: {
		email: string;
		onContinue: (email: string) => void;
		isLoading?: boolean;
		error?: string | null;
	} = $props();

	let formId = $state(Math.random().toString(36).substr(2, 9));

	function handleSubmit(event: Event) {
		event.preventDefault();
		
		if (!email) {
			return;
		}

		onContinue(email);
	}

	function validateEmail(email: string): boolean {
		const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
		return emailRegex.test(email);
	}

	let isValidEmail = $derived(email ? validateEmail(email) : false);
</script>

<div class="flex flex-col gap-4">
	<div class="flex flex-col items-center gap-2 text-center">
		<h1 class="text-2xl font-semibold tracking-tight">Welcome back</h1>
		<p class="text-sm text-muted-foreground">
			Enter your email to continue to your account
		</p>
	</div>

	{#if error}
		<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-600">
			{error}
		</div>
	{/if}

	<form onsubmit={handleSubmit} class="flex flex-col gap-4">
		<div class="grid gap-2">
			<Label for="email-{formId}">Email</Label>
			<Input
				id="email-{formId}"
				type="email"
				placeholder="name@example.com"
				bind:value={email}
				disabled={isLoading}
				required
				autocomplete="email"
				class="h-10"
			/>
		</div>

		<Button 
			type="submit" 
			class="w-full h-10" 
			disabled={isLoading || !isValidEmail}
		>
			{isLoading ? 'Please wait...' : 'Continue'}
		</Button>
	</form>

	<div class="text-center text-sm">
		<span class="text-muted-foreground">Don't have an account? </span>
		<a href="/signup" class="font-medium text-primary hover:underline">
			Sign up
		</a>
	</div>
</div>