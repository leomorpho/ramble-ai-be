<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { ArrowLeft, Key, Fingerprint, Shield } from 'lucide-svelte';
	import { webauthnLogin } from '$lib/pocketbase';

	let {
		email,
		onSuccess,
		onBack,
		onFallback,
		isLoading = $bindable(false),
		error = $bindable(null)
	}: {
		email: string;
		onSuccess: () => void;
		onBack: () => void;
		onFallback: () => void;
		isLoading?: boolean;
		error?: string | null;
	} = $props();

	async function handlePasskeyLogin() {
		isLoading = true;
		error = null;

		try {
			await webauthnLogin(email);
			onSuccess();
		} catch (err) {
			console.error('Passkey login error:', err);
			error = err instanceof Error ? err.message : 'Passkey authentication failed';
		} finally {
			isLoading = false;
		}
	}

	// Auto-trigger passkey login when component mounts
	$effect(() => {
		handlePasskeyLogin();
	});
</script>

<div class="flex flex-col gap-6">
	<div class="flex items-center gap-3">
		<Button variant="ghost" size="sm" onclick={onBack} disabled={isLoading}>
			<ArrowLeft class="h-4 w-4" />
		</Button>
		<div class="flex-1">
			<h1 class="text-xl font-semibold">Sign in with passkey</h1>
			<p class="text-sm text-muted-foreground">{email}</p>
		</div>
	</div>

	{#if error}
		<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-600">
			{error}
		</div>
	{/if}

	<div class="text-center space-y-4">
		{#if isLoading}
			<!-- Loading state with animation -->
			<div class="flex justify-center">
				<div class="relative">
					<Fingerprint class="h-16 w-16 text-primary animate-pulse" />
					<div class="absolute -inset-2 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
				</div>
			</div>
			
			<div class="space-y-2">
				<p class="text-lg font-medium">Use your passkey to sign in</p>
				<p class="text-sm text-muted-foreground">
					Use Touch ID, Face ID, Windows Hello, or your security key
				</p>
			</div>
		{:else if error}
			<!-- Error state -->
			<div class="space-y-4">
				<div class="flex justify-center">
					<Shield class="h-16 w-16 text-red-500" />
				</div>
				
				<div class="space-y-2">
					<p class="text-lg font-medium text-red-900">Passkey authentication failed</p>
					<p class="text-sm text-muted-foreground">
						You can try again or use your password instead
					</p>
				</div>

				<div class="flex flex-col gap-2">
					<Button onclick={handlePasskeyLogin} class="w-full">
						<Key class="h-4 w-4 mr-2" />
						Try passkey again
					</Button>
					<Button variant="outline" onclick={onFallback} class="w-full">
						Use password instead
					</Button>
				</div>
			</div>
		{/if}
	</div>

	{#if !error}
		<div class="text-center">
			<Button 
				variant="ghost" 
				onclick={onFallback}
				disabled={isLoading}
				class="text-sm text-muted-foreground hover:text-primary"
			>
				Use password instead
			</Button>
		</div>
	{/if}

	<div class="text-center text-sm">
		<span class="text-muted-foreground">Don't have an account? </span>
		<a href="/signup" class="font-medium text-primary hover:underline">
			Sign up
		</a>
	</div>
</div>