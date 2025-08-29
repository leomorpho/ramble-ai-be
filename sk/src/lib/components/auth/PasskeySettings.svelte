<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card/index.js';
	import { authModel } from '$lib/pocketbase';
	import { Key, Plus, Trash2, Shield } from 'lucide-svelte';
	import PasskeyRegistration from '../PasskeyRegistration.svelte';

	let showRegistration = $state(false);
	let user = $state(null);

	// Subscribe to auth changes
	$effect(() => {
		const unsubscribe = authModel.subscribe(value => {
			user = value;
		});
		
		return unsubscribe;
	});

	function handleAddPasskey() {
		showRegistration = true;
	}

	function handleRegistrationSuccess() {
		showRegistration = false;
		// Could refresh passkey list here
	}

	function handleRegistrationError(error: string) {
		console.error('Passkey registration failed:', error);
		// Could show error toast here
	}

	let webAuthnSupported = $derived(
		typeof navigator !== 'undefined' && 
		!!(navigator.credentials?.create && navigator.credentials?.get)
	);
</script>

<Card>
	<CardHeader>
		<CardTitle class="flex items-center gap-2">
			<Shield class="h-5 w-5" />
			Passkey Security
		</CardTitle>
		<CardDescription>
			Manage your passkeys for secure, passwordless sign-in using Touch ID, Face ID, Windows Hello, or security keys.
		</CardDescription>
	</CardHeader>
	<CardContent class="space-y-4">
		{#if !webAuthnSupported}
			<div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
				<p class="font-medium">WebAuthn not supported</p>
				<p>Your browser doesn't support passkeys. Try using a modern browser like Chrome, Safari, or Edge.</p>
			</div>
		{:else if !user}
			<div class="rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-800">
				<p>Please sign in to manage your passkeys.</p>
			</div>
		{:else if showRegistration}
			<div class="space-y-4">
				<div class="flex items-center justify-between">
					<h3 class="text-lg font-medium">Add a new passkey</h3>
					<Button variant="ghost" size="sm" onclick={() => showRegistration = false}>
						Cancel
					</Button>
				</div>
				<PasskeyRegistration 
					email={user.email}
					onSuccess={handleRegistrationSuccess}
					onError={handleRegistrationError}
				/>
			</div>
		{:else}
			<div class="space-y-4">
				<!-- Passkey list would go here -->
				<div class="flex items-center justify-between">
					<div>
						<h3 class="text-lg font-medium">Your passkeys</h3>
						<p class="text-sm text-muted-foreground">You don't have any passkeys set up yet.</p>
					</div>
				</div>

				<Button onclick={handleAddPasskey} class="w-full sm:w-auto">
					<Plus class="mr-2 h-4 w-4" />
					Add passkey
				</Button>

				<div class="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-800">
					<p class="font-medium">Why use passkeys?</p>
					<ul class="mt-1 list-disc list-inside space-y-1 text-sm">
						<li>Faster sign-in with biometrics or security keys</li>
						<li>More secure than passwords</li>
						<li>No password to remember or lose</li>
						<li>Protected against phishing attacks</li>
					</ul>
				</div>
			</div>
		{/if}
	</CardContent>
</Card>