import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			pages: '../backend/dist_frontend',
			assets: '../backend/dist_frontend',
			fallback: undefined,
			precompress: false,
			strict: true
		}),
		paths: {
			relative: false
		}
	}
};

export default config;
