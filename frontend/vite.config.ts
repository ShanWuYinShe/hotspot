import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import tailwindcss from '@tailwindcss/vite';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit(), svelteTesting()],
	optimizeDeps: {
		// @lucide/svelte 发布 .svelte 源文件，esbuild 预打包无法处理，
		// 排除后由 vite-plugin-svelte 直接转译（dev 必需，build 无预打包）
		exclude: ['@lucide/svelte']
	},
	test: {
		environment: 'jsdom',
		setupFiles: ['./vitest-setup.ts'],
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
