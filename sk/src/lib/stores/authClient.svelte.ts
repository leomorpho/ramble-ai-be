import { pb } from '$lib/pocketbase.js';
import type { AuthModel } from 'pocketbase';
import { browser } from '$app/environment';

class ClientAuthStore {
	#user = $state<AuthModel | null>(null);
	#isValid = $state(false);
	#isLoading = $state(true);
	#initialized = $state(false);

	constructor() {
		if (browser) {
			// Initialize immediately on browser
			this.#initializeAuth();
		}
	}

	#initializeAuth() {
		// PocketBase automatically loads from localStorage on instantiation
		// We just need to sync our reactive state
		this.#user = pb.authStore.model;
		this.#isValid = pb.authStore.isValid;
		this.#isLoading = false;
		this.#initialized = true;

		// Listen for PocketBase auth changes
		pb.authStore.onChange(() => {
			this.#user = pb.authStore.model;
			this.#isValid = pb.authStore.isValid;
		});

		// Try to refresh auth if we have a token
		if (pb.authStore.isValid) {
			this.#refreshAuth();
		} else if (pb.authStore.token) {
			// Clear any invalid tokens that might be stored
			console.log('🧹 Clearing invalid stored token');
			pb.authStore.clear();
			this.syncState();
		}
	}

	async #refreshAuth() {
		try {
			if (pb.authStore.isValid) {
				await pb.collection('users').authRefresh();
				console.log('🔄 Auth refresh successful');
			}
		} catch (error) {
			console.error('Auth refresh failed:', error);
			// Clear invalid tokens to prevent repeated refresh attempts
			if (error.status === 401 || error.status === 403) {
				console.log('🚫 Token invalid, clearing auth state');
				pb.authStore.clear();
				this.syncState();
			}
		}
	}

	get user() {
		// Ensure initialization if not done yet
		if (browser && !this.#initialized) {
			this.#initializeAuth();
		}
		return this.#user;
	}

	get isLoggedIn() {
		// Ensure initialization if not done yet
		if (browser && !this.#initialized) {
			this.#initializeAuth();
		}
		return this.#user !== null && this.#isValid;
	}

	get isLoading() {
		return this.#isLoading;
	}

	get initialized() {
		return this.#initialized;
	}

	async login(email: string, password: string) {
		try {
			console.log('🔐 Starting login...');
			const authData = await pb.collection('users').authWithPassword(email, password);
			console.log('🔐 Login successful, PocketBase state:', {
				isValid: pb.authStore.isValid,
				model: pb.authStore.model?.email
			});

			// Manual sync to ensure reactive state updates immediately
			this.syncState();
			console.log('🔐 After manual sync, auth store state:', {
				user: this.#user?.email,
				isValid: this.#isValid,
				isLoggedIn: this.isLoggedIn
			});

			// PocketBase automatically saves to localStorage and triggers onChange
			return { success: true, user: authData.record };
		} catch (error) {
			console.error('Login error:', error);
			return {
				success: false,
				error: error instanceof Error ? error.message : 'Login failed'
			};
		}
	}

	async signup(email: string, password: string, passwordConfirm: string, name?: string) {
		try {
			const data = {
				email,
				password,
				passwordConfirm,
				...(name && { name })
			};

			await pb.collection('users').create(data);

			// Auto-login after signup
			const authData = await pb.collection('users').authWithPassword(email, password);

			// Manual sync to ensure reactive state updates immediately
			this.syncState();

			// PocketBase automatically saves to localStorage and triggers onChange
			return { success: true, user: authData.record };
		} catch (error) {
			console.error('Signup error:', error);
			return {
				success: false,
				error: error instanceof Error ? error.message : 'Signup failed'
			};
		}
	}

	logout() {
		// PocketBase automatically clears localStorage and triggers onChange
		pb.authStore.clear();

		// Manual sync in case onChange doesn't fire immediately
		this.syncState();

		// Let route protection logic handle redirects appropriately
		// No forced redirect needed - protected routes will redirect to /login
		// and public routes will stay where they are
	}

	// Set auth data directly (for WebAuthn and other external auth)
	setAuthData(token: string, record: AuthModel) {
		if (browser) {
			// Set the auth token and model in PocketBase
			pb.authStore.save(token, record);
			
			// Manual sync to ensure reactive state updates immediately
			this.syncState();
		}
	}

	// Manual method to sync state (for edge cases)
	syncState() {
		if (browser) {
			this.#user = pb.authStore.model;
			this.#isValid = pb.authStore.isValid;
		}
	}
}

export const authStore = new ClientAuthStore();
