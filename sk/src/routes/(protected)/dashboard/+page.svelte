<script lang="ts">
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { subscriptionStore } from '$lib/stores/subscription.svelte.js';
	import { config } from '$lib/config.js';
	import { pb } from '$lib/pocketbase.js';
	import { Crown, Mail, Calendar, Settings, Edit3, Shield } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import APIKeyManager from '$lib/components/APIKeyManager.svelte';
	import OTPVerification from '$lib/components/OTPVerification.svelte';

	// Subscription data
	let subscriptionPlan = $state(null);
	let subscriptionData = $state(null);
	let usageData = $state(null);
	let isLoadingSubscription = $state(false);

	// Helper to format date
	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'long',
			day: 'numeric'
		});
	}

	// Track if we've already loaded data for current user
	let loadedForUserId = $state<string | null>(null);
	
	// Load subscription data
	async function loadSubscriptionData() {
		if (!authStore.user || isLoadingSubscription || loadedForUserId === authStore.user.id) {
			return;
		}

		isLoadingSubscription = true;
		loadedForUserId = authStore.user.id;
		
		try {
			// Get user's active subscription
			const subscriptions = await pb.collection('current_user_subscriptions').getFullList({
				filter: `user_id = "${authStore.user.id}" && status = "active"`,
				sort: '-created'
			}, {
				// Use different request key to avoid auto-cancellation
				requestKey: `subscription_${authStore.user.id}_${Date.now()}`
			});

			if (subscriptions.length > 0) {
				const subscription = subscriptions[0];
				subscriptionData = subscription;

				// Get the plan details
				if (subscription.plan_id) {
					const plans = await pb.collection('subscription_plans').getFullList({
						filter: `id = "${subscription.plan_id}"`
					}, {
						requestKey: `plan_${subscription.plan_id}_${Date.now()}`
					});

					if (plans.length > 0) {
						subscriptionPlan = plans[0];
					}
				}

				// Get usage data
				const now = new Date();
				const yearMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
				
				const usage = await pb.collection('monthly_usage').getFullList({
					filter: `user_id = "${authStore.user.id}" && year_month = "${yearMonth}"`,
					sort: '-last_processing_date'
				}, {
					requestKey: `usage_${authStore.user.id}_${yearMonth}_${Date.now()}`
				});

				if (usage.length > 0) {
					const hoursUsed = usage[0].hours_used || 0;
					const hoursLimit = subscriptionPlan?.hours_per_month || 10;
					usageData = {
						hours_used: hoursUsed,
						hours_limit: hoursLimit,
						usage_percentage: (hoursUsed / hoursLimit) * 100
					};
				}
			} else {
				// No subscription found - clear data
				subscriptionData = null;
				subscriptionPlan = null;
				usageData = null;
			}
		} catch (error) {
			// Only log non-cancellation errors
			if (!error.message?.includes('autocancelled')) {
				console.error('Failed to load subscription data:', error);
			}
		} finally {
			isLoadingSubscription = false;
		}
	}

	// Initialize subscription store
	subscriptionStore.initialize();

	// Watch for user changes
	$effect(() => {
		if (authStore.user && authStore.user.id !== loadedForUserId) {
			// User changed, reset and load new data
			subscriptionPlan = null;
			subscriptionData = null;
			usageData = null;
			loadedForUserId = null;
			loadSubscriptionData();
		} else if (!authStore.user) {
			// User logged out
			subscriptionPlan = null;
			subscriptionData = null;
			usageData = null;
			loadedForUserId = null;
		}
	});

	// Personal account editing state
	let isEditingProfile = $state(false);
	let editPersonalName = $state(authStore.user?.name || '');
	let editPersonalEmail = $state(authStore.user?.email || '');
	let isSavingPersonal = $state(false);
	let personalEditError = $state<string | null>(null);
	let personalEditSuccess = $state<string | null>(null);
	
	// Email verification state
	let showEmailVerification = $state(false);

	// Update edit values when user data changes
	$effect(() => {
		if (authStore.user) {
			editPersonalName = authStore.user.name || '';
			editPersonalEmail = authStore.user.email || '';
		}
	});

	// Handle personal account save
	async function handleSavePersonal() {
		if (!editPersonalName.trim()) {
			personalEditError = 'Name is required';
			return;
		}

		if (!editPersonalEmail.trim()) {
			personalEditError = 'Email is required';
			return;
		}

		isSavingPersonal = true;
		personalEditError = null;
		personalEditSuccess = null;

		try {
			const currentEmail = authStore.user!.email;
			const newEmail = editPersonalEmail.trim();
			
			// If email is changing, we need to request email change verification
			if (currentEmail !== newEmail) {
				// Request email change - this will send verification to new email
				await pb.collection('users').requestEmailChange(newEmail);
				
				// Update only the name for now - email will be updated after verification
				await pb.collection('users').update(authStore.user!.id, {
					name: editPersonalName.trim()
				});
				
				// Show success message about email verification
				personalEditSuccess = `A verification email has been sent to ${newEmail}. Please check your email and click the verification link to complete the email change.`;
				console.log('Email change verification sent to:', newEmail);
				
				// Reset email field to current value since change is pending
				editPersonalEmail = currentEmail;
			} else {
				// Only name is changing
				await pb.collection('users').update(authStore.user!.id, {
					name: editPersonalName.trim()
				});
			}

			// Update auth store
			authStore.syncState();
			isEditingProfile = false;
		} catch (error: any) {
			console.error('Failed to update profile:', error);
			personalEditError = error.message || 'Failed to update profile';
		} finally {
			isSavingPersonal = false;
		}
	}

	// Handle cancel personal edit
	function handleCancelPersonalEdit() {
		isEditingProfile = false;
		editPersonalName = authStore.user?.name || '';
		editPersonalEmail = authStore.user?.email || '';
		personalEditError = null;
		personalEditSuccess = null;
	}

	// Handle email verification success
	async function handleEmailVerificationSuccess() {
		showEmailVerification = false;
		
		// Fetch the updated user record from the server to get the latest verification status
		try {
			if (authStore.user) {
				const updatedUser = await pb.collection('users').getOne(authStore.user.id);
				// Update PocketBase's auth store with the fresh user data
				pb.authStore.save(pb.authStore.token, updatedUser);
				// Sync our reactive store
				authStore.syncState();
			}
		} catch (error) {
			console.error('Failed to refresh user data:', error);
		}
		
		personalEditSuccess = 'Email verified successfully!';
	}

	// Handle email verification cancel
	function handleEmailVerificationCancel() {
		showEmailVerification = false;
	}

	// Start email verification process
	function startEmailVerification() {
		showEmailVerification = true;
	}

</script>

<svelte:head>
	<title>Dashboard - {config.app.name}</title>
	<meta name="description" content="User dashboard" />
</svelte:head>

<!-- Hero Section -->
<section class="py-20 px-6">
	<div class="max-w-4xl mx-auto">
		<h1 class="text-4xl md:text-5xl font-bold mb-6">Dashboard</h1>
		<p class="text-xl text-muted-foreground">
			Welcome back, {authStore.user?.name || authStore.user?.email || 'User'}
		</p>
	</div>
</section>

<!-- Dashboard Content -->
<section class="py-20 border-t px-6">
	<div class="max-w-4xl mx-auto space-y-6">
		<!-- Profile Section -->
		<div class="border rounded-lg p-6">
			{#if showEmailVerification}
				<!-- Email Verification Step -->
				<OTPVerification 
					userID={authStore.user?.id || ''}
					email={authStore.user?.email || ''}
					purpose="signup_verification"
					onSuccess={handleEmailVerificationSuccess}
					onCancel={handleEmailVerificationCancel}
				/>
			{:else}
				<!-- Profile Header -->
				<div class="flex items-center justify-between mb-6">
					<h3 class="text-lg font-semibold">Profile</h3>
					<button
						onclick={() => isEditingProfile = !isEditingProfile}
						class="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1"
						disabled={isSavingPersonal}
					>
						<Edit3 class="w-4 h-4" />
						Edit
					</button>
				</div>

				<!-- Success Message -->
				{#if personalEditSuccess}
					<div class="mb-6 p-3 bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg">
						<p class="text-sm text-green-600 dark:text-green-300">{personalEditSuccess}</p>
					</div>
				{/if}

				{#if isEditingProfile}
					<!-- Edit Form -->
					<form onsubmit={(e) => { e.preventDefault(); handleSavePersonal(); }} class="space-y-4">
						{#if personalEditError}
							<div class="p-3 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg">
								<p class="text-sm text-red-600 dark:text-red-300">{personalEditError}</p>
							</div>
						{/if}

						<div class="space-y-2">
							<Label for="edit-personal-name">Name</Label>
							<Input
								id="edit-personal-name"
								type="text"
								bind:value={editPersonalName}
								disabled={isSavingPersonal}
								required
							/>
						</div>

						<div class="space-y-2">
							<Label for="edit-personal-email">Email</Label>
							<Input
								id="edit-personal-email"
								type="email"
								bind:value={editPersonalEmail}
								disabled={isSavingPersonal}
								required
							/>
							<p class="text-xs text-muted-foreground">Changing your email will require verification</p>
						</div>

						<div class="flex gap-2">
							<Button
								type="submit"
								size="sm"
								disabled={isSavingPersonal}
							>
								{isSavingPersonal ? 'Saving...' : 'Save Changes'}
							</Button>
							<Button
								type="button"
								variant="outline"
								size="sm"
								onclick={handleCancelPersonalEdit}
								disabled={isSavingPersonal}
							>
								Cancel
							</Button>
						</div>
					</form>
				{:else}
					<!-- Display Mode -->
					<div class="space-y-6">
						<!-- User Info -->
						<div>
							<div>
								<h4 class="text-lg font-semibold">
									{authStore.user?.name || 'User'}
								</h4>
								<div class="flex items-center gap-2 text-sm text-muted-foreground">
									<Mail class="w-4 h-4" />
									{authStore.user?.email}
									{#if !authStore.user?.verified}
										<span class="text-orange-600 dark:text-orange-400">⚠️ Not verified</span>
									{:else}
										<span class="text-green-600 dark:text-green-400">✅ Verified</span>
									{/if}
								</div>
								{#if authStore.user?.created}
									<div class="flex items-center gap-2 text-sm text-muted-foreground mt-1">
										<Calendar class="w-4 h-4" />
										Member since {formatDate(authStore.user.created)}
									</div>
								{/if}
							</div>
						</div>

						<!-- Account Status and Actions -->
						<div class="flex flex-col sm:flex-row gap-3">
							{#if subscriptionData?.status === 'active'}
								<div class="flex items-center gap-2 px-3 py-1 bg-green-50 dark:bg-green-950/30 text-green-800 dark:text-green-200 rounded-lg text-sm font-medium w-fit">
									<Crown class="w-4 h-4" />
									Premium Member
								</div>
							{:else}
								<button
									onclick={() => goto('/pricing')}
									class="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors text-sm font-medium w-fit"
								>
									<Crown class="w-4 h-4" />
									Upgrade to Premium
								</button>
							{/if}

							{#if !authStore.user?.verified}
								<Button
									size="sm"
									variant="outline"
									onclick={startEmailVerification}
									class="w-fit"
								>
									<Shield class="w-4 h-4 mr-2" />
									Verify Email
								</Button>
							{/if}
						</div>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Current Plan -->
		<div class="border rounded-lg p-6">
			<h3 class="text-lg font-semibold mb-4">Current Plan</h3>
			
			{#if isLoadingSubscription}
				<!-- Loading State -->
				<div class="flex items-center justify-between p-4 bg-muted/30 rounded-lg border">
					<div class="flex items-center gap-4">
						<div class="w-10 h-10 bg-muted rounded-lg flex items-center justify-center animate-pulse">
							<div class="w-5 h-5 bg-muted-foreground/30 rounded"></div>
						</div>
						<div class="space-y-2">
							<div class="h-5 w-24 bg-muted-foreground/20 rounded animate-pulse"></div>
							<div class="h-4 w-32 bg-muted-foreground/20 rounded animate-pulse"></div>
						</div>
					</div>
					<div class="h-10 w-24 bg-muted-foreground/20 rounded animate-pulse"></div>
				</div>
			{:else if subscriptionPlan && subscriptionData}
				<!-- Has Active Plan -->
				<div class="flex items-center justify-between p-4 bg-muted/30 rounded-lg border">
					<div class="flex items-center gap-4">
						<div class="w-10 h-10 bg-primary/10 rounded-lg flex items-center justify-center">
							<Crown class="w-5 h-5 text-primary" />
						</div>
						<div>
							<h4 class="text-lg font-semibold">{subscriptionPlan.name}</h4>
							<p class="text-sm text-muted-foreground">
								{subscriptionPlan.hours_per_month} hour{subscriptionPlan.hours_per_month !== 1 ? 's' : ''} per month
								• ${(subscriptionPlan.price_cents / 100).toFixed(2)} USD/{subscriptionPlan.billing_interval}
							</p>
							{#if usageData}
								<div class="mt-2">
									<p class="text-sm text-muted-foreground">
										{usageData.hours_used.toFixed(1)} / {usageData.hours_limit} hours used this month
									</p>
									<div class="w-full bg-muted rounded-full h-2 mt-1">
										<div 
											class="bg-primary h-2 rounded-full transition-all" 
											style="width: {Math.min(usageData.usage_percentage, 100)}%"
										></div>
									</div>
								</div>
							{/if}
							
							<!-- Upcoming Plan Info -->
							{#if subscriptionStore.getUpcomingPlan() && subscriptionStore.currentPeriodEnd}
								{@const upcomingPlan = subscriptionStore.getUpcomingPlan()}
								<div class="mt-3 pt-3 border-t border-muted">
									<p class="text-sm text-muted-foreground">
										Changes to <span class="text-foreground font-medium">{upcomingPlan?.name}</span> on {new Date(subscriptionStore.currentPeriodEnd).toLocaleDateString()}
									</p>
								</div>
							{/if}
						</div>
					</div>
					<button
						onclick={() => goto('/pricing')}
						class="flex items-center gap-2 px-4 py-2 border rounded-md hover:bg-muted transition-colors"
					>
						<Settings class="w-4 h-4" />
						Change Plan
					</button>
				</div>
			{:else if subscriptionStore.getEffectiveCurrentPlan()}
				<!-- Free Plan -->
				{@const freePlan = subscriptionStore.getEffectiveCurrentPlan()}
				<div class="flex items-center justify-between p-4 bg-muted/30 rounded-lg border">
					<div>
						<div>
							<h4 class="text-lg font-semibold">{freePlan?.name}</h4>
							<p class="text-sm text-muted-foreground">
								{freePlan?.hours_per_month} hour{freePlan?.hours_per_month !== 1 ? 's' : ''} per month
								• Free
							</p>
						</div>
					</div>
					<button
						onclick={() => goto('/pricing')}
						class="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
					>
						<Crown class="w-4 h-4" />
						Upgrade
					</button>
				</div>
			{:else}
				<!-- Fallback: No Plan Found -->
				<div class="flex items-center justify-between p-4 bg-muted/30 rounded-lg border">
					<div>
						<div>
							<h4 class="text-lg font-semibold">No Active Plan</h4>
							<p class="text-sm text-muted-foreground">Choose a plan to get started</p>
						</div>
					</div>
					<button
						onclick={() => goto('/pricing')}
						class="flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
					>
						<Crown class="w-4 h-4" />
						Choose Plan
					</button>
				</div>
			{/if}
		</div>

		<!-- API Key Management -->
		<APIKeyManager />
	</div>
</section>

