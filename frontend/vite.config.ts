import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import tailwindcss from '@tailwindcss/vite';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit(), svelteTesting()],
	server: {
		// 经 nginx 网关访问时 Host header 是 OrbStack 的本地域名，不放行会被 Vite 的
		// host 校验拒绝（403「Blocked request. This host ... is not allowed」）。
		// 用前导点的 `.orb.local` 而不是写死单个主机名——同一个 nginx 容器在 OrbStack
		// 下同时有多个域名，写死一个会让其余域名全部 403（实测）：
		//   nginx.hotspot.orb.local          服务名.项目名
		//   hotspot-nginx-1.orb.local        容器名
		//   nginx.<worktree 目录名>.orb.local  worktree 起的独立 compose 项目
		// Vite 对前导点的语义是「该域及其所有子域」（见 isHostAllowedInternal），
		// IP 与 localhost 本来就被无条件放行。放行范围仍限于 OrbStack 本地域
		// （只有本机 mDNS 会发布这些名字），不引入 DNS rebinding 风险。
		allowedHosts: ['.orb.local']
	},
	test: {
		environment: 'jsdom',
		setupFiles: ['./vitest-setup.ts'],
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
