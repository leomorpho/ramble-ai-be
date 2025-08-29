<script lang="ts">
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { pb } from '$lib/pocketbase.js';
	import { User, Camera, Upload, X } from 'lucide-svelte';
	import { getAvatarUrl } from '$lib/files.js';

	// Props
	let {
		size = 'lg',
		showUploadButton = true,
		onUploadComplete = null,
		onUploadError = null,
		className = ''
	}: {
		size?: 'sm' | 'md' | 'lg' | 'xl';
		showUploadButton?: boolean;
		onUploadComplete?: (() => void) | null;
		onUploadError?: ((error: Error) => void) | null;
		className?: string;
	} = $props();

	// State
	let showUploadDialog = $state(false);
	let isUploading = $state(false);
	let isDragOver = $state(false);
	let fileInput: HTMLInputElement;

	// Size configurations
	const sizeClasses = {
		sm: 'w-8 h-8',
		md: 'w-12 h-12', 
		lg: 'w-16 h-16',
		xl: 'w-24 h-24'
	};

	const iconSizes = {
		sm: 'h-3 w-3',
		md: 'h-4 w-4',
		lg: 'h-5 w-5', 
		xl: 'h-6 w-6'
	};

	// Map component size to avatar size
	const avatarSizeMap = {
		sm: 'small' as const,
		md: 'medium' as const,
		lg: 'medium' as const,
		xl: 'large' as const
	};

	// Handle file upload
	async function handleFileUpload(file: File) {
		if (!authStore.user) return;

		// Validate file
		const maxSize = 5 * 1024 * 1024; // 5MB
		const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp', 'image/gif'];
		
		if (file.size > maxSize) {
			const error = new Error('File size must be less than 5MB');
			if (onUploadError) onUploadError(error);
			return;
		}

		if (!allowedTypes.includes(file.type)) {
			const error = new Error('File must be an image (JPEG, PNG, WebP, or GIF)');
			if (onUploadError) onUploadError(error);
			return;
		}

		try {
			isUploading = true;

			// Create FormData and upload directly to PocketBase users collection
			const formData = new FormData();
			formData.append('avatar', file);

			// Update user record with new avatar
			const updatedUser = await pb.collection('users').update(authStore.user.id, formData);

			// Update auth store
			authStore.user = updatedUser;
			
			if (onUploadComplete) {
				onUploadComplete();
			}

			showUploadDialog = false;
		} catch (error) {
			console.error('Failed to upload avatar:', error);
			if (onUploadError) {
				onUploadError(error instanceof Error ? error : new Error('Failed to upload avatar'));
			}
		} finally {
			isUploading = false;
		}
	}

	// Handle file input change
	function handleFileChange(event: Event) {
		const target = event.target as HTMLInputElement;
		const file = target.files?.[0];
		if (file) {
			handleFileUpload(file);
		}
	}

	// Handle drag and drop
	function handleDrop(event: DragEvent) {
		event.preventDefault();
		isDragOver = false;
		
		const files = event.dataTransfer?.files;
		if (files && files.length > 0) {
			handleFileUpload(files[0]);
		}
	}

	function handleDragOver(event: DragEvent) {
		event.preventDefault();
		isDragOver = true;
	}

	function handleDragLeave(event: DragEvent) {
		event.preventDefault();
		isDragOver = false;
	}
</script>

<!-- Avatar Display -->
<div class="avatar-container {className}">
	<div class="relative inline-block">
		<!-- Avatar Image -->
		<div class="relative {sizeClasses[size]} rounded-full overflow-hidden bg-muted border-2 border-background shadow-sm">
			{#if getAvatarUrl(authStore.user, avatarSizeMap[size])}
				<img 
					src={getAvatarUrl(authStore.user, avatarSizeMap[size])} 
					alt="Avatar"
					class="w-full h-full object-cover"
				/>
			{:else}
				<div class="w-full h-full flex items-center justify-center bg-gradient-to-br from-primary/20 to-primary/10">
					<User class="{iconSizes[size]} text-primary" />
				</div>
			{/if}
			
			<!-- Loading overlay -->
			{#if isUploading}
				<div class="absolute inset-0 bg-black/50 flex items-center justify-center">
					<div class="animate-spin rounded-full h-1/2 w-1/2 border-2 border-white border-t-transparent"></div>
				</div>
			{/if}
		</div>

		<!-- Upload Button -->
		{#if showUploadButton && authStore.isLoggedIn}
			<button
				onclick={() => showUploadDialog = true}
				class="absolute -bottom-1 -right-1 bg-primary text-primary-foreground rounded-full p-1.5 shadow-lg hover:bg-primary/90 transition-colors"
				title="Change avatar"
				disabled={isUploading}
			>
				<Camera class="h-3 w-3" />
			</button>
		{/if}
	</div>
</div>

<!-- Upload Dialog -->
{#if showUploadDialog}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={(e) => e.target === e.currentTarget && (showUploadDialog = false)}>
		<div class="bg-background rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold">Upload Avatar</h3>
				<button
					onclick={() => showUploadDialog = false}
					class="text-muted-foreground hover:text-foreground"
					disabled={isUploading}
				>
					<X class="h-5 w-5" />
				</button>
			</div>
			
			<!-- Upload Area -->
			<div 
				class="border-2 border-dashed rounded-lg p-8 text-center transition-colors {isDragOver ? 'border-primary bg-primary/5' : 'border-muted-foreground/20'}"
				ondrop={handleDrop}
				ondragover={handleDragOver}
				ondragleave={handleDragLeave}
			>
				{#if isUploading}
					<div class="space-y-4">
						<div class="animate-spin rounded-full h-8 w-8 border-2 border-primary border-t-transparent mx-auto"></div>
						<p class="text-sm text-muted-foreground">Uploading avatar...</p>
					</div>
				{:else}
					<div class="space-y-4">
						<div class="w-12 h-12 bg-muted rounded-full flex items-center justify-center mx-auto">
							<Upload class="h-6 w-6 text-muted-foreground" />
						</div>
						<div>
							<p class="text-sm font-medium mb-1">Drop your image here, or click to browse</p>
							<p class="text-xs text-muted-foreground">JPEG, PNG, WebP or GIF up to 5MB</p>
						</div>
						<button
							onclick={() => fileInput.click()}
							class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors text-sm"
						>
							Choose File
						</button>
					</div>
				{/if}
			</div>

			<!-- Hidden file input -->
			<input
				bind:this={fileInput}
				type="file"
				accept="image/jpeg,image/jpg,image/png,image/webp,image/gif"
				onchange={handleFileChange}
				class="hidden"
				disabled={isUploading}
			/>

			<div class="flex justify-end space-x-2 mt-4">
				<button
					onclick={() => showUploadDialog = false}
					class="px-4 py-2 text-sm border border-muted-foreground/20 rounded-md hover:bg-muted transition-colors"
					disabled={isUploading}
				>
					Cancel
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.avatar-container {
		@apply inline-block;
	}
</style>