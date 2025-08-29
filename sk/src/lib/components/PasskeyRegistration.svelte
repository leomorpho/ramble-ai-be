<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { webauthnRegister } from '$lib/pocketbase.js';
	import { Key, Check, AlertCircle } from 'lucide-svelte';

	let { 
		email,
		onSuccess,
		onError 
	}: {
		email: string;
		onSuccess?: () => void;
		onError?: (error: string) => void;
	} = $props();

	let isLoading = $state(false);
	let isRegistered = $state(false);
	let error = $state<string | null>(null);
	let webAuthnSupported = $state(false);

	// Initialize WebAuthn support check
	$effect(() => {
		webAuthnSupported = typeof navigator !== 'undefined' && 
		   !!(navigator.credentials?.create && navigator.credentials?.get);
	});

	async function handleRegisterPasskey() {
		if (!email) {
			const errorMsg = 'Email is required for passkey registration';
			error = errorMsg;
			onError?.(errorMsg);
			return;
		}

		isLoading = true;
		error = null;

		try {
			await webauthnRegister(email);
			isRegistered = true;
			onSuccess?.();
		} catch (err) {
			const errorMsg = err instanceof Error ? err.message : 'Passkey registration failed';
			error = errorMsg;
			onError?.(errorMsg);
			console.error('Passkey registration error:', err);
		}

		isLoading = false;
	}
</script>

{#if webAuthnSupported}
	<div class="rounded-lg border border-dashed p-4">
		<div class="flex items-center gap-3">
			<div class="flex-shrink-0">
				{#if isRegistered}
					<div class="flex h-8 w-8 items-center justify-center rounded-full bg-green-100 text-green-600">
						<Check class="h-4 w-4" />
					</div>
				{:else if error}
					<div class="flex h-8 w-8 items-center justify-center rounded-full bg-red-100 text-red-600">
						<AlertCircle class="h-4 w-4" />
					</div>
				{:else}
					<div class="flex h-8 w-8 items-center justify-center rounded-full bg-blue-100 text-blue-600">
						<Key class="h-4 w-4" />
					</div>
				{/if}
			</div>
			
			<div class="flex-1 min-w-0">
				<div class="text-sm font-medium text-foreground">
					{#if isRegistered}
						Passkey Registered
					{:else}
						Set up Passkey Authentication
					{/if}
				</div>
				<div class="text-sm text-muted-foreground">
					{#if isRegistered}
						You can now sign in using your passkey for faster, more secure access.
					{:else}
						Use your device's biometrics or security key for secure, passwordless login.
					{/if}
				</div>
				{#if error}
					<div class="text-sm text-red-600 mt-1">
						{error}
					</div>
				{/if}
			</div>
			
			{#if !isRegistered}
				<Button
					variant="outline"
					size="sm"
					onclick={handleRegisterPasskey}
					disabled={isLoading || !email}
				>
					{isLoading ? 'Setting up...' : 'Set up'}
				</Button>
			{/if}
		</div>
	</div>
{/if}