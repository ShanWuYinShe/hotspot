<script lang="ts">
	import { Navigation2, ChevronRight } from '@lucide/svelte';
	import { themeStore } from '@/stores/theme.svelte';

	// 侧栏底部「位置选择」锚点卡片（迁移自旧版 DesktopLayout 内 LocationCard）。
	// 折叠态镜像侧栏导航项的 hover 语义：默认透明、hover 淡灰填充。

	let {
		onClick,
		collapsed = false
	}: {
		onClick: () => void;
		collapsed?: boolean;
	} = $props();

	let hovered = $state(false);
	const collapsedHover = $derived(themeStore.isDark ? 'rgba(255,255,255,0.045)' : 'rgba(0,0,0,0.035)');
</script>

<button
	type="button"
	class="neu-hover-btn flex items-center outline-none text-left {collapsed ? 'h-10 w-10 self-center justify-center rounded-xl' : 'w-full gap-3 rounded-[16px]'}"
	style="padding:{collapsed ? 0 : '10px 12px'};background:{hovered ? collapsedHover : collapsed ? 'transparent' : 'var(--neu-bg)'};border:1px solid transparent;box-shadow:none;transition:background 0.16s ease, color 0.16s ease;"
	title={collapsed ? '位置选择' : undefined}
	aria-label="位置选择"
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<div
		class="flex items-center justify-center rounded-full flex-shrink-0"
		style="width:{collapsed ? 30 : 34}px;height:{collapsed ? 30 : 34}px;background:var(--neu-accent-primary);box-shadow:none;"
	>
		<Navigation2 style="width:18px;height:18px;color:white;" strokeWidth={2} />
	</div>

	<div class={collapsed ? 'sr-only' : 'flex-1 min-w-0 text-left'}>
		<p style="font-size:12px;color:var(--neu-text);font-weight:500;">位置选择</p>
		<div class="flex items-center gap-1">
			<span class="inline-block rounded-full" style="width:5px;height:5px;background:var(--semantic-online);"></span>
			<p style="font-size:10px;line-height:1.4;color:var(--neu-text-muted);">徐州 · 云龙湖畔</p>
		</div>
	</div>

	{#if !collapsed}
		<ChevronRight style="width:13px;height:13px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
	{/if}
</button>
