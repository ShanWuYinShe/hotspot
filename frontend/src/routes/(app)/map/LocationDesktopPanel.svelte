<script lang="ts">
	import { Search, MapPin, Navigation, X, Users, ChevronRight, Clock, Star, TrendingUp, Crosshair, Check, Heart } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import BorderedList from '@/components/data-display/BorderedList.svelte';
	import { RECENT_LOCATIONS, HOT_LOCATIONS, type PinType, type Pin, type LocationResult } from './location-data';
	import { PIN_COLORS, PIN_LABELS, pinTone, getMarkerUrl } from './location-shared';

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
		hoveredListId = $bindable<string | null>(null),
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
		hoveredListId?: string | null;
		mapCounts?: Record<PinType, number>;
		/** 选中附近图钉：由父组件同步设置 selectedPin 并移动地图中心。 */
		onSelectPin: (pin: Pin) => void;
		onSelectLoc: (loc: LocationResult) => void;
		onReset: () => void;
		onConfirm: () => void;
	} = $props();
</script>

<!-- Header -->
<div class="shrink-0">
	<PageHeader title="位置选择" onBack={() => history.back()} subtitle="江苏·徐州" sticky>
		{#snippet trailing()}
			<button
				type="button"
				class="neu-hover-btn flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-circle-hover bg-[var(--neu-bg)]"
				style="width:36px;height:36px;border:1px solid var(--neu-border);box-shadow:4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light);"
				onclick={onReset}
			>
				<Crosshair style="width:15px;height:15px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
			</button>
		{/snippet}
	</PageHeader>
</div>

<!-- Search -->
<div class="shrink-0 px-4 pt-3 pb-2 relative">
	<div class="flex items-center gap-2.5 px-3.5 rounded-[14px] neu-input-surface" style="height:42px;">
		<Search style="width:14px;height:14px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
		<input
			bind:value={searchQuery}
			oninput={(e) => {
				searchQuery = e.currentTarget.value;
				selectedLoc = null;
			}}
			placeholder="搜索地点或地标…"
			class="flex-1 bg-transparent outline-none text-[var(--neu-text)] text-[13px]"
		/>
		{#if searchQuery}
			<button type="button" class="neu-hover-btn outline-none" onclick={() => (searchQuery = '')}>
				<X style="width:13px;height:13px;color:var(--neu-text-muted);" strokeWidth={2} />
			</button>
		{/if}
	</div>
	{#if searchResults.length > 0}
		<BorderedList class="absolute top-full left-4 right-4 z-30 mt-1" style="box-shadow:0 0 0 1px var(--neu-border), 0 10px 36px var(--neu-shadow-dark);">
			{#each searchResults as r}
				<button
					type="button"
					class="neu-row-hover flex w-full items-center gap-3 px-4 py-3 text-left outline-none"
					style="background:var(--neu-bg);"
					aria-pressed={selectedLoc?.displayName === r.displayName}
					onclick={() => onSelectLoc(r)}
				>
					<MapPin style="width:13px;height:13px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={1.5} />
					<span class="text-[var(--neu-text)]" style="font-size:13px;flex:1;">{r.displayName}</span>
					{#if selectedLoc?.displayName === r.displayName}
						<Check style="width:13px;height:13px;color:var(--neu-accent-primary);" strokeWidth={2.5} />
					{/if}
				</button>
			{/each}
		</BorderedList>
	{/if}
</div>

<!-- Tabs -->
<div class="flex-shrink-0 flex gap-1.5 px-4 pb-3">
	{#each [
		{ id: 'nearby', label: '附近', icon: TrendingUp },
		{ id: 'hot', label: '热门', icon: Star },
		{ id: 'recent', label: '最近', icon: Clock }
	] as tab}
		<button
			type="button"
			class="neu-hover-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none {activeTab === tab.id ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-[var(--neu-bg)] text-[var(--neu-text-muted)]'}"
			style={activeTab === tab.id ? 'font-size:12px;box-shadow:0 3px 10px var(--neu-glow);' : 'font-size:12px;border:1px solid var(--neu-border);'}
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
			<div class="flex items-center gap-3 p-3.5 rounded-[16px] bg-[var(--neu-accent-primary)] text-white">
				<div class="flex items-center justify-center rounded-full flex-shrink-0" style="width:42px;height:42px;background:rgba(255,255,255,0.2);">
					<Navigation style="width:17px;height:17px;" strokeWidth={2} />
				</div>
				<div class="flex-1">
					<p style="font-size:13px;">当前位置</p>
					<p style="font-size:11px;opacity:0.8;margin-top:1px;">江苏·徐州 · 精度 ±12m</p>
				</div>
				<div class="bg-[var(--semantic-online)]" style="width:8px;height:8px;border-radius:50%;box-shadow:0 0 6px var(--semantic-online);flex-shrink:0;"></div>
			</div>

			{#if nearbyPins.length}
				{#each nearbyPins as pin (pin.id)}
					<button
						type="button"
						class="neu-row-hover app-list-row app-list-row--standard outline-none"
						style="background:{selectedPin?.id === pin.id ? pinTone(pin.type, 10) : 'var(--neu-bg)'};"
						aria-pressed={selectedPin?.id === pin.id}
						onclick={() => onSelectPin(pin)}
					>
						{#if pin.type === 'user' && pin.avatar}
							<div class="relative flex-shrink-0">
								<img src={pin.avatar} alt={pin.name || pin.title || ''} class="rounded-full object-cover" style="width:42px;height:42px;" />
								{#if pin.online}
									<span class="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]" style="width:10px;height:10px;border:2px solid var(--neu-bg);"></span>
								{/if}
							</div>
						{:else if pin.type === 'moment' && pin.image}
							<img src={pin.image} alt={pin.title || pin.name || ''} class="rounded-[11px] object-cover flex-shrink-0" style="width:42px;height:42px;" />
						{:else}
							<div class="flex items-center justify-center rounded-[12px] flex-shrink-0" style="width:42px;height:42px;background:{pinTone(pin.type, 12)};">
								<img src={getMarkerUrl(pin.type, 1)} alt={pin.type} style="width:20px;height:27px;" />
							</div>
						{/if}
						<div class="flex-1 min-w-0">
							<p class="text-[var(--neu-text)] text-[13px]">{pin.name || pin.title}</p>
							<div class="flex items-center gap-2 mt-0.5 flex-wrap">
								<span style="font-size:10px;padding:1px 6px;border-radius:4px;background:{pinTone(pin.type, 15)};color:{PIN_COLORS[pin.type]};">
									{PIN_LABELS[pin.type]}
								</span>
								{#if pin.tag}
									<span class="text-[var(--neu-text-muted)] text-[11px]">#{pin.tag}</span>
								{/if}
								{#if pin.distance}
									<span class="text-[var(--neu-text-muted)] text-[11px]">{pin.distance}</span>
								{/if}
								{#if pin.members}
									<span class="text-[var(--neu-text-muted)] text-[11px]">{pin.members}人</span>
								{/if}
								{#if pin.likes}
									<span class="flex items-center gap-0.5 text-[var(--neu-text-muted)]">
										<Heart style="width:9px;height:9px;color:var(--neu-text-muted);" strokeWidth={1.5} />
										<span class="text-[11px]">{pin.likes}</span>
									</span>
								{/if}
							</div>
						</div>
						{#if selectedPin?.id === pin.id}
							<span class="rounded-full px-2 py-1" style="font-size:10px;color:white;background:{PIN_COLORS[pin.type]};flex-shrink:0;">已定位</span>
						{:else}
							<ChevronRight style="width:14px;height:14px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
						{/if}
					</button>
				{/each}
			{:else}
				<div class="flex flex-col items-center gap-2 py-10">
					<MapPin style="width:22px;height:22px;color:var(--neu-text-muted);opacity:0.3;" strokeWidth={1} />
					<p class="text-[var(--neu-text-muted)] text-xs">附近暂无内容，放大或移动地图后刷新</p>
				</div>
			{/if}
		</div>
	{:else if activeTab === 'hot'}
		<div class="animate-fade-in flex flex-col gap-2">
			{#if HOT_LOCATIONS.length}
				{#each HOT_LOCATIONS as loc, i}
					<button
						type="button"
						class="neu-row-hover app-list-row app-list-row--rich rounded-[16px] outline-none"
						style="background:{selectedLoc?.displayName === loc.displayName ? 'var(--neu-accent-soft)' : 'var(--neu-bg)'};box-shadow:{selectedLoc?.displayName === loc.displayName ? '0 0 0 1px var(--neu-accent-soft-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)' : '0 0 0 1px var(--neu-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)'};"
						aria-pressed={selectedLoc?.displayName === loc.displayName}
						onclick={() => onSelectLoc(loc)}
					>
						<div class="flex items-center justify-center rounded-[14px] flex-shrink-0 bg-[var(--neu-accent-primary)]" style="width:46px;height:46px;font-size:22px;opacity:0.88;">
							{loc.emoji}
						</div>
						<div class="flex-1 min-w-0">
							<p class="text-[var(--neu-text)] text-sm">{loc.displayName}</p>
							<p class="text-[var(--neu-text-muted)]" style="font-size:12px;margin-top:2px;">{loc.desc}</p>
						</div>
						<div class="flex flex-col items-end flex-shrink-0 gap-0.5">
							<div class="flex items-center gap-1">
								<Users style="width:11px;height:11px;color:var(--neu-accent-primary);" strokeWidth={1.5} />
								<span class="text-[var(--neu-accent-primary)] text-xs">{loc.users}</span>
							</div>
							<span class="text-[var(--neu-text-muted)] text-[10px]">在线</span>
						</div>
					</button>
				{/each}
			{:else}
				<div class="flex flex-col items-center gap-2 py-10">
					<Star style="width:22px;height:22px;color:var(--neu-text-muted);opacity:0.3;" strokeWidth={1} />
					<p class="text-[var(--neu-text-muted)] text-xs">暂无热门地点</p>
				</div>
			{/if}
		</div>
	{:else}
		<div class="animate-fade-in flex flex-col gap-2">
			{#if RECENT_LOCATIONS.length}
				{#each RECENT_LOCATIONS as loc, i}
					<button
						type="button"
						class="neu-row-hover app-list-row app-list-row--rich rounded-[16px] outline-none"
						style="background:{selectedLoc?.displayName === loc.displayName ? 'var(--neu-accent-soft)' : 'var(--neu-bg)'};box-shadow:{selectedLoc?.displayName === loc.displayName || hoveredListId === `recent-${i}` ? '0 0 0 1px var(--neu-accent-soft-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)' : '0 0 0 1px var(--neu-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)'};"
						aria-pressed={selectedLoc?.displayName === loc.displayName}
						onclick={() => onSelectLoc(loc)}
					>
						<div class="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-bg)]" style="width:44px;height:44px;box-shadow:3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light);border:1px solid var(--neu-border);font-size:20px;">
							{loc.emoji}
						</div>
						<div class="flex-1 min-w-0">
							<p class="text-[var(--neu-text)] text-sm">{loc.displayName}</p>
							<p class="text-[var(--neu-text-muted)]" style="font-size:12px;margin-top:2px;">{loc.desc}</p>
						</div>
						{#if selectedLoc?.displayName === loc.displayName}
							<Check style="width:14px;height:14px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={2.5} />
						{/if}
					</button>
				{/each}
			{:else}
				<div class="flex flex-col items-center gap-2 py-10">
					<Clock style="width:22px;height:22px;color:var(--neu-text-muted);opacity:0.3;" strokeWidth={1} />
					<p class="text-[var(--neu-text-muted)] text-xs">暂无最近记录</p>
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Confirm -->
<div style="flex-shrink:0;padding:10px 12px 16px;border-top:1px solid var(--neu-border);">
	{#if selectedLoc}
		<div class="flex items-center gap-2 mb-2 px-3 py-2.5 rounded-[13px] bg-[var(--neu-bg)]" style="box-shadow:0 0 0 1px var(--neu-border), inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light);">
			<Check style="width:13px;height:13px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={2.5} />
			<p class="flex-1 truncate text-[var(--neu-accent-primary)] text-xs">{selectedLoc.displayName}</p>
			<button
				type="button"
				class="neu-hover-btn outline-none"
				onclick={() => {
					selectedLoc = null;
					searchQuery = '';
				}}
			>
				<X style="width:12px;height:12px;color:var(--neu-text-muted);" strokeWidth={2} />
			</button>
		</div>
	{/if}
	<button
		type="button"
		class="neu-hover-btn w-full flex items-center justify-center gap-2 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white"
		style="height:48px;font-size:14px;box-shadow:0 6px 20px var(--neu-glow);"
		onclick={onConfirm}
	>
		<MapPin style="width:14px;height:14px;" strokeWidth={2} />
		{selectedLoc ? '确认此位置' : '设为浏览锚点'}
	</button>
</div>
