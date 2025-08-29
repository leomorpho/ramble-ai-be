<script lang="ts">
	import { Label } from '$lib/components/ui/label/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { ArrowLeft, Eye, EyeOff } from 'lucide-svelte';

	let {
		email,
		password = $bindable(''),
		onSignIn,
		onBack,
		isLoading = false,
		error = null
	}: {
		email: string;
		password: string;
		onSignIn: (email: string, password: string) => void;
		onBack: () => void;
		isLoading?: boolean;
		error?: string | null;
	} = $props();

	let showPassword = $state(false);
	let formId = $state(Math.random().toString(36).substr(2, 9));

	function handleSubmit(event: Event) {
		event.preventDefault();
		
		if (!password) {
			return;
		}

		onSignIn(email, password);
	}

	function togglePasswordVisibility() {
		showPassword = !showPassword;
	}
</script>

<div class="flex flex-col gap-4">
	<div class="flex items-center gap-3">
		<Button variant="ghost" size="sm" onclick={onBack} disabled={isLoading}>
			<ArrowLeft class="h-4 w-4" />
		</Button>
		<div class="flex-1">
			<h1 class="text-xl font-semibold">Enter your password</h1>
			<p class="text-sm text-muted-foreground">{email}</p>
		</div>
	</div>

	{#if error}
		<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-600">
			{error}
		</div>
	{/if}

	<form onsubmit={handleSubmit} class="flex flex-col gap-4">
		<div class="grid gap-2">
			<Label for="password-{formId}">Password</Label>
			<div class="relative">
				<Input
					id="password-{formId}"
					type={showPassword ? "text" : "password"}
					placeholder="Enter your password"
					bind:value={password}
					disabled={isLoading}
					required
					autocomplete="current-password"
					class="h-10 pr-10"
				/>
				<Button
					type="button"
					variant="ghost"
					size="sm"
					class="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
					onclick={togglePasswordVisibility}
					disabled={isLoading}
				>
					{#if showPassword}
						<EyeOff class="h-4 w-4" />
						<span class="sr-only">Hide password</span>
					{:else}
						<Eye class="h-4 w-4" />
						<span class="sr-only">Show password</span>
					{/if}
				</Button>
			</div>
		</div>

		<Button 
			type="submit" 
			class="w-full h-10" 
			disabled={isLoading || !password}
		>
			{isLoading ? 'Signing in...' : 'Sign in'}
		</Button>
	</form>

	<div class="text-center">
		<a 
			href="/forgot-password" 
			class="text-sm text-muted-foreground hover:text-primary hover:underline"
		>
			Forgot your password?
		</a>
	</div>

	<div class="text-center text-sm">
		<span class="text-muted-foreground">Don't have an account? </span>
		<a href="/signup" class="font-medium text-primary hover:underline">
			Sign up
		</a>
	</div>
</div>