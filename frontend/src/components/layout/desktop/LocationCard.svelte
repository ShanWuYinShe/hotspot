<script lang="ts">
	import { MapPin, ChevronRight } from '@lucide/svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { untrack } from 'svelte';
	import { LOCATION_ANCHOR_UPDATED_EVENT, readLastLocation } from '@/lib/map-auto-locate';

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
	// 锚点由地图页写入（localStorage），本卡常驻不重挂载——监听写入事件重读，
	// 否则地图页更新锚点后这里永远停留在挂载时的旧值（UI 走查确认）。
	let hasAnchor = $state(!!readLastLocation());
	let locationName = $state(readLastLocation()?.name ?? '');

	$effect(() => startDefaultLocation((loc) => {
		// hasAnchor 变化会重跑本 effect：untrack 避免重复订阅
		if (untrack(() => hasAnchor)) return;
		locationName = loc?.location_name ?? '';
	}));

	$effect(() => {
		const onAnchorUpdated = () => {
			const anchor = readLastLocation();
			hasAnchor = !!anchor;
			locationName = anchor?.name ?? '';
		};
		window.addEventListener(LOCATION_ANCHOR_UPDATED_EVENT, onAnchorUpdated);
		return () => window.removeEventListener(LOCATION_ANCHOR_UPDATED_EVENT, onAnchorUpdated);
	});
</script>

<button
	type="button"
	class="card-hover relative flex w-full items-center rounded-md bg-surface-subtle/60 p-1.5 text-left backdrop-blur-xs focus-ring hover:bg-surface-subtle {collapsed ? 'justify-center' : ''}"
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
			<span class="block truncate text-xxs leading-card text-text-secondary">{locationName || (hasAnchor ? '自定义位置' : '自动定位')}</span>
		</span>
		<ChevronRight class="ml-0.5 size-3.5 shrink-0 text-text-secondary" strokeWidth={1.5} />
	</span>
</button>
