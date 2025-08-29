<script lang="ts">
	import { authStore } from '$lib/stores/authClient.svelte.js';
	import { config } from '$lib/config.js';
</script>

<svelte:head>
	<title>Dashboard - {config.app.name}</title>
	<meta name="description" content="User dashboard" />
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<div class="mx-auto max-w-6xl">
		<!-- Header -->
		<div class="mb-8">
			<h1 class="text-4xl font-bold text-foreground mb-2">Dashboard</h1>
			<p class="text-muted-foreground">Welcome back, {authStore.user?.name || authStore.user?.email || 'User'}</p>
		</div>

		<div class="grid gap-8 lg:grid-cols-3">
			<!-- Profile Section -->
			<div class="lg:col-span-1">
				<div class="bg-card rounded-xl border border-border p-6 shadow-sm">
					<div class="text-center">
						<div class="w-24 h-24 rounded-full bg-muted border-4 border-background shadow-lg flex items-center justify-center mx-auto mb-4">
							<span class="text-2xl">👤</span>
						</div>

						<!-- User Name & Email -->
						<div class="space-y-2">
							<h2 class="text-xl font-semibold text-foreground">
								{authStore.user?.name || 'User'}
							</h2>
							<p class="text-muted-foreground">
								{authStore.user?.email}
							</p>
						</div>
					</div>
				</div>
			</div>

			<!-- Main Content -->
			<div class="lg:col-span-2 space-y-6">
				<!-- Welcome Message -->
				<div class="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-950/30 dark:to-indigo-950/30 rounded-xl border border-blue-200 dark:border-blue-800/50 p-6">
					<h3 class="text-lg font-semibold text-blue-900 dark:text-blue-100 mb-2">Welcome to your Dashboard!</h3>
					<p class="text-blue-700 dark:text-blue-300 mb-4">
						This is your personal space where you can manage your account and access features.
					</p>
					<div class="flex flex-wrap gap-2">
						<span class="px-3 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-800 dark:text-blue-200 rounded-full text-sm">
							✨ Simple Dashboard
						</span>
						<span class="px-3 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-800 dark:text-blue-200 rounded-full text-sm">
							🔒 Authenticated
						</span>
						<span class="px-3 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-800 dark:text-blue-200 rounded-full text-sm">
							⚡ Working
						</span>
					</div>
				</div>

				<!-- Debug Info -->
				<div class="bg-card rounded-xl border border-border p-6 shadow-sm">
					<h3 class="text-lg font-semibold text-foreground mb-4">User Info</h3>
					<pre class="text-xs bg-muted p-3 rounded overflow-auto">{JSON.stringify(authStore.user, null, 2)}</pre>
				</div>
			</div>
		</div>
	</div>
</div>