<script lang="ts">
	import { subscriptionStore } from '$lib/stores/subscription.svelte.ts';
	import { authStore } from '$lib/stores/authClient.svelte.ts';
	import { createCheckoutSession, cancelSubscription, changePlanDirect, createPortalLink } from '$lib/payment.ts';
	import { config } from '$lib/config.ts';
	import { Loader2, Check, Crown, Zap, AlertCircle } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	
	let isLoading = $state(false);
	let checkoutLoading = $state<string | null>(null);
	// Removed yearly support - only show monthly plans
	// let billingInterval = $state<'month' | 'year'>('month');
	
	// Dialog states
	let showErrorDialog = $state(false);
	let errorMessage = $state('');
	let showCancelDialog = $state(false);
	let pendingCancelPlan = $state<string | null>(null);
	let showPlanChangeDialog = $state(false);
	let pendingPlanChange = $state<{
		planId: string;
		checkoutUrl: string;
		paymentStatus: any;
	} | null>(null);

	// Subscription store is initialized in root layout
	onMount(() => {
		// Refresh data to ensure it's current
		subscriptionStore.refresh();
	});

	async function handleSubscribe(planId: string) {
		if (!authStore.isLoggedIn) {
			// Redirect to login
			window.location.href = '/login?redirect=/pricing';
			return;
		}

		const plan = subscriptionStore.getPlan(planId);
		const isDowngradeToFree = plan?.billing_interval === 'free' && subscriptionStore.isSubscribed;
		
		// Show confirmation dialog for cancellation/downgrade to free
		if (isDowngradeToFree) {
			pendingCancelPlan = planId;
			showCancelDialog = true;
			return;
		}

		// For upgrades or new subscriptions, use hybrid approach
		checkoutLoading = planId;
		try {
			const result = await createCheckoutSession(planId);
			
			// Check if hybrid approach is available (user has valid payment methods)
			if (result && typeof result === 'object' && result.can_use_direct_change) {
				// Show confirmation dialog for direct plan change
				pendingPlanChange = {
					planId: planId,
					checkoutUrl: result.checkout_url,
					paymentStatus: result.payment_status
				};
				showPlanChangeDialog = true;
			}
			// Otherwise, createCheckoutSession will have redirected to Stripe checkout
		} catch (error) {
			console.error('Error creating checkout session:', error);
			errorMessage = 'Failed to start checkout. Please try again.';
			showErrorDialog = true;
		} finally {
			checkoutLoading = null;
		}
	}

	function isCurrentPlan(planId: string): boolean {
		// Use effective current plan logic to include free plan when no subscription
		return subscriptionStore.isEffectivelyOnPlan(planId);
	}

	async function handleCancelConfirm() {
		if (!pendingCancelPlan) return;
		
		checkoutLoading = pendingCancelPlan;
		try {
			const result = await cancelSubscription();
			// Refresh subscription data to reflect the change
			subscriptionStore.refresh();
			showCancelDialog = false;
			pendingCancelPlan = null;
			
			// Show success message for immediate cancellation
			if (result) {
				errorMessage = `Subscription cancelled successfully. You've been switched to the Free plan and will receive a prorated refund for unused time.`;
				showErrorDialog = true; // Reusing error dialog for success message
			}
		} catch (error) {
			console.error('Error cancelling subscription:', error);
			errorMessage = 'Failed to cancel subscription. Please try again.';
			showErrorDialog = true;
			showCancelDialog = false;
		} finally {
			checkoutLoading = null;
		}
	}

	function handleCancelDecline() {
		showCancelDialog = false;
		pendingCancelPlan = null;
	}

	async function handlePlanChangeConfirm() {
		if (!pendingPlanChange) return;
		
		checkoutLoading = pendingPlanChange.planId;
		try {
			const result = await changePlanDirect(pendingPlanChange.planId);
			// Refresh subscription data to reflect the change
			subscriptionStore.refresh();
			showPlanChangeDialog = false;
			pendingPlanChange = null;
			checkoutLoading = null;
			
			// Show success message
			if (result && result.message) {
				errorMessage = result.message;
				showErrorDialog = true; // Reusing error dialog for success message
			}
		} catch (error) {
			console.error('Error changing plan directly:', error);
			// Keep loading state active while redirecting to checkout
			// Don't show error dialog - just keep the loading state
			
			// Immediately redirect to checkout on failure
			if (pendingPlanChange?.checkoutUrl) {
				// Keep the dialog open with loading state while redirecting
				window.location.href = pendingPlanChange.checkoutUrl;
			} else {
				// Only clear loading state if we can't redirect
				checkoutLoading = null;
				errorMessage = 'Failed to change plan. Please try again or contact support.';
				showErrorDialog = true;
				showPlanChangeDialog = false;
			}
		}
	}

	function handlePlanChangeDecline() {
		if (!pendingPlanChange) return;
		
		// Redirect to checkout portal instead
		if (pendingPlanChange.checkoutUrl) {
			window.location.href = pendingPlanChange.checkoutUrl;
		}
		
		showPlanChangeDialog = false;
		pendingPlanChange = null;
	}

	async function handleBillingPortal() {
		try {
			await createPortalLink();
		} catch (error) {
			console.error('Error creating billing portal link:', error);
			errorMessage = 'Failed to access billing portal. Please try again.';
			showErrorDialog = true;
		}
	}

	function getButtonText(planId: string): string {
		if (checkoutLoading === planId) return 'Processing...';
		if (isCurrentPlan(planId)) return 'Current Plan';
		
		const plan = subscriptionStore.getPlan(planId);
		if (plan?.billing_interval === 'free') {
			return subscriptionStore.isSubscribed ? 'Cancel Subscription' : 'Current Plan';
		}
		
		if (!authStore.isLoggedIn) return 'Sign Up to Subscribe';
		if (subscriptionStore.isSubscribed) return 'Switch Plan';
		return 'Subscribe';
	}

	function isButtonDisabled(planId: string): boolean {
		return checkoutLoading !== null || isCurrentPlan(planId);
	}

	function getPlanIcon(planName: string) {
		if (planName.toLowerCase().includes('pro')) return Crown;
		if (planName.toLowerCase().includes('basic')) return Zap;
		return Check;
	}

	function getMonthlyAndFreePlans() {
		return subscriptionStore.plans
			.filter(plan => plan.billing_interval === 'month' || plan.billing_interval === 'free')
			.sort((a, b) => a.display_order - b.display_order);
	}

	// Removed yearly-specific functions
	// function calculateSavings(monthlyPrice: number, yearlyPrice: number): number { ... }
	// function getMonthlyEquivalent(plan: any) { ... }
	// function hasYearlyPlans(): boolean { ... }
</script>

<svelte:head>
	<title>Pricing - {config.app.name}</title>
	<meta name="description" content="Choose the perfect plan for your needs" />
</svelte:head>

<!-- Hero Section -->
<section class="py-20 px-6">
	<div class="max-w-4xl mx-auto">
		<h1 class="text-4xl md:text-5xl font-bold mb-6">Choose Your Plan</h1>
		<p class="text-xl text-muted-foreground">
			Process more audio, get unlimited exports, and access premium features.
		</p>
	</div>
</section>

<!-- Pricing Plans -->
<section class="py-20 border-t px-6">
	<div class="max-w-4xl mx-auto">
		{#if subscriptionStore.isLoading}
			<div class="text-center py-8">
				<Loader2 class="h-6 w-6 animate-spin mx-auto mb-3" />
				<p class="text-sm text-muted-foreground">Loading pricing plans...</p>
			</div>
		{:else if subscriptionStore.plans.length === 0}
			<div class="text-center py-12">
				<p class="text-muted-foreground">No pricing plans available at the moment.</p>
				<p class="text-sm text-muted-foreground mt-2">
					Please check back later or contact support.
				</p>
			</div>
		{:else}
			<!-- Removed billing interval toggle - only showing monthly plans -->

			<!-- Plans Grid -->
			<div class="grid gap-6 md:grid-cols-3">
				{#each getMonthlyAndFreePlans() as plan (plan.id)}
					{@const isPopular = plan.name.toLowerCase().includes('basic')}
					{@const isPlanCurrent = isCurrentPlan(plan.id)}
					
					<div class="relative border rounded-lg p-6 {isPlanCurrent ? 'bg-green-50 dark:bg-green-900/30 border-green-200 dark:border-green-800' : ''}">
						{#if isPlanCurrent}
							<div class="absolute -top-3 left-1/2 transform -translate-x-1/2">
								<Badge class="bg-green-600 text-white px-4 py-1 text-sm font-semibold shadow-md">
									✓ Current Plan
								</Badge>
							</div>
						{:else if isPopular}
							<Badge class="mb-4">Most Popular</Badge>
						{/if}
						
						<div class="text-center mb-6 {isPlanCurrent ? 'mt-4' : ''}">
							<h3 class="text-xl font-semibold mb-2 {isPlanCurrent ? 'text-primary' : ''}">{plan.name}</h3>
							
							<div class="mb-4">
								{#if plan.billing_interval === 'free'}
									<div class="text-3xl font-bold">$0</div>
									<div class="text-sm text-muted-foreground">No monthly fee</div>
								{:else}
									<div class="text-3xl font-bold">
										{subscriptionStore.formatPrice(plan.price_cents)}
									</div>
									<div class="text-sm text-muted-foreground">
										USD per {plan.billing_interval}
									</div>
								{/if}
							</div>

							<div class="text-lg font-medium text-primary mb-4">
								{plan.hours_per_month} hour{plan.hours_per_month !== 1 ? 's' : ''} per month
							</div>
						</div>

						<ul class="space-y-2 mb-6">
							{#each plan.features as feature}
								<li class="flex items-start gap-2 text-sm">
									<Check class="h-4 w-4 text-green-600 mt-0.5 flex-shrink-0" />
									<span>{feature}</span>
								</li>
							{/each}
						</ul>

						<Button 
							class="w-full" 
							variant={isPlanCurrent ? "secondary" : "default"}
							disabled={isButtonDisabled(plan.id)}
							onclick={() => handleSubscribe(plan.id)}
						>
							{getButtonText(plan.id)}
						</Button>
					</div>
				{/each}
			</div>
		{/if}


		<!-- Usage Warning -->
		{#if subscriptionStore.usageWarning}
			<div class="mt-8">
				<div class="border rounded-lg p-4 bg-yellow-50 dark:bg-yellow-950/30 border-yellow-200 dark:border-yellow-800">
					<div class="text-yellow-800 dark:text-yellow-200">
						<strong>Usage Notice:</strong> {subscriptionStore.usageWarning.message}
					</div>
				</div>
			</div>
		{/if}
	</div>
</section>



<!-- Error Dialog -->
<Dialog bind:open={showErrorDialog}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Error</DialogTitle>
			<DialogDescription>
				{errorMessage}
			</DialogDescription>
		</DialogHeader>
		<DialogFooter>
			<Button onclick={() => showErrorDialog = false}>
				OK
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>

<!-- Cancel Subscription Dialog -->
<Dialog bind:open={showCancelDialog}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Cancel Subscription</DialogTitle>
			<DialogDescription>
				Are you sure you want to cancel your subscription? Your subscription will be cancelled immediately and you'll switch to the Free plan. You'll receive a prorated refund for the unused portion of your billing period. You can upgrade again anytime.
			</DialogDescription>
		</DialogHeader>
		<DialogFooter class="gap-2">
			<Button variant="outline" onclick={handleCancelDecline} disabled={checkoutLoading !== null}>
				Keep Subscription
			</Button>
			<Button 
				variant="destructive" 
				onclick={handleCancelConfirm}
				disabled={checkoutLoading !== null}
			>
				{#if checkoutLoading}
					<Loader2 class="h-4 w-4 animate-spin mr-2" />
				{/if}
				Cancel Subscription
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>

<!-- Plan Change Confirmation Dialog -->
<Dialog bind:open={showPlanChangeDialog}>
	<DialogContent>
		{#if checkoutLoading === pendingPlanChange?.planId}
			<!-- Loading overlay -->
			<div class="absolute inset-0 bg-background/80 backdrop-blur-sm rounded-lg flex items-center justify-center z-10">
				<div class="text-center">
					<Loader2 class="h-8 w-8 animate-spin mx-auto mb-3" />
					<p class="text-sm text-muted-foreground">Processing plan change...</p>
					<p class="text-xs text-muted-foreground mt-1">Please wait while we update your subscription</p>
				</div>
			</div>
		{/if}
		
		<DialogHeader>
			<DialogTitle>Confirm Plan Change</DialogTitle>
			<DialogDescription>
				{#if pendingPlanChange}
					{@const plan = subscriptionStore.getPlan(pendingPlanChange.planId)}
					{@const currentPlan = subscriptionStore.currentPlan}
					{#if plan && currentPlan}
						You're about to change from <strong>{subscriptionStore.currentPlan?.name}</strong> to <strong>{plan.name}</strong>.
						
						{#if plan.price_cents > (currentPlan.price_cents || 0)}
							<!-- Upgrade -->
							<br/><br/><strong>Immediate upgrade:</strong> You'll get {plan.name} features right away. Your payment will be prorated - you'll only pay for the time remaining in your billing period.
						{:else}
							<!-- Downgrade -->
							<br/><br/><strong>Immediate downgrade:</strong> You'll switch to {plan.name} features right away. You'll receive a prorated credit for the unused portion of your current plan.
						{/if}
						
						<br/><br/>We'll use your existing payment method. You can also use our secure checkout portal if you prefer.
					{/if}
				{/if}
			</DialogDescription>
		</DialogHeader>
		<DialogFooter class="gap-2">
			<Button variant="outline" onclick={handlePlanChangeDecline} disabled={checkoutLoading !== null}>
				Use Secure Checkout
			</Button>
			<Button 
				onclick={handlePlanChangeConfirm}
				disabled={checkoutLoading !== null}
			>
				{#if checkoutLoading === pendingPlanChange?.planId}
					<Loader2 class="h-4 w-4 animate-spin mr-2" />
					Processing...
				{:else}
					Confirm Plan Change
				{/if}
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>