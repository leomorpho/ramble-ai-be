<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { cn, type WithElementRef } from '$lib/utils.js';
	import { authStore } from '$lib/stores/authClient.svelte';
	import { goto } from '$app/navigation';
	import { webauthnLogin } from '$lib/pocketbase';
	import { Key, Eye, EyeOff } from 'lucide-svelte';
	import PasskeyRegistration from './PasskeyRegistration.svelte';
	import SignupForm from './SignupForm.svelte';

	let {
		ref = $bindable(null),
		class: className,
		initialTab = 'login',
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement> & { initialTab?: string }> = $props();

	let activeTab = $state(initialTab);
	
	// Update activeTab when initialTab prop changes
	$effect(() => {
		activeTab = initialTab;
	});
	
	// Form state - simplified to single step
	let email = $state('');
	let password = $state('');
	let showPassword = $state(false);
	let isLoading = $state(false);
	let error = $state<string | null>(null);
	
	// User state
	let hasPasskeys = $state(false);
	let webAuthnSupported = $state(false);
	let showPasskeyOption = $state(false);
	
	// Initialize WebAuthn support check on mount
	$effect(() => {
		webAuthnSupported = typeof navigator !== 'undefined' && 
		   !!(navigator.credentials?.create && navigator.credentials?.get);
	});

	// Generate unique ID for form fields
	let formId = $state(Math.random().toString(36).substr(2, 9));

	async function checkUserPasskeys(emailAddress: string): Promise<boolean> {
		try {
			const baseURL = 'http://localhost:8090'; // Use same as pocketbase.ts
			const response = await fetch(`${baseURL}/api/webauthn/login-options?usernameOrEmail=${encodeURIComponent(emailAddress)}`);
			return response.ok;
		} catch {
			return false;
		}
	}

	// Check for passkeys when email changes (with debounce)
	let checkPasskeysTimeout: number;
	$effect(() => {
		if (email && validateEmail(email)) {
			// Clear previous timeout
			if (checkPasskeysTimeout) {
				clearTimeout(checkPasskeysTimeout);
			}
			
			// Debounce the passkey check
			checkPasskeysTimeout = setTimeout(async () => {
				hasPasskeys = await checkUserPasskeys(email);
				showPasskeyOption = hasPasskeys && webAuthnSupported;
			}, 500);
		} else {
			showPasskeyOption = false;
		}
	});

	async function handlePasswordSignIn(e: Event) {
		e.preventDefault();
		
		if (!email || !password) return;
		
		isLoading = true;
		error = null;

		try {
			const result = await authStore.login(email, password);
			
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

	async function handlePasskeyLogin() {
		if (!email) {
			error = 'Please enter your email first';
			return;
		}

		isLoading = true;
		error = null;

		try {
			await webauthnLogin(email);
			// webauthnLogin handles saving to authStore internally
			goto('/dashboard');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Passkey authentication failed';
			console.error('Passkey login error:', err);
		} finally {
			isLoading = false;
		}
	}

	function validateEmail(email: string): boolean {
		const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
		return emailRegex.test(email);
	}

	function togglePasswordVisibility() {
		showPassword = !showPassword;
	}
</script>

<div class={cn('flex flex-col gap-6', className)} bind:this={ref} {...restProps}>
	<!-- App branding -->
	<div class="flex flex-col items-center space-y-2">
		<a href="/" class="flex items-center gap-2 font-medium">
			<div class="flex size-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="size-4">
					<rect width="7" height="18" x="3" y="3" rx="1"></rect>
					<rect width="7" height="7" x="14" y="3" rx="1"></rect>
					<rect width="7" height="7" x="14" y="14" rx="1"></rect>
				</svg>
			</div>
			<span class="sr-only">App Name</span>
		</a>
	</div>

	<Tabs bind:value={activeTab} class="w-full">
		<TabsList class="grid w-full grid-cols-2">
			<TabsTrigger value="login">Sign In</TabsTrigger>
			<TabsTrigger value="register">Create Account</TabsTrigger>
		</TabsList>
		<TabsContent value="login">
			<div class="flex flex-col gap-6">
				<!-- Single-step login form -->
				<div class="flex flex-col items-center gap-3 text-center">
					<h1 class="text-2xl font-semibold tracking-tight">Welcome back</h1>
					<p class="text-sm text-muted-foreground">
						Sign in to your account
					</p>
				</div>

				{#if error}
					<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-600">
						{error}
					</div>
				{/if}

				<form onsubmit={handlePasswordSignIn} class="flex flex-col gap-4">
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
								{:else}
									<Eye class="h-4 w-4" />
								{/if}
							</Button>
						</div>
					</div>

					<Button 
						type="submit" 
						class="w-full h-10" 
						disabled={isLoading || !email || !password || !validateEmail(email)}
					>
						{isLoading ? 'Signing in...' : 'Sign in'}
					</Button>
				</form>

				<!-- Passkey option (if available) -->
				{#if showPasskeyOption && !isLoading}
					<div class="relative">
						<div class="absolute inset-0 flex items-center">
							<span class="w-full border-t" />
						</div>
						<div class="relative flex justify-center text-xs uppercase">
							<span class="bg-background px-2 text-muted-foreground">Or</span>
						</div>
					</div>

					<Button 
						variant="outline" 
						onclick={handlePasskeyLogin} 
						class="w-full h-10" 
						disabled={isLoading || !email || !validateEmail(email)}
					>
						<Key class="h-4 w-4 mr-2" />
						Use passkey instead
					</Button>
				{/if}

				<div class="text-center">
					<a 
						href="/forgot-password" 
						class="text-sm text-muted-foreground hover:text-primary hover:underline"
					>
						Forgot your password?
					</a>
				</div>

				<!-- Show passkey registration option after successful login flow -->
				{#if email && webAuthnSupported && !isLoading && !showPasskeyOption}
					<div class="mt-4">
						<div class="text-sm font-medium text-foreground mb-3">Optional: Enhanced Security</div>
						<PasskeyRegistration 
							{email}
							onSuccess={() => {
								console.log('Passkey registered successfully');
								// Refresh passkey status
								checkUserPasskeys(email).then(result => {
									hasPasskeys = result;
									showPasskeyOption = result && webAuthnSupported;
								});
							}}
							onError={(error) => {
								console.error('Passkey registration failed:', error);
							}}
						/>
					</div>
				{/if}
			</div>
		</TabsContent>
		<TabsContent value="register">
			<SignupForm />
		</TabsContent>
	</Tabs>
</div>