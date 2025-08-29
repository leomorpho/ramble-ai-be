<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/authClient.svelte';
	import { client } from '$lib/pocketbase';
	
	import EmailStep from './EmailStep.svelte';
	import AuthMethodSelector from './AuthMethodSelector.svelte';
	import PasswordStep from './PasswordStep.svelte';
	import PasskeyStep from './PasskeyStep.svelte';
	import OAuthOptions from './OAuthOptions.svelte';

	type FlowStep = 'email' | 'method-select' | 'password' | 'passkey';

	let currentStep = $state<FlowStep>('email');
	let email = $state('');
	let password = $state('');
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	
	// User state
	let hasPasskeys = $state(false);
	let webAuthnSupported = $state(false);
	
	// Check WebAuthn support on mount
	$effect(() => {
		webAuthnSupported = typeof navigator !== 'undefined' && 
		   !!(navigator.credentials?.create && navigator.credentials?.get);
	});

	async function checkUserPasskeys(emailAddress: string): Promise<boolean> {
		try {
			// Try to fetch login options to see if user has passkeys
			const baseURL = client.baseUrl || (typeof window !== 'undefined' ? 
				(window.location.hostname === 'localhost' ? 'http://localhost:8090' : window.location.origin) : 
				undefined);
			
			if (!baseURL) return false;
			
			const response = await fetch(`${baseURL}/api/webauthn/login-options?usernameOrEmail=${encodeURIComponent(emailAddress)}`);
			return response.ok;
		} catch {
			return false;
		}
	}

	async function handleEmailContinue(emailAddress: string) {
		email = emailAddress;
		isLoading = true;
		error = null;

		try {
			// Check if user has passkeys registered
			hasPasskeys = await checkUserPasskeys(emailAddress);
			currentStep = 'method-select';
		} catch (err) {
			error = 'Failed to check user information. Please try again.';
		} finally {
			isLoading = false;
		}
	}

	function handleMethodSelect(method: 'password' | 'passkey') {
		error = null;
		currentStep = method;
	}

	async function handlePasswordSignIn(emailAddress: string, userPassword: string) {
		isLoading = true;
		error = null;

		try {
			const result = await authStore.login(emailAddress, userPassword);
			
			if (result.success) {
				goto('/dashboard');
			} else {
				error = result.error || 'Invalid email or password';
			}
		} catch (err) {
			error = 'Login failed. Please try again.';
		} finally {
			isLoading = false;
		}
	}

	function handlePasskeySuccess() {
		goto('/dashboard');
	}

	function handlePasskeyFallback() {
		currentStep = 'password';
	}

	function handleBackToEmail() {
		currentStep = 'email';
		error = null;
	}

	function handleBackToMethodSelect() {
		currentStep = 'method-select';
		error = null;
	}

	function handleOAuthLogin(provider: 'google' | 'apple') {
		// TODO: Implement OAuth login
		console.log(`OAuth login with ${provider}`);
	}
</script>

<div class="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
	<!-- App branding -->
	<div class="flex flex-col items-center space-y-2">
		<a href="/" class="flex items-center gap-2 font-medium">
			<div class="flex size-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					class="size-4"
				>
					<rect width="7" height="18" x="3" y="3" rx="1" />
					<rect width="7" height="7" x="14" y="3" rx="1" />
					<rect width="7" height="7" x="14" y="14" rx="1" />
				</svg>
			</div>
			<span class="sr-only">App Name</span>
		</a>
	</div>

	<!-- Main flow content -->
	<div class="flex flex-col space-y-4">
		{#if currentStep === 'email'}
			<EmailStep 
				bind:email
				onContinue={handleEmailContinue}
				{isLoading}
				{error}
			/>
		{:else if currentStep === 'method-select'}
			<AuthMethodSelector
				{email}
				{hasPasskeys}
				{webAuthnSupported}
				onSelectMethod={handleMethodSelect}
				onBack={handleBackToEmail}
				{isLoading}
				{error}
			/>
		{:else if currentStep === 'password'}
			<PasswordStep
				{email}
				bind:password
				onSignIn={handlePasswordSignIn}
				onBack={handleBackToMethodSelect}
				{isLoading}
				{error}
			/>
		{:else if currentStep === 'passkey'}
			<PasskeyStep
				{email}
				onSuccess={handlePasskeySuccess}
				onBack={handleBackToMethodSelect}
				onFallback={handlePasskeyFallback}
			/>
		{/if}
	</div>

	<!-- OAuth options - show on email and method-select steps -->
	{#if currentStep === 'email' || currentStep === 'method-select'}
		<OAuthOptions 
			{isLoading}
			onProviderLogin={handleOAuthLogin}
		/>
	{/if}

	<!-- Terms and privacy -->
	<div class="px-8 text-center text-xs text-muted-foreground">
		By continuing, you agree to our{' '}
		<a href="/terms" class="underline underline-offset-4 hover:text-primary">
			Terms of Service
		</a>{' '}
		and{' '}
		<a href="/privacy" class="underline underline-offset-4 hover:text-primary">
			Privacy Policy
		</a>
		.
	</div>
</div>