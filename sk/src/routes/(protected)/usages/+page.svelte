<!-- 
🔗 CROSS-REFERENCE: This component is mirrored in the Wails frontend
📁 Location: frontend/src/routes/usage/+page.svelte
⚠️  Any changes to this usage statistics component should be reflected in the Wails version
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { authStore } from '$lib/stores/authClient.svelte';
	import { pb } from '$lib/pocketbase';
	import { Card } from '$lib/components/ui/card';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { 
		BarChart, 
		FileAudio, 
		Clock, 
		TrendingUp, 
		Calendar,
		CheckCircle2,
		AlertCircle,
		Loader2,
		Shield
	} from 'lucide-svelte';

	interface ProcessedFile {
		id: string;
		filename: string;
		file_size_bytes: number;
		duration_seconds: number;
		processing_time_ms: number;
		status: 'completed' | 'processing' | 'failed';
		transcript_length: number;
		words_count: number;
		model_used: string;
		created: string;
		updated: string;
	}

	interface UsageSummary {
		total_files: number;
		total_duration_seconds: number;
		total_duration_minutes: number;
		total_duration_hours: number;
		total_file_size_bytes: number;
		total_file_size_mb: number;
		total_processing_time_ms: number;
		avg_processing_time_ms: number;
		status_breakdown: {
			completed: number;
			processing: number;
			failed: number;
		};
		success_rate: number;
	}

	let processedFiles = $state<ProcessedFile[]>([]);
	let currentMonthSummary = $state<UsageSummary | null>(null);
	let allTimeSummary = $state<UsageSummary | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let selectedMonth = $state(new Date().toISOString().slice(0, 7)); // YYYY-MM

	onMount(async () => {
		await loadUsageData();
		
		// Subscribe to real-time updates
		pb.collection('processed_files').subscribe('*', (data) => {
			// Refresh data when changes occur
			loadUsageData();
		});
	});

	async function loadUsageData() {
		if (!authStore.user) return;

		try {
			isLoading = true;
			error = null;

			// Load processed files (recent ones for the table)  
			const filesResult = await pb.collection('processed_files').getList(1, 50, {
				filter: `user_id="${authStore.user.id}"`,
			});

			processedFiles = filesResult.items as unknown as ProcessedFile[];

			// Debug: let's see what fields are available in the response
			if (filesResult.items.length > 0) {
				console.log('Sample processed file record:', filesResult.items[0]);
			}

			// For now, let's use all files for both summaries until we get the basic query working
			// We'll add date filtering later once we confirm the basic query works
			const allTimeFiles = await pb.collection('processed_files').getFullList({
				filter: `user_id="${authStore.user.id}"`,
			});

			// Use same data for both summaries for now
			allTimeSummary = calculateSummary(allTimeFiles as unknown as ProcessedFile[]);
			currentMonthSummary = calculateSummary(allTimeFiles as unknown as ProcessedFile[]);

		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load usage data';
			console.error('Error loading usage data:', err);
		} finally {
			isLoading = false;
		}
	}

	function calculateSummary(files: ProcessedFile[]): UsageSummary {
		const totalFiles = files.length;
		let totalDuration = 0;
		let totalFileSize = 0;
		let totalProcessingTime = 0;
		const statusCounts = { completed: 0, processing: 0, failed: 0 };

		for (const file of files) {
			totalDuration += file.duration_seconds || 0;
			totalFileSize += file.file_size_bytes || 0;
			totalProcessingTime += file.processing_time_ms || 0;
			
			if (statusCounts.hasOwnProperty(file.status)) {
				statusCounts[file.status]++;
			}
		}

		const totalMinutes = totalDuration / 60;
		const totalHours = totalMinutes / 60;
		const avgProcessingTime = totalFiles > 0 ? totalProcessingTime / totalFiles : 0;
		const successRate = totalFiles > 0 ? (statusCounts.completed / totalFiles) * 100 : 0;

		return {
			total_files: totalFiles,
			total_duration_seconds: totalDuration,
			total_duration_minutes: totalMinutes,
			total_duration_hours: totalHours,
			total_file_size_bytes: totalFileSize,
			total_file_size_mb: totalFileSize / (1024 * 1024),
			total_processing_time_ms: totalProcessingTime,
			avg_processing_time_ms: avgProcessingTime,
			status_breakdown: statusCounts,
			success_rate: successRate,
		};
	}

	function getNextMonth(month: string): string {
		const date = new Date(month + '-01');
		date.setMonth(date.getMonth() + 1);
		return date.toISOString().slice(0, 7);
	}

	function formatDuration(seconds: number): string {
		if (seconds < 60) {
			return `${Math.round(seconds)}s`;
		} else if (seconds < 3600) {
			return `${Math.round(seconds / 60)}m`;
		} else {
			const hours = Math.floor(seconds / 3600);
			const minutes = Math.round((seconds % 3600) / 60);
			return `${hours}h ${minutes}m`;
		}
	}

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes}B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
		if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
		return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)}GB`;
	}

	function formatDate(dateString: string): string {
		if (!dateString) return 'Unknown';
		
		const date = new Date(dateString);
		
		// Check if date is valid
		if (isNaN(date.getTime())) {
			console.warn('Invalid date received:', dateString);
			return 'Invalid Date';
		}
		
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
		});
	}

	function getStatusIcon(status: string) {
		switch (status) {
			case 'completed': return CheckCircle2;
			case 'processing': return Loader2;
			case 'failed': return AlertCircle;
			default: return AlertCircle;
		}
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'completed': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
			case 'processing': return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
			case 'failed': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
			default: return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
		}
	}
</script>

<svelte:head>
	<title>Usage Statistics</title>
	<meta name="description" content="View your video processing usage statistics and history" />
</svelte:head>

<!-- Hero Section -->
<section class="py-20 px-6">
	<div class="max-w-4xl mx-auto">
		<div class="flex items-center gap-3 mb-6">
			<BarChart class="h-8 w-8 text-primary" />
			<h1 class="text-4xl md:text-5xl font-bold">Usage Statistics</h1>
		</div>
		<p class="text-xl text-muted-foreground mb-6">
			Track your video processing usage and history
		</p>
		<div class="border rounded-lg p-4 bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800">
			<div class="flex items-start gap-3 text-sm text-green-800 dark:text-green-200">
				<Shield class="h-4 w-4 mt-0.5 flex-shrink-0" />
				<p>
					<strong>Privacy First:</strong> All audio and video processing happens locally on your machine. 
					We never store your files on our servers - only processing metadata is tracked for usage statistics.
				</p>
			</div>
		</div>
	</div>
</section>

{#if isLoading}
	<section class="py-20 px-6">
		<div class="max-w-4xl mx-auto">
			<div class="flex items-center justify-center min-h-[400px]">
				<div class="flex items-center gap-3 text-muted-foreground">
					<Loader2 class="h-6 w-6 animate-spin" />
					<span class="text-lg">Loading usage statistics...</span>
				</div>
			</div>
		</div>
	</section>
{:else if error}
	<section class="py-20 px-6">
		<div class="max-w-4xl mx-auto">
			<div class="rounded-lg border border-red-200 bg-red-50 p-6 text-red-600 dark:border-red-900 dark:bg-red-950 dark:text-red-400">
				<div class="flex items-center gap-3 mb-4">
					<AlertCircle class="h-6 w-6" />
					<span class="text-lg font-semibold">Error loading usage data: {error}</span>
				</div>
				<Button onclick={loadUsageData} variant="outline">
					Retry
				</Button>
			</div>
		</div>
	</section>
{:else}
	<!-- Summary Cards Section -->
	<section class="py-20 border-t px-6">
		<div class="max-w-4xl mx-auto">
			<h2 class="text-3xl md:text-4xl font-bold mb-8">Overview</h2>
			<div class="grid gap-6 md:grid-cols-3">
				<!-- This Month Card -->
				<div class="border rounded-lg p-6">
					<div class="flex items-center justify-between mb-3">
						<span class="text-sm font-medium text-muted-foreground">This Month</span>
						<Calendar class="h-4 w-4 text-muted-foreground" />
					</div>
					<div class="text-2xl font-bold">{currentMonthSummary ? formatDuration(currentMonthSummary.total_duration_seconds) : '0s'}</div>
				</div>

				<!-- Success Rate Card -->
				<div class="border rounded-lg p-6">
					<div class="flex items-center justify-between mb-3">
						<span class="text-sm font-medium text-muted-foreground">Success Rate</span>
						<TrendingUp class="h-4 w-4 text-muted-foreground" />
					</div>
					<div class="text-2xl font-bold">{currentMonthSummary ? currentMonthSummary.success_rate.toFixed(1) : '0.0'}%</div>
				</div>

				<!-- Total Processing Card -->
				<div class="border rounded-lg p-6">
					<div class="flex items-center justify-between mb-3">
						<span class="text-sm font-medium text-muted-foreground">Total Processing</span>
						<Clock class="h-4 w-4 text-muted-foreground" />
					</div>
					<div class="text-2xl font-bold">{allTimeSummary ? formatDuration(allTimeSummary.total_duration_seconds) : '0s'}</div>
				</div>
			</div>
		</div>
	</section>

	<!-- Recent Files Section -->
	<section class="py-20 border-t px-6">
		<div class="max-w-4xl mx-auto">
			<h2 class="text-3xl md:text-4xl font-bold mb-8">Recent Processing History</h2>
			
			<div class="border rounded-lg overflow-hidden">
				<div class="p-6">
				
				{#if processedFiles.length === 0}
					<div class="text-center py-12 text-muted-foreground">
						<FileAudio class="h-8 w-8 mx-auto mb-4 opacity-50" />
						<p class="font-medium mb-2">No files processed yet</p>
						<p class="text-sm">Your transcription history will appear here once you start processing audio files.</p>
					</div>
				{:else}
					<div class="overflow-x-auto">
						<table class="w-full">
							<thead>
								<tr class="border-b text-left">
									<th class="pb-3 text-sm font-medium text-muted-foreground">File</th>
									<th class="pb-3 text-sm font-medium text-muted-foreground">Duration</th>
									<th class="pb-3 text-sm font-medium text-muted-foreground">Size</th>
									<th class="pb-3 text-sm font-medium text-muted-foreground">Status</th>
									<th class="pb-3 text-sm font-medium text-muted-foreground">Processed</th>
								</tr>
							</thead>
							<tbody>
								{#each processedFiles as file}
									<tr class="border-b border-border/50 last:border-0">
										<td class="py-4">
											<div class="flex items-center gap-3">
												<FileAudio class="h-4 w-4 text-muted-foreground" />
												<div>
													<div class="font-medium text-sm">{file.filename}</div>
													<div class="text-xs text-muted-foreground">
														{file.words_count} words, {file.transcript_length} chars
													</div>
												</div>
											</div>
										</td>
										<td class="py-4 text-sm">
											{formatDuration(file.duration_seconds)}
										</td>
										<td class="py-4 text-sm">
											{formatFileSize(file.file_size_bytes)}
										</td>
										<td class="py-4">
											<Badge class={`${getStatusColor(file.status)} border-0`}>
												{@const IconComponent = getStatusIcon(file.status)}
												<IconComponent class="h-3 w-3 mr-1 {file.status === 'processing' ? 'animate-spin' : ''}" />
												{file.status}
											</Badge>
										</td>
										<td class="py-4 text-sm text-muted-foreground">
											{formatDate(file.created)}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
				</div>
			</div>
		</div>
	</section>
{/if}