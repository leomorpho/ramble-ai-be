<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { pb } from '$lib/pocketbase.js';
	import { Mail, RefreshCw } from 'lucide-svelte';

	interface Props {
		userID: string;
		email: string;
		purpose: 'signup_verification' | 'email_change' | 'password_reset';
		onSuccess: () => void;
		onCancel?: () => void;
	}

	let { userID, email, purpose, onSuccess, onCancel }: Props = $props();

	// State
	let otpCode = $state('');
	let isVerifying = $state(false);
	let isSendingOTP = $state(false);
	let error = $state<string | null>(null);
	let success = $state<string | null>(null);
	let timeLeft = $state(600); // 10 minutes in seconds
	let canResend = $state(false);

	// Format time remaining
	let timeFormatted = $derived.by(() => {
		const minutes = Math.floor(timeLeft / 60);
		const seconds = timeLeft % 60;
		return `${minutes}:${seconds.toString().padStart(2, '0')}`;
	});

	// Start countdown timer
	function startTimer() {
		timeLeft = 600; // Reset to 10 minutes
		canResend = false;
		
		const timer = setInterval(() => {
			timeLeft--;
			if (timeLeft <= 0) {
				clearInterval(timer);
				canResend = true;
			}
		}, 1000);
	}

	// Send OTP on component mount
	$effect(() => {
		sendOTP();
	});

	// Send OTP code
	async function sendOTP() {
		isSendingOTP = true;
		error = null;
		success = null;

		try {
			const response = await fetch(`${pb.baseUrl}/send-otp`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					user_id: userID,
					email: email,
					purpose: purpose
				}),
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.message || 'Failed to send OTP');
			}

			success = `Verification code sent to ${email}`;
			startTimer();
		} catch (err: any) {
			error = err.message || 'Failed to send OTP';
		} finally {
			isSendingOTP = false;
		}
	}

	// Verify OTP code
	async function verifyOTP() {
		if (!otpCode || otpCode.length !== 6) {
			error = 'Please enter a valid 6-digit code';
			return;
		}

		isVerifying = true;
		error = null;

		try {
			const response = await fetch(`${pb.baseUrl}/verify-otp`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					user_id: userID,
					otp_code: otpCode,
					purpose: purpose
				}),
			});

			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.message || 'Invalid verification code');
			}

			success = 'Verification successful!';
			setTimeout(onSuccess, 1000); // Call success callback after showing message
		} catch (err: any) {
			error = err.message || 'Verification failed';
		} finally {
			isVerifying = false;
		}
	}

	// Format OTP input (add spaces for readability)
	function formatOTPInput(value: string) {
		// Remove all non-digits and limit to 6 digits
		const cleaned = value.replace(/\D/g, '').slice(0, 6);
		// Add spaces every 3 digits for readability
		return cleaned.replace(/(\d{3})(\d{1,3})/, '$1 $2');
	}

	// Handle OTP input
	function handleOTPInput(event: Event) {
		const target = event.target as HTMLInputElement;
		const formatted = formatOTPInput(target.value);
		target.value = formatted;
		otpCode = formatted.replace(/\s/g, ''); // Store without spaces
	}

	// Get title based on purpose
	let title = $derived.by(() => {
		switch (purpose) {
			case 'signup_verification':
				return 'Verify Your Email';
			case 'email_change':
				return 'Verify New Email';
			case 'password_reset':
				return 'Reset Password';
			default:
				return 'Verify Code';
		}
	});

	// Get description based on purpose
	let description = $derived.by(() => {
		switch (purpose) {
			case 'signup_verification':
				return 'Enter the 6-digit verification code sent to your email to complete your account setup.';
			case 'email_change':
				return 'Enter the 6-digit verification code sent to your new email address.';
			case 'password_reset':
				return 'Enter the 6-digit verification code sent to your email to reset your password.';
			default:
				return 'Enter the verification code sent to your email.';
		}
	});
</script>

<div class="mx-auto max-w-md">
	<div class="bg-card rounded-xl border border-border p-6 shadow-sm">
		<!-- Header -->
		<div class="text-center mb-6">
			<div class="w-12 h-12 bg-primary/10 rounded-full flex items-center justify-center mx-auto mb-4">
				<Mail class="w-6 h-6 text-primary" />
			</div>
			<h2 class="text-xl font-semibold text-foreground mb-2">{title}</h2>
			<p class="text-sm text-muted-foreground">{description}</p>
		</div>

		<!-- Success Message -->
		{#if success}
			<div class="mb-4 p-3 bg-green-50 dark:bg-green-950/50 border border-green-200 dark:border-green-800 rounded-lg">
				<p class="text-sm text-green-600 dark:text-green-400">{success}</p>
			</div>
		{/if}

		<!-- Error Message -->
		{#if error}
			<div class="mb-4 p-3 bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-800 rounded-lg">
				<p class="text-sm text-red-600 dark:text-red-400">{error}</p>
			</div>
		{/if}

		<!-- OTP Input -->
		<form onsubmit={(e) => { e.preventDefault(); verifyOTP(); }} class="space-y-4">
			<div class="space-y-2">
				<Label for="otp-code">Verification Code</Label>
				<Input
					id="otp-code"
					type="text"
					placeholder="123 456"
					maxlength="7"
					oninput={handleOTPInput}
					disabled={isVerifying}
					class="text-center text-lg tracking-widest font-mono"
					required
				/>
				<p class="text-xs text-muted-foreground text-center">
					Enter the 6-digit code sent to {email}
				</p>
			</div>

			<!-- Timer -->
			{#if !canResend}
				<div class="text-center">
					<p class="text-sm text-muted-foreground">
						Code expires in {timeFormatted}
					</p>
				</div>
			{/if}

			<!-- Action Buttons -->
			<div class="space-y-3">
				<Button
					type="submit"
					class="w-full"
					disabled={isVerifying || otpCode.length !== 6}
				>
					{isVerifying ? 'Verifying...' : 'Verify Code'}
				</Button>

				<!-- Resend Button -->
				{#if canResend}
					<Button
						type="button"
						variant="outline"
						class="w-full"
						onclick={sendOTP}
						disabled={isSendingOTP}
					>
						{#if isSendingOTP}
							<RefreshCw class="w-4 h-4 mr-2 animate-spin" />
							Sending...
						{:else}
							Resend Code
						{/if}
					</Button>
				{/if}

				<!-- Cancel Button -->
				{#if onCancel}
					<Button
						type="button"
						variant="ghost"
						class="w-full"
						onclick={onCancel}
						disabled={isVerifying || isSendingOTP}
					>
						Cancel
					</Button>
				{/if}
			</div>
		</form>

		<!-- Help Text -->
		<div class="mt-6 text-center">
			<p class="text-xs text-muted-foreground">
				Didn't receive the code? Check your spam folder or wait for the timer to resend.
			</p>
		</div>
	</div>
</div>