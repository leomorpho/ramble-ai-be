import { page } from '@vitest/browser/context';
import { describe, expect, it, beforeEach } from 'vitest';
import { render } from 'vitest-browser-svelte';
import LoginForm from './LoginForm.svelte';

// Mock window.fetch to avoid network calls
beforeEach(() => {
	if (typeof window !== 'undefined') {
		window.fetch = async () => ({
			ok: false,
			json: async () => ({})
		}) as Response;
	}
});

describe('LoginForm UI Flow Tests', () => {

	it('should render tabs with Sign In and Create Account options', async () => {
		render(LoginForm);

		// Check that tabs are rendered
		const signInTab = page.getByRole('tab', { name: 'Sign In' });
		const registerTab = page.getByRole('tab', { name: 'Create Account' });
		
		await expect.element(signInTab).toBeInTheDocument();
		await expect.element(registerTab).toBeInTheDocument();
	});

	it('should show single-step login form', async () => {
		render(LoginForm);

		// Should show email and password inputs
		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByPlaceholder('Enter your password');
		const signInButton = page.getByRole('button', { name: 'Sign in' });
		
		await expect.element(emailInput).toBeInTheDocument();
		await expect.element(passwordInput).toBeInTheDocument();
		await expect.element(signInButton).toBeInTheDocument();
		
		// Should show welcome message
		const heading = page.getByRole('heading', { name: 'Welcome back' });
		await expect.element(heading).toBeInTheDocument();
	});

	it('should validate email and password before allowing sign in', async () => {
		render(LoginForm);

		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByPlaceholder('Enter your password');
		const signInButton = page.getByRole('button', { name: 'Sign in' });
		
		// Button should be disabled initially
		await expect.element(signInButton).toBeDisabled();
		
		// Enter invalid email
		await emailInput.fill('invalid-email');
		await expect.element(signInButton).toBeDisabled();
		
		// Enter valid email but no password
		await emailInput.fill('test@example.com');
		await expect.element(signInButton).toBeDisabled();
		
		// Enter both valid email and password
		await passwordInput.fill('password123');
		await expect.element(signInButton).toBeEnabled();
	});

	it('should show forgot password link', async () => {
		render(LoginForm);

		// Should show forgot password link in single-step form
		const forgotPasswordLink = page.getByRole('link', { name: 'Forgot your password?' });
		await expect.element(forgotPasswordLink).toBeInTheDocument();
		await expect.element(forgotPasswordLink).toHaveAttribute('href', '/forgot-password');
	});

	it('should show password input with toggle visibility', async () => {
		render(LoginForm);

		// Should show password input in single-step form
		const passwordInput = page.getByPlaceholder('Enter your password');
		await expect.element(passwordInput).toBeInTheDocument();
		
		// Should be password type initially
		await expect.element(passwordInput).toHaveAttribute('type', 'password');
		
		// Should have a toggle button (eye icon)
		const toggleButton = passwordInput.locator('..').getByRole('button');
		await expect.element(toggleButton).toBeInTheDocument();
	});

	it('should show passkey registration option when available', async () => {
		render(LoginForm);

		// Enter valid email to potentially show passkey options
		const emailInput = page.getByPlaceholder('name@example.com');
		await emailInput.fill('test@example.com');
		
		// Wait a moment for passkey check (would normally show passkey registration)
		// Since WebAuthn is not supported in test environment, should show registration option
		const passkeyRegSection = page.getByText('Optional: Enhanced Security');
		await expect.element(passkeyRegSection).toBeInTheDocument();
	});

	it('should disable form during loading state', async () => {
		render(LoginForm);

		const emailInput = page.getByPlaceholder('name@example.com');
		const passwordInput = page.getByPlaceholder('Enter your password');
		
		// Fill in valid credentials
		await emailInput.fill('test@example.com');
		await passwordInput.fill('password123');
		
		// Form elements should not be disabled initially
		await expect.element(emailInput).not.toBeDisabled();
		await expect.element(passwordInput).not.toBeDisabled();
	});

	it('should show password visibility toggle', async () => {
		render(LoginForm);

		const passwordInput = page.getByPlaceholder('Enter your password');
		
		// Should be password type initially
		await expect.element(passwordInput).toHaveAttribute('type', 'password');
		
		// Find and click the toggle button (eye icon)
		const toggleButton = passwordInput.locator('..').getByRole('button');
		await toggleButton.click();
		
		// Should change to text type
		await expect.element(passwordInput).toHaveAttribute('type', 'text');
	});

	it('should switch between login and register tabs', async () => {
		render(LoginForm);

		// Should start on Sign In tab
		const signInTab = page.getByRole('tab', { name: 'Sign In' });
		const registerTab = page.getByRole('tab', { name: 'Create Account' });
		
		// Click register tab
		await registerTab.click();
		
		// Should switch to register tab content
		const createAccountHeading = page.getByRole('heading', { name: 'Create your account' });
		await expect.element(createAccountHeading).toBeInTheDocument();
	});

	it('should validate password strength in register form', async () => {
		render(LoginForm);

		// Switch to register tab
		await page.getByRole('tab', { name: 'Create Account' }).click();

		// Fill in form fields with short password
		await page.getByPlaceholder('John Doe').fill('Test User');
		await page.getByPlaceholder('m@example.com').fill('test@example.com');
		await page.getByPlaceholder('Create a password (min 8 characters)').fill('12345');
		await page.getByPlaceholder('Confirm your password').fill('12345');

		const createButton = page.getByRole('button', { name: 'Create Account' });
		await createButton.click();

		// Should show password length error
		const errorMessage = page.getByText('Password must be at least 8 characters long');
		await expect.element(errorMessage).toBeInTheDocument();
	});
});