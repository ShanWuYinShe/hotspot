<script lang="ts">
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import GroupChatView from '@/components/messages/GroupChatView.svelte';
	import { smartBack } from '@/lib/smart-back';
	import { ROUTES } from '@/lib/routes';

	// 群聊窗口（静态段 group 优先于私聊 [peerId] 参数段）。
	// 桌面端与宽平板保持 Master-Detail 两栏布局（重定向至分栏视图，对齐桌面 IM 客户端体验）；
	// 移动端与窄平板保持单栏推入与 showBackButton。

	let { params }: { params: { groupId: string } } = $props();

	const useSplit = $derived(platform.platform === 'desktop' || platform.isWideTablet);

	$effect(() => {
		if (useSplit) {
			void goto(`${ROUTES.messages}?group=${encodeURIComponent(params.groupId ?? '')}`, { replaceState: true });
		}
	});
</script>
<svelte:head>
	<title>群聊 · HotSpot</title>
</svelte:head>


{#if !useSplit}
	<div class="flex flex-col" style="height:100%;background-color:var(--cst-surface-chat);">
		<GroupChatView groupId={params.groupId ?? ''} showBackButton onBack={() => smartBack()} />
	</div>
{/if}
