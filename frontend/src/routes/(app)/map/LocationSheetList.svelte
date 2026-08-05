<script lang="ts">
	import { MapPin, X, Users, ChevronRight, Clock, Star, Check } from '@lucide/svelte';
	import EmbeddedList from '@/components/data-display/EmbeddedList.svelte';
	import { HOT_LOCATIONS, type PinType, type Pin, type LocationResult } from './location-data';
	import { PIN_COLORS, PIN_LABELS, pinTone, getMarkerUrl } from './location-shared';

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
	<EmbeddedList class="animate-fade-in">
		{#each searchResults as r}
			<button
				type="button"
				class="app-list-row app-list-row--standard px-5 outline-none transition-[background-color,transform] duration-150 hover:bg-interactive-hover-fill focus-visible:bg-interactive-hover-fill active:scale-[0.98] active:duration-100"
				aria-pressed={selectedLoc?.displayName === r.displayName}
				onclick={() => onSelectLoc(r)}
			>
				<div class="flex items-center justify-center rounded-full flex-shrink-0 bg-accent-default" style="width:34px;height:34px;opacity:0.12">
					<MapPin style="width:14px;height:14px;color:var(--accent-default)" strokeWidth={1.75} />
				</div>
				<span class="text-text-primary" style="font-size:14px;flex:1">{r.displayName}</span>
				{#if selectedLoc?.displayName === r.displayName}
					<Check style="width:14px;height:14px;color:var(--accent-default);flex-shrink:0" strokeWidth={2.5} />
				{/if}
			</button>
		{/each}
	</EmbeddedList>
{:else if searchQuery && searchResults.length === 0}
	<div class="flex flex-col items-center py-6">
		<MapPin style="width:26px;height:26px;color:var(--text-secondary);opacity:0.28;" strokeWidth={1} />
		<p class="text-text-secondary" style="font-size:13px;margin-top:8px;">未找到匹配地点</p>
	</div>
{:else}
	<div class="animate-fade-in">
		<!-- Selected pin mini-card -->
		{#if selectedPin}
			<div class="flex items-center gap-3 mx-4 mb-2 p-3 rounded-[15px]" style="background:{pinTone(selectedPin.type, 10)};border:1px solid {pinTone(selectedPin.type, 30)};">
				{#if selectedPin.type === 'user' && selectedPin.avatar}
					<div class="relative flex-shrink-0">
						<img src={selectedPin.avatar} alt={selectedPin.name || selectedPin.title || ''} class="rounded-full object-cover" style="width:42px;height:42px;" />
						{#if selectedPin.online}
							<span class="absolute bottom-0 right-0 rounded-full bg-status-online" style="width:10px;height:10px;border:2px solid var(--surface-primary);"></span>
						{/if}
					</div>
				{:else if selectedPin.type === 'moment' && selectedPin.image}
					<img src={selectedPin.image} alt={selectedPin.name || selectedPin.title || ''} class="rounded-[10px] object-cover flex-shrink-0" style="width:42px;height:42px;" />
				{:else}
					<img src={getMarkerUrl(selectedPin.type, selectedPin.count ?? 1)} alt="" style="width:22px;height:29px;flex-shrink:0;" />
				{/if}
				<div class="flex-1 min-w-0">
					<p class="text-text-primary text-[13px]">{selectedPin.name || selectedPin.title}</p>
					<p class="text-text-secondary" style="font-size:11px;margin-top:1px;">
						{PIN_LABELS[selectedPin.type]}
						{selectedPin.distance ? ` · ${selectedPin.distance}` : ''}
						{selectedPin.tag ? ` · #${selectedPin.tag}` : ''}
					</p>
				</div>
				<button type="button" class="hover-btn px-2.5 py-1.5 rounded-full outline-none flex-shrink-0" style="background:{PIN_COLORS[selectedPin.type]};color:white;font-size:11px;" onclick={() => onOpenPinDetail(selectedPin)}>
					查看
				</button>
				<button type="button" class="hover-btn outline-none flex-shrink-0 ml-0.5" onclick={onClearPin}>
					<X style="width:13px;height:13px;color:var(--text-secondary);" strokeWidth={2} />
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
					<tab.icon style="width:11px;height:11px;" strokeWidth={activeTab === tab.id ? 2.5 : 1.5} />
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
				<EmbeddedList>
					{#each recentItems as loc}
						<button
							type="button"
							class="app-list-row app-list-row--standard px-4 outline-none transition-[background-color,transform] duration-150 hover:bg-interactive-hover-fill focus-visible:bg-interactive-hover-fill active:scale-[0.98] active:duration-100"
							aria-pressed={selectedLoc?.displayName === loc.displayName}
							onclick={() => onSelectLoc(loc)}
						>
							<div class="flex items-center justify-center rounded-full flex-shrink-0 bg-surface-primary" style="width:36px;height:36px;border:1px solid var(--border-default);box-shadow:var(--shadow-neu-xs);font-size:18px;">
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate text-text-primary text-sm">{loc.displayName}</p>
								<p class="text-text-secondary" style="font-size:11px;margin-top:1px;">{loc.desc}</p>
							</div>
							{#if selectedLoc?.displayName === loc.displayName}
								<Check style="width:14px;height:14px;color:var(--accent-default);flex-shrink:0;" strokeWidth={2.5} />
							{:else}
								<ChevronRight style="width:13px;height:13px;color:var(--border-default);flex-shrink:0;" strokeWidth={1.5} />
							{/if}
						</button>
					{/each}
				</EmbeddedList>
			{:else}
				<p class="text-text-secondary" style="font-size:13px;padding:12px 20px;">暂无最近记录</p>
			{/if}
		{:else}
			{#if HOT_LOCATIONS.length}
				<EmbeddedList>
					{#each HOT_LOCATIONS as loc}
						<button
							type="button"
							class="app-list-row app-list-row--standard px-4 outline-none transition-[background-color,transform] duration-150 hover:bg-interactive-hover-fill focus-visible:bg-interactive-hover-fill active:scale-[0.98] active:duration-100"
							aria-pressed={selectedLoc?.displayName === loc.displayName}
							onclick={() => onSelectLoc(loc)}
						>
							<div class="flex items-center justify-center rounded-[13px] flex-shrink-0 bg-accent-default" style="width:36px;height:36px;font-size:18px;opacity:0.88;">
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate text-text-primary text-sm">{loc.displayName}</p>
								<p class="text-text-secondary" style="font-size:11px;margin-top:1px;">{loc.desc}</p>
							</div>
							<div class="flex items-center gap-1 flex-shrink-0">
								<Users style="width:11px;height:11px;color:var(--accent-default);" strokeWidth={1.5} />
								<span class="text-accent-default text-xs">{loc.users}</span>
							</div>
						</button>
					{/each}
				</EmbeddedList>
			{:else}
				<div class="flex flex-col items-center gap-2 py-8">
					<Star style="width:20px;height:20px;color:var(--text-secondary);opacity:0.3;" strokeWidth={1} />
					<p class="text-text-secondary text-xs">暂无热门地点</p>
				</div>
			{/if}
		{/if}
	</div>
{/if}
