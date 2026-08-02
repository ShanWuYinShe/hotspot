<script lang="ts">
	import { platformStore } from '@/stores/platform.svelte';
	import LocationDesktop from './LocationDesktop.svelte';
	import LocationMobile from './LocationMobile.svelte';

	// 地图探索入口：平台分派（桌面/移动）。
</script>

{#if platformStore.platform !== 'mobile'}
	<LocationDesktop />
{:else}
	<LocationMobile />
{/if}
