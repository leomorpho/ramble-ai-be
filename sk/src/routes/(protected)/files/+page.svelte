<script lang="ts">
	import { onMount } from 'svelte';
	import FileUpload from '$lib/components/FileUpload.svelte';
	import AvatarUpload from '$lib/components/AvatarUpload.svelte';
	import DocumentUpload from '$lib/components/DocumentUpload.svelte';
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { getUserFiles, getThumbnailUrl, type FileUploadRecord } from '$lib/files.js';
	import { config } from '$lib/config.js';
	import { FileText, Image, User } from 'lucide-svelte';

	// State
	let userFiles = $state<FileUploadRecord[]>([]);
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	// Load user files on mount
	onMount(async () => {
		if (authStore.isLoggedIn) {
			await loadUserFiles();
		}
	});

	// Load user files
	async function loadUserFiles() {
		try {
			isLoading = true;
			error = null;
			const result = await getUserFiles();
			userFiles = result.items;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load files';
			console.error('Failed to load user files:', err);
		} finally {
			isLoading = false;
		}
	}

	// Handle upload completion
	function handleUploadComplete(record: FileUploadRecord) {
		console.log('Upload completed:', record);
		userFiles = [record, ...userFiles];
	}

	// Handle upload error
	function handleUploadError(error: Error) {
		console.error('Upload error:', error);
	}

	// Check if file is an image
	function isImageFile(file: FileUploadRecord): boolean {
		if (!file.file) return false;
		const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg'];
		const fileName = file.file.toLowerCase();
		return imageExtensions.some(ext => fileName.endsWith(ext));
	}
</script>

<svelte:head>
	<title>File Uploads - {config.app.name}</title>
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<div class="max-w-4xl mx-auto">
		<!-- Header -->
		<div class="mb-8">
			<h1 class="text-3xl font-bold mb-2">File Uploads</h1>
			<p class="text-muted-foreground">
				Test the TUS resumable upload system with PocketBase integration
			</p>
		</div>

		{#if !authStore.isLoggedIn}
			<div class="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-8">
				<p class="text-yellow-800">
					Please <a href="/login" class="text-primary hover:underline">log in</a> to test file uploads.
				</p>
			</div>
		{:else}
			<!-- Upload Sections -->
			<div class="space-y-8">
				<!-- Avatar Upload -->
				<section class="bg-card rounded-lg border p-6">
					<div class="flex items-center space-x-2 mb-4">
						<User class="h-5 w-5 text-primary" />
						<h2 class="text-xl font-semibold">Avatar Upload</h2>
					</div>
					
					<div class="flex items-start space-x-6">
						<AvatarUpload 
							size="xl"
							onUploadComplete={handleUploadComplete}
							onUploadError={handleUploadError}
						/>
						<div class="flex-1">
							<p class="text-sm text-muted-foreground mb-2">
								Upload a profile picture. The image will be automatically resized and optimized.
							</p>
							<ul class="text-xs text-muted-foreground space-y-1">
								<li>• Supported formats: JPEG, PNG, WebP, GIF</li>
								<li>• Maximum size: 5MB</li>
								<li>• Automatically resized to 200x200 with thumbnail generation</li>
							</ul>
						</div>
					</div>
				</section>

				<!-- Document Upload -->
				<section class="bg-card rounded-lg border p-6">
					<DocumentUpload
						category="test_documents"
						onUploadComplete={handleUploadComplete}
						onUploadError={handleUploadError}
					/>
				</section>

				<!-- Media Upload -->
				<section class="bg-card rounded-lg border p-6">
					<div class="flex items-center space-x-2 mb-4">
						<Image class="h-5 w-5 text-primary" />
						<h2 class="text-xl font-semibold">Media Upload</h2>
					</div>
					
					<FileUpload
						fileType="media"
						category="gallery"
						maxFiles={5}
						multiple={true}
						allowedTypes={['image/*', 'video/mp4', 'video/webm']}
						maxFileSize={50 * 1024 * 1024}
						processAfterUpload={['thumbnail', 'resize:1920x1080']}
						visibility="private"
						placeholder="Upload images or videos for your gallery"
						showPreview={true}
						onUploadComplete={handleUploadComplete}
						onUploadError={handleUploadError}
					/>
				</section>

				<!-- Generic File Upload -->
				<section class="bg-card rounded-lg border p-6">
					<div class="flex items-center space-x-2 mb-4">
						<FileText class="h-5 w-5 text-primary" />
						<h2 class="text-xl font-semibold">Generic File Upload</h2>
					</div>
					
					<FileUpload
						fileType="temp"
						category="test_files"
						maxFiles={3}
						multiple={true}
						maxFileSize={25 * 1024 * 1024}
						autoUpload={false}
						visibility="private"
						placeholder="Upload any files (manual upload mode)"
						showPreview={true}
						onUploadComplete={handleUploadComplete}
						onUploadError={handleUploadError}
					/>
				</section>
			</div>

			<!-- User Files List -->
			<section class="mt-12">
				<h2 class="text-2xl font-semibold mb-4">Your Uploaded Files</h2>
				
				{#if isLoading}
					<div class="text-center py-8">
						<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
						<p class="mt-2 text-muted-foreground">Loading files...</p>
					</div>
				{:else if error}
					<div class="bg-red-50 border border-red-200 rounded-lg p-4">
						<p class="text-red-800">{error}</p>
						<button 
							onclick={loadUserFiles}
							class="mt-2 text-sm text-primary hover:underline"
						>
							Retry
						</button>
					</div>
				{:else if userFiles.length === 0}
					<div class="text-center py-8 bg-muted/50 rounded-lg">
						<p class="text-muted-foreground">No files uploaded yet.</p>
					</div>
				{:else}
					<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
						{#each userFiles as file (file.id)}
							<div class="bg-card border rounded-lg p-4">
								<!-- Image Thumbnail -->
								{#if isImageFile(file)}
									<div class="mb-3">
										<img
											src={getThumbnailUrl(file, '200x150f')}
											alt={file.original_name || 'File thumbnail'}
											class="w-full h-32 object-cover rounded-lg bg-muted"
											loading="lazy"
										/>
									</div>
								{:else}
									<div class="mb-3 w-full h-32 bg-muted rounded-lg flex items-center justify-center">
										<FileText class="h-8 w-8 text-muted-foreground" />
									</div>
								{/if}
								
								<div class="mb-2">
									<h3 class="font-medium text-sm">
										{file.original_name || 'Unnamed file'}
									</h3>
									<p class="text-xs text-muted-foreground">
										{file.file_type} • {file.category}
									</p>
								</div>
								
								<div class="flex items-center justify-between">
									<span class="text-xs px-2 py-1 rounded-full
										{file.processing_status === 'completed' ? 'bg-green-100 text-green-800' :
										 file.processing_status === 'processing' ? 'bg-blue-100 text-blue-800' :
										 file.processing_status === 'failed' ? 'bg-red-100 text-red-800' :
										 'bg-gray-100 text-gray-800'}">
										{file.processing_status}
									</span>
									
									<span class="text-xs text-muted-foreground">
										{file.visibility}
									</span>
								</div>
								
								<div class="mt-2 text-xs text-muted-foreground">
									{new Date(file.created).toLocaleDateString()}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</section>
		{/if}
	</div>
</div>