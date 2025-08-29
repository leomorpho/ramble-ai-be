import { browser } from '$app/environment';
import { authStore } from './stores/authClient.svelte.js';
import { pb } from './pocketbase.js';

// Helper to create checkout session for a subscription plan
export async function createCheckoutSession(planId: string) {
	if (!browser) return null;
	
	const user = authStore.user;
	if (!user) {
		throw new Error('User must be logged in to create checkout session');
	}

	const response = await fetch(`${pb.baseUrl}/create-checkout-session`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify({
			plan_id: planId,
			user_id: user.id
		})
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.error || 'Failed to create checkout session');
	}

	const { url } = await response.json();
	
	// Redirect to Stripe checkout
	if (url) {
		window.location.href = url;
	}
	
	return url;
}

// Helper to create billing portal link
export async function createPortalLink() {
	if (!browser) return null;
	
	const user = authStore.user;
	if (!user) {
		throw new Error('User must be logged in to access billing portal');
	}

	const response = await fetch(`${pb.baseUrl}/create-portal-link`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		body: JSON.stringify({
			user_id: user.id
		})
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.error || 'Failed to create portal link');
	}

	const { url } = await response.json();
	
	// Redirect to Stripe billing portal
	if (url) {
		window.location.href = url;
	}
	
	return url;
}

// Helper to format price from cents
export function formatPrice(amountInCents: number, currency: string = 'usd'): string {
	return new Intl.NumberFormat('en-US', {
		style: 'currency',
		currency: currency.toUpperCase(),
	}).format(amountInCents / 100);
}

// Helper to format billing interval
export function formatInterval(interval: string, intervalCount: number = 1): string {
	if (intervalCount === 1) {
		return `per ${interval}`;
	}
	return `every ${intervalCount} ${interval}s`;
}