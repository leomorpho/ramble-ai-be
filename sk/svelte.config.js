import adapter from '@sveltejs/adapter-cloudflare';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			// Enable static SPA mode for Cloudflare Pages
			routes: {
				include: ['/*'],
				exclude: []
			}
		}),
		alias: {
			'@/*': 'src/lib/*'
		}
	}
};

export default config;
