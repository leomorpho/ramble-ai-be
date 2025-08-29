import { startAuthentication, startRegistration } from '@simplewebauthn/browser';
import type { AuthenticationResponseJSON, RegistrationResponseJSON } from '@simplewebauthn/browser';

// Use the same base URL as the rest of the app
const API_BASE = 'http://localhost:8090';

export interface WebAuthnResult {
	success: boolean;
	error?: string;
	token?: string;
	record?: any;
}

/**
 * Register a new passkey for the user
 */
export async function registerPasskey(usernameOrEmail: string): Promise<WebAuthnResult> {
	console.log('=== PASSKEY REGISTRATION START ===', usernameOrEmail);
	try {
		// Get registration options from the server
		console.log('Fetching registration options for:', usernameOrEmail);
		const optionsResponse = await fetch(
			`${API_BASE}/api/webauthn/registration-options?usernameOrEmail=${encodeURIComponent(usernameOrEmail)}`
		);

		if (!optionsResponse.ok) {
			const errorText = await optionsResponse.text();
			console.error('Registration options failed:', optionsResponse.status, errorText);
			throw new Error(`Failed to get registration options: ${errorText}`);
		}

		const optionsJSON = await optionsResponse.json();
		console.log('Registration options received:', optionsJSON);

		// Start the registration process
		let registrationResponse: RegistrationResponseJSON;
		try {
			console.log('Starting WebAuthn registration...');
			registrationResponse = await startRegistration({ optionsJSON });
			console.log('WebAuthn registration completed:', registrationResponse);
		} catch (error) {
			console.error('WebAuthn registration error:', error);
			console.error('Error details:', error.name, error.message, error.stack);
			// User likely cancelled or doesn't have WebAuthn support
			return {
				success: false,
				error: `Registration failed: ${error.message || 'Unknown error'}`
			};
		}

		// Send the registration response to the server
		const verificationResponse = await fetch(`${API_BASE}/api/webauthn/register`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				...registrationResponse,
				usernameOrEmail
			})
		});

		if (!verificationResponse.ok) {
			throw new Error('Registration verification failed');
		}

		const result = await verificationResponse.json();

		return {
			success: true
		};
	} catch (error) {
		console.error('WebAuthn registration error:', error);
		return {
			success: false,
			error: error instanceof Error ? error.message : 'Registration failed'
		};
	}
}

/**
 * Authenticate using a passkey
 */
export async function authenticateWithPasskey(usernameOrEmail: string): Promise<WebAuthnResult> {
	try {
		// Get authentication options from the server
		const optionsResponse = await fetch(
			`${API_BASE}/api/webauthn/login-options?usernameOrEmail=${encodeURIComponent(usernameOrEmail)}`
		);

		if (!optionsResponse.ok) {
			throw new Error('Failed to get authentication options');
		}

		const optionsJSON = await optionsResponse.json();

		// Start the authentication process
		let authenticationResponse: AuthenticationResponseJSON;
		try {
			authenticationResponse = await startAuthentication({ optionsJSON });
		} catch (error) {
			// User likely cancelled or doesn't have WebAuthn support
			return {
				success: false,
				error: 'Authentication cancelled or not supported'
			};
		}

		// Send the authentication response to the server
		const verificationResponse = await fetch(`${API_BASE}/api/webauthn/login`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify({
				...authenticationResponse,
				usernameOrEmail
			})
		});

		if (!verificationResponse.ok) {
			throw new Error('Authentication verification failed');
		}

		const result = await verificationResponse.json();

		return {
			success: true,
			token: result.token,
			record: result.record
		};
	} catch (error) {
		console.error('WebAuthn authentication error:', error);
		return {
			success: false,
			error: error instanceof Error ? error.message : 'Authentication failed'
		};
	}
}

/**
 * Check if WebAuthn is supported in the current browser
 */
export function isWebAuthnSupported(): boolean {
	return typeof navigator !== 'undefined' && 
		   !!(navigator.credentials?.create && navigator.credentials?.get);
}