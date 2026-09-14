<script lang="ts">
	import { Search, MapPin, Navigation, X, Users, ChevronRight, Clock, Star, TrendingUp, Check, Heart } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { RECENT_LOCATIONS, HOT_LOCATIONS, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { PIN_COLORS, PIN_LABELS, PIN_TONES } from '../shared/location-shared';
import PinMedia from '../shared/PinMedia.svelte';
import { smartBack } from '@/lib/smart-back';

	// 桌面地图「左侧面板」：标题/搜索/Tab 列表/确认。
	// 页面私有组件（colocate），由 map/LocationDesktop.svelte 渲染。
	// 选中态经 $bindable 双向绑定（selectedPin/selectedLoc/searchQuery/activeTab 等），
	// 仅跨状态联动（选地点→移动地图中心）与外部动作（重置/确认）走回调。

	let {
		searchQuery = $bindable(''),
		searchResults = $bindable<LocationResult[]>([]),
		nearbyPins = $bindable<Pin[]>([]),
		activeTab = $bindable<'nearby' | 'hot' | 'recent'>('nearby'),
		selectedPin = $bindable<Pin | null>(null),
		selectedLoc = $bindable<LocationResult | null>(null),
		mapCounts = $bindable<Record<PinType, number>>({ user: 0, group: 0, moment: 0 }),
		onSelectPin,
		onSelectLoc,
		onConfirm
	}: {
		searchQuery?: string;
		searchResults?: LocationResult[];
		nearbyPins?: Pin[];
		activeTab?: 'nearby' | 'hot' | 'recent';
		selectedPin?: Pin | null;
		selectedLoc?: LocationResult | null;
		mapCounts?: Record<PinType, number>;
		/** 选中附近图钉：由父组件同步设置 selectedPin 并移动地图中心。 */
		onSelectPin: (pin: Pin) => void;
		onSelectLoc: (loc: LocationResult) => void;
		onConfirm: () => void;
	} = $props();

	let locationName = $state('');
	$effect(() => startDefaultLocation((loc) => {
locationName = loc?.location_name ?? '';
}));
</script>

<!-- Header -->
<div class="shrink-0">
	<PageHeader title="位置选择" onBack={() => smartBack()} subtitle={locationName || '位置选择'} sticky contentClassName="px-4" />
</div>

<!-- Search -->
<div class="shrink-0 px-4 pt-3 pb-2 relative">
	<div class="flex items-center gap-2.5 px-3.5 rounded-md input-surface h-10.5" >
		<Search class="size-3.5 shrink-0 text-text-secondary"   strokeWidth={1.5} />
		<input
			bind:value={searchQuery}
			oninput={(e) => {
				searchQuery = e.currentTarget.value;
				selectedLoc = null;
			}}
			placeholder="搜索地点或地标…"
			class="flex-1 bg-transparent text-text-primary text-sm"
		/>
		{#if searchQuery}
			<button
				type="button"
				class="hover-action -m-1.5 flex items-center justify-center rounded-full p-1.5 focus-ring"
				aria-label="清空搜索"
				onclick={() => (searchQuery = '')}
			>
				<X class="size-3.5 text-text-secondary" strokeWidth={2} />
			</button>
		{/if}
	</div>
	{#if searchResults.length > 0}
		<ListContainer bordered class="absolute top-full left-4 right-4 z-30 mt-1 shadow-menu">
		{#each searchResults as r}
			{@const isSelected = selectedLoc?.displayName === r.displayName}
			<ListItem
				pressed={isSelected}
				onClick={() => onSelectLoc(r)}
				ariaLabel={`选择地点 ${r.displayName}`}
				class="flex list-row-standard items-center gap-3 px-3.5 py-3"
			>
				<MapPin class="size-3.5 shrink-0 text-accent-default"   strokeWidth={1.5} />
				<span class="{isSelected ? 'text-text-on-accent' : 'text-text-primary'} flex-1 text-sm" >{r.displayName}</span>
				{#if isSelected}
					<Check class="size-3.5 text-text-on-accent"   strokeWidth={2.5} />
				{/if}
			</ListItem>
		{/each}
		</ListContainer>
	{/if}
</div>

<!-- Tabs -->
<div class="shrink-0 px-4 pb-3">
	<!-- Tab row：统一 TabBar（分段选择器） -->
	<TabBar
		variant="segmented"
		items={[
			{ id: 'nearby', label: '附近', icon: TrendingUp },
			{ id: 'hot', label: '热门', icon: Star },
			{ id: 'recent', label: '最近', icon: Clock }
		]}
		value={activeTab}
		onChange={(id) => (activeTab = id as 'nearby' | 'hot' | 'recent')}
		class="w-full"
	/>
</div>

<!-- List -->
<div class="flex-1 min-h-0 overflow-y-auto no-scrollbar px-4 pb-6">
	{#if activeTab === 'nearby'}
		<div class="animate-fade-in flex flex-col gap-2">
			<!-- Current location card -->
			<div class="flex items-center gap-3 p-3.5 rounded-md bg-accent-default text-text-on-accent">
				<div class="flex items-center justify-center rounded-full shrink-0 size-10.5"  style="background:var(--cst-surface-map-control)">
					<Navigation class="size-4.5"  strokeWidth={2} />
				</div>
				<div class="flex-1">
					<p class="text-sm" >当前位置</p>
					<p class="text-xs mt-px opacity-emphasis">{locationName || '自动定位'}</p>
				</div>
				<div class="bg-status-online size-2 shrink-0 rounded-full"  style="box-shadow:var(--cst-marker-halo) var(--cst-status-online)"></div>
			</div>

			{#if nearbyPins.length}
				<ListContainer>
					{#each nearbyPins as pin (pin.id)}
						{@const isSelected = selectedPin?.id === pin.id}
						<ListItem
							pressed={isSelected}
							onClick={() => onSelectPin(pin)}
							ariaLabel={`选择 ${pin.name || pin.title || ''}`}
							class="flex list-row-standard items-center gap-3 px-3.5 py-3"
						>
						<PinMedia {pin} size={42} selected={isSelected} />
						<div class="flex-1 min-w-0">
							<p class="{isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm">{pin.name || pin.title}</p>
							<div class="flex items-center gap-2 mt-0.5 flex-wrap">
								<!-- 类型徽章：选中行背景为 accent 实色，type 色与 accent 同色（user），
								     反色适配为 on-accent 底 + accent 字（对齐 ChatListItem 未读徽章模式） -->
								<span class="text-xs" style="padding:1px 6px;border-radius:var(--cst-shape-sm);background:{isSelected ? 'var(--cst-text-on-accent)' : PIN_TONES[pin.type].bg};color:{isSelected ? 'var(--cst-accent-default)' : PIN_COLORS[pin.type]};">
									{PIN_LABELS[pin.type]}
								</span>
								{#if pin.tag}
									<span class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs">#{pin.tag}</span>
								{/if}
								{#if pin.distance}
									<span class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs">{pin.distance}</span>
								{/if}
								{#if pin.members}
									<span class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs">{pin.members}人</span>
								{/if}
								{#if pin.likes}
									<span class="flex items-center gap-0.5 {isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'}">
										<Heart class="size-2.5"  strokeWidth={1.5} />
										<span class="text-xs">{pin.likes}</span>
									</span>
								{/if}
							</div>
						</div>
						{#if isSelected}
							<span class="inline-flex h-5 shrink-0 items-center justify-center rounded-full bg-text-on-accent px-2 text-xxs font-medium leading-none text-accent-default">已定位</span>
						{:else}
							<ChevronRight class="size-3.5 shrink-0 text-text-secondary"   strokeWidth={1.5} />
						{/if}
					</ListItem>
				{/each}
				</ListContainer>
			{:else}
				<EmptyState bare title="附近暂无内容，放大或移动地图后刷新">
					{#snippet icon()}
						<MapPin class="size-5.5 opacity-faint text-text-secondary" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{/if}
		</div>
	{:else if activeTab === 'hot'}
		<div class="animate-fade-in">
			{#if HOT_LOCATIONS.length}
				<ListContainer>
					{#each HOT_LOCATIONS as loc}
						{@const isSelected = selectedLoc?.displayName === loc.displayName}
						<ListItem
							pressed={isSelected}
							onClick={() => onSelectLoc(loc)}
							ariaLabel={`选择热门地点 ${loc.displayName}`}
							class="flex list-row-standard items-center gap-3 px-3.5 py-3"
						>
							<div class="icon-badge flex shrink-0 items-center justify-center rounded-sm size-9 text-lg" aria-hidden="true">
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate {isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm">{loc.displayName}</p>
								<p class="truncate {isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
							</div>
							<div class="flex shrink-0 items-center gap-1">
								<Users class="size-3" style="color:{isSelected ? 'var(--cst-text-on-accent)' : 'var(--cst-accent-default)'};" strokeWidth={1.5} />
								<span class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-accent-default'} text-xs">{loc.users}</span>
							</div>
							{#if isSelected}
								<Check class="size-3.5 shrink-0 text-text-on-accent"   strokeWidth={2.5} />
							{:else}
								<ChevronRight class="size-3.5 shrink-0 text-border-default" strokeWidth={1.5} />
							{/if}
						</ListItem>
					{/each}
				</ListContainer>
			{:else}
				<EmptyState bare title="暂无热门地点">
					{#snippet icon()}
						<Star class="size-5.5 opacity-faint text-text-secondary" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{/if}
		</div>
	{:else}
		<div class="animate-fade-in">
			{#if RECENT_LOCATIONS.length}
				<ListContainer>
					{#each RECENT_LOCATIONS as loc}
						{@const isSelected = selectedLoc?.displayName === loc.displayName}
						<ListItem
							pressed={isSelected}
							onClick={() => onSelectLoc(loc)}
							ariaLabel={`选择最近地点 ${loc.displayName}`}
							class="flex list-row-standard items-center gap-3 px-3.5 py-3"
						>
							<div class="flex items-center justify-center rounded-full shrink-0 bg-surface-primary size-9 text-lg border border-border-default shadow-raised-sm"  >
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate {isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm">{loc.displayName}</p>
								<p class="truncate {isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
							</div>
							{#if isSelected}
								<Check class="size-3.5 shrink-0 text-text-on-accent"   strokeWidth={2.5} />
							{:else}
								<ChevronRight class="size-3.5 shrink-0 text-border-default" strokeWidth={1.5} />
							{/if}
						</ListItem>
					{/each}
				</ListContainer>
			{:else}
				<EmptyState bare title="暂无最近记录">
					{#snippet icon()}
						<Clock class="size-5.5 opacity-faint text-text-secondary" strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{/if}
		</div>
	{/if}
</div>

<!-- Confirm -->
<div class="shrink-0 border-t border-border-default px-4 pt-2.5 pb-4">
	{#if selectedLoc}
		<div class="flex items-center gap-2 mb-2 px-3 py-2.5 rounded-sm bg-surface-primary" style="box-shadow:var(--shadow-inset-sm);">
			<Check class="size-3.5 shrink-0 text-accent-default"   strokeWidth={2.5} />
			<p class="flex-1 truncate text-accent-default text-xs">{selectedLoc.displayName}</p>
			<button
				type="button"
				class="hover-action -m-1.5 flex items-center justify-center rounded-full p-1.5 focus-ring"
				aria-label="移除所选地点"
				onclick={() => {
					selectedLoc = null;
					searchQuery = '';
				}}
			>
				<X class="size-3 text-text-secondary" strokeWidth={2} />
			</button>
		</div>
	{/if}
	<button
		type="button"
		class="hover-btn w-full flex items-center justify-center gap-2 rounded-full focus-ring bg-accent-default text-text-on-accent text-sm h-11 shadow-raised-md hover:shadow-raised-lg"
		onclick={onConfirm}
	>
		<MapPin class="size-3.5"  strokeWidth={2} />
		{selectedLoc ? '确认此位置' : '设为浏览锚点'}
	</button>
</div>
