<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import { uploadFile, validateFile, formatFileSize, type UploadMetadata, type FileUploadRecord, type UploadProgress } from '$lib/files.js';
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { Loader2, Upload, CheckCircle, XCircle, File, Image, Video, Music } from 'lucide-svelte';

	// Props
	let {
		fileType,
		category = '',
		maxFiles = 1,
		allowedTypes = [],
		allowedExtensions = [],
		maxFileSize = 50 * 1024 * 1024, // 50MB default
		processAfterUpload = [],
		visibility = 'private',
		customMetadata = {},
		autoUpload = true,
		showPreview = true,
		multiple = false,
		placeholder = 'Drag and drop files here or click to browse',
		onUploadComplete = null,
		onUploadProgress = null,
		onUploadError = null,
		onFileSelect = null,
		disabled = false,
		className = ''
	}: {
		fileType: 'avatar' | 'document' | 'media' | 'temp';
		category?: string;
		maxFiles?: number;
		allowedTypes?: string[];
		allowedExtensions?: string[];
		maxFileSize?: number;
		processAfterUpload?: string[];
		visibility?: 'public' | 'private' | 'shared';
		customMetadata?: Record<string, any>;
		autoUpload?: boolean;
		showPreview?: boolean;
		multiple?: boolean;
		placeholder?: string;
		onUploadComplete?: ((record: FileUploadRecord) => void) | null;
		onUploadProgress?: ((progress: UploadProgress) => void) | null;
		onUploadError?: ((error: Error) => void) | null;
		onFileSelect?: ((files: File[]) => void) | null;
		disabled?: boolean;
		className?: string;
	} = $props();

	// State
	let isDragOver = $state(false);
	let isUploading = $state(false);
	let uploadProgress = $state<UploadProgress | null>(null);
	let selectedFiles = $state<File[]>([]);
	let uploadedFiles = $state<FileUploadRecord[]>([]);
	let errors = $state<string[]>([]);
	let fileInputElement: HTMLInputElement;

	// Computed
	let canUpload = $derived(
		authStore.isLoggedIn && 
		!disabled && 
		!isUploading && 
		(multiple ? selectedFiles.length > 0 : selectedFiles.length === 1)
	);

	let acceptedFileTypes = $derived(
		allowedTypes.length > 0 ? allowedTypes.join(',') : '*'
	);

	// Handle file selection
	function handleFileSelect(files: FileList | null) {
		if (!files || files.length === 0) return;

		const fileArray = Array.from(files);
		const validFiles: File[] = [];
		const newErrors: string[] = [];

		// Validate each file
		fileArray.forEach(file => {
			const validation = validateFile(file, {
				maxSize: maxFileSize,
				allowedTypes: allowedTypes.length > 0 ? allowedTypes : undefined,
				allowedExtensions: allowedExtensions.length > 0 ? allowedExtensions : undefined
			});

			if (validation.valid) {
				validFiles.push(file);
			} else {
				newErrors.push(`${file.name}: ${validation.error}`);
			}
		});

		// Check file count limit
		if (!multiple && validFiles.length > 1) {
			newErrors.push('Only one file is allowed');
			validFiles.splice(1);
		} else if (validFiles.length > maxFiles) {
			newErrors.push(`Maximum ${maxFiles} files allowed`);
			validFiles.splice(maxFiles);
		}

		// Update state
		errors = newErrors;
		selectedFiles = multiple ? [...selectedFiles, ...validFiles] : validFiles;

		// Notify parent
		if (onFileSelect) {
			onFileSelect(selectedFiles);
		}

		// Auto upload if enabled
		if (autoUpload && validFiles.length > 0 && !newErrors.length) {
			handleUpload();
		}
	}

	// Handle drag and drop
	function handleDragOver(event: DragEvent) {
		event.preventDefault();
		isDragOver = true;
	}

	function handleDragLeave(event: DragEvent) {
		event.preventDefault();
		isDragOver = false;
	}

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		isDragOver = false;
		
		if (disabled || !authStore.isLoggedIn) return;

		const files = event.dataTransfer?.files;
		handleFileSelect(files);
	}

	// Handle upload
	async function handleUpload() {
		if (!canUpload || selectedFiles.length === 0) return;

		isUploading = true;
		errors = [];

		try {
			const metadata: UploadMetadata = {
				fileType,
				category: category || undefined,
				userId: authStore.user?.id,
				visibility,
				processAfterUpload,
				...customMetadata
			};

			// Upload files (one at a time for now)
			for (const file of selectedFiles) {
				const record = await uploadFile(
					file,
					metadata,
					(progress) => {
						uploadProgress = progress;
						if (onUploadProgress) {
							onUploadProgress(progress);
						}
					},
					(error) => {
						errors = [...errors, error.message];
						if (onUploadError) {
							onUploadError(error);
						}
					}
				);

				if (record) {
					uploadedFiles = [...uploadedFiles, record];
					if (onUploadComplete) {
						onUploadComplete(record);
					}
				}
			}

			// Clear selected files after successful upload
			if (!multiple) {
				selectedFiles = [];
			}
		} catch (error) {
			const errorMessage = error instanceof Error ? error.message : 'Upload failed';
			errors = [...errors, errorMessage];
			if (onUploadError) {
				onUploadError(error instanceof Error ? error : new Error(errorMessage));
			}
		} finally {
			isUploading = false;
			uploadProgress = null;
		}
	}

	// Remove selected file
	function removeFile(index: number) {
		selectedFiles = selectedFiles.filter((_, i) => i !== index);
		if (onFileSelect) {
			onFileSelect(selectedFiles);
		}
	}

	// Get file icon based on type
	function getFileIcon(file: File) {
		if (file.type.startsWith('image/')) return Image;
		if (file.type.startsWith('video/')) return Video;
		if (file.type.startsWith('audio/')) return Music;
		return File;
	}

	// Clear all
	function clearAll() {
		selectedFiles = [];
		uploadedFiles = [];
		errors = [];
		uploadProgress = null;
		if (onFileSelect) {
			onFileSelect([]);
		}
	}
</script>

<!-- Upload Area -->
<div class="file-upload-container {className}">
	<!-- Drop Zone -->
	<div
		class="border-2 border-dashed rounded-lg p-6 text-center transition-colors
			{isDragOver ? 'border-primary bg-primary/5' : 'border-muted-foreground/20'}
			{disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer hover:border-primary hover:bg-primary/5'}
			{errors.length > 0 ? 'border-destructive' : ''}"
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		ondrop={handleDrop}
		onclick={() => !disabled && fileInputElement?.click()}
		role="button"
		tabindex="0"
		onkeydown={(e) => {
			if ((e.key === 'Enter' || e.key === ' ') && !disabled) {
				e.preventDefault();
				fileInputElement?.click();
			}
		}}
	>
		<!-- Upload Icon -->
		<div class="mb-4">
			{#if isUploading}
				<Loader2 class="h-8 w-8 animate-spin mx-auto text-primary" />
			{:else}
				<Upload class="h-8 w-8 mx-auto text-muted-foreground" />
			{/if}
		</div>

		<!-- Upload Text -->
		<div class="space-y-2">
			<p class="text-sm font-medium">
				{isUploading ? 'Uploading...' : placeholder}
			</p>
			
			{#if !isUploading}
				<p class="text-xs text-muted-foreground">
					{#if allowedTypes.length > 0}
						Supported formats: {allowedTypes.join(', ')}
					{/if}
					{#if maxFileSize}
						• Max size: {formatFileSize(maxFileSize)}
					{/if}
					{#if maxFiles > 1}
						• Max files: {maxFiles}
					{/if}
				</p>
			{/if}
		</div>

		<!-- Progress Bar -->
		{#if isUploading && uploadProgress}
			<div class="mt-4">
				<div class="w-full bg-muted rounded-full h-2">
					<div 
						class="bg-primary h-2 rounded-full transition-all duration-300"
						style="width: {uploadProgress.percentage}%"
					></div>
				</div>
				<p class="text-xs text-muted-foreground mt-1">
					{uploadProgress.percentage}% - {formatFileSize(uploadProgress.bytesUploaded)} / {formatFileSize(uploadProgress.bytesTotal)}
				</p>
			</div>
		{/if}
	</div>

	<!-- Hidden File Input -->
	<input
		bind:this={fileInputElement}
		type="file"
		accept={acceptedFileTypes}
		{multiple}
		onchange={(e) => handleFileSelect(e.target?.files)}
		class="hidden"
		{disabled}
	/>

	<!-- Selected Files List -->
	{#if selectedFiles.length > 0 && showPreview}
		<div class="mt-4 space-y-2">
			<h4 class="text-sm font-medium">Selected Files:</h4>
			{#each selectedFiles as file, index}
				{@const FileIcon = getFileIcon(file)}
				<div class="flex items-center justify-between p-3 bg-muted rounded-lg">
					<div class="flex items-center space-x-3">
						<FileIcon class="h-4 w-4 text-muted-foreground" />
						<div>
							<p class="text-sm font-medium">{file.name}</p>
							<p class="text-xs text-muted-foreground">{formatFileSize(file.size)}</p>
						</div>
					</div>
					{#if !isUploading}
						<button
							onclick={() => removeFile(index)}
							class="text-muted-foreground hover:text-destructive"
							type="button"
						>
							<XCircle class="h-4 w-4" />
						</button>
					{/if}
				</div>
			{/each}
		</div>
	{/if}

	<!-- Uploaded Files List -->
	{#if uploadedFiles.length > 0 && showPreview}
		<div class="mt-4 space-y-2">
			<h4 class="text-sm font-medium">Uploaded Files:</h4>
			{#each uploadedFiles as record}
				<div class="flex items-center justify-between p-3 bg-green-50 border border-green-200 rounded-lg">
					<div class="flex items-center space-x-3">
						<CheckCircle class="h-4 w-4 text-green-600" />
						<div>
							<p class="text-sm font-medium">{record.original_name || 'Uploaded file'}</p>
							<p class="text-xs text-muted-foreground">
								Status: {record.processing_status}
							</p>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Errors -->
	{#if errors.length > 0}
		<div class="mt-4 space-y-1">
			{#each errors as error}
				<div class="flex items-center space-x-2 text-sm text-destructive">
					<XCircle class="h-4 w-4" />
					<span>{error}</span>
				</div>
			{/each}
		</div>
	{/if}

	<!-- Manual Upload Button -->
	{#if !autoUpload && selectedFiles.length > 0}
		<div class="mt-4 flex space-x-2">
			<button
				onclick={handleUpload}
				disabled={!canUpload}
				class="flex-1 bg-primary text-primary-foreground hover:bg-primary/90 
					disabled:opacity-50 disabled:cursor-not-allowed 
					px-4 py-2 rounded-md text-sm font-medium transition-colors"
			>
				{#if isUploading}
					<Loader2 class="h-4 w-4 animate-spin mr-2 inline" />
					Uploading...
				{:else}
					Upload {selectedFiles.length} file{selectedFiles.length === 1 ? '' : 's'}
				{/if}
			</button>
			
			<button
				onclick={clearAll}
				disabled={isUploading}
				class="px-4 py-2 border border-muted-foreground/20 hover:bg-muted 
					disabled:opacity-50 disabled:cursor-not-allowed
					rounded-md text-sm font-medium transition-colors"
			>
				Clear
			</button>
		</div>
	{/if}

	<!-- Auth Required Message -->
	{#if !authStore.isLoggedIn}
		<div class="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
			<p class="text-sm text-yellow-800">
				Please log in to upload files.
			</p>
		</div>
	{/if}
</div>

<style>
	.file-upload-container {
		@apply w-full;
	}

	/* Uppy.io custom styling to match our design */
	:global(.uppy-Dashboard-inner) {
		border-radius: 0.5rem !important;
	}

	:global(.uppy-Dashboard-AddFiles-title) {
		font-size: 0.875rem !important;
		font-weight: 500 !important;
	}
</style>