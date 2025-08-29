<script lang="ts">
	import { subscriptionStore } from '$lib/stores/subscription.svelte.js';
	import { config } from '$lib/config.js';
	import { Crown, Zap, Shield, Star } from 'lucide-svelte';

	$: currentProduct = subscriptionStore.userSubscription 
		? subscriptionStore.getProduct(
			subscriptionStore.getPrice(subscriptionStore.userSubscription.price_id)?.product_id || ''
		) : null;
</script>

<svelte:head>
	<title>Premium Features - {config.app.name}</title>
	<meta name="description" content="Access exclusive premium features" />
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<div class="mx-auto max-w-4xl">
		<div class="text-center mb-8">
			<div class="inline-flex items-center rounded-full bg-gradient-to-r from-yellow-400 to-orange-500 px-4 py-2 text-white mb-4">
				<Crown class="h-5 w-5 mr-2" />
				<span class="font-semibold">Premium User</span>
			</div>
			<h1 class="text-4xl font-bold mb-4">Premium Features</h1>
			<p class="text-xl text-muted-foreground">
				Welcome to your exclusive premium experience!
			</p>
		</div>

		{#if currentProduct}
			<div class="rounded-lg bg-gradient-to-r from-blue-50 to-purple-50 border p-6 mb-8">
				<div class="flex items-center mb-2">
					<Star class="h-5 w-5 text-yellow-500 mr-2" />
					<h2 class="text-lg font-semibold">Your Plan: {currentProduct.name}</h2>
				</div>
				{#if currentProduct.description}
					<p class="text-muted-foreground">{currentProduct.description}</p>
				{/if}
			</div>
		{/if}

		<div class="grid gap-6 md:grid-cols-2 lg:grid-cols-3 mb-12">
			<div class="rounded-lg border bg-card p-6">
				<div class="flex items-center mb-4">
					<div class="rounded-lg bg-blue-100 p-3 mr-4">
						<Zap class="h-6 w-6 text-blue-600" />
					</div>
					<h3 class="text-lg font-semibold">Advanced Analytics</h3>
				</div>
				<p class="text-muted-foreground mb-4">
					Get detailed insights and analytics about your usage and performance.
				</p>
				<div class="rounded-lg bg-blue-50 p-4">
					<div class="text-2xl font-bold text-blue-600">127%</div>
					<div class="text-sm text-blue-600">Growth this month</div>
				</div>
			</div>

			<div class="rounded-lg border bg-card p-6">
				<div class="flex items-center mb-4">
					<div class="rounded-lg bg-green-100 p-3 mr-4">
						<Shield class="h-6 w-6 text-green-600" />
					</div>
					<h3 class="text-lg font-semibold">Priority Support</h3>
				</div>
				<p class="text-muted-foreground mb-4">
					Get priority access to our support team with faster response times.
				</p>
				<div class="rounded-lg bg-green-50 p-4">
					<div class="text-2xl font-bold text-green-600">&lt; 2h</div>
					<div class="text-sm text-green-600">Average response time</div>
				</div>
			</div>

			<div class="rounded-lg border bg-card p-6">
				<div class="flex items-center mb-4">
					<div class="rounded-lg bg-purple-100 p-3 mr-4">
						<Crown class="h-6 w-6 text-purple-600" />
					</div>
					<h3 class="text-lg font-semibold">Exclusive Features</h3>
				</div>
				<p class="text-muted-foreground mb-4">
					Access to beta features and premium-only functionality.
				</p>
				<div class="rounded-lg bg-purple-50 p-4">
					<div class="text-2xl font-bold text-purple-600">5+</div>
					<div class="text-sm text-purple-600">Beta features available</div>
				</div>
			</div>
		</div>

		<!-- Demo Premium Content -->
		<div class="rounded-lg border bg-card p-8 mb-8">
			<h2 class="text-2xl font-bold mb-6">Premium Dashboard</h2>
			
			<div class="grid gap-6 md:grid-cols-2">
				<div class="space-y-4">
					<h3 class="text-lg font-semibold">Usage Statistics</h3>
					<div class="space-y-2">
						<div class="flex justify-between">
							<span>API Calls</span>
							<span class="font-semibold">15,247</span>
						</div>
						<div class="w-full bg-gray-200 rounded-full h-2">
							<div class="bg-blue-600 h-2 rounded-full" style="width: 76%"></div>
						</div>
					</div>
					<div class="space-y-2">
						<div class="flex justify-between">
							<span>Storage Used</span>
							<span class="font-semibold">4.2 GB</span>
						</div>
						<div class="w-full bg-gray-200 rounded-full h-2">
							<div class="bg-green-600 h-2 rounded-full" style="width: 42%"></div>
						</div>
					</div>
				</div>

				<div class="space-y-4">
					<h3 class="text-lg font-semibold">Recent Activity</h3>
					<div class="space-y-3">
						<div class="flex items-center text-sm">
							<div class="w-2 h-2 bg-green-500 rounded-full mr-3"></div>
							<span>Data export completed</span>
							<span class="text-muted-foreground ml-auto">2 min ago</span>
						</div>
						<div class="flex items-center text-sm">
							<div class="w-2 h-2 bg-blue-500 rounded-full mr-3"></div>
							<span>API key refreshed</span>
							<span class="text-muted-foreground ml-auto">1 hour ago</span>
						</div>
						<div class="flex items-center text-sm">
							<div class="w-2 h-2 bg-purple-500 rounded-full mr-3"></div>
							<span>Beta feature enabled</span>
							<span class="text-muted-foreground ml-auto">3 hours ago</span>
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="text-center">
			<h3 class="text-lg font-semibold mb-4">Enjoying Premium?</h3>
			<div class="flex justify-center space-x-4">
				<a 
					href="/billing" 
					class="inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 transition-colors"
				>
					Manage Subscription
				</a>
				<a 
					href="/pricing" 
					class="inline-flex items-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
				>
					Upgrade Plan
				</a>
			</div>
		</div>
	</div>
</div>