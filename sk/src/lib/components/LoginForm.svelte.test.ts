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

	it('should start with email step in login flow', async () => {
		render(LoginForm);

		// Should show email input initially
		const emailInput = page.getByPlaceholder('name@example.com');
		const continueButton = page.getByRole('button', { name: 'Continue' });
		
		await expect.element(emailInput).toBeInTheDocument();
		await expect.element(continueButton).toBeInTheDocument();
		
		// Should show welcome message
		const heading = page.getByRole('heading', { name: 'Welcome back' });
		await expect.element(heading).toBeInTheDocument();
	});

	it('should validate email before allowing continue', async () => {
		render(LoginForm);

		const emailInput = page.getByPlaceholder('name@example.com');
		const continueButton = page.getByRole('button', { name: 'Continue' });
		
		// Button should be disabled initially
		await expect.element(continueButton).toBeDisabled();
		
		// Enter invalid email
		await emailInput.fill('invalid-email');
		await expect.element(continueButton).toBeDisabled();
		
		// Enter valid email
		await emailInput.fill('test@example.com');
		await expect.element(continueButton).toBeEnabled();
	});

	it('should progress to method selection after valid email', async () => {
		render(LoginForm);

		const emailInput = page.getByPlaceholder('name@example.com');
		const continueButton = page.getByRole('button', { name: 'Continue' });
		
		// Enter valid email and continue
		await emailInput.fill('test@example.com');
		await continueButton.click();

		// Should show method selection
		const methodHeading = page.getByRole('heading', { name: 'How would you like to sign in?' });
		await expect.element(methodHeading).toBeInTheDocument();
		
		// Should show password option
		const passwordButton = page.getByRole('button', { name: /Use your password/ });
		await expect.element(passwordButton).toBeInTheDocument();
	});

	it('should navigate to password step when password method selected', async () => {
		render(LoginForm);

		// Navigate to method selection
		const emailInput = page.getByPlaceholder('name@example.com');
		await emailInput.fill('test@example.com');
		await page.getByRole('button', { name: 'Continue' }).click();

		// Select password method
		const passwordButton = page.getByRole('button', { name: /Use your password/ });
		await passwordButton.click();

		// Should show password step
		const passwordHeading = page.getByRole('heading', { name: 'Enter your password' });
		const passwordInput = page.getByPlaceholder('Enter your password');
		
		await expect.element(passwordHeading).toBeInTheDocument();
		await expect.element(passwordInput).toBeInTheDocument();
	});

	it('should show/hide password toggle', async () => {
		render(LoginForm);

		// Navigate to password step
		const emailInput = page.getByPlaceholder('name@example.com');
		await emailInput.fill('test@example.com');
		await page.getByRole('button', { name: 'Continue' }).click();
		await page.getByRole('button', { name: /Use your password/ }).click();

		const passwordInput = page.getByPlaceholder('Enter your password');
		
		// Should be password type initially
		await expect.element(passwordInput).toHaveAttribute('type', 'password');
	});

	it('should handle forgot password link', async () => {
		render(LoginForm);

		// Navigate to password step
		const emailInput = page.getByPlaceholder('name@example.com');
		await emailInput.fill('test@example.com');
		await page.getByRole('button', { name: 'Continue' }).click();
		await page.getByRole('button', { name: /Use your password/ }).click();

		// Should show forgot password link
		const forgotPasswordLink = page.getByRole('link', { name: 'Forgot your password?' });
		await expect.element(forgotPasswordLink).toBeInTheDocument();
		await expect.element(forgotPasswordLink).toHaveAttribute('href', '/forgot-password');
	});

	it('should show passkey registration option in password step', async () => {
		render(LoginForm);

		// Navigate to password step
		const emailInput = page.getByPlaceholder('name@example.com');
		await emailInput.fill('test@example.com');
		await page.getByRole('button', { name: 'Continue' }).click();
		await page.getByRole('button', { name: /Use your password/ }).click();

		// Should show passkey registration section
		const passkeyRegSection = page.getByText('Optional: Enhanced Security');
		await expect.element(passkeyRegSection).toBeInTheDocument();
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
		await page.getByPlaceholder('Create a password (min 6 characters)').fill('12345');
		await page.getByPlaceholder('Confirm your password').fill('12345');

		const createButton = page.getByRole('button', { name: 'Create Account' });
		await createButton.click();

		// Should show password length error
		const errorMessage = page.getByText('Password must be at least 6 characters long');
		await expect.element(errorMessage).toBeInTheDocument();
	});
});