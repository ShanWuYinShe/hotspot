<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import MessagesDesktop from '../desktop/MessagesDesktop.svelte';
	import MessagesMobile from '../mobile/MessagesMobile.svelte';
	import type { ChatListData } from '../shared/useChatListData.svelte';

	let { data }: { data: ChatListData } = $props();

	// 平板端定位契约：
	// 宽平板（≥960px，如横屏 iPad Pro）开启 Master-Detail 双栏分栏；
	// 紧凑平板（<960px）使用触控单栏视图，点入会话推入聊天详情。
	const isWide = $derived(platform.isWideTablet);
</script>

{#if isWide}
	<MessagesDesktop {data} />
{:else}
	<MessagesMobile {data} />
{/if}
