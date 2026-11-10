<script lang="ts">
	import { MapPin, ChevronRight } from '@lucide/svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { untrack } from 'svelte';
	import { LOCATION_ANCHOR_UPDATED_EVENT, readLastLocation } from '@/lib/map-auto-locate';

	// 侧栏底部「位置选择」锚点入口（地图门户）。
	// R13：从「独立浮起小卡」收敛为安静的定位胶囊——去掉 backdrop 模糊与 hover 抬升投影，
	// 只保留淡底 + hover 淡 accent 填充，避免和上方导航抢视觉重量；
	// 折叠时文字随侧栏宽度同步收起，图标磁贴居中不位移。

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

{#if collapsed}
	<!-- 折叠态：44px 圆形锚点，和折叠导航同为圆形 icon 节奏 -->
	<button
		type="button"
		class="group mx-auto flex size-11 items-center justify-center rounded-full border border-border-default/50 bg-surface-primary/60 text-accent-default backdrop-blur-md transition-colors duration-rapid focus-ring hover:border-accent-border hover:bg-accent-soft"
		title="位置选择"
		aria-label="位置选择"
		onclick={onClick}
	>
		<MapPin class="size-4.5" strokeWidth={1.75} />
	</button>
{:else}
	<button
		type="button"
		class="group relative flex w-full items-center rounded-xl border border-border-default/50 bg-surface-primary/70 backdrop-blur-md p-2 text-left transition-colors duration-rapid focus-ring hover:bg-surface-primary hover:border-border-default"
		aria-label="位置选择"
		onclick={onClick}
	>
		<span
			class="flex size-7 shrink-0 items-center justify-center rounded-lg bg-accent-soft text-accent-default transition-colors group-hover:bg-accent-default group-hover:text-text-on-accent"
		>
			<MapPin class="size-3.5" strokeWidth={1.75} />
		</span>

		<span
			class="flex min-w-0 flex-1 items-center overflow-hidden whitespace-nowrap transition-[max-width,opacity,margin] duration-standard ease-standard ml-2.5 max-w-full opacity-full"
		>
			<span class="min-w-0 flex-1">
				<span class="block text-xs font-semibold text-text-primary group-hover:text-accent-default transition-colors">位置选择</span>
				<span class="block truncate text-xxs leading-card text-text-secondary">{locationName || (hasAnchor ? '自定义位置' : '自动定位')}</span>
			</span>
			<ChevronRight class="ml-0.5 size-3.5 shrink-0 text-text-secondary transition-transform duration-rapid group-hover:translate-x-0.5" strokeWidth={1.5} />
		</span>
	</button>
{/if}
