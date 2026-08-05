<script lang="ts">
	import { Search, MapPin, Navigation, X, Users, ChevronRight, Clock, Star, TrendingUp, Crosshair, Check, Heart } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import BorderedList from '@/components/data-display/BorderedList.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import { RECENT_LOCATIONS, HOT_LOCATIONS, type PinType, type Pin, type LocationResult } from '../shared/location-data';
	import { PIN_COLORS, PIN_LABELS, pinTone, getMarkerUrl } from '../shared/location-shared';

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
				<Crosshair style="width:15px;height:15px;color:var(--accent-default);" strokeWidth={1.75} />
			</button>
		{/snippet}
	</PageHeader>
</div>

<!-- Search -->
<div class="shrink-0 px-4 pt-3 pb-2 relative">
	<div class="flex items-center gap-2.5 px-3.5 rounded-xl input-surface" style="height:42px;">
		<Search style="width:14px;height:14px;color:var(--text-secondary);shrink:0;" strokeWidth={1.5} />
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
				<X style="width:13px;height:13px;color:var(--text-secondary);" strokeWidth={2} />
			</button>
		{/if}
	</div>
	{#if searchResults.length > 0}
		<BorderedList class="absolute top-full left-4 right-4 z-30 mt-1" style="box-shadow:0 0 0 1px var(--border-default), 0 10px 36px var(--elevation-shadow-dark);">
			{#each searchResults as r}
				<button
					type="button"
					class="bg-surface-primary flex w-full items-center gap-3 px-4 py-3 text-left outline-none transition-[background-color,opacity] duration-150 hover:bg-interactive-hover-fill focus-visible:bg-interactive-hover-fill active:opacity-80"
					aria-pressed={selectedLoc?.displayName === r.displayName}
					onclick={() => onSelectLoc(r)}
				>
					<MapPin style="width:13px;height:13px;color:var(--accent-default);shrink:0;" strokeWidth={1.5} />
					<span class="text-text-primary" style="font-size:13px;flex:1;">{r.displayName}</span>
					{#if selectedLoc?.displayName === r.displayName}
						<Check style="width:13px;height:13px;color:var(--accent-default);" strokeWidth={2.5} />
					{/if}
				</button>
			{/each}
		</BorderedList>
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
			class="hover-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none {activeTab === tab.id ? 'bg-accent-default text-white' : 'bg-surface-primary text-text-secondary'}"
			style={activeTab === tab.id ? 'font-size:12px;box-shadow:0 3px 10px var(--glow-accent);' : 'font-size:12px;border:1px solid var(--border-default);'}
			onclick={() => (activeTab = tab.id as 'nearby' | 'hot' | 'recent')}
		>
			<tab.icon style="width:11px;height:11px;" strokeWidth={activeTab === tab.id ? 2.5 : 1.5} />
			{tab.label}
		</button>
	{/each}
</div>

<!-- List -->
<div class="flex-1 min-h-0 overflow-y-auto no-scrollbar px-3 pb-3">
	{#if activeTab === 'nearby'}
		<div class="animate-fade-in flex flex-col gap-2">
			<!-- Current location card -->
			<div class="flex items-center gap-3 p-3.5 rounded-control bg-accent-default text-white">
				<div class="flex items-center justify-center rounded-full shrink-0" style="width:42px;height:42px;background:rgba(255,255,255,0.2);">
					<Navigation style="width:17px;height:17px;" strokeWidth={2} />
				</div>
				<div class="flex-1">
					<p style="font-size:13px;">当前位置</p>
					<p style="font-size:11px;opacity:0.8;margin-top:1px;">{locationName || '自动定位'}</p>
				</div>
				<div class="bg-status-online" style="width:8px;height:8px;border-radius:50%;box-shadow:0 0 6px var(--status-online);shrink:0;"></div>
			</div>

			{#if nearbyPins.length}
				{#each nearbyPins as pin (pin.id)}
					{@const isSelected = selectedPin?.id === pin.id}
					<button
						type="button"
						class="app-list-row app-list-row--standard outline-none transition-[background-color,opacity] duration-150 {isSelected ? '' : 'hover:bg-interactive-hover-fill focus-visible:bg-interactive-hover-fill'} active:opacity-80"
						style={isSelected ? `background:${pinTone(pin.type, 10)};` : ''}
						aria-pressed={isSelected}
						onclick={() => onSelectPin(pin)}
					>
						{#if pin.type === 'user' && pin.avatar}
							<div class="relative shrink-0">
								<img src={pin.avatar} alt={pin.name || pin.title || ''} class="rounded-full object-cover" style="width:42px;height:42px;" />
								{#if pin.online}
									<span class="absolute bottom-0 right-0 rounded-full bg-status-online" style="width:10px;height:10px;border:2px solid var(--surface-primary);"></span>
								{/if}
							</div>
						{:else if pin.type === 'moment' && pin.image}
							<img src={pin.image} alt={pin.title || pin.name || ''} class="rounded-[11px] object-cover shrink-0" style="width:42px;height:42px;" />
						{:else}
							<div class="flex items-center justify-center rounded-small shrink-0" style="width:42px;height:42px;background:{pinTone(pin.type, 12)};">
								<img src={getMarkerUrl(pin.type, 1)} alt={pin.type} style="width:20px;height:27px;" />
							</div>
						{/if}
						<div class="flex-1 min-w-0">
							<p class="text-text-primary text-base">{pin.name || pin.title}</p>
							<div class="flex items-center gap-2 mt-0.5 flex-wrap">
								<span style="font-size:10px;padding:1px 6px;border-radius:4px;background:{pinTone(pin.type, 15)};color:{PIN_COLORS[pin.type]};">
									{PIN_LABELS[pin.type]}
								</span>
								{#if pin.tag}
									<span class="text-text-secondary text-xs">#{pin.tag}</span>
								{/if}
								{#if pin.distance}
									<span class="text-text-secondary text-xs">{pin.distance}</span>
								{/if}
								{#if pin.members}
									<span class="text-text-secondary text-xs">{pin.members}人</span>
								{/if}
								{#if pin.likes}
									<span class="flex items-center gap-0.5 text-text-secondary">
										<Heart style="width:9px;height:9px;color:var(--text-secondary);" strokeWidth={1.5} />
										<span class="text-xs">{pin.likes}</span>
									</span>
								{/if}
							</div>
						</div>
						{#if isSelected}
							<span class="rounded-full px-2 py-1" style="font-size:10px;color:white;background:{PIN_COLORS[pin.type]};shrink:0;">已定位</span>
						{:else}
							<ChevronRight style="width:14px;height:14px;color:var(--text-secondary);shrink:0;" strokeWidth={1.5} />
						{/if}
					</button>
				{/each}
			{:else}
				<div class="flex flex-col items-center gap-2 py-10">
					<MapPin style="width:22px;height:22px;color:var(--text-secondary);opacity:0.3;" strokeWidth={1} />
					<p class="text-text-secondary text-xs">附近暂无内容，放大或移动地图后刷新</p>
				</div>
			{/if}
		</div>
	{:else if activeTab === 'hot'}
		<div class="animate-fade-in flex flex-col gap-2">
			{#if HOT_LOCATIONS.length}
				{#each HOT_LOCATIONS as loc}
					{@const isSelected = selectedLoc?.displayName === loc.displayName}
					<button
						type="button"
						class="bg-surface-primary flex w-full items-center gap-3 rounded-control p-3.5 text-left outline-none transition-[background-color,box-shadow,opacity] duration-150 {isSelected ? '' : 'hover:bg-interactive-hover-fill focus-visible:bg-interactive-hover-fill'} active:opacity-80"
						style="box-shadow:{isSelected ? '0 0 0 1px var(--accent-border), var(--shadow-neu-md)' : 'var(--shadow-neu-md)'};border:1px solid {isSelected ? 'var(--accent-border)' : 'var(--border-default)'};"
						aria-pressed={isSelected}
						onclick={() => onSelectLoc(loc)}
					>
						<div class="flex items-center justify-center rounded-xl shrink-0 bg-accent-default" style="width:46px;height:46px;font-size:22px;opacity:0.88;">
							{loc.emoji}
						</div>
						<div class="flex-1 min-w-0">
							<p class="text-text-primary text-sm">{loc.displayName}</p>
							<p class="text-text-secondary" style="font-size:12px;margin-top:2px;">{loc.desc}</p>
						</div>
						<div class="flex flex-col items-end shrink-0 gap-0.5">
							<div class="flex items-center gap-1">
								<Users style="width:11px;height:11px;color:var(--accent-default);" strokeWidth={1.5} />
								<span class="text-accent-default text-xs">{loc.users}</span>
							</div>
							<span class="text-text-secondary text-xxs">在线</span>
						</div>
					</button>
				{/each}
			{:else}
				<div class="flex flex-col items-center gap-2 py-10">
					<Star style="width:22px;height:22px;color:var(--text-secondary);opacity:0.3;" strokeWidth={1} />
					<p class="text-text-secondary text-xs">暂无热门地点</p>
				</div>
			{/if}
		</div>
	{:else}
		<div class="animate-fade-in flex flex-col gap-2">
			{#if RECENT_LOCATIONS.length}
				{#each RECENT_LOCATIONS as loc}
					{@const isSelected = selectedLoc?.displayName === loc.displayName}
					<button
						type="button"
						class="bg-surface-primary flex w-full items-center gap-3 rounded-control p-3.5 text-left outline-none transition-[background-color,box-shadow,opacity] duration-150 {isSelected ? '' : 'hover:bg-interactive-hover-fill focus-visible:bg-interactive-hover-fill'} active:opacity-80"
						style="box-shadow:var(--shadow-neu-md);border:1px solid var(--border-default);"
						aria-pressed={isSelected}
						onclick={() => onSelectLoc(loc)}
					>
						<div class="flex items-center justify-center rounded-full shrink-0 bg-surface-primary" style="width:44px;height:44px;box-shadow:var(--shadow-neu-xs);border:1px solid var(--border-default);font-size:20px;">
							{loc.emoji}
						</div>
						<div class="flex-1 min-w-0">
							<p class="text-text-primary text-sm">{loc.displayName}</p>
							<p class="text-text-secondary" style="font-size:12px;margin-top:2px;">{loc.desc}</p>
						</div>
						{#if isSelected}
							<Check style="width:14px;height:14px;color:var(--accent-default);shrink:0;" strokeWidth={2.5} />
						{/if}
					</button>
				{/each}
			{:else}
				<div class="flex flex-col items-center gap-2 py-10">
					<Clock style="width:22px;height:22px;color:var(--text-secondary);opacity:0.3;" strokeWidth={1} />
					<p class="text-text-secondary text-xs">暂无最近记录</p>
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Confirm -->
<div style="shrink:0;padding:10px 12px 16px;border-top:1px solid var(--border-default);">
	{#if selectedLoc}
		<div class="flex items-center gap-2 mb-2 px-3 py-2.5 rounded-[13px] bg-surface-primary" style="box-shadow:0 0 0 1px var(--border-default), var(--shadow-neu-inset-xs);">
			<Check style="width:13px;height:13px;color:var(--accent-default);shrink:0;" strokeWidth={2.5} />
			<p class="flex-1 truncate text-accent-default text-xs">{selectedLoc.displayName}</p>
			<button
				type="button"
				class="hover-btn outline-none"
				onclick={() => {
					selectedLoc = null;
					searchQuery = '';
				}}
			>
				<X style="width:12px;height:12px;color:var(--text-secondary);" strokeWidth={2} />
			</button>
		</div>
	{/if}
	<button
		type="button"
		class="hover-btn w-full flex items-center justify-center gap-2 rounded-full outline-none bg-accent-default text-white"
		style="height:48px;font-size:14px;box-shadow:0 6px 20px var(--glow-accent);"
		onclick={onConfirm}
	>
		<MapPin style="width:14px;height:14px;" strokeWidth={2} />
		{selectedLoc ? '确认此位置' : '设为浏览锚点'}
	</button>
</div>
