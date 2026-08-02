<script lang="ts">
	import { Search, MapPin, Navigation, X, Users, ChevronRight, Clock, Star, TrendingUp, Crosshair, Check } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import { ROUTES } from '@/lib/routes';
	import { XUZHOU_CENTER, RECENT_LOCATIONS, HOT_LOCATIONS, SEARCH_DB, type PinType, type Pin, type LocationResult } from './location-data';
	import {
		PIN_COLORS, PIN_LABELS, pinTone, buildMapData, getMarkerUrl, fetchMapCanvas,
		type MapHeatBucket,
		markerUserUrl, markerGroupUrl, markerMomentUrl
	} from './location-shared';
	import MapPinMarker from './MapPinMarker.svelte';
	import MapHeatDot from './MapHeatDot.svelte';

	// 桌面地图探索：左侧栏（统计/搜索/Tab 列表/确认）+ 地图面板（OSM iframe + 图钉）。
	// 图钉与统计来自 POST /api/v1/map/canvas（spec: api_spec_map.md）。

	let searchQuery = $state('');
	let selectedPin = $state<Pin | null>(null);
	let selectedLoc = $state<LocationResult | null>(null);
	let activeFilters = $state<Set<PinType>>(new Set(['user', 'group', 'post']));
	let activeTab = $state<'nearby' | 'hot' | 'recent'>('nearby');
	let hoveredListId = $state<string | null>(null);
	const zoom = 0.045;
	let mapCenter = $state(XUZHOU_CENTER);

	// map/canvas 数据
	let mapPins = $state<Pin[]>([]);
	let heatBuckets = $state<MapHeatBucket[]>([]);
	let mapCounts = $state<Record<PinType, number>>({ user: 0, group: 0, post: 0 });
	let mapLoading = $state(true);

	// 视口中心或筛选变化时重新拉取
	$effect(() => {
		const center = mapCenter;
		const types = [...activeFilters];
		let cancelled = false;
		mapLoading = true;
		void fetchMapCanvas(center, zoom, types).then((r) => {
			if (cancelled) return;
			mapPins = r.pins;
			heatBuckets = r.heatBuckets;
			mapCounts = r.counts;
			mapLoading = false;
		});
		return () => {
			cancelled = true;
		};
	});

	// 图钉详情跳转：user/group/post 分别路由到对应详情页
	function openPinDetail(pin: Pin) {
		if (pin.type === 'user') void goto(ROUTES.user(pin.id));
		else if (pin.type === 'group') void goto(ROUTES.group(pin.id));
		else void goto(ROUTES.moment(pin.id));
	}

	function toggleFilter(type: PinType) {
		const next = new Set(activeFilters);
		next.has(type) ? next.delete(type) : next.add(type);
		activeFilters = next;
	}

	const searchResults = $derived(
		searchQuery.trim() ? SEARCH_DB.filter((s) => s.displayName.includes(searchQuery)).slice(0, 6) : []
	);
	const nearbyPins = $derived(mapPins.filter((p) => (p.count ?? 1) === 1));

	const { url: embedUrl, bounds } = $derived(buildMapData(mapCenter, zoom));

	const userCount = $derived(mapCounts.user);
	const groupCount = $derived(mapCounts.group);
	const postCount = $derived(mapCounts.post);

	function handleSelectLoc(loc: LocationResult) {
		selectedLoc = loc;
		searchQuery = loc.displayName;
		mapCenter = { lat: loc.lat, lng: loc.lng };
		selectedPin = null;
	}

	function handleConfirm() {
		if (selectedLoc) {
			try {
				localStorage.setItem('hotspot_last_location', JSON.stringify([selectedLoc.lat, selectedLoc.lng]));
			} catch {}
		}
		history.back();
	}
</script>

<div class="flex h-full bg-[var(--neu-bg)]">
	<!-- ══ LEFT SIDEBAR ══ -->
	<div class="flex flex-col flex-shrink-0" style="width:340px;border-right:1px solid var(--neu-border);">
		<!-- Header -->
		<div class="flex-shrink-0">
			<PageHeader title="地图探索" onBack={() => history.back()} subtitle="江苏·徐州" sticky>
				{#snippet trailing()}
					<button
						type="button"
						class="neu-hover-btn flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-circle-hover bg-[var(--neu-bg)]"
						style="width:36px;height:36px;border:1px solid var(--neu-border);box-shadow:4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light);"
						onclick={() => {
							mapCenter = XUZHOU_CENTER;
							selectedLoc = null;
							searchQuery = '';
						}}
					>
						<Crosshair style="width:15px;height:15px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
					</button>
				{/snippet}
			</PageHeader>

			<!-- Type stat chips -->
			<div class="flex gap-2 px-5 pb-3">
				{#each [
					{ type: 'user', count: userCount, color: PIN_COLORS.user, label: '用户', icon: markerUserUrl },
					{ type: 'group', count: groupCount, color: PIN_COLORS.group, label: '群组', icon: markerGroupUrl },
					{ type: 'post', count: postCount, color: PIN_COLORS.post, label: '动态', icon: markerMomentUrl }
				] as chip}
					{@const active = activeFilters.has(chip.type as PinType)}
					<button
						type="button"
						class="neu-hover-btn flex-1 flex items-center gap-1.5 rounded-[13px] px-2 py-2 outline-none"
						style="background:{active ? pinTone(chip.type as PinType, 14) : 'var(--neu-bg)'};border:1px solid {active ? pinTone(chip.type as PinType, 40) : 'var(--neu-border)'};box-shadow:{active ? `0 2px 8px ${pinTone(chip.type as PinType, 25)}` : '4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)'};"
						onclick={() => toggleFilter(chip.type as PinType)}
					>
						<img src={chip.icon} alt={chip.label} style="width:14px;height:19px;flex-shrink:0;" />
						<div class="flex-1 min-w-0">
							<p style="font-size:15px;color:{active ? chip.color : 'var(--neu-text)'};line-height:1;">{chip.count}</p>
							<p class="text-[var(--neu-text-muted)]" style="font-size:10px;margin-top:1px;">{chip.label}</p>
						</div>
					</button>
				{/each}
			</div>
		</div>

		<!-- Search -->
		<div class="flex-shrink-0 px-4 pt-3 pb-2 relative">
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
				<div class="absolute top-full left-4 right-4 z-30 mt-1 rounded-[16px] bg-[var(--neu-bg)] overflow-hidden" style="box-shadow:0 0 0 1px var(--neu-border), 0 10px 36px var(--neu-shadow-dark);">
					<div style="overflow:hidden;border-radius:16px;">
						{#each searchResults as r, i}
							<button
								type="button"
								class="neu-hover-bg neu-row-hover flex w-full items-center gap-3 px-4 py-3 text-left outline-none"
								style="background:var(--neu-bg);border-bottom:{i < searchResults.length - 1 ? '1px solid var(--neu-border)' : 'none'};"
								aria-pressed={selectedLoc?.displayName === r.displayName}
								onclick={() => handleSelectLoc(r)}
							>
								<MapPin style="width:13px;height:13px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={1.5} />
								<span class="text-[var(--neu-text)]" style="font-size:13px;flex:1;">{r.displayName}</span>
								{#if selectedLoc?.displayName === r.displayName}
									<Check style="width:13px;height:13px;color:var(--neu-accent-primary);" strokeWidth={2.5} />
								{/if}
							</button>
						{/each}
					</div>
				</div>
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
							class="neu-row-hover app-list-row app-list-row--rich neu-row-hover rounded-[15px] outline-none"
							style="background:{selectedPin?.id === pin.id ? pinTone(pin.type, 10) : 'var(--neu-bg)'};box-shadow:{selectedPin?.id === pin.id ? `0 0 0 1px ${pinTone(pin.type, 52)}, 0 6px 18px ${pinTone(pin.type, 32)}, 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)` : '0 0 0 1px var(--neu-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)'};"
							aria-pressed={selectedPin?.id === pin.id}
							onclick={() => {
								selectedPin = pin;
								mapCenter = { lat: pin.lat, lng: pin.lng };
							}}
						>
							{#if pin.type === 'user' && pin.avatar}
								<div class="relative flex-shrink-0">
									<img src={pin.avatar} alt={pin.name || pin.title || ''} class="rounded-full object-cover" style="width:42px;height:42px;" />
									{#if pin.online}
										<span class="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]" style="width:10px;height:10px;border:2px solid var(--neu-bg);"></span>
									{/if}
								</div>
							{:else if pin.type === 'post' && pin.image}
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
							class="neu-hover-card neu-row-hover app-list-row app-list-row--rich neu-row-hover rounded-[16px] outline-none"
							style="background:{selectedLoc?.displayName === loc.displayName ? 'var(--neu-accent-soft)' : 'var(--neu-bg)'};box-shadow:{selectedLoc?.displayName === loc.displayName ? '0 0 0 1px var(--neu-accent-soft-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)' : '0 0 0 1px var(--neu-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)'};"
							aria-pressed={selectedLoc?.displayName === loc.displayName}
							onclick={() => handleSelectLoc(loc)}
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
							class="neu-hover-card neu-row-hover app-list-row app-list-row--rich neu-row-hover rounded-[16px] outline-none"
							style="background:{selectedLoc?.displayName === loc.displayName ? 'var(--neu-accent-soft)' : 'var(--neu-bg)'};box-shadow:{selectedLoc?.displayName === loc.displayName || hoveredListId === `recent-${i}` ? '0 0 0 1px var(--neu-accent-soft-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)' : '0 0 0 1px var(--neu-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)'};"
							aria-pressed={selectedLoc?.displayName === loc.displayName}
							onclick={() => handleSelectLoc(loc)}
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
				onclick={handleConfirm}
			>
				<MapPin style="width:14px;height:14px;" strokeWidth={2} />
				{selectedLoc ? '确认此位置' : '设为浏览锚点'}
			</button>
		</div>
	</div>

	<!-- ══ MAP PANEL ══ -->
	<div
		class="flex-1 relative overflow-hidden"
		role="button"
		tabindex="0"
		aria-label="地图（点击空白处取消选中）"
		onclick={(e) => {
			// 详情浮层内的点击不视为地图空白点击
			if ((e.target as HTMLElement).closest('[data-map-popup]')) return;
			selectedPin = null;
		}}
		onkeydown={(e) => {
			if (e.target !== e.currentTarget) return;
			if (e.key === 'Enter' || e.key === ' ' || e.key === 'Escape') {
				e.preventDefault();
				selectedPin = null;
			}
		}}
	>
		<iframe src={embedUrl} class="w-full h-full border-0" title="地图" style="pointer-events:none;"></iframe>

		<!-- Current location dot -->
		<div style="position:absolute;left:50%;top:50%;transform:translate(-50%,-50%);pointer-events:none;z-index:5;">
			<div style="width:18px;height:18px;border-radius:50%;background:{PIN_COLORS.user};border:3px solid var(--neu-bg);box-shadow:0 2px 10px {pinTone('user', 55)};position:relative;">
				<span class="absolute rounded-full" style="inset:-8px;background:{pinTone('user', 18)};border:1px solid {pinTone('user', 25)};"></span>
			</div>
		</div>

		<!-- Heatmap buckets（大视口 / 高密度时服务端返回热力统计） -->
		{#each heatBuckets as bucket (bucket.cell_token)}
			<MapHeatDot {bucket} {bounds} />
		{/each}

		<!-- Pin markers -->
		{#each mapPins as pin (pin.id)}
			<MapPinMarker
				pin={pin}
				{bounds}
				isSelected={selectedPin?.id === pin.id}
				onSelect={(e) => {
					e.stopPropagation();
					selectedPin = selectedPin?.id === pin.id ? null : pin;
					mapCenter = { lat: pin.lat, lng: pin.lng };
				}}
			/>
		{/each}

		{#if mapLoading}
			<!-- 加载中的轻量指示 -->
			<div class="absolute top-3 left-1/2 -translate-x-1/2 z-20 rounded-full bg-[var(--neu-bg)] px-3 py-1 text-[11px] text-[var(--neu-text-muted)]" style="box-shadow:0 2px 12px var(--neu-shadow-dark);border:1px solid var(--neu-border);">
				加载地图数据…
			</div>
		{/if}


		<!-- Selected pin popup -->
		{#if selectedPin}
			<div
				data-map-popup
				class="bg-[var(--neu-bg)]"
				style="position:absolute;bottom:20px;left:20px;right:20px;z-index:25;border-radius:20px;padding:16px;box-shadow:0 0 0 1px var(--neu-border), 0 16px 48px var(--neu-shadow-dark);"
			>
				<div class="flex items-center gap-3">
					{#if selectedPin.type === 'user' && selectedPin.avatar}
						<div class="relative flex-shrink-0">
							<img src={selectedPin.avatar} alt={selectedPin.name || selectedPin.title || ''} class="rounded-full object-cover" style="width:52px;height:52px;" />
							{#if selectedPin.online}
								<span class="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]" style="width:12px;height:12px;border:2.5px solid var(--neu-bg);"></span>
							{/if}
						</div>
					{:else if selectedPin.type === 'post' && selectedPin.image}
						<img src={selectedPin.image} alt={selectedPin.name || selectedPin.title || ''} class="rounded-[14px] object-cover flex-shrink-0" style="width:52px;height:52px;" />
					{:else}
						<div class="flex items-center justify-center rounded-[15px] flex-shrink-0" style="width:52px;height:52px;background:{pinTone(selectedPin.type, 14)};">
							<img src={getMarkerUrl(selectedPin.type, selectedPin.count ?? 1)} alt="" style="width:26px;height:35px;" />
						</div>
					{/if}
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2">
							<p class="text-[var(--neu-text)] text-[15px]">{selectedPin.name || selectedPin.title}</p>
							{#if (selectedPin.count ?? 1) > 1}
								<span style="font-size:10px;padding:1px 6px;border-radius:4px;background:{PIN_COLORS[selectedPin.type]};color:white;">
									{selectedPin.count}个
								</span>
							{/if}
						</div>
						<div class="flex items-center gap-2 mt-1 flex-wrap">
							<span style="font-size:11px;padding:1px 7px;border-radius:4px;background:{pinTone(selectedPin.type, 15)};color:{PIN_COLORS[selectedPin.type]};">
								{PIN_LABELS[selectedPin.type]}
							</span>
							{#if selectedPin.tag}
								<span class="text-[var(--neu-text-muted)] text-xs">#{selectedPin.tag}</span>
							{/if}
							{#if selectedPin.distance}
								<span class="text-[var(--neu-text-muted)] text-xs">{selectedPin.distance}</span>
							{/if}
						</div>
					</div>
					<div class="flex gap-2 flex-shrink-0">
						<button
							type="button"
							class="neu-hover-btn flex items-center gap-1.5 px-4 py-2 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white"
							style="font-size:13px;box-shadow:0 4px 14px var(--neu-glow);"
							onclick={() => selectedPin && openPinDetail(selectedPin)}
						>
							查看详情
							<ChevronRight style="width:13px;height:13px;" strokeWidth={2} />
						</button>
						<button
							type="button"
							class="neu-hover-btn outline-none flex items-center justify-center rounded-full bg-[var(--neu-bg)]"
							style="width:36px;height:36px;border:1px solid var(--neu-border);box-shadow:3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light);"
							onclick={() => (selectedPin = null)}
						>
							<X style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={2} />
						</button>
					</div>
				</div>
			</div>
		{/if}

		<!-- Attribution -->
		<div class="text-[var(--neu-text-muted)]" style="position:absolute;bottom:4px;left:8px;font-size:9px;opacity:0.38;pointer-events:none;z-index:1;">
			© OpenStreetMap contributors
		</div>
	</div>
</div>
