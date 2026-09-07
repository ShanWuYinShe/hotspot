<script lang="ts">
	import { MapPin, X, Users, ChevronRight, Clock, Star, Check } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import { HOT_LOCATIONS, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { PIN_COLORS, PIN_LABELS, PIN_TONES } from '../shared/location-shared';
import PinMedia from '../shared/PinMedia.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';

	// 移动端地图「底部 Sheet 列表内容」：搜索建议 / 选中图钉 mini-card / Tab / 最近·热门列表。
	// 页面私有组件（colocate），由 map/LocationMobile.svelte 渲染（仅 snap !== 'min' 时显示）。
	// 选中态经 $bindable 双向绑定；跨状态联动（选地点→移动地图中心）走 onSelectLoc 回调。

	let {
		searchQuery = $bindable(''),
		searchResults = $bindable<LocationResult[]>([]),
		selectedPin = $bindable<Pin | null>(null),
		selectedLoc = $bindable<LocationResult | null>(null),
		activeTab = $bindable<'recent' | 'hot'>('recent'),
		recentItems = $bindable<LocationResult[]>([]),
		onSelectLoc,
		onOpenPinDetail,
		onClearPin
	}: {
		searchQuery?: string;
		searchResults?: LocationResult[];
		selectedPin?: Pin | null;
		selectedLoc?: LocationResult | null;
		activeTab?: 'recent' | 'hot';
		recentItems?: LocationResult[];
		onSelectLoc: (loc: LocationResult) => void;
		onOpenPinDetail: (pin: Pin) => void;
		onClearPin: () => void;
	} = $props();
</script>

{#if searchQuery && searchResults.length > 0}
	<ListContainer class="animate-fade-in">
		{#each searchResults as r}
			{@const isSelected = selectedLoc?.displayName === r.displayName}
			<ListItem
				pressed={isSelected}
				onClick={() => onSelectLoc(r)}
				ariaLabel={`选择地点 ${r.displayName}`}
				class="flex list-row-standard items-center gap-3 px-4 py-3"
			>
				<div class="flex items-center justify-center rounded-full shrink-0 size-8.5 {isSelected ? 'bg-surface-primary' : 'bg-accent-soft'}">
					<MapPin class="size-3.5 text-accent-default"   strokeWidth={1.75} />
				</div>
				<span class="{isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm"  style="flex:1">{r.displayName}</span>
				{#if isSelected}
					<Check class="size-3.5 shrink-0 text-text-on-accent"   strokeWidth={2.5} />
				{/if}
			</ListItem>
		{/each}
	</ListContainer>
{:else if searchQuery && searchResults.length === 0}
	<!-- 空态统一 EmptyState（bare + 紧凑 py-8：地图 sheet 不是整页，不套 py-20） -->
	<EmptyState bare title="未找到匹配地点" class="py-8">
		{#snippet icon()}
			<MapPin size={24} class="opacity-faint text-text-secondary"   strokeWidth={1.5} />
		{/snippet}
	</EmptyState>
{:else}
	<div class="animate-fade-in">
		<!-- Selected pin mini-card -->
		{#if selectedPin}
			<div class="flex items-center gap-3 mx-4 mb-2 p-3 rounded-md" style="background:{PIN_TONES[selectedPin.type].bg};border:var(--cst-border-width-hairline) solid {PIN_TONES[selectedPin.type].border};">
				<PinMedia pin={selectedPin} size={42} />
				<div class="flex-1 min-w-0">
					<p class="text-text-primary text-sm">{selectedPin.name || selectedPin.title}</p>
					<p class="text-text-secondary text-xs mt-px" >
						{PIN_LABELS[selectedPin.type]}
						{selectedPin.distance ? ` · ${selectedPin.distance}` : ''}
						{selectedPin.tag ? ` · #${selectedPin.tag}` : ''}
					</p>
				</div>
				<button type="button" class="hover-btn scale-btn px-2.5 py-1.5 rounded-full focus-ring shrink-0 text-sm" style="background:{PIN_COLORS[selectedPin.type]};color:var(--cst-content-on-marker);" onclick={() => onOpenPinDetail(selectedPin)}>
					查看
				</button>
				<button type="button" class="hover-btn hit-slop focus-ring shrink-0 ml-0.5" aria-label="清除选中地点" onclick={onClearPin}>
					<X class="size-3.5 text-text-secondary"   strokeWidth={2} />
				</button>
			</div>
		{/if}

		<!-- Tab row：统一 TabBar（分段） -->
		<div class="flex items-center gap-2 px-4 pb-2">
			<TabBar
				variant="segmented"
				items={[
					{ id: 'recent', label: '最近使用', icon: Clock },
					{ id: 'hot', label: '热门地点', icon: Star }
				]}
				value={activeTab}
				onChange={(id) => (activeTab = id as 'recent' | 'hot')}
				class="flex-1"
			/>
			{#if activeTab === 'recent' && recentItems.length > 0}
				<!-- 一键清空不可恢复：二次确认防误触 -->
				<button
					type="button"
					class="link-btn ml-auto -my-1 px-2 py-1 focus-ring text-xs text-text-secondary hover:text-accent-default"
					onclick={() => {
						if (window.confirm('清空全部最近记录？此操作不可恢复。')) recentItems = [];
					}}
				>
					清空
				</button>
			{/if}
		</div>

		<!-- List items -->
		{#if activeTab === 'recent'}
			{#if recentItems.length > 0}
				<ListContainer>
					{#each recentItems as loc}
						{@const isSelected = selectedLoc?.displayName === loc.displayName}
						<ListItem
							pressed={isSelected}
							onClick={() => onSelectLoc(loc)}
							ariaLabel={`选择最近地点 ${loc.displayName}`}
							class="flex list-row-standard items-center gap-3 px-4 py-3"
						>
							<div class="flex items-center justify-center rounded-full shrink-0 bg-surface-primary size-9 text-lg border border-border-default shadow-raised-sm"  >
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate {isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm">{loc.displayName}</p>
								<p class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
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
				<EmptyState bare title="暂无最近记录" class="py-8">
					{#snippet icon()}
						<Clock size={24} class="opacity-faint text-text-secondary"   strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{/if}
		{:else}
			{#if HOT_LOCATIONS.length}
				<ListContainer>
					{#each HOT_LOCATIONS as loc}
						{@const isSelected = selectedLoc?.displayName === loc.displayName}
						<ListItem
							pressed={isSelected}
							onClick={() => onSelectLoc(loc)}
							ariaLabel={`选择热门地点 ${loc.displayName}`}
							class="flex list-row-standard items-center gap-3 px-4 py-3"
						>
							<div class="flex items-center justify-center rounded-sm shrink-0 size-9 text-lg {isSelected ? 'bg-surface-primary' : 'bg-accent-default'} opacity-hover">
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate {isSelected ? 'text-text-on-accent' : 'text-text-primary'} text-sm">{loc.displayName}</p>
								<p class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
							</div>
							<div class="flex items-center gap-1 shrink-0">
								<Users class="size-3" style="color:{isSelected ? 'var(--cst-text-on-accent)' : 'var(--cst-accent-default)'};" strokeWidth={1.5} />
								<span class="{isSelected ? 'text-text-on-accent opacity-emphasis' : 'text-accent-default'} text-xs">{loc.users}</span>
							</div>
						</ListItem>
					{/each}
				</ListContainer>
			{:else}
				<EmptyState bare title="暂无热门地点" class="py-8">
					{#snippet icon()}
						<Star size={24} class="opacity-faint text-text-secondary"  strokeWidth={1.5} />
					{/snippet}
				</EmptyState>
			{/if}
		{/if}
	</div>
{/if}
