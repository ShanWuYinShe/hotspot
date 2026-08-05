<script lang="ts">
	import type { Snippet } from 'svelte';
	import { platform } from '@/stores/platform.svelte';
	import DesktopLayout from '@/components/layout/desktop/DesktopLayout.svelte';
	import TabletLayout from '@/components/layout/tablet/TabletLayout.svelte';
	import MobileLandscape from '@/components/layout/mobile/MobileLandscape.svelte';
	import MobilePortrait from '@/components/layout/mobile/MobilePortrait.svelte';

	// MainLayout — 四端壳分派：桌面 → DesktopLayout，平板 → TabletLayout，
	// 移动横屏 → MobileLandscape，移动竖屏 → MobilePortrait。

	let { children }: { children: Snippet } = $props();

	const isMobileLandscape = $derived(platform.orientation === 'landscape');
</script>

{#if platform.platform === 'desktop'}
	<DesktopLayout>{@render children()}</DesktopLayout>
{:else if platform.platform === 'tablet'}
	<TabletLayout>{@render children()}</TabletLayout>
{:else if isMobileLandscape}
	<MobileLandscape>{@render children()}</MobileLandscape>
{:else}
	<MobilePortrait>{@render children()}</MobilePortrait>
{/if}
