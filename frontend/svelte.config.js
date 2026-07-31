import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),

	// @lucide/svelte 以 .svelte 源文件发布，需显式纳入编译
	//（vite-plugin-svelte 默认排除 node_modules）
	vitePlugin: {
		include: [/\.svelte$/, /node_modules\/@lucide\/svelte\/.*\.svelte$/],
		exclude: [/\.svelte\.md$/]
	},

	kit: {
		// SPA 模式：全部页面客户端渲染，nginx try_files 回退 index.html
		adapter: adapter({ fallback: 'index.html' }),
		alias: {
			'@/*': './src/*'
		}
	}
};

export default config;
