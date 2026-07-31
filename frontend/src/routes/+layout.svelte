<script lang="ts">
	import type { Snippet } from 'svelte';
	import '../styles/index.css';
	import { setApiBaseUrl } from '@/api';
	import { initAuth } from '@/stores/auth.svelte';
	import { initTheme } from '@/stores/theme.svelte';
	import { initPlatform } from '@/stores/platform.svelte';
	import AuthGate from '@/components/auth/AuthGate.svelte';
	import { initMock } from '@/mocks/toggle';

	let { children }: { children: Snippet } = $props();

	// SPA 模式（ssr=false）：本组件仅在客户端实例化，可直接访问 window。
	// 若无 PUBLIC_API_URL，回退同源（经 nginx 网关反代 /api）。
	setApiBaseUrl(import.meta.env.PUBLIC_API_URL ?? window.location.origin);
	initTheme();
	void initPlatform();
	void initAuth();

	// MSW mock（仅 dev；PUBLIC_API_MOCK !== 'false' 时启用，localStorage 可覆盖）
	if (import.meta.env.DEV && import.meta.env.PUBLIC_API_MOCK !== 'false') {
		void import('@/mocks/browser').then(({ worker }) => {
			initMock(worker, true);
		});
	}
</script>

{@render children()}
<AuthGate />
