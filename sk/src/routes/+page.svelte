<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import RambleLogo from '$lib/components/RambleLogo.svelte';
	import DownloadButton from '$lib/components/DownloadButton.svelte';
	import { config } from '$lib/config.ts';
	import { Check, Crown, Zap } from 'lucide-svelte';
	import { goto } from '$app/navigation';

	let billingInterval = $state<'month' | 'year'>('month');

	const faqs = [
		{
			question: 'How is Ramble different from traditional video editors?',
			answer: 'Ramble is a preprocessing tool, not a video editor. We optimize your talking head content into perfect scripts BEFORE editing.'
		},
		{
			question: 'How much time will Ramble save me?',
			answer: 'Most users report 60-80% reduction in post-production time.'
		},
		{
			question: 'What do I get when I export?',
			answer: 'Optimized script files with precise timestamps, individual video clips ready for assembly, and transcripts.'
		},
		{
			question: 'How good is the AI?',
			answer: '90%+ accuracy in creating coherent scripts from disorganized footage.'
		},
		{
			question: 'Is my content secure?',
			answer: 'Yes. Everything processes locally on your computer - nothing uploads to servers.'
		}
	];
</script>

<svelte:head>
	<title>Ramble • AI Script Optimization for Talking Head Videos</title>
	<meta name="description" content="Transform rambling videos into compelling scripts. Cut editing time by 60%." />
	<meta property="og:title" content="Ramble • AI Script Optimization for Talking Head Videos" />
	<meta property="og:description" content="Transform rambling videos into compelling scripts. Cut editing time by 60%." />
	<meta property="og:image" content="/og-image.png" />
	<meta property="og:type" content="website" />
</svelte:head>

<!-- Hero -->
<section class="min-h-[70vh] flex items-center px-6">
	<div class="max-w-4xl mx-auto">
		<h1 class="text-5xl md:text-7xl font-bold mb-6">
			Turn rambling into compelling
		</h1>
		<p class="text-xl md:text-2xl text-muted-foreground mb-8 max-w-2xl">
			AI finds your best clips and reorders them into perfect scripts. Cut editing time by 60%.
		</p>
		<div class="flex flex-col sm:flex-row gap-4">
			<DownloadButton text="Download Ramble" />
			<Button 
				variant="outline" 
				size="lg"
				onclick={() => document.getElementById('demo-video')?.scrollIntoView({ behavior: 'smooth' })}
			>
				See How It Works
			</Button>
		</div>
	</div>
</section>

<!-- Product Demo Video -->
<section id="demo-video" class="py-20 border-t">
	<div class="max-w-4xl mx-auto px-6">
		<h2 class="text-3xl md:text-4xl font-bold mb-8">Watch it in action</h2>
		<div class="aspect-video bg-black rounded-lg overflow-hidden">
			<iframe 
				src="https://www.youtube.com/embed/dQw4w9WgXcQ"
				title="Ramble Product Demo"
				class="w-full h-full"
				frameborder="0"
				allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
				allowfullscreen
			></iframe>
		</div>
	</div>
</section>

<!-- How it works -->
<section class="py-20 border-t">
	<div class="max-w-4xl mx-auto px-6">
		<h2 class="text-3xl md:text-4xl font-bold mb-12">Simple workflow</h2>
		<div class="space-y-8">
			<div class="flex gap-4">
				<div class="w-8 h-8 rounded-full bg-foreground text-background flex items-center justify-center font-bold shrink-0">1</div>
				<div>
					<h3 class="font-semibold mb-1">Upload Footage</h3>
					<p class="text-muted-foreground">Upload your raw talking head video. AI analyzes speech patterns and content.</p>
				</div>
			</div>
			<div class="flex gap-4">
				<div class="w-8 h-8 rounded-full bg-foreground text-background flex items-center justify-center font-bold shrink-0">2</div>
				<div>
					<h3 class="font-semibold mb-1">Smart Selection</h3>
					<p class="text-muted-foreground">AI identifies the best clips based on quality and narrative value.</p>
				</div>
			</div>
			<div class="flex gap-4">
				<div class="w-8 h-8 rounded-full bg-foreground text-background flex items-center justify-center font-bold shrink-0">3</div>
				<div>
					<h3 class="font-semibold mb-1">Script Reordering</h3>
					<p class="text-muted-foreground">AI reorders selected clips into a coherent, engaging script.</p>
				</div>
			</div>
			<div class="flex gap-4">
				<div class="w-8 h-8 rounded-full bg-foreground text-background flex items-center justify-center font-bold shrink-0">4</div>
				<div>
					<h3 class="font-semibold mb-1">Export & Handoff</h3>
					<p class="text-muted-foreground">Export optimized scripts and clips to your favorite video editor.</p>
				</div>
			</div>
		</div>
	</div>
</section>

<!-- Features -->
<section class="py-20 border-t">
	<div class="max-w-4xl mx-auto px-6">
		<h2 class="text-3xl md:text-4xl font-bold mb-12">Features</h2>
		<div class="grid md:grid-cols-2 gap-8">
			<div>
				<h3 class="font-semibold mb-2">Smart Clip Selection</h3>
				<p class="text-muted-foreground text-sm">AI identifies the best parts of your talking head videos automatically.</p>
			</div>
			<div>
				<h3 class="font-semibold mb-2">AI Script Reordering</h3>
				<p class="text-muted-foreground text-sm">Transform clips into coherent scripts using any AI model.</p>
			</div>
			<div>
				<h3 class="font-semibold mb-2">Speech Transcription</h3>
				<p class="text-muted-foreground text-sm">Word-perfect transcripts with precise timing.</p>
			</div>
			<div>
				<h3 class="font-semibold mb-2">Workflow Integration</h3>
				<p class="text-muted-foreground text-sm">Fits perfectly as the first step in your editing workflow.</p>
			</div>
			<div>
				<h3 class="font-semibold mb-2">Export & Handoff</h3>
				<p class="text-muted-foreground text-sm">Export optimized scripts ready for your editor.</p>
			</div>
			<div>
				<h3 class="font-semibold mb-2">60-80% Time Savings</h3>
				<p class="text-muted-foreground text-sm">Dramatically reduce post-production time.</p>
			</div>
		</div>
	</div>
</section>

<!-- AI Flexibility -->
<section class="py-20 border-t">
	<div class="max-w-4xl mx-auto px-6">
		<h2 class="text-3xl md:text-4xl font-bold mb-8">Your AI, Your Choice</h2>
		<p class="text-muted-foreground mb-6">Works with any AI model through OpenRouter. Bring your own API keys.</p>
		<ul class="space-y-2 text-sm">
			<li>• No subscription lock-in</li>
			<li>• Control your AI costs and usage</li>
			<li>• 200+ AI Models (GPT-4, Claude, Gemini, Llama)</li>
			<li>• Built-in guide walks you through setup in under 5 minutes</li>
		</ul>
	</div>
</section>

<!-- Pricing -->
<section class="py-20 border-t">
	<div class="max-w-6xl mx-auto px-6">
		<div class="text-center mb-12">
			<h2 class="text-3xl md:text-4xl font-bold mb-4">Simple, Transparent Pricing</h2>
			<p class="text-muted-foreground">Choose the plan that fits your workflow</p>
		</div>

		<!-- Billing Toggle -->
		<div class="flex justify-center mb-12">
			<div class="flex items-center bg-muted p-1 rounded-lg">
				<button
					class="px-6 py-2 rounded-md text-sm font-medium transition-colors {billingInterval === 'month' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
					onclick={() => billingInterval = 'month'}
				>
					Monthly
				</button>
				<button
					class="px-6 py-2 rounded-md text-sm font-medium transition-colors {billingInterval === 'year' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
					onclick={() => billingInterval = 'year'}
				>
					Yearly
					<Badge variant="secondary" class="ml-2">Save 20%</Badge>
				</button>
			</div>
		</div>
		
		<div class="grid md:grid-cols-3 gap-6 max-w-5xl mx-auto">
			<!-- Free Plan -->
			<Card class="relative">
				<CardHeader class="text-center">
					<div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
						<Check class="h-6 w-6 text-primary" />
					</div>
					<CardTitle class="text-2xl">Free</CardTitle>
					<div class="mt-4">
						<div class="text-4xl font-bold">$0</div>
						<div class="text-muted-foreground">Always free</div>
					</div>
				</CardHeader>
				<CardContent class="space-y-6">
					<div class="text-center">
						<div class="text-2xl font-semibold text-primary">1 hour</div>
						<div class="text-sm text-muted-foreground">of media processing per month</div>
					</div>

					<ul class="space-y-3">
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">1 hour of media processing</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Unlimited video quality exports</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Basic support</span>
						</li>
					</ul>

					<Button class="w-full" variant="outline" onclick={() => goto('/login?tab=signup')}>
						Get Started
					</Button>
				</CardContent>
			</Card>

			<!-- Basic Plan -->
			<Card class="relative ring-2 ring-primary">
				<Badge class="absolute -top-3 left-1/2 -translate-x-1/2">
					Most Popular
				</Badge>
				<CardHeader class="text-center">
					<div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
						<Zap class="h-6 w-6 text-primary" />
					</div>
					<CardTitle class="text-2xl">Basic</CardTitle>
					<div class="mt-4">
						{#if billingInterval === 'month'}
							<div class="text-4xl font-bold">$5</div>
							<div class="text-muted-foreground">per month</div>
						{:else}
							<div class="text-4xl font-bold">$48</div>
							<div class="text-muted-foreground">per year</div>
							<div class="text-sm text-green-600 mt-1">
								($4 per month)
							</div>
						{/if}
					</div>
				</CardHeader>
				<CardContent class="space-y-6">
					<div class="text-center">
						<div class="text-2xl font-semibold text-primary">10 hours</div>
						<div class="text-sm text-muted-foreground">of media processing per month</div>
					</div>

					<ul class="space-y-3">
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">10 hours of media processing</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Unlimited video quality exports</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Email support</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Priority processing</span>
						</li>
						{#if billingInterval === 'year'}
							<li class="flex items-center">
								<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
								<span class="text-sm">2 months free</span>
							</li>
						{/if}
					</ul>

					<Button class="w-full" onclick={() => goto('/login?tab=signup')}>
						Get Started
					</Button>
				</CardContent>
			</Card>

			<!-- Pro Plan -->
			<Card class="relative">
				<CardHeader class="text-center">
					<div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
						<Crown class="h-6 w-6 text-primary" />
					</div>
					<CardTitle class="text-2xl">Pro</CardTitle>
					<div class="mt-4">
						{#if billingInterval === 'month'}
							<div class="text-4xl font-bold">$15</div>
							<div class="text-muted-foreground">per month</div>
						{:else}
							<div class="text-4xl font-bold">$144</div>
							<div class="text-muted-foreground">per year</div>
							<div class="text-sm text-green-600 mt-1">
								($12 per month)
							</div>
						{/if}
					</div>
				</CardHeader>
				<CardContent class="space-y-6">
					<div class="text-center">
						<div class="text-2xl font-semibold text-primary">30 hours</div>
						<div class="text-sm text-muted-foreground">of media processing per month</div>
					</div>

					<ul class="space-y-3">
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">30 hours of media processing</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Unlimited video quality exports</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Priority support</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Advanced AI models</span>
						</li>
						<li class="flex items-center">
							<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
							<span class="text-sm">Bulk processing</span>
						</li>
						{#if billingInterval === 'year'}
							<li class="flex items-center">
								<Check class="h-4 w-4 text-green-600 mr-3 flex-shrink-0" />
								<span class="text-sm">2 months free</span>
							</li>
						{/if}
					</ul>

					<Button class="w-full" onclick={() => goto('/login?tab=signup')}>
						Get Started
					</Button>
				</CardContent>
			</Card>
		</div>
	</div>
</section>

<!-- FAQ -->
<section class="py-20 border-t">
	<div class="max-w-4xl mx-auto px-6">
		<h2 class="text-3xl md:text-4xl font-bold mb-12">FAQ</h2>
		<div class="space-y-6">
			{#each faqs as faq}
				<details class="border-b pb-4">
					<summary class="font-semibold cursor-pointer hover:text-primary">{faq.question}</summary>
					<p class="text-muted-foreground mt-2 text-sm">{faq.answer}</p>
				</details>
			{/each}
		</div>
	</div>
</section>

<!-- CTA -->
<section class="py-20 border-t">
	<div class="max-w-4xl mx-auto px-6 text-center">
		<h2 class="text-3xl md:text-4xl font-bold mb-4">Ready to optimize your scripts?</h2>
		<p class="text-muted-foreground mb-8">Join creators who save 60% of their editing time.</p>
		<DownloadButton text="Download Ramble" />
	</div>
</section>

<!-- Footer -->
<footer class="border-t py-12">
	<div class="max-w-4xl mx-auto px-6">
		<div class="flex flex-col md:flex-row justify-between items-center gap-4">
			<div class="flex items-center gap-2">
				<RambleLogo href="/" showText={false} />
				<span class="font-bold">RAMBLE</span>
			</div>
			<a href="mailto:support@ramble.app" class="text-primary hover:underline">
				support@ramble.app
			</a>
			<p class="text-sm text-muted-foreground">© {config.getCurrentYear()} Ramble</p>
		</div>
	</div>
</footer>