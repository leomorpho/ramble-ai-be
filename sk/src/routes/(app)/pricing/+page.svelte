<script lang="ts">
	import { subscriptionStore } from '$lib/stores/subscription.svelte.ts';
	import { authStore } from '$lib/stores/authClient.svelte.ts';
	import { createCheckoutSession, createPortalLink } from '$lib/stripe.ts';
	import { config } from '$lib/config.ts';
	import { Loader2, Check, Crown, Zap } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	
	let isLoading = $state(false);
	let checkoutLoading = $state<string | null>(null);
	// Removed yearly support - only show monthly plans
	// let billingInterval = $state<'month' | 'year'>('month');
	
	// Dialog states
	let showFreeDowngradeDialog = $state(false);
	let showErrorDialog = $state(false);
	let errorMessage = $state('');

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
		
		// Handle free plan switching
		if (plan?.billing_interval === 'free') {
			if (subscriptionStore.isSubscribed) {
				// User wants to downgrade to free - show confirmation dialog
				showFreeDowngradeDialog = true;
			}
			return;
		}

		checkoutLoading = planId;
		try {
			await createCheckoutSession(planId);
		} catch (error) {
			console.error('Error creating checkout session:', error);
			errorMessage = 'Failed to start checkout. Please try again.';
			showErrorDialog = true;
		} finally {
			checkoutLoading = null;
		}
	}

	function isCurrentPlan(planId: string): boolean {
		return subscriptionStore.isCurrentPlan(planId);
	}

	async function handleFreeDowngrade() {
		showFreeDowngradeDialog = false;
		try {
			await createPortalLink();
		} catch (error) {
			console.error('Error accessing billing portal:', error);
			errorMessage = 'Failed to access billing portal. Please try again.';
			showErrorDialog = true;
		}
	}

	function getButtonText(planId: string): string {
		if (checkoutLoading === planId) return 'Processing...';
		if (isCurrentPlan(planId)) return 'Current Plan';
		
		const plan = subscriptionStore.getPlan(planId);
		if (plan?.billing_interval === 'free') {
			return subscriptionStore.isSubscribed ? 'Switch to Free' : 'Select Free Plan';
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
					{@const isCurrentPlan = subscriptionStore.isCurrentPlan(plan.id)}
					
					<div class="relative border rounded-lg p-6 {isCurrentPlan ? 'bg-green-50 dark:bg-green-900/30' : ''}">
						{#if isCurrentPlan}
							<div class="absolute -top-3 left-1/2 transform -translate-x-1/2">
								<Badge class="bg-green-600 text-white px-4 py-1 text-sm font-semibold shadow-md">
									✓ Current Plan
								</Badge>
							</div>
						{:else if isPopular}
							<Badge class="mb-4">Most Popular</Badge>
						{/if}
						
						<div class="text-center mb-6 {isCurrentPlan ? 'mt-4' : ''}">
							<h3 class="text-xl font-semibold mb-2 {isCurrentPlan ? 'text-primary' : ''}">{plan.name}</h3>
							
							<div class="mb-4">
								{#if plan.billing_interval === 'free'}
									<div class="text-3xl font-bold">Free</div>
									<div class="text-sm text-muted-foreground">Always free</div>
								{:else}
									<div class="text-3xl font-bold">
										{subscriptionStore.formatPrice(plan.price_cents)}
									</div>
									<div class="text-sm text-muted-foreground">
										per {plan.billing_interval}
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
							variant={isCurrentPlan ? "secondary" : "default"}
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

<!-- FAQ/Features Section -->
<section class="py-20 border-t px-6">
	<div class="max-w-4xl mx-auto">
		<h2 class="text-3xl md:text-4xl font-bold mb-12">All Plans Include</h2>
		
		<div class="grid md:grid-cols-2 gap-8 mb-12">
			<div>
				<h3 class="text-lg font-semibold mb-4">Core Features</h3>
				<ul class="space-y-3">
					<li class="flex items-center gap-2">
						<Check class="h-4 w-4 text-green-600" />
						<span>High-quality audio transcription</span>
					</li>
					<li class="flex items-center gap-2">
						<Check class="h-4 w-4 text-green-600" />
						<span>Unlimited video quality exports</span>
					</li>
					<li class="flex items-center gap-2">
						<Check class="h-4 w-4 text-green-600" />
						<span>Multiple export formats</span>
					</li>
					<li class="flex items-center gap-2">
						<Check class="h-4 w-4 text-green-600" />
						<span>Secure file processing</span>
					</li>
				</ul>
			</div>
			
			<div>
				<h3 class="text-lg font-semibold mb-4">Support</h3>
				<ul class="space-y-3">
					<li class="flex items-center gap-2">
						<Check class="h-4 w-4 text-green-600" />
						<span>Email support</span>
					</li>
					<li class="flex items-center gap-2">
						<Check class="h-4 w-4 text-green-600" />
						<span>Cancel anytime</span>
					</li>
					<li class="flex items-center gap-2">
						<Check class="h-4 w-4 text-green-600" />
						<span>No long-term contracts</span>
					</li>
				</ul>
			</div>
		</div>
		
		<div class="border rounded-lg p-6 text-center">
			<h3 class="text-lg font-semibold mb-2">Questions?</h3>
			<p class="text-muted-foreground">
				Need help choosing the right plan? Contact our support team for assistance.
			</p>
		</div>
	</div>
</section>

<!-- Free Plan Downgrade Confirmation Dialog -->
<Dialog bind:open={showFreeDowngradeDialog}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Switch to Free Plan</DialogTitle>
			<DialogDescription>
				Switching to the Free plan will cancel your current subscription at the end of the billing period. You can manage this change in the billing portal.
			</DialogDescription>
		</DialogHeader>
		<DialogFooter>
			<Button variant="outline" onclick={() => showFreeDowngradeDialog = false}>
				Cancel
			</Button>
			<Button onclick={handleFreeDowngrade}>
				Continue to Billing Portal
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>

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