<script lang="ts">
	import { Label } from '$lib/components/ui/label/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Edit3, Shield } from 'lucide-svelte';
	import { pb } from '$lib/pocketbase.js';
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import OTPVerification from '../OTPVerification.svelte';

	// Personal account editing state
	let isEditingProfile = $state(false);
	let editPersonalName = $state(authStore.user?.name || '');
	let editPersonalEmail = $state(authStore.user?.email || '');
	let isSavingPersonal = $state(false);
	let personalEditError = $state<string | null>(null);
	let personalEditSuccess = $state<string | null>(null);
	
	// Email verification state
	let showEmailVerification = $state(false);

	// Update edit values when user data changes
	$effect(() => {
		if (authStore.user) {
			editPersonalName = authStore.user.name || '';
			editPersonalEmail = authStore.user.email || '';
		}
	});

	// Handle personal account save
	async function handleSavePersonal() {
		if (!editPersonalName.trim()) {
			personalEditError = 'Name is required';
			return;
		}

		if (!editPersonalEmail.trim()) {
			personalEditError = 'Email is required';
			return;
		}

		isSavingPersonal = true;
		personalEditError = null;
		personalEditSuccess = null;

		try {
			const currentEmail = authStore.user!.email;
			const newEmail = editPersonalEmail.trim();
			
			// If email is changing, we need to request email change verification
			if (currentEmail !== newEmail) {
				// Request email change - this will send verification to new email
				await pb.collection('users').requestEmailChange(newEmail);
				
				// Update only the name for now - email will be updated after verification
				await pb.collection('users').update(authStore.user!.id, {
					name: editPersonalName.trim()
				});
				
				// Show success message about email verification
				personalEditSuccess = `A verification email has been sent to ${newEmail}. Please check your email and click the verification link to complete the email change.`;
				console.log('Email change verification sent to:', newEmail);
				
				// Reset email field to current value since change is pending
				editPersonalEmail = currentEmail;
			} else {
				// Only name is changing
				await pb.collection('users').update(authStore.user!.id, {
					name: editPersonalName.trim()
				});
			}

			// Update auth store
			authStore.syncState();
			isEditingProfile = false;
		} catch (error: any) {
			console.error('Failed to update profile:', error);
			personalEditError = error.message || 'Failed to update profile';
		} finally {
			isSavingPersonal = false;
		}
	}

	// Handle cancel personal edit
	function handleCancelPersonalEdit() {
		isEditingProfile = false;
		editPersonalName = authStore.user?.name || '';
		editPersonalEmail = authStore.user?.email || '';
		personalEditError = null;
		personalEditSuccess = null;
	}

	// Handle email verification success
	async function handleEmailVerificationSuccess() {
		showEmailVerification = false;
		
		// Fetch the updated user record from the server to get the latest verification status
		try {
			if (authStore.user) {
				const updatedUser = await pb.collection('users').getOne(authStore.user.id);
				// Update PocketBase's auth store with the fresh user data
				pb.authStore.save(pb.authStore.token, updatedUser);
				// Sync our reactive store
				authStore.syncState();
			}
		} catch (error) {
			console.error('Failed to refresh user data:', error);
		}
		
		personalEditSuccess = 'Email verified successfully!';
	}

	// Handle email verification cancel
	function handleEmailVerificationCancel() {
		showEmailVerification = false;
	}

	// Start email verification process
	function startEmailVerification() {
		showEmailVerification = true;
	}
</script>

<div class="bg-card rounded-xl border border-border p-6 shadow-sm">
	{#if showEmailVerification}
		<!-- Email Verification Step -->
		<OTPVerification 
			userID={authStore.user?.id || ''}
			email={authStore.user?.email || ''}
			purpose="signup_verification"
			onSuccess={handleEmailVerificationSuccess}
			onCancel={handleEmailVerificationCancel}
		/>
	{:else}
		<!-- Regular Personal Account View -->
		<div class="flex items-center justify-between mb-4">
			<h3 class="text-lg font-semibold text-foreground">Personal Account</h3>
			<button
				onclick={() => isEditingProfile = !isEditingProfile}
				class="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1"
				disabled={isSavingPersonal}
			>
				<Edit3 class="w-4 h-4" />
				Edit
			</button>
		</div>

		<!-- Success Message for Email Verification -->
		{#if personalEditSuccess}
			<div class="mb-4 p-3 bg-green-50 dark:bg-green-950/50 border border-green-200 dark:border-green-800 rounded-lg">
				<p class="text-sm text-green-600 dark:text-green-400">{personalEditSuccess}</p>
			</div>
		{/if}

	{#if isEditingProfile}
		<!-- Edit Form -->
		<form onsubmit={(e) => { e.preventDefault(); handleSavePersonal(); }} class="space-y-4">
			{#if personalEditError}
				<div class="p-3 bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-800 rounded-lg">
					<p class="text-sm text-red-600 dark:text-red-400">{personalEditError}</p>
				</div>
			{/if}

			{#if personalEditSuccess}
				<div class="p-3 bg-blue-50 dark:bg-blue-950/50 border border-blue-200 dark:border-blue-800 rounded-lg">
					<p class="text-sm text-blue-600 dark:text-blue-400">{personalEditSuccess}</p>
				</div>
			{/if}

			<div class="space-y-2">
				<Label for="edit-personal-name">Name</Label>
				<Input
					id="edit-personal-name"
					type="text"
					bind:value={editPersonalName}
					disabled={isSavingPersonal}
					required
				/>
			</div>

			<div class="space-y-2">
				<Label for="edit-personal-email">Email</Label>
				<Input
					id="edit-personal-email"
					type="email"
					bind:value={editPersonalEmail}
					disabled={isSavingPersonal}
					required
				/>
				<p class="text-xs text-muted-foreground">Changing your email will require verification</p>
			</div>

			<div class="flex gap-2">
				<Button
					type="submit"
					size="sm"
					disabled={isSavingPersonal}
				>
					{isSavingPersonal ? 'Saving...' : 'Save Changes'}
				</Button>
				<Button
					type="button"
					variant="outline"
					size="sm"
					onclick={handleCancelPersonalEdit}
					disabled={isSavingPersonal}
				>
					Cancel
				</Button>
			</div>
		</form>
	{:else}
		<!-- Display Mode -->
		<div class="grid gap-4 sm:grid-cols-2">
			<div class="p-4 bg-muted/50 rounded-lg">
				<h4 class="text-sm font-medium text-muted-foreground uppercase tracking-wide mb-1">Email</h4>
				<p class="text-foreground">{authStore.user?.email}</p>
				{#if !authStore.user?.verified}
					<div class="flex items-center justify-between mt-2">
						<p class="text-xs text-orange-600 dark:text-orange-400">⚠️ Email not verified</p>
						<Button
							size="sm"
							variant="outline"
							onclick={startEmailVerification}
							class="text-xs h-6 px-2"
						>
							<Shield class="w-3 h-3 mr-1" />
							Verify Now
						</Button>
					</div>
				{:else}
					<p class="text-xs text-green-600 dark:text-green-400 mt-1">✅ Email verified</p>
				{/if}
			</div>
			
			<div class="p-4 bg-muted/50 rounded-lg">
				<h4 class="text-sm font-medium text-muted-foreground uppercase tracking-wide mb-1">Name</h4>
				<p class="text-foreground">{authStore.user?.name || 'Not set'}</p>
			</div>
		</div>
	{/if}
	{/if}
</div>