<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import { useChatListData } from './shared/useChatListData.svelte';
	import MessagesDesktop from './desktop/MessagesDesktop.svelte';
	import MessagesMobile from './mobile/MessagesMobile.svelte';
	import MessagesTablet from './tablet/MessagesTablet.svelte';

	// ═══ 消息列表控制器（Controller-View 分层架构）═══
	// 架构重构：消灭 645 行巨石混杂组件。
	// 数据状态/WS 订阅/会话置顶免打扰逻辑收敛至 useChatListData；
	// 视图层按端分发至 MessagesDesktop / MessagesTablet / MessagesMobile。

	const chatData = useChatListData();
	const device = $derived(platform.platform);
</script>

<svelte:head>
	<title>消息 · HotSpot</title>
</svelte:head>

{#if device === 'desktop'}
	<MessagesDesktop data={chatData} />
{:else if device === 'tablet'}
	<MessagesTablet data={chatData} />
{:else}
	<MessagesMobile data={chatData} />
{/if}
