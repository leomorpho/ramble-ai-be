<script lang="ts">
	import ErrorTestComponent from '$lib/components/ErrorTestComponent.svelte';
	import * as m from '$lib/paraglide/messages.js';
</script>

<svelte:head>
	<title>Sentry Error Monitoring Test</title>
</svelte:head>

<div class="max-w-4xl mx-auto p-8">
	<div class="text-center mb-8">
		<h1 class="text-3xl font-bold tracking-tight mb-4">
			Sentry Error Monitoring Test
		</h1>
		<p class="text-muted-foreground text-lg">
			Test the error monitoring integration with various error scenarios
		</p>
	</div>

	<div class="space-y-8">
		<!-- Test Component -->
		<ErrorTestComponent />
		
		<!-- Setup Instructions -->
		<div class="border rounded-lg p-6 bg-muted/50">
			<h2 class="text-xl font-semibold mb-4">Setup Instructions</h2>
			<div class="space-y-4">
				<div>
					<h3 class="font-medium mb-2">1. Create a Sentry Project</h3>
					<p class="text-sm text-muted-foreground">
						Go to <a href="https://sentry.io" class="text-primary hover:underline" target="_blank">sentry.io</a> 
						and create a new SvelteKit project.
					</p>
				</div>
				
				<div>
					<h3 class="font-medium mb-2">2. Configure Environment Variables</h3>
					<p class="text-sm text-muted-foreground mb-2">
						Copy your DSN from Sentry and add it to your <code class="bg-muted px-1 py-0.5 rounded">.env</code> file:
					</p>
					<pre class="bg-black text-green-400 p-3 rounded text-xs overflow-x-auto"><code>VITE_SENTRY_DSN=https://your-actual-dsn@sentry.io/project-id
VITE_ENVIRONMENT=development</code></pre>
				</div>
				
				<div>
					<h3 class="font-medium mb-2">3. Test Error Monitoring</h3>
					<p class="text-sm text-muted-foreground">
						Use the buttons above to trigger different types of errors and logs. 
						Check your Sentry dashboard to see them captured.
					</p>
				</div>
			</div>
		</div>

		<!-- Features Overview -->
		<div class="grid md:grid-cols-2 gap-6">
			<div class="border rounded-lg p-6">
				<h3 class="font-semibold mb-3">✅ What's Included</h3>
				<ul class="space-y-2 text-sm text-muted-foreground">
					<li>• Console error capturing</li>
					<li>• Unhandled exception tracking</li>
					<li>• Structured logging with different levels</li>
					<li>• Session replay (10% sample rate)</li>
					<li>• Performance monitoring</li>
					<li>• User context tracking</li>
					<li>• Custom error boundaries</li>
				</ul>
			</div>
			
			<div class="border rounded-lg p-6">
				<h3 class="font-semibold mb-3">🛠 Usage in Your Code</h3>
				<pre class="bg-muted p-3 rounded text-xs overflow-x-auto"><code>import &#123; Logger &#125; from '$lib/logger';

// Log errors with context
Logger.error('Payment failed', error, &#123;
  userId: user.id,
  amount: paymentAmount
&#125;);

// Set user context
Logger.setUser(&#123; 
  id: user.id, 
  email: user.email 
&#125;);</code></pre>
			</div>
		</div>
	</div>
</div>