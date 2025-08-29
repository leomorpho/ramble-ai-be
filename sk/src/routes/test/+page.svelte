<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
</script>

<div class="p-8 space-y-4">
	<h1 class="text-4xl font-bold">Test Page</h1>
	
	<Badge>Test Badge</Badge>
	
	<Button>Test Button</Button>
	
	<Card>
		<CardContent class="p-6">
			<p>Test Card Content</p>
		</CardContent>
	</Card>
	
	<div class="w-full h-96 bg-gradient-to-r from-blue-400 to-purple-600 rounded-lg flex items-center justify-center">
		<p class="text-white text-2xl font-bold">Gradient Background Test</p>
	</div>
</div>