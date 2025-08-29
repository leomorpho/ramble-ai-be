<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Key, ArrowLeft } from 'lucide-svelte';

	let {
		email,
		hasPasskeys = false,
		webAuthnSupported = false,
		onSelectMethod,
		onBack,
		isLoading = false,
		error = null
	}: {
		email: string;
		hasPasskeys: boolean;
		webAuthnSupported: boolean;
		onSelectMethod: (method: 'password' | 'passkey') => void;
		onBack: () => void;
		isLoading?: boolean;
		error?: string | null;
	} = $props();

	function selectPasskey() {
		onSelectMethod('passkey');
	}

	function selectPassword() {
		onSelectMethod('password');
	}
</script>

<div class="flex flex-col gap-4">
	<div class="flex items-center gap-3">
		<Button variant="ghost" size="sm" onclick={onBack} disabled={isLoading}>
			<ArrowLeft class="h-4 w-4" />
		</Button>
		<div class="flex-1">
			<h1 class="text-xl font-semibold">Choose how to sign in</h1>
			<p class="text-sm text-muted-foreground">{email}</p>
		</div>
	</div>

	{#if error}
		<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-600">
			{error}
		</div>
	{/if}

	<div class="flex flex-col gap-3">
		{#if hasPasskeys && webAuthnSupported}
			<!-- Passkey option - primary for users who have them -->
			<Button 
				onclick={selectPasskey}
				disabled={isLoading}
				class="relative h-16 justify-start gap-4 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0 shadow-lg hover:shadow-xl transition-all duration-200"
			>
				<div class="flex items-center justify-center w-10 h-10 bg-white/20 rounded-lg backdrop-blur-sm">
					<Key class="h-5 w-5" />
				</div>
				<div class="text-left flex-1">
					<div class="font-semibold text-base">Use your passkey</div>
					<div class="text-sm text-white/80">Touch ID, Face ID, or security key</div>
				</div>
				<div class="flex items-center justify-center w-8 h-8 bg-white/10 rounded-full">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
				</div>
			</Button>

			<!-- Password fallback -->
			<Button 
				variant="outline"
				onclick={selectPassword}
				disabled={isLoading}
				class="h-10"
			>
				Use password instead
			</Button>
		{:else}
			<!-- Password primary for users without passkeys -->
			<Button 
				onclick={selectPassword}
				disabled={isLoading}
				class="h-10"
			>
				Continue with password
			</Button>

			{#if webAuthnSupported}
				<div class="text-center">
					<p class="text-xs text-muted-foreground">
						You can set up passkeys after signing in for faster, more secure access
					</p>
				</div>
			{/if}
		{/if}
	</div>

	<div class="text-center text-sm">
		<span class="text-muted-foreground">New here? </span>
		<a href="/signup" class="font-medium text-primary hover:underline">
			Create an account
		</a>
	</div>
</div>