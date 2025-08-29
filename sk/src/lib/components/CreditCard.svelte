<script lang="ts">
	import { Loader2 } from 'lucide-svelte';
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

	let {
		price,
		checkoutLoading,
		isButtonDisabled,
		onPurchase
	}: {
		price: Price;
		checkoutLoading: string | null;
		isButtonDisabled: (priceId: string) => boolean;
		onPurchase: (priceId: string) => void;
	} = $props();

	function handlePurchase() {
		onPurchase(price.price_id);
	}
</script>

<div class="rounded-lg border bg-card p-4 text-center hover:shadow-md transition-shadow">
	<div class="mb-3">
		<div class="text-2xl font-bold">
			{formatPrice(price.unit_amount, price.currency)}
		</div>
		<div class="text-sm text-muted-foreground">{price.description}</div>
		{#if price.metadata?.bonus}
			<div class="text-xs text-green-600 mt-1">{price.metadata.bonus}</div>
		{/if}
	</div>
	
	<button
		onclick={handlePurchase}
		disabled={isButtonDisabled(price.price_id)}
		class="w-full rounded-md px-3 py-2 text-sm font-medium transition-colors
			border border-primary text-primary hover:bg-primary hover:text-primary-foreground
			disabled:opacity-50 disabled:cursor-not-allowed"
	>
		{#if checkoutLoading === price.price_id}
			<Loader2 class="h-4 w-4 animate-spin inline mr-2" />
		{/if}
		Purchase Credits
	</button>
</div>