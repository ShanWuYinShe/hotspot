import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import tailwindcss from '@tailwindcss/vite';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit(), svelteTesting()],
	server: {
		// 经 nginx 网关访问时 Host header 为 nginx.hotspot.orb.local（OrbStack 本地域），
		// 需显式允许，否则 Vite 7 的 host 校验会拒绝请求。
		allowedHosts: ['nginx.hotspot.orb.local']
	},
	test: {
		environment: 'jsdom',
		setupFiles: ['./vitest-setup.ts'],
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
