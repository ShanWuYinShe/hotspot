<script lang="ts">
	import type { Snippet } from 'svelte';
	import '../styles/index.css';
	import { setApiBaseUrl } from '@/api';
	import { auth, initAuth } from '@/stores/auth.svelte';
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

	/* Boot 门控：auth 首次解析（session 检查）完成前不挂载路由页面。
	   否则页面会以「匿名」变体抢先请求双版本端点（pick()），viewer 态字段
	   （viewer_state/is_favorited/reacted_by_me…）拿到错误数据且不会重取。
	   3s 兜底：会话检查异常挂起时不至于白屏（退化为旧行为）。 */
	let bootEscaped = $state(false);
	$effect(() => {
		const t = setTimeout(() => (bootEscaped = true), 3000);
		return () => clearTimeout(t);
	});
	const showApp = $derived(auth.status !== 'loading' || bootEscaped);
</script>

{#if showApp}
	{@render children()}
{:else}
	<!-- 启动占位：token 化 spinner（.spinner 自带 reduced-motion 降级），
	     避免未解析 auth 时页面闪错数据 -->
	<div class="fixed inset-0 grid place-items-center" style="background:var(--cst-surface-page);">
		<div class="spinner size-7" role="status" aria-label="加载中"></div>
	</div>
{/if}
<AuthGate />
<ToastViewport />
