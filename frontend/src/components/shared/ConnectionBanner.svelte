<script lang="ts">
	import { ws } from '@/stores/ws.svelte';
	import { auth } from '@/stores/auth.svelte';

	// ConnectionBanner — WS 断线提示条（顶部细条）。
	// 显示条件：曾成功连接过（everConnected），当前不在连接态，且仍处于登录态
	// （登出是手动断开、不会重连，不提示"正在重连"）。

	let everConnected = $state(false);
	$effect(() => {
		if (ws.status === 'connected') everConnected = true;
	});

	const show = $derived(
		auth.isAuthenticated && everConnected && ws.status !== 'connected' && ws.status !== 'idle'
	);
</script>

{#if show}
	<div
		class="fixed inset-x-0 top-0 z-[var(--cst-z-toast)] flex items-center justify-center gap-2"
		style="height:28px;background:var(--cst-status-warning);color:#fff;font-size:12px;letter-spacing:0.04em;box-shadow:0 2px 10px rgba(0,0,0,0.18);"
	>
		网络连接已断开，正在重连…
	</div>
{/if}
