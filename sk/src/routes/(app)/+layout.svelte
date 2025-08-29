<script lang="ts">
	let { children } = $props();
</script>

<div class="container mx-auto px-4 py-8">
	{@render children()}
</div>