<script lang="ts">
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import ChatView from '@/components/messages/ChatView.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES } from '@/lib/routes';

	// 私聊窗口（peerId 即对端 user_id，spec: target_user_id）。
	// 桌面端与宽平板保持 Master-Detail 两栏布局（重定向至分栏视图，对齐桌面 IM 客户端体验）；
	// 移动端与窄平板保持单栏推入与 showBackButton。

	let { params }: { params: { peerId: string } } = $props();

	const useSplit = $derived(platform.platform === 'desktop' || platform.isWideTablet);

	$effect(() => {
		if (useSplit) {
			void goto(`${ROUTES.messages}?peer=${encodeURIComponent(params.peerId ?? '')}`, { replaceState: true });
		}
	});
</script>
<svelte:head>
	<title>私聊 · HotSpot</title>
</svelte:head>


{#if !useSplit}
	<div class="flex h-full flex-col bg-surface-page">
		<ChatView peerUserId={params.peerId ?? ''} showBackButton onBack={() => smartBack()} />
	</div>
{:else}
	<!-- 桌面端与宽平板重定向过渡骨架（防白屏闪烁） -->
	<div class="flex h-full flex-1 gap-3 overflow-hidden bg-surface-page p-3 animate-pulse" aria-busy="true" aria-label="正在进入会话">
		<div class="flex w-80 shrink-0 flex-col overflow-hidden rounded-lg bg-surface-island shadow-xs p-4 space-y-4">
			<div class="flex items-center justify-between pb-2">
				<div class="h-6 w-20 rounded-md bg-surface-subtle"></div>
				<div class="h-8 w-16 rounded-lg bg-surface-subtle"></div>
			</div>
			<div class="h-9 w-full rounded-lg bg-surface-subtle"></div>
			<div class="space-y-3 pt-2">
				{#each Array(6) as _}
					<div class="flex items-center gap-3 p-2 rounded-lg bg-surface-page/50">
						<div class="h-10 w-10 shrink-0 rounded-full bg-surface-subtle"></div>
						<div class="flex-1 space-y-1.5">
							<div class="h-3.5 w-24 rounded bg-surface-subtle"></div>
							<div class="h-2.5 w-36 rounded bg-surface-subtle"></div>
						</div>
					</div>
				{/each}
			</div>
		</div>
		<div class="relative flex min-w-0 flex-1 flex-col justify-between overflow-hidden rounded-lg bg-surface-island shadow-xs p-4">
			<div class="flex items-center gap-3 pb-3 border-b border-border-subtle">
				<div class="h-9 w-9 rounded-full bg-surface-subtle"></div>
				<div class="h-4 w-32 rounded bg-surface-subtle"></div>
			</div>
			<div class="space-y-4 py-6">
				<div class="h-10 w-48 rounded-xl bg-surface-subtle"></div>
				<div class="h-14 w-64 rounded-xl bg-surface-subtle ml-auto"></div>
				<div class="h-10 w-40 rounded-xl bg-surface-subtle"></div>
			</div>
			<div class="h-12 w-full rounded-xl bg-surface-subtle"></div>
		</div>
	</div>
{/if}
