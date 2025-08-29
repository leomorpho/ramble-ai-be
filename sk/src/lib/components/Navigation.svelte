<script lang="ts">
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import DownloadButton from '$lib/components/DownloadButton.svelte';
	import { Home, LogIn, LogOut, User, CreditCard, Crown, BarChart } from 'lucide-svelte';
	import type { AuthModel } from 'pocketbase';
	import { config } from '$lib/config.js';
	import { getAvatarUrl } from '$lib/files.js';

	let {
		isLoggedIn = false,
		user = null,
		isSubscribed = false,
		onLogout
	}: {
		isLoggedIn?: boolean;
		user?: AuthModel | null;
		isSubscribed?: boolean;
		onLogout?: () => void;
	} = $props();

	function handleLogout() {
		onLogout?.();
	}
</script>

<header class="bg-background/80 fixed left-0 right-0 top-0 z-50 border-b backdrop-blur-lg">
	<div class="container mx-auto px-4 py-4">
		<nav class="flex items-center justify-between">
			<div class="flex items-center space-x-4">
				<a
					href="/"
					class="flex cursor-pointer items-center space-x-3 transition-opacity hover:opacity-80"
				>
					<img src="/logo-128.png" alt="Ramble logo" class="h-8 w-8 rounded-lg" />
					<span class="text-xl font-bold tracking-tight">
						<span class="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent"
							>RAMBLE</span
						>
					</span>
				</a>
			</div>

			<div class="flex items-center space-x-2">
				{#if isLoggedIn}
					<a
						href="/usages"
						class="hover:bg-accent hover:text-accent-foreground inline-flex h-9 w-9 items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors"
						title="Usage Statistics"
					>
						<BarChart class="h-4 w-4" />
					</a>
					<a
						href="/dashboard"
						class="hover:bg-accent hover:text-accent-foreground flex items-center space-x-2 rounded-md px-2 py-1 text-sm transition-colors"
					>
						{#if getAvatarUrl(user, 'small')}
							<img
								src={getAvatarUrl(user, 'small')}
								alt="Profile"
								class="border-border h-6 w-6 rounded-full border object-cover"
							/>
						{:else}
							<div
								class="bg-muted border-border flex h-6 w-6 items-center justify-center rounded-full border"
							>
								<User class="text-muted-foreground h-3 w-3" />
							</div>
						{/if}
						<span class="hidden sm:inline">{user?.name || user?.email}</span>
					</a>
					<button
						onclick={handleLogout}
						class="hover:bg-accent hover:text-accent-foreground inline-flex h-9 w-9 items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors"
						title="Sign Out"
					>
						<LogOut class="h-4 w-4" />
					</button>
				{:else}
					<a
						href="/login"
						class="hover:bg-accent hover:text-accent-foreground inline-flex items-center space-x-2 whitespace-nowrap rounded-md px-3 py-2 text-sm font-medium transition-colors"
						title="Sign In"
					>
						<LogIn class="h-4 w-4" />
						<span class="hidden sm:inline">Login/Register</span>
					</a>
				{/if}

				<ThemeToggle />
			</div>
		</nav>
	</div>
</header>
