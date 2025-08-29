import { pb } from '$lib/pocketbase.js';
import { authStore } from '$lib/stores/authClient.svelte.js';

// Types for file upload system
export interface UploadMetadata {
	fileType: 'avatar' | 'document' | 'media' | 'temp';
	category?: string;
	userId?: string;
	visibility?: 'public' | 'private' | 'shared';
	processAfterUpload?: string[];
	customMetadata?: Record<string, any>;
}

export interface FileUploadRecord {
	id: string;
	collectionId: string;
	collectionName: string;
	upload_id: string;
	file: string;
	metadata: Record<string, any>;
	processing_status: 'pending' | 'processing' | 'completed' | 'failed';
	file_type: string;
	category?: string;
	user: string;
	visibility: 'public' | 'private' | 'shared';
	processed_variants?: Record<string, any>;
	original_name?: string;
	created: string;
	updated: string;
}

export interface UploadProgress {
	uploadId: string;
	bytesUploaded: number;
	bytesTotal: number;
	percentage: number;
}

// PocketBase file URL generation
export function getFileUrl(record: FileUploadRecord, filename?: string): string {
	if (!record.file) return '';
	
	const actualFilename = filename || record.file;
	return pb.files.getUrl(record, actualFilename);
}

// Comprehensive thumbnail size types
export type ThumbnailSize = 
	| '32x32' | '64x64' | '100x100' | '128x128' | '200x200' | '300x300' | '400x400'
	| '600x400' | '400x600' 
	| '32x0' | '64x0' | '128x0' | '200x0'
	| '0x32' | '0x64' | '0x128' | '0x200'
	| '800x600f' | '400x300f' | '200x150f';

export type AvatarSize = 'small' | 'medium' | 'large' | 'xl';

// Get thumbnail URL for file uploads with comprehensive size options
export function getThumbnailUrl(record: FileUploadRecord, size: ThumbnailSize = '100x100'): string {
	if (!record.file) return '';
	
	return pb.files.getUrl(record, record.file, { thumb: size });
}

// Get avatar URL with appropriate thumbnail size
export function getAvatarUrl(user: any, size: AvatarSize = 'medium'): string | null {
	if (!user?.avatar) return null;
	
	// Define thumbnail sizes based on use case
	const thumbSizes: Record<AvatarSize, ThumbnailSize> = {
		small: '64x64',    // Navigation, small cards
		medium: '128x128', // Profile cards, medium displays  
		large: '200x200',  // Profile pages, large displays
		xl: '200x0'        // Extra large, width-constrained
	};
	
	return pb.files.getUrl(user, user.avatar, { thumb: thumbSizes[size] });
}

// Upload file using TUS protocol
export async function uploadFile(
	file: File,
	metadata: UploadMetadata,
	onProgress?: (progress: UploadProgress) => void,
	onError?: (error: Error) => void
): Promise<FileUploadRecord | null> {
	if (!authStore.isLoggedIn || !authStore.user) {
		throw new Error('Authentication required for file upload');
	}

	try {
		// Dynamically import Uppy modules (to avoid SSR issues)
		const [{ default: Uppy }, { default: Tus }] = await Promise.all([
			import('@uppy/core'),
			import('@uppy/tus')
		]);

		// Create Uppy instance
		const uppy = new Uppy({
			restrictions: {
				maxNumberOfFiles: 1,
				maxFileSize: 100 * 1024 * 1024 // 100MB
			},
			autoProceed: true
		});

		// Configure TUS plugin
		uppy.use(Tus, {
			endpoint: `${pb.baseUrl}/tus/`,
			headers: {
				Authorization: `Bearer ${pb.authStore.token}`
			},
			metadata: {
				filename: file.name,
				filetype: file.type,
				fileType: metadata.fileType,
				category: metadata.category || '',
				userId: authStore.user.id,
				visibility: metadata.visibility || 'private',
				processAfterUpload: JSON.stringify(metadata.processAfterUpload || []),
				...metadata.customMetadata
			},
			chunkSize: 1024 * 1024, // 1MB chunks
			retryDelays: [0, 1000, 3000, 5000],
			removeFingerprintOnSuccess: true
		});

		return new Promise((resolve, reject) => {
			// Track upload progress
			uppy.on('upload-progress', (file, progress) => {
				if (onProgress) {
					onProgress({
						uploadId: file.id,
						bytesUploaded: progress.bytesUploaded,
						bytesTotal: progress.bytesTotal,
						percentage: Math.round((progress.bytesUploaded / progress.bytesTotal) * 100)
					});
				}
			});

			// Handle upload completion
			uppy.on('upload-success', async (file, response) => {
				try {
					// Get the upload ID from the response
					const uploadId = response.uploadURL?.split('/').pop();
					if (!uploadId) {
						throw new Error('Failed to get upload ID from response');
					}

					// Wait a moment for the backend to process
					await new Promise(resolve => setTimeout(resolve, 1000));

					// Find the record by upload_id
					const record = await pb.collection('file_uploads').getFirstListItem(
						`upload_id = "${uploadId}"`
					);

					resolve(record);
				} catch (error) {
					reject(error);
				}
			});

			// Handle upload errors
			uppy.on('upload-error', (file, error) => {
				if (onError) {
					onError(error);
				}
				reject(error);
			});

			// Start upload
			uppy.addFile({
				name: file.name,
				type: file.type,
				data: file
			});
		});
	} catch (error) {
		if (onError) {
			onError(error as Error);
		}
		throw error;
	}
}

// Get user's uploaded files
export async function getUserFiles(
	fileType?: string,
	category?: string,
	page = 1,
	perPage = 50
): Promise<{ items: FileUploadRecord[]; totalItems: number; totalPages: number }> {
	if (!authStore.isLoggedIn || !authStore.user) {
		throw new Error('Authentication required');
	}

	let filter = `user = "${authStore.user.id}"`;
	
	if (fileType) {
		filter += ` && file_type = "${fileType}"`;
	}
	
	if (category) {
		filter += ` && category = "${category}"`;
	}

	return pb.collection('file_uploads').getList(page, perPage, {
		filter,
		sort: '-created'
	});
}

// Delete a file upload
export async function deleteFile(recordId: string): Promise<boolean> {
	try {
		await pb.collection('file_uploads').delete(recordId);
		return true;
	} catch (error) {
		console.error('Failed to delete file:', error);
		return false;
	}
}

// Check processing status
export async function getProcessingStatus(uploadId: string): Promise<string | null> {
	try {
		const record = await pb.collection('file_uploads').getFirstListItem(
			`upload_id = "${uploadId}"`
		);
		return record.processing_status;
	} catch (error) {
		return null;
	}
}

// Subscribe to file processing updates
export function subscribeToFileUpdates(
	recordId: string,
	callback: (record: FileUploadRecord) => void
): () => void {
	return pb.collection('file_uploads').subscribe(recordId, callback);
}

// Process file with specific instructions
export async function processFile(
	recordId: string,
	instructions: string[]
): Promise<boolean> {
	try {
		// This would call a custom endpoint for processing
		const response = await fetch(`${pb.baseUrl}/api/files/process/${recordId}`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${pb.authStore.token}`
			},
			body: JSON.stringify({ instructions })
		});

		return response.ok;
	} catch (error) {
		console.error('Failed to process file:', error);
		return false;
	}
}

// Validate file before upload
export function validateFile(
	file: File,
	options: {
		maxSize?: number; // in bytes
		allowedTypes?: string[];
		allowedExtensions?: string[];
	} = {}
): { valid: boolean; error?: string } {
	const { maxSize = 100 * 1024 * 1024, allowedTypes, allowedExtensions } = options;

	// Check file size
	if (file.size > maxSize) {
		return {
			valid: false,
			error: `File size (${formatFileSize(file.size)}) exceeds maximum allowed size (${formatFileSize(maxSize)})`
		};
	}

	// Check MIME type
	if (allowedTypes && allowedTypes.length > 0) {
		const isTypeAllowed = allowedTypes.some(type => {
			if (type.endsWith('/*')) {
				return file.type.startsWith(type.slice(0, -1));
			}
			return file.type === type;
		});

		if (!isTypeAllowed) {
			return {
				valid: false,
				error: `File type "${file.type}" is not allowed. Allowed types: ${allowedTypes.join(', ')}`
			};
		}
	}

	// Check file extension
	if (allowedExtensions && allowedExtensions.length > 0) {
		const extension = file.name.split('.').pop()?.toLowerCase();
		if (!extension || !allowedExtensions.includes(extension)) {
			return {
				valid: false,
				error: `File extension ".${extension}" is not allowed. Allowed extensions: ${allowedExtensions.join(', ')}`
			};
		}
	}

	return { valid: true };
}

// Format file size for display
export function formatFileSize(bytes: number): string {
	if (bytes === 0) return '0 Bytes';

	const k = 1024;
	const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));

	return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Get file type from MIME type
export function getFileTypeFromMime(mimeType: string): 'image' | 'video' | 'audio' | 'document' | 'unknown' {
	if (mimeType.startsWith('image/')) return 'image';
	if (mimeType.startsWith('video/')) return 'video';
	if (mimeType.startsWith('audio/')) return 'audio';
	if (mimeType.includes('pdf') || mimeType.includes('document') || mimeType.includes('text')) {
		return 'document';
	}
	return 'unknown';
}

// Generate a preview URL for different file types
export function getFilePreviewUrl(record: FileUploadRecord): string | null {
	if (!record.file) return null;

	const fileUrl = getFileUrl(record);
	
	// For images, return thumbnail
	if (record.file.toLowerCase().match(/\.(jpg|jpeg|png|gif|webp)$/)) {
		return getThumbnailUrl(record, '300x300');
	}

	// For other file types, you might return a placeholder or the file URL
	return fileUrl;
}