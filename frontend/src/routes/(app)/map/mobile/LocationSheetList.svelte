<script lang="ts">
	import { MapPin, X, Users, ChevronRight, Clock, Star, Check } from '@lucide/svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { HOT_LOCATIONS, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { PIN_COLORS, PIN_LABELS, pinTone } from '../shared/location-shared';
import PinMedia from '../shared/PinMedia.svelte';

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
				class="app-list-row app-list-row--standard rounded-control px-5"
			>
				<div class="flex items-center justify-center rounded-full shrink-0 bg-accent-default size-8.5"  style="opacity:0.12">
					<MapPin class="size-3.5"  style="color:var(--accent-default)" strokeWidth={1.75} />
				</div>
				<span class="{isSelected ? 'text-content-inverse' : 'text-text-primary'} text-sm"  style="flex:1">{r.displayName}</span>
				{#if isSelected}
					<Check class="size-3.5 shrink-0"  style="color:var(--content-inverse)" strokeWidth={2.5} />
				{/if}
			</ListItem>
		{/each}
	</ListContainer>
{:else if searchQuery && searchResults.length === 0}
	<div class="flex flex-col items-center py-6">
		<MapPin class="size-6.5"  style="color:var(--text-secondary);opacity:0.28" strokeWidth={1} />
		<p class="text-text-secondary text-base mt-2" >未找到匹配地点</p>
	</div>
{:else}
	<div class="animate-fade-in">
		<!-- Selected pin mini-card -->
		{#if selectedPin}
			<div class="flex items-center gap-3 mx-4 mb-2 p-3 rounded-[15px]" style="background:{pinTone(selectedPin.type, 10)};border:1px solid {pinTone(selectedPin.type, 30)};">
				<PinMedia pin={selectedPin} size={42} />
				<div class="flex-1 min-w-0">
					<p class="text-text-primary text-base">{selectedPin.name || selectedPin.title}</p>
					<p class="text-text-secondary text-xs mt-px" >
						{PIN_LABELS[selectedPin.type]}
						{selectedPin.distance ? ` · ${selectedPin.distance}` : ''}
						{selectedPin.tag ? ` · #${selectedPin.tag}` : ''}
					</p>
				</div>
				<button type="button" class="hover-btn px-2.5 py-1.5 rounded-full outline-none shrink-0" style="background:{PIN_COLORS[selectedPin.type]};color:white;font-size:11px;" onclick={() => onOpenPinDetail(selectedPin)}>
					查看
				</button>
				<button type="button" class="hover-btn outline-none shrink-0 ml-0.5" onclick={onClearPin}>
					<X class="size-3.25"  style="color:var(--text-secondary)" strokeWidth={2} />
				</button>
			</div>
		{/if}

		<!-- Tab row -->
		<div class="flex items-center gap-2 px-4 pb-2">
			{#each [
				{ id: 'recent', label: '最近使用', icon: Clock },
				{ id: 'hot', label: '热门地点', icon: Star }
			] as tab}
				<button
					type="button"
					class="hover-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none {activeTab === tab.id ? 'bg-accent-default text-white' : 'bg-surface-primary text-text-secondary'}"
					style={activeTab === tab.id ? 'font-size:12px;box-shadow:0 3px 10px var(--glow-accent);' : 'font-size:12px;border:1px solid var(--border-default);'}
					onclick={() => (activeTab = tab.id as 'recent' | 'hot')}
				>
					<tab.icon class="size-2.75"  strokeWidth={activeTab === tab.id ? 2.5 : 1.5} />
					{tab.label}
				</button>
			{/each}
			{#if activeTab === 'recent' && recentItems.length > 0}
				<button type="button" class="hover-btn ml-auto outline-none text-text-secondary text-xs" onclick={() => (recentItems = [])}>
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
							class="app-list-row app-list-row--standard rounded-control px-4"
						>
							<div class="flex items-center justify-center rounded-full shrink-0 bg-surface-primary size-9 text-lg border border-border-default shadow-neu-xs"  >
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate {isSelected ? 'text-content-inverse' : 'text-text-primary'} text-sm">{loc.displayName}</p>
								<p class="{isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
							</div>
							{#if isSelected}
								<Check class="size-3.5 shrink-0"  style="color:var(--content-inverse)" strokeWidth={2.5} />
							{:else}
								<ChevronRight class="size-3.25 shrink-0"  style="color:var(--border-default)" strokeWidth={1.5} />
							{/if}
						</ListItem>
					{/each}
				</ListContainer>
			{:else}
				<p class="text-text-secondary text-base"  style="padding:12px 20px">暂无最近记录</p>
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
							class="app-list-row app-list-row--standard rounded-control px-4"
						>
							<div class="flex items-center justify-center rounded-[13px] shrink-0 bg-accent-default size-9 text-lg"  style="opacity:0.88">
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate {isSelected ? 'text-content-inverse' : 'text-text-primary'} text-sm">{loc.displayName}</p>
								<p class="{isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
							</div>
							<div class="flex items-center gap-1 shrink-0">
								<Users style="width:11px;height:11px;color:{isSelected ? 'var(--content-inverse)' : 'var(--accent-default)'};" strokeWidth={1.5} />
								<span class="{isSelected ? 'text-content-inverse-secondary' : 'text-accent-default'} text-xs">{loc.users}</span>
							</div>
						</ListItem>
					{/each}
				</ListContainer>
			{:else}
				<div class="flex flex-col items-center gap-2 py-8">
					<Star class="size-5"  style="color:var(--text-secondary);opacity:0.3" strokeWidth={1} />
					<p class="text-text-secondary text-xs">暂无热门地点</p>
				</div>
			{/if}
		{/if}
	</div>
{/if}
