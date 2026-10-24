<script lang="ts">
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import GroupChatView from '@/components/messages/GroupChatView.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES } from '@/lib/routes';

	// 群聊窗口（静态段 group 优先于私聊 [peerId] 参数段）。
	// 桌面端保持 Master-Detail 两栏布局（重定向至分栏视图，对齐桌面 IM 客户端体验）；
	// 移动端保持单栏推入与 showBackButton。

	let { params }: { params: { groupId: string } } = $props();

	$effect(() => {
		if (platform.platform === 'desktop') {
			void goto(`${ROUTES.messages}?group=${encodeURIComponent(params.groupId ?? '')}`, { replaceState: true });
		}
	});
</script>
<svelte:head>
	<title>群聊 · HotSpot</title>
</svelte:head>


{#if platform.platform !== 'desktop'}
	<div class="flex flex-col" style="height:100%;background-color:var(--cst-surface-chat);">
		<GroupChatView groupId={params.groupId ?? ''} showBackButton onBack={() => smartBack()} />
	</div>
{/if}
