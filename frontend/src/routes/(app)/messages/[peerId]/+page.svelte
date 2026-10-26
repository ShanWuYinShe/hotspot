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
	<div class="flex flex-col" style="height:100%;background-color:var(--cst-surface-chat);">
		<ChatView peerUserId={params.peerId ?? ''} showBackButton onBack={() => smartBack()} />
	</div>
{/if}
