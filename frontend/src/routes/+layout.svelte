<script lang="ts">
	import type { Snippet } from 'svelte';
	import '../styles/index.css';
	import { setApiBaseUrl } from '@/api';
	import { auth, initAuth } from '@/stores/auth.svelte';
	import { initWs } from '@/stores/ws.svelte';
	import { initPresence } from '@/stores/presence.svelte';
	import { initUnread } from '@/stores/unread.svelte';
	import { initTheme } from '@/stores/theme.svelte';
	import { initMapSource } from '@/stores/mapSource.svelte';
	import { initPlatform } from '@/stores/platform.svelte';
	import AuthGate from '@/components/auth/AuthGate.svelte';
	import BootSplash from '@/components/shared/BootSplash.svelte';
	import ToastViewport from '@/components/overlays/ToastViewport.svelte';

	let { children }: { children: Snippet } = $props();

	// SPA 模式（ssr=false）：本组件仅在客户端实例化，可直接访问 window。
	// 若无 PUBLIC_API_URL，回退同源（经 nginx 网关反代 /api）。
	setApiBaseUrl(import.meta.env.PUBLIC_API_URL ?? window.location.origin);
	// 明暗三档：initTheme 读取持久化初值 + 订阅系统明暗变化，返回清理函数；
	// $effect 保证注册在组件初始化上下文，销毁时移除 matchMedia 监听。
	$effect(() => initTheme());
	// 地图源偏好（高德/OSM）持久化初值，设置页与地图页共享。
	initMapSource();
	void initPlatform();
	void initAuth();
	// WS 实时通道：监听 auth store，登录后自动连接、登出后断开。
	// 与 initAuth 一样在 script 顶层调用（非 onMount），保证内部 $effect 注册在组件初始化上下文。
	initWs();
	// 在线状态实时化：UserOnline/UserOffline → onlineUsers（供消息列表/私聊/群成员消费）。
	initPresence();
	// 全局未读徽标：WS 通知/消息事件维护计数（冷启动校准在 auth store 的
	// bootstrap 流程内完成；回写收敛在 notify/messages 页）。
	initUnread();

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
	<!-- 启动占位：logo 基座 SVG 动画（BootSplash，reduced-motion 由全局规则降级），
	     避免未解析 auth 时页面闪错数据 -->
	<div class="fixed inset-0 grid place-items-center bg-surface-page">
		<BootSplash />
	</div>
{/if}
<AuthGate />
<ToastViewport />
