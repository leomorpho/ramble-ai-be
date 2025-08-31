import { browser } from '$app/environment';
import { authStore } from './stores/authClient.svelte.js';
import { pb } from './pocketbase.js';

// Payment method status interface
interface PaymentMethodStatus {
	has_valid_payment_method: boolean;
	payment_methods_count: number;
	default_payment_method?: string;
	last_used?: string;
	requires_update: boolean;
	can_process_payments: boolean;
}

// Helper to check if user has valid payment methods
export async function checkPaymentMethods(): Promise<PaymentMethodStatus> {
	if (!browser) throw new Error('Not running in browser');
	
	const user = authStore.user;
	if (!user) {
		throw new Error('User must be logged in to check payment methods');
	}

	const response = await fetch(`${pb.baseUrl}/api/payment/check-method`, {
		method: 'GET',
		headers: {
			'Authorization': `Bearer ${pb.authStore.token}`,
			'Content-Type': 'application/json',
		},
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.error || 'Failed to check payment methods');
	}

	return await response.json();
}

// Helper to create checkout session for a subscription plan with hybrid approach
export async function createCheckoutSession(planId: string) {
	if (!browser) return null;
	
	const user = authStore.user;
	if (!user) {
		throw new Error('User must be logged in to create checkout session');
	}

	// First check if this is a free plan
	const response = await fetch(`${pb.baseUrl}/api/payment/checkout`, {
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

	const data = await response.json();
	
	// Check if this is a free plan
	if (data.is_free) {
		// For free plans, use the change-plan endpoint instead
		return changePlan(planId);
	}

	// For paid plans, check if user has valid payment methods for hybrid approach
	try {
		const paymentStatus = await checkPaymentMethods();
		
		if (paymentStatus.has_valid_payment_method && paymentStatus.can_process_payments) {
			// User has valid payment methods - offer direct plan change option
			// We'll return a special flag to indicate hybrid approach is available
			return {
				checkout_url: data.url,
				can_use_direct_change: true,
				payment_status: paymentStatus
			};
		}
	} catch (error) {
		// If payment method check fails, fall back to checkout flow
		console.warn('Payment method check failed, falling back to checkout:', error);
	}
	
	// Redirect to payment checkout (default behavior)
	if (data.url) {
		window.location.href = data.url;
	}
	
	return data.url;
}

// Helper to create billing portal link
export async function createPortalLink() {
	if (!browser) return null;
	
	const user = authStore.user;
	if (!user) {
		throw new Error('User must be logged in to access billing portal');
	}

	const response = await fetch(`${pb.baseUrl}/api/payment/portal`, {
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
	
	// Redirect to billing portal
	if (url) {
		window.location.href = url;
	}
	
	return url;
}

// Helper to cancel subscription (preserves benefits until period end)
export async function cancelSubscription() {
	if (!browser) return null;
	
	const user = authStore.user;
	if (!user) {
		throw new Error('User must be logged in to cancel subscription');
	}

	try {
		const result = await pb.send('/api/subscription/cancel', {
			method: 'POST'
		});
		return result;
	} catch (error: any) {
		throw new Error(error?.message || 'Failed to cancel subscription');
	}
}

// Helper to change plan directly with confirmation (for existing customers with valid payment methods)
export async function changePlanDirect(planId: string) {
	if (!browser) return null;
	
	const user = authStore.user;
	if (!user) {
		throw new Error('User must be logged in to change plan');
	}

	// Use the enhanced change-plan endpoint that handles upgrade/downgrade logic
	const result = await changePlan(planId);
	return result;
}

// Helper to change plan directly (for paid plan upgrades and downgrades)
export async function changePlan(planId: string) {
	if (!browser) return null;
	
	const user = authStore.user;
	if (!user) {
		throw new Error('User must be logged in to change plan');
	}

	const response = await fetch(`${pb.baseUrl}/api/payment/change-plan`, {
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
		throw new Error(error.error || 'Failed to change plan');
	}

	const result = await response.json();
	return result;
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