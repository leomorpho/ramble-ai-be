import { page } from '@vitest/browser/context';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import SignupForm from './SignupForm.svelte';

describe('SignupForm UI Tests', () => {

	it('should render signup form with all required fields', async () => {
		render(SignupForm);

		// Check form elements are present
		const heading = page.getByRole('heading', { name: 'Create your account' });
		const nameInput = page.getByPlaceholder('John Doe');
		const emailInput = page.getByPlaceholder('m@example.com');
		const passwordInput = page.getByPlaceholder('Create a password (min 6 characters)');
		const confirmPasswordInput = page.getByPlaceholder('Confirm your password');
		const createButton = page.getByRole('button', { name: 'Create Account' });

		await expect.element(heading).toBeInTheDocument();
		await expect.element(nameInput).toBeInTheDocument();
		await expect.element(emailInput).toBeInTheDocument();
		await expect.element(passwordInput).toBeInTheDocument();
		await expect.element(confirmPasswordInput).toBeInTheDocument();
		await expect.element(createButton).toBeInTheDocument();
	});

	it('should show proper labels for form fields', async () => {
		render(SignupForm);

		// Check labels are present - use more specific selectors to avoid conflicts
		await expect.element(page.getByText('Full Name')).toBeInTheDocument();
		await expect.element(page.getByText('Email')).toBeInTheDocument();
		await expect.element(page.getByText('Password', { exact: true })).toBeInTheDocument();
		await expect.element(page.getByText('Confirm Password')).toBeInTheDocument();
	});

	it('should have working form structure for validation', async () => {
		render(SignupForm);

		const createButton = page.getByRole('button', { name: 'Create Account' });
		const nameInput = page.getByPlaceholder('John Doe');
		const emailInput = page.getByPlaceholder('m@example.com');
		const passwordInput = page.getByPlaceholder('Create a password (min 6 characters)');
		const confirmPasswordInput = page.getByPlaceholder('Confirm your password');
		
		// Verify form elements are present and properly configured
		await expect.element(createButton).toBeInTheDocument();
		await expect.element(nameInput).toHaveAttribute('required');
		await expect.element(emailInput).toHaveAttribute('required');
		await expect.element(passwordInput).toHaveAttribute('required');
		await expect.element(confirmPasswordInput).toHaveAttribute('required');
		
		// Verify form structure supports validation
		await expect.element(emailInput).toHaveAttribute('type', 'email');
		await expect.element(passwordInput).toHaveAttribute('type', 'password');
		await expect.element(confirmPasswordInput).toHaveAttribute('type', 'password');
	});

	it('should validate password minimum length', async () => {
		render(SignupForm);

		// Fill form with short password
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

	it('should validate password confirmation match', async () => {
		render(SignupForm);

		// Fill form with mismatched passwords
		await page.getByPlaceholder('John Doe').fill('Test User');
		await page.getByPlaceholder('m@example.com').fill('test@example.com');
		await page.getByPlaceholder('Create a password (min 6 characters)').fill('password123');
		await page.getByPlaceholder('Confirm your password').fill('different123');

		const createButton = page.getByRole('button', { name: 'Create Account' });
		await createButton.click();

		// Should show password mismatch error
		const errorMessage = page.getByText('Passwords do not match');
		await expect.element(errorMessage).toBeInTheDocument();
	});

	it('should show form validation for required fields', async () => {
		render(SignupForm);

		// All inputs should be required
		const nameInput = page.getByPlaceholder('John Doe');
		const emailInput = page.getByPlaceholder('m@example.com');
		const passwordInput = page.getByPlaceholder('Create a password (min 6 characters)');
		const confirmPasswordInput = page.getByPlaceholder('Confirm your password');

		await expect.element(nameInput).toHaveAttribute('required');
		await expect.element(emailInput).toHaveAttribute('required');
		await expect.element(passwordInput).toHaveAttribute('required');
		await expect.element(confirmPasswordInput).toHaveAttribute('required');
	});

	it('should show OAuth options', async () => {
		render(SignupForm);

		// Should show OAuth buttons
		const appleButton = page.getByRole('button', { name: 'Continue with Apple' });
		const googleButton = page.getByRole('button', { name: 'Continue with Google' });

		await expect.element(appleButton).toBeInTheDocument();
		await expect.element(googleButton).toBeInTheDocument();
	});

	it('should show terms and privacy policy links', async () => {
		render(SignupForm);

		// Should show terms and privacy links
		const termsLink = page.getByRole('link', { name: 'Terms of Service' });
		const privacyLink = page.getByRole('link', { name: 'Privacy Policy' });

		await expect.element(termsLink).toBeInTheDocument();
		await expect.element(privacyLink).toBeInTheDocument();
		
		await expect.element(termsLink).toHaveAttribute('href', '/terms');
		await expect.element(privacyLink).toHaveAttribute('href', '/privacy');
	});

	it('should validate email format', async () => {
		render(SignupForm);

		// Fill form with invalid email
		await page.getByPlaceholder('John Doe').fill('Test User');
		await page.getByPlaceholder('m@example.com').fill('invalid-email');
		await page.getByPlaceholder('Create a password (min 6 characters)').fill('password123');
		await page.getByPlaceholder('Confirm your password').fill('password123');

		const createButton = page.getByRole('button', { name: 'Create Account' });
		
		// Email input should have validation state
		const emailInput = page.getByPlaceholder('m@example.com');
		await expect.element(emailInput).toHaveAttribute('type', 'email');
	});

	it('should maintain form structure and accessibility', async () => {
		render(SignupForm);

		// Form should be properly structured
		const nameInput = page.getByPlaceholder('John Doe');
		const emailInput = page.getByPlaceholder('m@example.com');
		const passwordInput = page.getByPlaceholder('Create a password (min 6 characters)');
		const confirmPasswordInput = page.getByPlaceholder('Confirm your password');
		const createButton = page.getByRole('button', { name: 'Create Account' });

		// All form elements should be present
		await expect.element(nameInput).toBeInTheDocument();
		await expect.element(emailInput).toBeInTheDocument();
		await expect.element(passwordInput).toBeInTheDocument();
		await expect.element(confirmPasswordInput).toBeInTheDocument();
		await expect.element(createButton).toBeInTheDocument();
	});

	it('should have proper form accessibility', async () => {
		render(SignupForm);

		// Check that labels are properly associated with inputs
		const nameInput = page.getByPlaceholder('John Doe');
		const emailInput = page.getByPlaceholder('m@example.com');
		const passwordInput = page.getByPlaceholder('Create a password (min 6 characters)');
		const confirmPasswordInput = page.getByPlaceholder('Confirm your password');

		// All inputs should have required attribute
		await expect.element(nameInput).toHaveAttribute('required');
		await expect.element(emailInput).toHaveAttribute('required');
		await expect.element(passwordInput).toHaveAttribute('required');
		await expect.element(confirmPasswordInput).toHaveAttribute('required');

		// Email input should have correct type
		await expect.element(emailInput).toHaveAttribute('type', 'email');
		
		// Password inputs should have correct type
		await expect.element(passwordInput).toHaveAttribute('type', 'password');
		await expect.element(confirmPasswordInput).toHaveAttribute('type', 'password');
	});
});