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
		class="fixed inset-x-0 top-0 z-[var(--cst-z-toast)] flex h-7 items-center justify-center gap-2 text-sm tracking-wide"
		style="background:var(--cst-status-warning);color:var(--cst-text-on-status);box-shadow:var(--cst-elevation-menu);"
	>
		网络连接已断开，正在重连…
	</div>
{/if}
