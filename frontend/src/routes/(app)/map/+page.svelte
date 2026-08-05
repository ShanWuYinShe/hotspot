<script lang="ts">
	import { platform } from '@/stores/platform.svelte';
	import LocationDesktop from './desktop/LocationDesktop.svelte';
	import LocationMobile from './mobile/LocationMobile.svelte';

	// 地图探索入口：平台分派（桌面/移动）。
</script>

{#if platform.platform !== 'mobile'}
	<LocationDesktop />
{:else}
	<LocationMobile />
{/if}
