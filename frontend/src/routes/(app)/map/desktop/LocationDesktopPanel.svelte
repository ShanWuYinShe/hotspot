<script lang="ts">
	import { MapPin, Navigation, X, ChevronRight, Check, Heart } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import Button from '@/components/ui/button.svelte';
	import { type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { PIN_COLORS, PIN_LABELS, PIN_TONES } from '../shared/location-shared';
	import PinMedia from '../shared/PinMedia.svelte';
	import { smartBack } from '@/lib/smart-back';

	// 桌面地图「左侧面板」：标题/搜索/附近列表/确认。
	// 页面私有组件（colocate），由 map/LocationDesktop.svelte 渲染。
	// 选中态经 $bindable 双向绑定（selectedPin/selectedLoc/searchQuery 等），
	// 仅跨状态联动（选地点→移动地图中心）与外部动作（重置/确认）走回调。
	// 原「热门/最近」Tab 为静态演示数据脚手架，已按产品决定移除。
	//
	// R13：面板本身是悬浮内容岛（外壳负责投影/圆角），内部收敛为
	// 「位置头 → 搜索 → 当前位置实底卡 → 类型图标砖列表 → 粘底确认」一条纵向节奏；
	// 类型信息用 marker 语义色（不再每个字段一套灰底胶囊），确认动作走按钮原语。

	let {
		searchQuery = $bindable(''),
		searchResults = $bindable<LocationResult[]>([]),
		searching = false,
		nearbyPins = $bindable<Pin[]>([]),
		selectedPin = $bindable<Pin | null>(null),
		selectedLoc = $bindable<LocationResult | null>(null),
		mapCounts = $bindable<Record<PinType, number>>({ user: 0, group: 0, moment: 0 }),
		// 实际视口中心的地名（父组件按选中/逆地理/锚点/默认链解析）；
		// 此前这里直读服务端静态默认名，定位/拖拽后永远对不上实际位置。
		locationName = '',
		onSelectPin,
		onSelectLoc,
		onConfirm
	}: {
		searchQuery?: string;
		searchResults?: LocationResult[];
		/** 地理编码请求进行中（上游偶发数秒慢，需「搜索中」指示）。 */
		searching?: boolean;
		nearbyPins?: Pin[];
		selectedPin?: Pin | null;
		selectedLoc?: LocationResult | null;
		mapCounts?: Record<PinType, number>;
		locationName?: string;
		/** 选中附近图钉：由父组件同步设置 selectedPin 并移动地图中心。 */
		onSelectPin: (pin: Pin) => void;
		onSelectLoc: (loc: LocationResult) => void;
		onConfirm: () => void;
	} = $props();
</script>

<!-- Header -->
<div class="shrink-0">
	<PageHeader title="位置选择" onBack={() => smartBack()} subtitle={locationName || '位置选择'} sticky contentClassName="px-4" titleSize="xl" />
</div>

<!-- Search -->
<div class="relative shrink-0 px-4 pb-3">
	<SearchInput
		value={searchQuery}
		onChange={(v) => {
			searchQuery = v;
			selectedLoc = null;
		}}
		placeholder="搜索地点或地标…"
		ariaLabel="搜索地点或地标"
	/>
	{#if searchResults.length > 0}
		<ListContainer bordered class="absolute left-4 right-4 top-full z-30 mt-1 shadow-menu">
			{#each searchResults as r}
				{@const isSelected = selectedLoc?.displayName === r.displayName}
				<ListItem
					pressed={isSelected}
					onClick={() => onSelectLoc(r)}
					ariaLabel={`选择地点 ${r.displayName}`}
					class="flex list-row-standard items-center gap-3 px-3.5 py-3"
				>
					<MapPin class="size-3.5 shrink-0 text-accent-default" strokeWidth={1.5} />
					<span class="{isSelected ? 'text-text-on-accent' : 'text-text-primary'} min-w-0 flex-1 truncate text-sm">{r.displayName}</span>
					{#if isSelected}
						<Check class="size-3.5 text-text-on-accent" strokeWidth={2.5} />
					{/if}
				</ListItem>
			{/each}
		</ListContainer>
	{:else if searching}
		<!-- 搜索中指示：上游偶发数秒级慢响应（OSM 代理），零反馈会被当成搜索失效 -->
		<div class="absolute top-full left-4 right-4 z-30 mt-1 flex items-center gap-2 rounded-md bg-surface-primary px-3.5 py-2.5 shadow-menu">
			<span class="spinner size-3.5" role="status" aria-label="搜索中"></span>
			<span class="text-sm text-text-secondary">搜索中…</span>
		</div>
	{:else if searchQuery.trim()}
		<!-- 搜索终态空态：失败与零结果都落到这里（与移动端 sheet 同文案）。
		     此前无此分支：请求完成后下拉直接消失，用户无法区分“搜完无果”与“没在搜”。 -->
		<div class="absolute top-full left-4 right-4 z-30 mt-1 rounded-md bg-surface-primary px-3.5 py-2 shadow-menu">
			<EmptyState bare title="未找到匹配地点" class="py-5" />
		</div>
	{/if}
</div>

<!-- List -->
<!-- 底部渐隐：滚动内容探入粘底确认条下方时柔和收尾，避免行被硬切 -->
<div
	class="no-scrollbar min-h-0 flex-1 overflow-y-auto px-4 pb-4"
	style="mask-image:linear-gradient(to bottom, black calc(100% - 24px), transparent 100%);-webkit-mask-image:linear-gradient(to bottom, black calc(100% - 24px), transparent 100%);"
>
	<div class="animate-fade-in flex flex-col gap-3">
		<!-- 当前位置卡：R13 改 accent-soft + accent 图标——实底 accent 只留给底部主 CTA，
		     一屏一个主操作（设计评审：双实底抢焦点）。 -->
		<div class="flex items-center gap-3 rounded-md bg-accent-soft p-3">
			<span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-surface-primary shadow-raised-sm">
				<Navigation class="size-4.5 text-accent-default" strokeWidth={2} aria-hidden="true" />
			</span>
			<span class="min-w-0 flex-1">
				<span class="block text-xxs font-medium uppercase tracking-eyebrow text-accent-default">当前位置</span>
				<span class="mt-0.5 block truncate text-md font-medium text-text-primary">{locationName || '自动定位'}</span>
			</span>
			<span
				class="size-2 shrink-0 rounded-full bg-status-online"
				style="box-shadow:var(--cst-marker-halo) var(--cst-status-online);"
				aria-hidden="true"
			></span>
		</div>

		{#if nearbyPins.length}
			<ListContainer bordered>
				{#each nearbyPins as pin (pin.id)}
					{@const isSelected = selectedPin?.id === pin.id}
					<ListItem
						pressed={isSelected}
						onClick={() => onSelectPin(pin)}
						ariaLabel={pin.isAggregate
							? `定位到 ${pin.name || pin.title || '附近区域'}`
							: `选择 ${pin.name || pin.title || ''}`}
						class="flex list-row-standard items-center gap-3 px-3 py-2.5"
					>
						<PinMedia {pin} size={42} selected={isSelected} />
						<span class="min-w-0 flex-1">
							<span
								class="block truncate text-md {isSelected ? 'text-text-on-accent' : 'text-text-primary'}"
							>
								{pin.name || pin.title}
							</span>
							<span class="mt-0.5 flex min-w-0 items-center gap-1.5 text-xs">
								<span
									class="shrink-0 font-medium"
									style="color:{isSelected ? 'var(--cst-text-on-accent)' : PIN_COLORS[pin.type]};"
								>
									{PIN_LABELS[pin.type]}
								</span>
								{#if pin.tag}
									<span class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} truncate">
										#{pin.tag}
									</span>
								{/if}
								{#if pin.members}
									<span class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} shrink-0">
										{pin.members}人
									</span>
								{/if}
								{#if pin.likes}
									<span class="flex shrink-0 items-center gap-0.5 {isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'}">
										<Heart class="size-2.5" strokeWidth={1.5} />
										<span class="tabular-nums">{pin.likes}</span>
									</span>
								{/if}
								{#if pin.distance}
									<span class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} ml-auto shrink-0 pl-1 tabular-nums">
										{pin.distance}
									</span>
								{/if}
							</span>
						</span>
						{#if isSelected}
							<span class="inline-flex h-5 shrink-0 items-center justify-center rounded-full bg-text-on-accent px-2 text-xxs font-medium leading-none text-accent-default">
								已定位
							</span>
						{:else if pin.isAggregate}
							<!-- 聚合桶：该网格内对象总数。热力桶不是单个实体，箭头会误导成「可进详情」 -->
							<span
								class="inline-flex h-5 shrink-0 items-center justify-center rounded-full px-2 text-xxs font-medium leading-none tabular-nums"
								style="background:{PIN_TONES[pin.type].bg};color:{PIN_COLORS[pin.type]};"
							>
								{pin.count ?? 0} 项
							</span>
						{:else}
							<ChevronRight class="size-3.5 shrink-0 text-text-secondary" strokeWidth={1.5} />
						{/if}
					</ListItem>
				{/each}
			</ListContainer>
		{:else}
			<EmptyState
				iconTone="accent"
				title="附近暂无内容"
				description="放大或移动地图后，再看看这里"
				iconSize={56}
				class="py-10"
			>
				{#snippet icon()}
					<MapPin class="size-6" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{/if}
	</div>
</div>

<!-- Confirm — 粘底操作条（功能线白名单：粘底条允许 border-t） -->
<div class="shrink-0 border-t border-border-subtle px-4 pb-4 pt-3">
	{#if selectedLoc}
		<div class="mb-2 flex items-center gap-2 rounded-md bg-surface-subtle/60 px-3 py-2">
			<Check class="size-3.5 shrink-0 text-accent-default" strokeWidth={2.5} />
			<p class="min-w-0 flex-1 truncate text-xs text-accent-default">{selectedLoc.displayName}</p>
			<button
				type="button"
				class="hover-action -m-1.5 flex items-center justify-center rounded-full p-1.5 focus-ring"
				aria-label="移除所选地点"
				title="移除"
				onclick={() => {
					selectedLoc = null;
					searchQuery = '';
				}}
			>
				<X class="size-3 text-text-secondary" strokeWidth={1.75} />
			</button>
		</div>
	{/if}
	<Button variant="primary" size="lg" class="w-full" onclick={onConfirm}>
		<MapPin class="size-4" strokeWidth={1.75} aria-hidden="true" />
		{selectedLoc ? '确认此位置' : '设为浏览锚点'}
	</Button>
</div>
