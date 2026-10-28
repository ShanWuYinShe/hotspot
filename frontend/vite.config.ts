import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';
import tailwindcss from '@tailwindcss/vite';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit(), svelteTesting()],
	server: {
		fs: {
			allow: ['..', '/Users/liyou/work/hotspot']
		},
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
		allowedHosts: ['.orb.local'],
		// dev 冷路由预热（Vite 8 `server.warmup.clientFiles`，语义：dev server 启动后
		// 异步预 transform 这些文件及其直接 import，结果写入模块缓存，从而消除「第一次
		// 访问某路由时按需编译」的一拍迟滞；相对 project root，glob 由 Vite 内置的
		// tinyglobby 解析，见 node_modules/vite/dist/node/chunks/node.js warmupFiles/mapFiles）。
		// 注意：glob 里 `(app)` 的圆括号是分组语法，必须转义成 `\(app\)` 才能匹配——实测
		// 不转义时匹配 0 个文件；这里改用 `src/routes/*/**` 覆盖全部路由目录（含 (app)、
		// 登录组与 share），不依赖转义写法，后续新增路由目录自动纳入。
		// 2026-10-28 实测（worktree 8898 dev）：route +page.svelte 冷 transform 58–720ms，
		// 预热后 restart 完成即 5–20ms（.workwork/perf/{before,after}/server-*.json）。
		warmup: {
			clientFiles: [
				'src/routes/*/**/+page.svelte', // 全部 34 个路由页面（覆盖 (app) 下全部 +page.svelte）
				'src/routes/+layout.svelte',
				'src/routes/*/+layout.svelte', // (app) 组布局
				'src/components/layout/**/*.svelte', // 布局外壳（桌面/平板/移动 + page 原语）
				'src/components/messages/**/*.svelte', // 消息高频组件
				'src/components/navigation/**/*.svelte' // 导航原语（ActionIconButton/BackToTop/TabBar）
			]
		}
	},
	test: {
		environment: 'jsdom',
		setupFiles: ['./vitest-setup.ts'],
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
