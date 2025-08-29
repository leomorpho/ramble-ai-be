import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			// Configure as SPA with fallback to index.html
			fallback: 'index.html',
			precompress: false,
			strict: false
		}),
		alias: {
			'@/*': 'src/lib/*'
		}
	}
};

export default config;
