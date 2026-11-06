<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import { useHomeFeedData } from '@/components/home/shared/home-feed-data.svelte';
	import HomeDesktop from '@/components/home/desktop/HomeDesktop.svelte';
	import HomeMobile from '@/components/home/mobile/HomeMobile.svelte';
	import HomeTablet from '@/components/home/tablet/HomeTablet.svelte';

	// ═══ 首页控制器（Controller-View 分层架构）═══
	// 架构重构：消灭 700+ 行巨石混杂组件。
	// 状态/游标分页/话题派生收敛至 useHomeFeedData 单一事实源；
	// 视图层按端分发至 HomeDesktop / HomeTablet / HomeMobile，各端纯粹无干扰。

	const feedData = useHomeFeedData();
	const device = $derived(platform.platform);
</script>

<svelte:head>
	<title>首页 · HotSpot</title>
</svelte:head>

{#if device === 'desktop'}
	<HomeDesktop data={feedData} />
{:else if device === 'tablet'}
	<HomeTablet data={feedData} />
{:else}
	<HomeMobile data={feedData} />
{/if}
