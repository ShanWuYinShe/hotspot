<script lang="ts">
	import GroupChatView from '@/components/messages/GroupChatView.svelte';

	// 群聊窗口（静态段 group 优先于私聊 [peerId] 参数段）。
	// 与 messages/[peerId] 私聊页对称；桌面分栏复用同一 GroupChatView 组件。

	let { params }: { params: { groupId: string } } = $props();
</script>

<div class="flex flex-col" style="height:100%;background-color:var(--cst-surface-primary);">
	<GroupChatView groupId={params.groupId ?? ''} showBackButton onBack={() => history.back()} />
</div>
