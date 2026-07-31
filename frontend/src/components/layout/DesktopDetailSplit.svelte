<script lang="ts">
	import type { Snippet } from 'svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import ContextRail from '@/components/layout/ContextRail.svelte';

	// 桌面详情页容器：左主体 + 右 320px 陪伴栏（非桌面仅主体）。
	// 迁移自旧版 DetailRail 内 DesktopDetailSplit。

	let { main, rail }: { main: Snippet; rail: Snippet } = $props();
</script>

{#if platformStore.platform !== 'desktop'}
	{@render main()}
{:else}
	<div class="flex h-full w-full">
		<div class="flex-1 min-w-0 flex flex-col">{@render main()}</div>
		<ContextRail attached>{@render rail()}</ContextRail>
	</div>
{/if}
