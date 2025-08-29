<script lang="ts">
	import { subscriptionStore } from '$lib/stores/subscription.svelte.js';
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { createPortalLink, formatPrice } from '$lib/stripe.js';
	import { config } from '$lib/config.js';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { CreditCard, Calendar, AlertCircle, Loader2, ExternalLink } from 'lucide-svelte';

	let isLoading = $state(false);

	onMount(() => {
		// Redirect if not logged in
		if (!authStore.isLoggedIn) {
			goto('/login?redirect=/billing');
			return;
		}

		subscriptionStore.refresh();
	});

	async function handleManageBilling() {
		isLoading = true;
		try {
			await createPortalLink();
		} catch (error) {
			console.error('Error creating portal link:', error);
			alert('Failed to open billing portal. Please try again.');
		} finally {
			isLoading = false;
		}
	}

	function formatDate(timestamp: number): string {
		return new Date(timestamp * 1000).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'long',
			day: 'numeric'
		});
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'active': return 'text-green-600 bg-green-100';
			case 'trialing': return 'text-blue-600 bg-blue-100';
			case 'canceled': return 'text-red-600 bg-red-100';
			case 'past_due': return 'text-yellow-600 bg-yellow-100';
			default: return 'text-gray-600 bg-gray-100';
		}
	}

	function getStatusText(status: string): string {
		switch (status) {
			case 'active': return 'Active';
			case 'trialing': return 'Trial';
			case 'canceled': return 'Canceled';
			case 'past_due': return 'Past Due';
			case 'incomplete': return 'Incomplete';
			case 'incomplete_expired': return 'Expired';
			default: return status;
		}
	}

	let currentPrice = $derived(subscriptionStore.userSubscription 
		? subscriptionStore.getPrice(subscriptionStore.userSubscription.price_id)
		: null);

	let currentProduct = $derived(currentPrice 
		? subscriptionStore.getProduct(currentPrice.product_id)
		: null);
</script>

<svelte:head>
	<title>Billing - {config.app.name}</title>
	<meta name="description" content="Manage your subscription and billing" />
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<div class="mx-auto max-w-4xl">
		<div class="mb-8">
			<h1 class="text-3xl font-bold mb-2">Billing & Subscription</h1>
			<p class="text-muted-foreground">
				Manage your subscription, payment methods, and billing history.
			</p>
		</div>

		{#if subscriptionStore.isLoading}
			<div class="text-center py-12">
				<Loader2 class="h-8 w-8 animate-spin mx-auto mb-4" />
				<p>Loading subscription details...</p>
			</div>
		{:else if subscriptionStore.isSubscribed && subscriptionStore.userSubscription}
			<!-- Current Subscription -->
			<div class="grid gap-6 md:grid-cols-2 mb-8">
				<div class="rounded-lg border bg-card p-6">
					<div class="flex items-center mb-4">
						<CreditCard class="h-5 w-5 mr-2" />
						<h2 class="text-lg font-semibold">Current Plan</h2>
					</div>
					
					{#if currentProduct && currentPrice}
						<div class="space-y-3">
							<div>
								<div class="text-2xl font-bold">{currentProduct.name}</div>
								{#if currentProduct.description}
									<div class="text-sm text-muted-foreground">{currentProduct.description}</div>
								{/if}
							</div>
							
							<div class="flex items-baseline">
								<span class="text-xl font-semibold">
									{formatPrice(currentPrice.unit_amount, currentPrice.currency)}
								</span>
								<span class="text-muted-foreground ml-2">
									per {currentPrice.interval}
								</span>
							</div>

							<div class="flex items-center">
								<span class={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(subscriptionStore.userSubscription.status)}`}>
									{getStatusText(subscriptionStore.userSubscription.status)}
								</span>
								{#if subscriptionStore.userSubscription.cancel_at_period_end}
									<span class="ml-2 text-sm text-red-600">
										(Cancels at period end)
									</span>
								{/if}
							</div>
						</div>
					{/if}
				</div>

				<div class="rounded-lg border bg-card p-6">
					<div class="flex items-center mb-4">
						<Calendar class="h-5 w-5 mr-2" />
						<h2 class="text-lg font-semibold">Billing Cycle</h2>
					</div>
					
					<div class="space-y-3">
						<div>
							<div class="text-sm text-muted-foreground">Current period started</div>
							<div class="font-medium">
								{formatDate(subscriptionStore.userSubscription.current_period_start)}
							</div>
						</div>
						
						<div>
							<div class="text-sm text-muted-foreground">
								{subscriptionStore.userSubscription.cancel_at_period_end ? 'Subscription ends' : 'Next billing date'}
							</div>
							<div class="font-medium">
								{formatDate(subscriptionStore.userSubscription.current_period_end)}
							</div>
						</div>

						{#if subscriptionStore.userSubscription.trial_end && subscriptionStore.userSubscription.trial_end > Date.now() / 1000}
							<div>
								<div class="text-sm text-blue-600">Trial ends</div>
								<div class="font-medium text-blue-600">
									{formatDate(subscriptionStore.userSubscription.trial_end)}
								</div>
							</div>
						{/if}
					</div>
				</div>
			</div>

			<!-- Manage Subscription -->
			<div class="rounded-lg border bg-card p-6 mb-8">
				<h2 class="text-lg font-semibold mb-4">Manage Subscription</h2>
				<p class="text-muted-foreground mb-4">
					Update your payment method, download invoices, change your plan, or cancel your subscription.
				</p>
				
				<button
					onclick={handleManageBilling}
					disabled={isLoading}
					class="inline-flex items-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				>
					{#if isLoading}
						<Loader2 class="h-4 w-4 animate-spin mr-2" />
						Opening...
					{:else}
						<ExternalLink class="h-4 w-4 mr-2" />
						Manage Billing
					{/if}
				</button>
			</div>

			<!-- Warnings -->
			{#if subscriptionStore.userSubscription.status === 'past_due'}
				<div class="rounded-lg border border-yellow-200 bg-yellow-50 p-4 mb-6">
					<div class="flex items-start">
						<AlertCircle class="h-5 w-5 text-yellow-600 mr-3 mt-0.5" />
						<div>
							<h3 class="text-sm font-semibold text-yellow-800">Payment Issue</h3>
							<p class="text-sm text-yellow-700 mt-1">
								Your subscription payment failed. Please update your payment method to avoid service interruption.
							</p>
						</div>
					</div>
				</div>
			{/if}

			{#if subscriptionStore.userSubscription.cancel_at_period_end}
				<div class="rounded-lg border border-red-200 bg-red-50 p-4 mb-6">
					<div class="flex items-start">
						<AlertCircle class="h-5 w-5 text-red-600 mr-3 mt-0.5" />
						<div>
							<h3 class="text-sm font-semibold text-red-800">Subscription Ending</h3>
							<p class="text-sm text-red-700 mt-1">
								Your subscription will end on {formatDate(subscriptionStore.userSubscription.current_period_end)}. 
								You can reactivate it anytime before then.
							</p>
						</div>
					</div>
				</div>
			{/if}
		{:else}
			<!-- No Subscription -->
			<div class="text-center py-12">
				<div class="rounded-lg border-2 border-dashed border-gray-300 p-8">
					<CreditCard class="h-12 w-12 text-gray-400 mx-auto mb-4" />
					<h2 class="text-xl font-semibold mb-2">No Active Subscription</h2>
					<p class="text-muted-foreground mb-6">
						You don't have an active subscription. Choose a plan to get started.
					</p>
					<a 
						href="/pricing"
						class="inline-flex items-center rounded-md bg-primary px-6 py-3 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
					>
						View Pricing Plans
					</a>
				</div>
			</div>
		{/if}

		<!-- Additional Actions -->
		<div class="border-t pt-6">
			<h3 class="text-lg font-semibold mb-4">Need Help?</h3>
			<div class="grid gap-4 md:grid-cols-1">
				<a 
					href="/pricing" 
					class="rounded-lg border p-4 hover:bg-accent transition-colors"
				>
					<h4 class="font-medium mb-1">Change Plan</h4>
					<p class="text-sm text-muted-foreground">Upgrade or downgrade your subscription</p>
				</a>
			</div>
		</div>
	</div>
</div>