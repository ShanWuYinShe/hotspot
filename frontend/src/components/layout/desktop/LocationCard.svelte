<script lang="ts">
	import { Navigation2, ChevronRight } from '@lucide/svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';

	// 侧栏底部「位置选择」锚点卡片。
	// 折叠态镜像侧栏导航项的 hover 语义：默认透明、hover 淡灰填充。

	let {
		onClick,
		collapsed = false
	}: {
		onClick: () => void;
		collapsed?: boolean;
	} = $props();

	let hovered = $state(false);
	let locationName = $state('');
	const collapsedHover = $derived('var(--cst-interactive-hover-fill)');

	$effect(() => startDefaultLocation((loc) => {
locationName = loc?.location_name ?? '';
}));
</script>

<button
	type="button"
	class="hover-btn card-hover flex items-center outline-none text-left {collapsed ? 'h-10 w-10 self-center justify-center rounded-sm' : 'w-full gap-3 rounded-md'}"
	style="padding:{collapsed ? 0 : '10px 12px'};background:{hovered ? collapsedHover : collapsed ? 'transparent' : 'var(--cst-surface-primary)'};border:1px solid var(--cst-border-default);box-shadow:{collapsed ? 'none' : 'var(--shadow-raised-md)'};transform:{hovered ? 'translateY(-2px)' : 'translateY(0)'};transition:background 0.16s var(--cst-ease-standard), transform 0.22s var(--cst-ease-standard), box-shadow 0.22s var(--cst-ease-standard);"
	title={collapsed ? '位置选择' : undefined}
	aria-label="位置选择"
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<div
		class="flex items-center justify-center rounded-full shrink-0 {collapsed ? 'size-7.5' : 'size-8.5'}"
		style="background:var(--cst-accent-default);box-shadow:none;"
	>
		<Navigation2 class="size-4.5"  style="color:var(--cst-text-on-accent)" strokeWidth={2} />
	</div>

	<div class={collapsed ? 'sr-only' : 'flex-1 min-w-0 text-left'}>
		<p class="text-xs font-medium" style="color:var(--cst-text-primary)">位置选择</p>
		<div class="flex items-center gap-1">
			<span class="inline-block rounded-full size-1.25"  style="background:var(--cst-status-online)"></span>
			<p class="text-xxs leading-card" style="color:var(--cst-text-secondary)">{locationName || '自动定位'}</p>
		</div>
	</div>

	{#if !collapsed}
		<ChevronRight class="size-3.25 shrink-0"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
	{/if}
</button>
