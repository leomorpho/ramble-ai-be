import { browser } from '$app/environment';
import { pb } from '$lib/pocketbase';
import { authStore } from './authClient.svelte';

interface SubscriptionPlan {
	id: string;
	name: string;
	price_cents: number;
	currency: string;
	billing_interval: 'free' | 'month' | 'year';
	hours_per_month: number;
	stripe_price_id?: string;
	stripe_product_id?: string;
	is_active: boolean;
	display_order: number;
	features: string[];
}

interface UserSubscription {
	id: string;
	user_id: string;
	plan_id: string;
	stripe_subscription_id?: string;
	status: 'active' | 'cancelled' | 'past_due' | 'trialing';
	current_period_start: string;
	current_period_end: string;
	cancel_at_period_end: boolean;
	canceled_at?: string;
	trial_end?: string;
}

interface UsageInfo {
	user_id: string;
	current_plan_id: string;
	plan_name: string;
	hours_limit: number;
	hours_used: number;
	hours_remaining: number;
	usage_percentage: number;
	files_processed: number;
	period_start: string;
	period_end: string;
	is_over_limit: boolean;
	can_process_more: boolean;
	subscription_status: string;
	billing_interval: string;
}

interface PlansByTier {
	[tier: string]: SubscriptionPlan[];
}

class SubscriptionStore {
	#plans = $state<SubscriptionPlan[]>([]);
	#plansByTier = $state<PlansByTier>({});
	#userSubscription = $state<UserSubscription | null>(null);
	#currentPlan = $state<SubscriptionPlan | null>(null);
	#usage = $state<UsageInfo | null>(null);
	#isLoading = $state(false);
	#isUsageLoading = $state(false);
	#initialized = $state(false);

	constructor() {
		// Initialize will be called from components
	}

	// Initialize the store with effect tracking - call this from components
	initialize() {
		if (browser && !this.#initialized) {
			this.#initialized = true;
			
			// Load subscription plans immediately (public data)
			this.loadPlans();
			
			// Watch auth state changes to load/clear user-specific data
			$effect(() => {
				if (authStore.isLoggedIn) {
					this.loadUserData();
				} else {
					this.clearUserData();
				}
			});
		}
	}

	// Getters
	get plans() {
		return this.#plans;
	}

	get plansByTier() {
		return this.#plansByTier;
	}

	get userSubscription() {
		return this.#userSubscription;
	}

	get currentPlan() {
		return this.#currentPlan;
	}

	get usage() {
		return this.#usage;
	}

	get isLoading() {
		return this.#isLoading;
	}

	get isUsageLoading() {
		return this.#isUsageLoading;
	}

	get isSubscribed() {
		return this.#userSubscription?.status === 'active' || 
		       this.#userSubscription?.status === 'trialing';
	}

	get subscriptionStatus() {
		return this.#userSubscription?.status || 'none';
	}

	get canProcessFiles() {
		return this.#usage?.can_process_more ?? false;
	}

	get usageWarning() {
		if (!this.#usage) return null;
		
		if (this.#usage.usage_percentage >= 90) {
			return {
				type: 'danger',
				message: `You've used ${this.#usage.usage_percentage.toFixed(1)}% of your monthly quota. Consider upgrading your plan.`
			};
		}
		
		if (this.#usage.usage_percentage >= 75) {
			return {
				type: 'warning', 
				message: `You've used ${this.#usage.hours_used.toFixed(1)} hours of your ${this.#usage.hours_limit} hour monthly quota.`
			};
		}

		return null;
	}

	// Load subscription plans (public data)
	async loadPlans() {
		if (!browser) return;

		this.#isLoading = true;

		try {
			// Use PocketBase SDK to fetch subscription plans
			const plans = await pb.collection('subscription_plans').getFullList<SubscriptionPlan>({
				filter: 'is_active = true',
				sort: '+display_order'
			});
			
			this.#plans = plans;
			
			// Group plans by tier
			const plansByTier: PlansByTier = {};
			for (const plan of plans) {
				let tier = 'free';
				if (plan.billing_interval === 'free') {
					tier = 'free';
				} else if (plan.name.toLowerCase().includes('basic')) {
					tier = 'basic';
				} else if (plan.name.toLowerCase().includes('pro')) {
					tier = 'pro';
				}
				
				if (!plansByTier[tier]) {
					plansByTier[tier] = [];
				}
				plansByTier[tier].push(plan);
			}
			
			this.#plansByTier = plansByTier;
		} catch (error) {
			console.debug('Subscription plans not available yet:', error);
			this.#plans = [];
			this.#plansByTier = {};
		} finally {
			this.#isLoading = false;
		}
	}

	// Load user-specific subscription data
	async loadUserData() {
		if (!browser || !authStore.user) return;

		this.#isLoading = true;

		try {
			// Get user's subscription
			const subscriptions = await pb.collection('user_subscriptions').getFullList<UserSubscription>({
				filter: `user_id = "${authStore.user.id}"`,
				sort: '-created'
			});
			
			if (subscriptions.length > 0) {
				this.#userSubscription = subscriptions[0];
				
				// Get the plan details
				if (this.#userSubscription.plan_id) {
					const plan = await pb.collection('subscription_plans').getOne<SubscriptionPlan>(
						this.#userSubscription.plan_id
					);
					this.#currentPlan = plan;
				}
				
				// Load usage info
				await this.loadUsage();
			} else {
				this.clearUserData();
			}
		} catch (error: any) {
			console.debug('No subscription found for user:', error);
			this.clearUserData();
		} finally {
			this.#isLoading = false;
		}
	}

	// Load usage statistics
	async loadUsage() {
		if (!browser || !authStore.user) return;

		this.#isUsageLoading = true;

		try {
			// Get current month's usage
			const now = new Date();
			const yearMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
			
			const usageRecords = await pb.collection('monthly_usage').getFullList({
				filter: `user_id = "${authStore.user.id}" && year_month = "${yearMonth}"`,
				sort: '-last_processing_date'
			});
			
			if (usageRecords.length > 0 && this.#currentPlan) {
				const usage = usageRecords[0];
				// Calculate usage info similar to backend
				const hoursUsed = usage.hours_used || 0;
				const hoursLimit = this.#currentPlan.hours_per_month || 1;
				const hoursRemaining = Math.max(0, hoursLimit - hoursUsed);
				const usagePercentage = (hoursUsed / hoursLimit) * 100;
				
				this.#usage = {
					user_id: authStore.user.id,
					current_plan_id: this.#currentPlan.id,
					plan_name: this.#currentPlan.name,
					hours_limit: hoursLimit,
					hours_used: hoursUsed,
					hours_remaining: hoursRemaining,
					usage_percentage: usagePercentage,
					files_processed: usage.files_processed || 0,
					period_start: `${yearMonth}-01`,
					period_end: new Date(now.getFullYear(), now.getMonth() + 1, 0).toISOString().split('T')[0],
					is_over_limit: hoursUsed >= hoursLimit,
					can_process_more: hoursUsed < hoursLimit,
					subscription_status: this.#userSubscription?.status || 'none',
					billing_interval: this.#currentPlan.billing_interval
				};
			} else if (this.#currentPlan) {
				// No usage records yet, create default
				const hoursLimit = this.#currentPlan.hours_per_month || 1;
				this.#usage = {
					user_id: authStore.user.id,
					current_plan_id: this.#currentPlan.id,
					plan_name: this.#currentPlan.name,
					hours_limit: hoursLimit,
					hours_used: 0,
					hours_remaining: hoursLimit,
					usage_percentage: 0,
					files_processed: 0,
					period_start: `${yearMonth}-01`,
					period_end: new Date(now.getFullYear(), now.getMonth() + 1, 0).toISOString().split('T')[0],
					is_over_limit: false,
					can_process_more: true,
					subscription_status: this.#userSubscription?.status || 'none',
					billing_interval: this.#currentPlan.billing_interval
				};
			} else {
				this.#usage = null;
			}
		} catch (error) {
			console.debug('Failed to load usage stats:', error);
			this.#usage = null;
		} finally {
			this.#isUsageLoading = false;
		}
	}

	// Clear user-specific data
	clearUserData() {
		this.#userSubscription = null;
		this.#currentPlan = null;
		this.#usage = null;
	}

	// Get plan by ID
	getPlan(planId: string): SubscriptionPlan | undefined {
		return this.#plans.find(plan => plan.id === planId);
	}

	// Get plans for a specific tier
	getPlansForTier(tier: string): SubscriptionPlan[] {
		return this.#plansByTier[tier] || [];
	}

	// Get available upgrade options
	async getUpgradeOptions(): Promise<SubscriptionPlan[]> {
		if (!browser || !authStore.user || !this.#currentPlan) return [];

		try {
			// Get plans with higher hours limit than current plan
			const currentHours = this.#currentPlan.hours_per_month || 0;
			
			const upgrades = await pb.collection('subscription_plans').getFullList<SubscriptionPlan>({
				filter: `is_active = true && hours_per_month > ${currentHours}`,
				sort: '+display_order'
			});

			return upgrades;
		} catch (error) {
			console.debug('Failed to load upgrade options:', error);
			return [];
		}
	}

	// Check if a plan is the user's current plan
	isCurrentPlan(planId: string): boolean {
		return this.#currentPlan?.id === planId;
	}

	// Check if user has access to features
	hasAccess(): boolean {
		return this.isSubscribed;
	}

	// Get formatted price
	formatPrice(priceCents: number, currency = 'usd'): string {
		// Handle the 1 cent workaround for free plans
		if (priceCents <= 1) {
			return 'Free';
		}
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: currency.toUpperCase(),
		}).format(priceCents / 100);
	}

	// Calculate savings for yearly plans
	calculateYearlySavings(monthlyPrice: number, yearlyPrice: number): number {
		const monthlyTotal = monthlyPrice * 12;
		return Math.round(((monthlyTotal - yearlyPrice) / monthlyTotal) * 100);
	}

	// Refresh all data
	async refresh() {
		await Promise.all([
			this.loadPlans(),
			authStore.isLoggedIn ? this.loadUserData() : Promise.resolve()
		]);
	}

	// Refresh only usage data
	async refreshUsage() {
		if (authStore.isLoggedIn) {
			await this.loadUsage();
		}
	}
}

export const subscriptionStore = new SubscriptionStore();