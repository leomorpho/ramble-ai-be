<script lang="ts">
	import { Check, Loader2 } from 'lucide-svelte';
	import { formatPrice } from '$lib/stripe.js';

	interface Price {
		id: string;
		price_id: string;
		product_id: string;
		active: boolean;
		currency: string;
		unit_amount: number;
		type: string;
		interval?: string;
		interval_count?: number;
		trial_period_days?: number;
		metadata?: any;
	}

	interface Product {
		id: string;
		product_id: string;
		active: boolean;
		name: string;
		description: string;
		image?: string;
		metadata?: any;
		product_order?: number;
	}

	let {
		product,
		monthlyPrice,
		yearlyPrice,
		isPopular = false,
		popularLabel = "Most Popular",
		isCurrentPlan,
		checkoutLoading,
		getButtonText,
		isButtonDisabled,
		onSubscribe
	}: {
		product: Product;
		monthlyPrice?: Price;
		yearlyPrice?: Price;
		isPopular?: boolean;
		popularLabel?: string;
		isCurrentPlan: (priceId: string) => boolean;
		checkoutLoading: string | null;
		getButtonText: (priceId: string) => string;
		isButtonDisabled: (priceId: string) => boolean;
		onSubscribe: (priceId: string) => void;
	} = $props();

	function handleSubscribe(priceId: string) {
		onSubscribe(priceId);
	}
</script>

<div class="relative rounded-xl border bg-card shadow-sm hover:shadow-md transition-shadow
	{isPopular ? 'border-primary ring-2 ring-primary/20' : ''}">
	
	{#if isPopular}
		<div class="absolute -top-3 left-1/2 transform -translate-x-1/2">
			<span class="bg-primary text-primary-foreground text-xs font-medium px-3 py-1 rounded-full">
				{popularLabel}
			</span>
		</div>
	{/if}

	<div class="p-6">
		<div class="text-center mb-6">
			<h3 class="text-xl font-bold mb-2">{product.name}</h3>
			<p class="text-sm text-muted-foreground">{product.description}</p>
		</div>

		{#if monthlyPrice}
			<div class="text-center mb-6">
				<div class="flex items-baseline justify-center mb-1">
					<span class="text-3xl font-bold">
						{formatPrice(monthlyPrice.unit_amount, monthlyPrice.currency)}
					</span>
					<span class="text-muted-foreground ml-1">/month</span>
				</div>
				{#if yearlyPrice}
					<p class="text-sm text-muted-foreground">
						or {formatPrice(Math.floor(yearlyPrice.unit_amount / 12), yearlyPrice.currency)}/month billed yearly
					</p>
				{/if}
			</div>

			{#if monthlyPrice.trial_period_days}
				<div class="text-center mb-4">
					<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
						{monthlyPrice.trial_period_days} day free trial
					</span>
				</div>
			{/if}

			<!-- Features from metadata -->
			{#if monthlyPrice.metadata?.features}
				<div class="mb-6">
					<ul class="text-sm space-y-2">
						{#each monthlyPrice.metadata.features.split(', ') as feature}
							<li class="flex items-start">
								<Check class="h-4 w-4 text-green-500 mr-2 mt-0.5 flex-shrink-0" />
								<span>{feature}</span>
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			<div class="space-y-2">
				<button
					onclick={() => handleSubscribe(monthlyPrice.price_id)}
					disabled={isButtonDisabled(monthlyPrice.price_id)}
					class="w-full rounded-lg px-4 py-2.5 text-sm font-medium transition-colors
						{isCurrentPlan(monthlyPrice.price_id) 
							? 'bg-green-100 text-green-800 cursor-not-allowed' 
							: 'bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50'
						}"
				>
					{#if checkoutLoading === monthlyPrice.price_id}
						<Loader2 class="h-4 w-4 animate-spin inline mr-2" />
					{/if}
					{getButtonText(monthlyPrice.price_id)} - Monthly
				</button>

				{#if yearlyPrice}
					<button
						onclick={() => handleSubscribe(yearlyPrice.price_id)}
						disabled={isButtonDisabled(yearlyPrice.price_id)}
						class="w-full rounded-lg px-4 py-2.5 text-sm font-medium transition-colors border
							{isCurrentPlan(yearlyPrice.price_id) 
								? 'bg-green-100 text-green-800 cursor-not-allowed border-green-200' 
								: 'border-muted-foreground/20 text-muted-foreground hover:border-primary hover:text-primary disabled:opacity-50'
							}"
					>
						{#if checkoutLoading === yearlyPrice.price_id}
							<Loader2 class="h-4 w-4 animate-spin inline mr-2" />
						{/if}
						{getButtonText(yearlyPrice.price_id)} - Yearly
						{#if yearlyPrice.metadata?.discount}
							<span class="text-xs text-green-600 ml-1">({yearlyPrice.metadata.discount})</span>
						{/if}
					</button>
				{/if}
			</div>

			{#if isCurrentPlan(monthlyPrice.price_id) || (yearlyPrice && isCurrentPlan(yearlyPrice.price_id))}
				<div class="flex items-center justify-center mt-3 text-green-600">
					<Check class="h-4 w-4 mr-1" />
					<span class="text-sm">Active subscription</span>
				</div>
			{/if}
		{/if}
	</div>
</div>