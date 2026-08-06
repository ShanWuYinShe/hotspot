<script lang="ts">
	import { Search, MapPin, Navigation, X, Users, ChevronRight, Clock, Star, TrendingUp, Crosshair, Check, Heart } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import { RECENT_LOCATIONS, HOT_LOCATIONS, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { PIN_COLORS, PIN_LABELS, pinTone } from '../shared/location-shared';
import PinMedia from '../shared/PinMedia.svelte';

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
		onReset,
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
		onReset: () => void;
		onConfirm: () => void;
	} = $props();

	let locationName = $state('');
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

<!-- Header -->
<div class="shrink-0">
	<PageHeader title="位置选择" onBack={() => history.back()} subtitle={locationName || '位置选择'} sticky>
		{#snippet trailing()}
			<button
				type="button"
				class="hover-btn flex items-center justify-center rounded-full outline-none shrink-0 circle-hover bg-surface-primary"
				style="width:36px;height:36px;border:1px solid var(--border-default);box-shadow:var(--shadow-neu-md);"
				onclick={onReset}
			>
				<Crosshair class="size-3.75"  style="color:var(--accent-default)" strokeWidth={1.75} />
			</button>
		{/snippet}
	</PageHeader>
</div>

<!-- Search -->
<div class="shrink-0 px-4 pt-3 pb-2 relative">
	<div class="flex items-center gap-2.5 px-3.5 rounded-xl input-surface h-10.5" >
		<Search class="size-3.5 shrink-0"  style="color:var(--text-secondary)" strokeWidth={1.5} />
		<input
			bind:value={searchQuery}
			oninput={(e) => {
				searchQuery = e.currentTarget.value;
				selectedLoc = null;
			}}
			placeholder="搜索地点或地标…"
			class="flex-1 bg-transparent outline-none text-text-primary text-base"
		/>
		{#if searchQuery}
			<button type="button" class="hover-btn outline-none" onclick={() => (searchQuery = '')}>
				<X class="size-3.25"  style="color:var(--text-secondary)" strokeWidth={2} />
			</button>
		{/if}
	</div>
	{#if searchResults.length > 0}
		<ListContainer bordered class="absolute top-full left-4 right-4 z-30 mt-1" style="box-shadow:0 0 0 1px var(--border-default), 0 10px 36px var(--elevation-shadow-dark);">
		{#each searchResults as r}
			{@const isSelected = selectedLoc?.displayName === r.displayName}
			<ListItem
				pressed={isSelected}
				onClick={() => onSelectLoc(r)}
				ariaLabel={`选择地点 ${r.displayName}`}
				class="app-list-row app-list-row--standard"
			>
				<MapPin class="size-3.25 shrink-0"  style="color:var(--accent-default)" strokeWidth={1.5} />
				<span class="{isSelected ? 'text-content-inverse' : 'text-text-primary'} flex-1 text-base" >{r.displayName}</span>
				{#if isSelected}
					<Check class="size-3.25"  style="color:var(--content-inverse)" strokeWidth={2.5} />
				{/if}
			</ListItem>
		{/each}
		</ListContainer>
	{/if}
</div>

<!-- Tabs -->
<div class="shrink-0 flex gap-1.5 px-4 pb-3">
	{#each [
		{ id: 'nearby', label: '附近', icon: TrendingUp },
		{ id: 'hot', label: '热门', icon: Star },
		{ id: 'recent', label: '最近', icon: Clock }
	] as tab}
		<button
			type="button"
			class="hover-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none {activeTab === tab.id ? 'bg-accent-default text-text-on-accent' : 'bg-surface-primary text-text-secondary'}"
			style={activeTab === tab.id ? 'font-size:12px;box-shadow:0 3px 10px var(--glow-accent);' : 'font-size:12px;border:1px solid var(--border-default);'}
			onclick={() => (activeTab = tab.id as 'nearby' | 'hot' | 'recent')}
		>
			<tab.icon class="size-2.75"  strokeWidth={activeTab === tab.id ? 2.5 : 1.5} />
			{tab.label}
		</button>
	{/each}
</div>

<!-- List -->
<div class="flex-1 min-h-0 overflow-y-auto no-scrollbar px-3 pb-3">
	{#if activeTab === 'nearby'}
		<div class="animate-fade-in flex flex-col gap-2">
			<!-- Current location card -->
			<div class="flex items-center gap-3 p-3.5 rounded-control bg-accent-default text-text-on-accent">
				<div class="flex items-center justify-center rounded-full shrink-0 size-10.5"  style="background:var(--surface-map-control)">
					<Navigation class="size-4.25"  strokeWidth={2} />
				</div>
				<div class="flex-1">
					<p class="text-base" >当前位置</p>
					<p class="text-xs mt-px"  style="opacity:0.8">{locationName || '自动定位'}</p>
				</div>
				<div class="bg-status-online size-2 shrink-0"  style="border-radius:50%;box-shadow:0 0 6px var(--status-online)"></div>
			</div>

			{#if nearbyPins.length}
				<ListContainer>
					{#each nearbyPins as pin (pin.id)}
						{@const isSelected = selectedPin?.id === pin.id}
						<ListItem
							pressed={isSelected}
							onClick={() => onSelectPin(pin)}
							ariaLabel={`选择 ${pin.name || pin.title || ''}`}
							class="app-list-row app-list-row--standard rounded-control"
						>
						<PinMedia {pin} size={42} />
						<div class="flex-1 min-w-0">
							<p class="{isSelected ? 'text-content-inverse' : 'text-text-primary'} text-base">{pin.name || pin.title}</p>
							<div class="flex items-center gap-2 mt-0.5 flex-wrap">
								<span style="font-size:10px;padding:1px 6px;border-radius:4px;background:{pinTone(pin.type, 15)};color:{PIN_COLORS[pin.type]};">
									{PIN_LABELS[pin.type]}
								</span>
								{#if pin.tag}
									<span class="{isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'} text-xs">#{pin.tag}</span>
								{/if}
								{#if pin.distance}
									<span class="{isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'} text-xs">{pin.distance}</span>
								{/if}
								{#if pin.members}
									<span class="{isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'} text-xs">{pin.members}人</span>
								{/if}
								{#if pin.likes}
									<span class="flex items-center gap-0.5 {isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'}">
										<Heart class="size-2.25"  strokeWidth={1.5} />
										<span class="text-xs">{pin.likes}</span>
									</span>
								{/if}
							</div>
						</div>
						{#if isSelected}
							<span class="rounded-full px-2 py-1" style="font-size:10px;color:white;background:{PIN_COLORS[pin.type]};shrink:0;">已定位</span>
						{:else}
							<ChevronRight class="size-3.5 shrink-0"  style="color:var(--text-secondary)" strokeWidth={1.5} />
						{/if}
					</ListItem>
				{/each}
				</ListContainer>
			{:else}
				<div class="flex flex-col items-center gap-2 py-10">
					<MapPin class="size-5.5"  style="color:var(--text-secondary);opacity:0.3" strokeWidth={1} />
					<p class="text-text-secondary text-xs">附近暂无内容，放大或移动地图后刷新</p>
				</div>
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
							class="app-list-row app-list-row--standard rounded-control"
						>
							<div class="flex items-center justify-center rounded-small shrink-0 bg-accent-default size-9 text-lg"  style="opacity:0.88">
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate {isSelected ? 'text-content-inverse' : 'text-text-primary'} text-sm">{loc.displayName}</p>
								<p class="truncate {isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
							</div>
							<div class="flex shrink-0 items-center gap-1">
								<Users style="width:11px;height:11px;color:{isSelected ? 'var(--content-inverse)' : 'var(--accent-default)'};" strokeWidth={1.5} />
								<span class="{isSelected ? 'text-content-inverse-secondary' : 'text-accent-default'} text-xs">{loc.users}</span>
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
				<div class="flex flex-col items-center gap-2 py-10">
					<Star class="size-5.5"  style="color:var(--text-secondary);opacity:0.3" strokeWidth={1} />
					<p class="text-text-secondary text-xs">暂无热门地点</p>
				</div>
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
							class="app-list-row app-list-row--standard rounded-control"
						>
							<div class="flex items-center justify-center rounded-full shrink-0 bg-surface-primary size-9 text-lg border border-border-default shadow-neu-xs"  >
								{loc.emoji}
							</div>
							<div class="flex-1 min-w-0">
								<p class="truncate {isSelected ? 'text-content-inverse' : 'text-text-primary'} text-sm">{loc.displayName}</p>
								<p class="truncate {isSelected ? 'text-content-inverse-secondary' : 'text-text-secondary'} text-xs mt-px" >{loc.desc}</p>
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
				<div class="flex flex-col items-center gap-2 py-10">
					<Clock class="size-5.5"  style="color:var(--text-secondary);opacity:0.3" strokeWidth={1} />
					<p class="text-text-secondary text-xs">暂无最近记录</p>
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Confirm -->
<div class="shrink-0 border-t border-border-default"   style="padding:10px 12px 16px">
	{#if selectedLoc}
		<div class="flex items-center gap-2 mb-2 px-3 py-2.5 rounded-[13px] bg-surface-primary" style="box-shadow:0 0 0 1px var(--border-default), var(--shadow-neu-inset-xs);">
			<Check class="size-3.25 shrink-0"  style="color:var(--accent-default)" strokeWidth={2.5} />
			<p class="flex-1 truncate text-accent-default text-xs">{selectedLoc.displayName}</p>
			<button
				type="button"
				class="hover-btn outline-none"
				onclick={() => {
					selectedLoc = null;
					searchQuery = '';
				}}
			>
				<X class="size-3"  style="color:var(--text-secondary)" strokeWidth={2} />
			</button>
		</div>
	{/if}
	<button
		type="button"
		class="hover-btn w-full flex items-center justify-center gap-2 rounded-full outline-none bg-accent-default text-text-on-accent"
		style="height:48px;font-size:14px;box-shadow:0 6px 20px var(--glow-accent);"
		onclick={onConfirm}
	>
		<MapPin class="size-3.5"  strokeWidth={2} />
		{selectedLoc ? '确认此位置' : '设为浏览锚点'}
	</button>
</div>
