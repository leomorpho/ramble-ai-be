<script lang="ts">
	import { pb } from '$lib/pocketbase';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { Download, Monitor, Apple, Loader2, ExternalLink, AlertCircle } from 'lucide-svelte';
	import { onMount } from 'svelte';

	interface AppVersion {
		id: string;
		version: string;
		platform: 'windows' | 'macos' | 'linux';
		architecture: string;
		download_url: string;
		file_size: number;
		checksum_sha256: string;
		release_notes?: string;
		is_latest: boolean;
		is_released: boolean;
		is_prerelease: boolean;
		minimum_os_version?: string;
		download_count: number;
		created: string;
		updated: string;
	}

	let versions = $state<AppVersion[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let downloading = $state<string | null>(null);

	// Detect user's platform
	function detectPlatform(): 'windows' | 'macos' | 'linux' {
		const userAgent = navigator.userAgent.toLowerCase();
		if (userAgent.indexOf('win') !== -1) return 'windows';
		if (userAgent.indexOf('mac') !== -1) return 'macos';
		return 'linux';
	}

	const userPlatform = detectPlatform();

	// Platform icons and names
	const platformInfo = {
		windows: { icon: Monitor, name: 'Windows', ext: '.exe' },
		macos: { icon: Apple, name: 'macOS', ext: '.dmg' },
		linux: { icon: Monitor, name: 'Linux', ext: '.AppImage' }
	};

	// Format file size
	function formatFileSize(bytes: number): string {
		const mb = bytes / (1024 * 1024);
		return `${mb.toFixed(1)} MB`;
	}

	// Group versions by platform
	function groupVersionsByPlatform(versions: AppVersion[]): Record<string, AppVersion[]> {
		const grouped: Record<string, AppVersion[]> = {};
		versions.forEach(version => {
			if (!grouped[version.platform]) {
				grouped[version.platform] = [];
			}
			grouped[version.platform].push(version);
		});
		return grouped;
	}

	// Load versions from PocketBase
	async function loadVersions() {
		try {
			loading = true;
			error = null;
			
			const records = await pb.collection('app_versions').getList<AppVersion>(1, 50, {
				filter: 'is_released = true && is_latest = true',
				sort: '-created'
			});
			
			versions = records.items;
		} catch (err) {
			console.error('Failed to load versions:', err);
			error = 'Failed to load download links. Please try again later.';
		} finally {
			loading = false;
		}
	}

	// Handle download
	async function handleDownload(version: AppVersion) {
		downloading = version.id;
		
		try {
			// Increment download count
			await pb.collection('app_versions').update(version.id, {
				'download_count+': 1
			});
			
			// Open download URL
			window.open(version.download_url, '_blank');
		} catch (err) {
			console.error('Download error:', err);
		} finally {
			setTimeout(() => {
				downloading = null;
			}, 2000);
		}
	}

	onMount(() => {
		loadVersions();
	});

	let groupedVersions = $derived(groupVersionsByPlatform(versions));
	let userPlatformVersions = $derived(groupedVersions[userPlatform] || []);
</script>

<svelte:head>
	<title>Download Ramble • AI Script Optimization</title>
	<meta name="description" content="Download Ramble for Windows, macOS, or Linux and start optimizing your video scripts today." />
</svelte:head>

<div class="container mx-auto px-4 py-12 max-w-6xl">
	<div class="text-center mb-12">
		<h1 class="text-4xl md:text-5xl font-bold mb-4">Download Ramble</h1>
		<p class="text-lg text-muted-foreground max-w-2xl mx-auto">
			Transform your rambling videos into compelling scripts. Available for Windows, macOS, and Linux.
		</p>
	</div>

	{#if loading}
		<div class="grid gap-6 md:grid-cols-3">
			{#each ['windows', 'macos', 'linux'] as platform}
				<Card>
					<CardHeader>
						<Skeleton class="h-6 w-24 mb-2" />
						<Skeleton class="h-4 w-32" />
					</CardHeader>
					<CardContent>
						<Skeleton class="h-10 w-full mb-4" />
						<Skeleton class="h-4 w-20" />
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else if error}
		<div class="flex items-center gap-2 p-4 rounded-lg border border-destructive bg-destructive/10 text-destructive">
			<AlertCircle class="h-4 w-4" />
			<p>{error}</p>
		</div>
	{:else if versions.length === 0}
		<div class="flex items-center gap-2 p-4 rounded-lg border bg-muted/50">
			<AlertCircle class="h-4 w-4" />
			<p>No downloads available yet. Please check back later.</p>
		</div>
	{:else}
		<!-- Recommended download for user's platform -->
		{#if userPlatformVersions.length > 0}
			<div class="mb-8">
				<h2 class="text-2xl font-semibold mb-4">Recommended for your system</h2>
				<Card class="border-primary">
					<CardHeader>
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-3">
								<svelte:component this={platformInfo[userPlatform].icon} class="h-8 w-8" />
								<div>
									<CardTitle>{platformInfo[userPlatform].name}</CardTitle>
									<CardDescription>Version {userPlatformVersions[0].version}</CardDescription>
								</div>
							</div>
							{#if userPlatformVersions[0].is_prerelease}
								<Badge variant="secondary">Beta</Badge>
							{/if}
						</div>
					</CardHeader>
					<CardContent>
						<div class="space-y-4">
							{#each userPlatformVersions as version}
								<div class="flex items-center justify-between p-3 rounded-lg bg-muted/50">
									<div>
										<p class="font-medium">
											{version.architecture === 'universal' ? 'Universal' : version.architecture.toUpperCase()}
										</p>
										<p class="text-sm text-muted-foreground">
											{formatFileSize(version.file_size)}
										</p>
									</div>
									<Button 
										onclick={() => handleDownload(version)}
										disabled={downloading === version.id}
									>
										{#if downloading === version.id}
											<Loader2 class="mr-2 h-4 w-4 animate-spin" />
											Downloading...
										{:else}
											<Download class="mr-2 h-4 w-4" />
											Download
										{/if}
									</Button>
								</div>
							{/each}
							{#if userPlatformVersions[0].minimum_os_version}
								<p class="text-xs text-muted-foreground">
									Requires {platformInfo[userPlatform].name} {userPlatformVersions[0].minimum_os_version} or later
								</p>
							{/if}
						</div>
					</CardContent>
				</Card>
			</div>
		{/if}

		<!-- All platforms -->
		<div>
			<h2 class="text-2xl font-semibold mb-4">All platforms</h2>
			<div class="grid gap-6 md:grid-cols-3">
				{#each ['windows', 'macos', 'linux'] as platform}
					{@const platformVersions = groupedVersions[platform] || []}
					<Card class={userPlatform === platform ? 'opacity-60' : ''}>
						<CardHeader>
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-3">
									<svelte:component this={platformInfo[platform].icon} class="h-6 w-6" />
									<div>
										<CardTitle class="text-lg">{platformInfo[platform].name}</CardTitle>
										{#if platformVersions.length > 0}
											<CardDescription>Version {platformVersions[0].version}</CardDescription>
										{/if}
									</div>
								</div>
								{#if platformVersions[0]?.is_prerelease}
									<Badge variant="secondary" class="text-xs">Beta</Badge>
								{/if}
							</div>
						</CardHeader>
						<CardContent>
							{#if platformVersions.length === 0}
								<p class="text-sm text-muted-foreground">Coming soon</p>
							{:else}
								<div class="space-y-3">
									{#each platformVersions as version}
										<div class="space-y-2">
											<Button 
												onclick={() => handleDownload(version)}
												disabled={downloading === version.id}
												variant={userPlatform === platform ? "secondary" : "default"}
												class="w-full"
											>
												{#if downloading === version.id}
													<Loader2 class="mr-2 h-4 w-4 animate-spin" />
													Downloading...
												{:else}
													<Download class="mr-2 h-4 w-4" />
													{version.architecture === 'universal' ? 'Download' : version.architecture.toUpperCase()}
												{/if}
											</Button>
											<p class="text-xs text-muted-foreground text-center">
												{formatFileSize(version.file_size)}
												{#if version.minimum_os_version}
													• Requires {version.minimum_os_version}+
												{/if}
											</p>
										</div>
									{/each}
								</div>
							{/if}
						</CardContent>
					</Card>
				{/each}
			</div>
		</div>

		<!-- System requirements -->
		<div class="mt-12 p-6 rounded-lg bg-muted/50">
			<h3 class="text-lg font-semibold mb-3">System Requirements</h3>
			<div class="grid gap-4 md:grid-cols-3 text-sm">
				<div>
					<p class="font-medium mb-1">Windows</p>
					<ul class="text-muted-foreground space-y-1">
						<li>• Windows 10 or later</li>
						<li>• 4GB RAM minimum</li>
						<li>• 500MB disk space</li>
					</ul>
				</div>
				<div>
					<p class="font-medium mb-1">macOS</p>
					<ul class="text-muted-foreground space-y-1">
						<li>• macOS 11 Big Sur or later</li>
						<li>• Apple Silicon or Intel</li>
						<li>• 4GB RAM minimum</li>
					</ul>
				</div>
				<div>
					<p class="font-medium mb-1">Linux</p>
					<ul class="text-muted-foreground space-y-1">
						<li>• Ubuntu 20.04+ or equivalent</li>
						<li>• 4GB RAM minimum</li>
						<li>• X11 or Wayland</li>
					</ul>
				</div>
			</div>
		</div>

		<!-- Installation help -->
		<div class="mt-8 text-center">
			<p class="text-sm text-muted-foreground mb-2">
				Need help installing? Check out our
			</p>
			<Button variant="link" onclick={() => window.open('https://docs.ramble.ai/installation', '_blank')}>
				<ExternalLink class="mr-2 h-3 w-3" />
				Installation Guide
			</Button>
		</div>
	{/if}
</div>