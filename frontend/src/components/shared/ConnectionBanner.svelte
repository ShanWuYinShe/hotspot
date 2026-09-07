<script lang="ts">
	import { WifiOff } from '@lucide/svelte';
	import { ws } from '@/stores/ws.svelte';
	import { auth } from '@/stores/auth.svelte';

	// ConnectionBanner — WS 断线提示胶囊（居中悬浮呼吸微徽章）。
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
		role="status"
		aria-live="polite"
		class="pointer-events-none fixed inset-x-0 top-3 z-[var(--cst-z-toast)] flex justify-center px-4"
	>
		<div
			class="pointer-events-auto flex items-center gap-2 rounded-full border border-status-warning/40 bg-surface-primary/95 px-3.5 py-1.5 shadow-raised-md backdrop-blur-md animate-in fade-in-0 slide-in-from-top-2"
		>
			<span class="relative flex size-2 shrink-0">
				<span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-status-warning opacity-75"></span>
				<span class="relative inline-flex size-2 rounded-full bg-status-warning"></span>
			</span>
			<WifiOff class="size-3.5 shrink-0 text-status-warning" strokeWidth={2.2} />
			<span class="text-xs font-medium text-text-primary tracking-tight">网络连接已断开，正在重连…</span>
		</div>
	</div>
{/if}
