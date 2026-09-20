<script lang="ts">
	import { MapPin, ChevronRight } from '@lucide/svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { readLastLocation } from '@/lib/map-auto-locate';

	// 侧栏底部「位置选择」锚点入口（地图门户）。
	// 独立小卡片形态（tinted 底 + hairline 边框），与上方导航行明确区隔：
	// hover 沿 card-hover 契约抬升投影；折叠时文字随侧栏宽度同步收起，图标磁贴居中不位移。

	let {
		onClick,
		collapsed = false
	}: {
		onClick: () => void;
		collapsed?: boolean;
	} = $props();

	// 与附近页同一口径：本机浏览锚点优先，无锚点才回退服务端默认位置。
	const savedAnchor = readLastLocation();
	let locationName = $state(savedAnchor?.name ?? '');

	$effect(() => startDefaultLocation((loc) => {
		if (savedAnchor) return;
		locationName = loc?.location_name ?? '';
	}));
</script>

<button
	type="button"
	class="surface-tinted card-hover relative flex w-full items-center rounded-md border border-border-default p-1.5 text-left focus-ring {collapsed ? 'justify-center' : ''}"
	title={collapsed ? '位置选择' : undefined}
	aria-label="位置选择"
	onclick={onClick}
>
	<span
		class="flex size-8 shrink-0 items-center justify-center rounded-sm transition-colors duration-standard ease-standard"
		style="background:var(--cst-accent-soft);"
	>
		<MapPin class="size-4 text-accent-default" strokeWidth={1.75} />
	</span>

	<span
		class="flex min-w-0 flex-1 items-center overflow-hidden whitespace-nowrap transition-[max-width,opacity,margin] duration-standard ease-standard {collapsed ? 'ml-0 max-w-0 opacity-none' : 'ml-2.5 max-w-full opacity-full'}"
	>
		<span class="min-w-0 flex-1">
			<span class="block text-xs font-medium text-text-primary">位置选择</span>
			<span class="block truncate text-xxs leading-card text-text-secondary">{locationName || '自动定位'}</span>
		</span>
		<ChevronRight class="ml-0.5 size-3.5 shrink-0 text-text-secondary" strokeWidth={1.5} />
	</span>
</button>
