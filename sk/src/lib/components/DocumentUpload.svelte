<script lang="ts">
	import FileUpload from './FileUpload.svelte';
	import { type FileUploadRecord } from '$lib/files.js';
	import { FileText } from 'lucide-svelte';

	// Props
	let {
		category = 'document',
		allowedTypes = [
			'application/pdf',
			'application/msword',
			'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
			'text/plain',
			'image/jpeg',
			'image/png'
		],
		maxFileSize = 25 * 1024 * 1024, // 25MB
		maxFiles = 5,
		multiple = true,
		visibility = 'private',
		processAfterUpload = ['extract_text', 'thumbnail'],
		placeholder = 'Upload documents (PDF, DOC, images)',
		showPreview = true,
		onUploadComplete = null,
		onUploadError = null,
		onFileSelect = null,
		customMetadata = {},
		className = ''
	}: {
		category?: string;
		allowedTypes?: string[];
		maxFileSize?: number;
		maxFiles?: number;
		multiple?: boolean;
		visibility?: 'public' | 'private' | 'shared';
		processAfterUpload?: string[];
		placeholder?: string;
		showPreview?: boolean;
		onUploadComplete?: ((record: FileUploadRecord) => void) | null;
		onUploadError?: ((error: Error) => void) | null;
		onFileSelect?: ((files: File[]) => void) | null;
		customMetadata?: Record<string, any>;
		className?: string;
	} = $props();

	// State
	let uploadedDocuments = $state<FileUploadRecord[]>([]);

	// Handle document upload
	function handleDocumentUpload(record: FileUploadRecord) {
		uploadedDocuments = [...uploadedDocuments, record];
		
		if (onUploadComplete) {
			onUploadComplete(record);
		}
	}

	// Handle upload error
	function handleUploadError(error: Error) {
		console.error('Document upload error:', error);
		if (onUploadError) {
			onUploadError(error);
		}
	}

	// Handle file selection
	function handleFileSelect(files: File[]) {
		if (onFileSelect) {
			onFileSelect(files);
		}
	}
</script>

<div class="document-upload {className}">
	<!-- Upload Header -->
	<div class="flex items-center space-x-2 mb-4">
		<FileText class="h-5 w-5 text-primary" />
		<h3 class="text-lg font-semibold">Document Upload</h3>
	</div>

	<!-- File Upload Component -->
	<FileUpload
		fileType="document"
		{category}
		{allowedTypes}
		{maxFileSize}
		{maxFiles}
		{multiple}
		{visibility}
		{processAfterUpload}
		{placeholder}
		{showPreview}
		customMetadata={{
			...customMetadata,
			documentType: category
		}}
		onUploadComplete={handleDocumentUpload}
		onUploadError={handleUploadError}
		onFileSelect={handleFileSelect}
	/>

	<!-- Document Guidelines -->
	<div class="mt-4 p-4 bg-muted/50 rounded-lg">
		<h4 class="text-sm font-medium mb-2">Supported Document Types:</h4>
		<ul class="text-xs text-muted-foreground space-y-1">
			<li>• PDF documents (.pdf)</li>
			<li>• Microsoft Word documents (.doc, .docx)</li>
			<li>• Plain text files (.txt)</li>
			<li>• Images with text (.jpg, .png) - text will be extracted</li>
		</ul>
		
		<div class="mt-3 pt-3 border-t border-muted-foreground/20">
			<p class="text-xs text-muted-foreground">
				<strong>Processing:</strong> Documents are automatically processed for text extraction and thumbnail generation.
				Large files may take a few moments to process completely.
			</p>
		</div>
	</div>

	<!-- Recent Uploads Summary -->
	{#if uploadedDocuments.length > 0}
		<div class="mt-4">
			<h4 class="text-sm font-medium mb-2">Recently Uploaded ({uploadedDocuments.length})</h4>
			<div class="space-y-1 max-h-24 overflow-y-auto">
				{#each uploadedDocuments.slice(-3) as doc}
					<div class="text-xs text-muted-foreground flex justify-between">
						<span>{doc.original_name || 'Unnamed document'}</span>
						<span class="capitalize">{doc.processing_status}</span>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>

<style>
	.document-upload {
		@apply w-full;
	}
</style>