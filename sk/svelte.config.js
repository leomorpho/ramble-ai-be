import adapter from '@sveltejs/adapter-cloudflare';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			// Configure for Cloudflare Pages static deployment
			fallback: 'index.html'
		}),
		alias: {
			'@/*': 'src/lib/*'
		}
	}
};

export default config;
