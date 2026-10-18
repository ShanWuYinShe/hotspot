<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import LocationDesktop from './desktop/LocationDesktop.svelte';
	import LocationMobile from './mobile/LocationMobile.svelte';

	// 地图探索入口：平台分派。平板改用移动版全宽画布 + 底部面板——桌面 340px 左栏
	// 在 820 宽平板上占 41.5%，地图可视区过小。
</script>
<svelte:head>
	<title>位置选择 · HotSpot</title>
</svelte:head>


{#if platform.platform === 'desktop'}
	<LocationDesktop />
{:else}
	<LocationMobile />
{/if}
