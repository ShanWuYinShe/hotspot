<script lang="ts">
	import ChatView from '@/components/messages/ChatView.svelte';

	// 私聊窗口（peerId 即对端 user_id，spec: target_user_id）。

	let { params }: { params: { peerId: string } } = $props();
</script>

<div class="flex flex-col" style="height:100%;background-color:var(--cst-surface-primary);">
	<ChatView peerUserId={params.peerId ?? ''} showBackButton onBack={() => history.back()} />
</div>
