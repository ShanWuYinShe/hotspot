<script lang="ts">
	import type { Snippet } from 'svelte';
	import '../styles/index.css';
	import { setApiBaseUrl } from '@/api';
	import { initAuth } from '@/stores/auth.svelte';
	import { initWs } from '@/stores/ws.svelte';
	import { initPresence } from '@/stores/presence.svelte';
	import { initTheme } from '@/stores/theme.svelte';
	import { initPlatform } from '@/stores/platform.svelte';
	import AuthGate from '@/components/auth/AuthGate.svelte';
	import ToastViewport from '@/components/overlays/ToastViewport.svelte';

	let { children }: { children: Snippet } = $props();

	// SPA 模式（ssr=false）：本组件仅在客户端实例化，可直接访问 window。
	// 若无 PUBLIC_API_URL，回退同源（经 nginx 网关反代 /api）。
	setApiBaseUrl(import.meta.env.PUBLIC_API_URL ?? window.location.origin);
	initTheme();
	void initPlatform();
	void initAuth();
	// WS 实时通道：监听 auth store，登录后自动连接、登出后断开。
	// 与 initAuth 一样在 script 顶层调用（非 onMount），保证内部 $effect 注册在组件初始化上下文。
	initWs();
	// 在线状态实时化：UserOnline/UserOffline → onlineUsers（供消息列表/私聊/群成员消费）。
	initPresence();
</script>

{@render children()}
<AuthGate />
<ToastViewport />
