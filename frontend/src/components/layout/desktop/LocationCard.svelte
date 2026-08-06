<script lang="ts">
	import { Navigation2, ChevronRight } from '@lucide/svelte';
	import { loadDefaultLocation } from '@/lib/default-location';

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
	const collapsedHover = $derived('var(--interactive-hover-fill)');

	$effect(() => {
		let cancelled = false;
		void loadDefaultLocation().then((loc) => {
			if (!cancelled) locationName = loc?.location_name ?? '';
		});
		return () => {
			cancelled = true;
		};
	});
</script>

<button
	type="button"
	class="hover-btn card-hover flex items-center outline-none text-left {collapsed ? 'h-10 w-10 self-center justify-center rounded-xl' : 'w-full gap-3 rounded-control'}"
	style="padding:{collapsed ? 0 : '10px 12px'};background:{hovered ? collapsedHover : collapsed ? 'transparent' : 'var(--surface-primary)'};border:1px solid var(--border-default);box-shadow:{collapsed ? 'none' : 'var(--shadow-neu-lg)'};transform:{hovered ? 'translateY(-2px)' : 'translateY(0)'};transition:background 0.16s ease, transform 0.22s cubic-bezier(0.22,1,0.36,1), box-shadow 0.22s cubic-bezier(0.22,1,0.36,1);"
	title={collapsed ? '位置选择' : undefined}
	aria-label="位置选择"
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<div
		class="flex items-center justify-center rounded-full shrink-0"
		style="width:{collapsed ? 30 : 34}px;height:{collapsed ? 30 : 34}px;background:var(--accent-default);box-shadow:none;"
	>
		<Navigation2 class="size-4.5"  style="color:var(--text-on-accent)" strokeWidth={2} />
	</div>

	<div class={collapsed ? 'sr-only' : 'flex-1 min-w-0 text-left'}>
		<p class="text-xs"  style="color:var(--text-primary);font-weight:500">位置选择</p>
		<div class="flex items-center gap-1">
			<span class="inline-block rounded-full size-1.25"  style="background:var(--status-online)"></span>
			<p class="text-xxs"  style="line-height:1.4;color:var(--text-secondary)">{locationName || '自动定位'}</p>
		</div>
	</div>

	{#if !collapsed}
		<ChevronRight class="size-3.25 shrink-0"  style="color:var(--text-secondary)" strokeWidth={1.5} />
	{/if}
</button>
