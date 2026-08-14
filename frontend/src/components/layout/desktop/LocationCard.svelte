<script lang="ts">
	import { Navigation2, ChevronRight } from '@lucide/svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';

	// 侧栏底部「位置选择」锚点入口。
	// 与 SidebarNavItem 同一交互语义：默认透明融入侧栏（无常驻边框/阴影/卡片底），
	// hover 淡填充；折叠时文字随侧栏宽度同步收起，图标列固定 40px 居中不位移。

	let {
		onClick,
		collapsed = false
	}: {
		onClick: () => void;
		collapsed?: boolean;
	} = $props();

	let hovered = $state(false);
	let locationName = $state('');

	$effect(() => startDefaultLocation((loc) => {
		locationName = loc?.location_name ?? '';
	}));
</script>

<button
	type="button"
	class="relative flex h-10 w-full items-center rounded-sm focus-ring text-left {collapsed ? 'justify-center' : ''}"
	style="background:{hovered ? 'var(--cst-surface-navigation-hover)' : 'transparent'};transition:background var(--cst-motion-rapid) var(--cst-ease-standard);"
	title={collapsed ? '位置选择' : undefined}
	aria-label="位置选择"
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<span class="flex size-10 shrink-0 items-center justify-center">
		<span class="flex size-8.5 items-center justify-center rounded-full bg-accent-default">
			<Navigation2 class="size-4.5 text-text-on-accent" strokeWidth={2} />
		</span>
	</span>

	<span
		class="flex min-w-0 flex-1 items-center overflow-hidden whitespace-nowrap transition-[max-width,opacity,margin] duration-standard ease-standard {collapsed ? 'ml-0 max-w-0 opacity-none' : 'ml-1.5 max-w-full opacity-full'}"
	>
		<span class="min-w-0 flex-1">
			<span class="block text-xs font-medium text-text-primary">位置选择</span>
			<span class="flex items-center gap-1">
				<span class="inline-block rounded-full size-1.25 bg-status-online"></span>
				<span class="text-xxs leading-card text-text-secondary">{locationName || '自动定位'}</span>
			</span>
		</span>
		<ChevronRight class="ml-0.5 size-3.5 shrink-0 text-text-secondary" strokeWidth={1.5} />
	</span>
</button>
