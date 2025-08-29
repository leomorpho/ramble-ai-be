import adapter from '@sveltejs/adapter-cloudflare';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			// Static mode - no server-side functions
			platformProxy: {
				persist: false
			}
		}),
		alias: {
			'@/*': 'src/lib/*'
		}
	}
};

export default config;
