<script lang="ts">
	import ChatView from '@/components/messages/ChatView.svelte';

	// 私聊窗口（chatId 实为对端 user_id，spec: target_user_id）。
	// 迁移自 React ChatDetailPage.tsx。

	let { params }: { params: { chatId: string } } = $props();
</script>

<div class="flex flex-col" style="height:100%;background-color:var(--neu-bg);">
	<ChatView peerUserId={params.chatId ?? ''} showBackButton onBack={() => history.back()} />
</div>
