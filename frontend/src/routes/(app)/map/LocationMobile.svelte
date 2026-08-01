<script lang="ts">
	import { Search, MapPin, X, Users, ChevronRight, Clock, Star, SlidersHorizontal, Crosshair, Check } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import PageHeader from '@/components/shared/PageHeader.svelte';
	import { ROUTES } from '@/lib/routes';
	import * as Popover from '@/components/ui/popover';
	import { XUZHOU_CENTER, RECENT_LOCATIONS, HOT_LOCATIONS, SEARCH_DB, type PinType, type Pin, type LocationResult } from './location-data';
	import { SNAP_MIN, SNAP_MID, SNAP_MAX, SNAP_HEIGHTS, PIN_COLORS, PIN_LABELS, pinTone, floatBtnStyle, buildMapData, getMarkerUrl, fetchMapCanvas, type MapHeatBucket, markerUserUrl, markerGroupUrl, markerMomentUrl, type SnapState } from './location-shared';
	import MapPinMarker from './MapPinMarker.svelte';
	import MapHeatDot from './MapHeatDot.svelte';

	// 移动端地图探索：全屏 OSM + 可拖拽吸附底部 Sheet（148/318/540 三档）。
	// 迁移自旧版 LocationMobile（motion spring 吸附改 CSS transition）。

	/* ── Sheet drag ── */
	let sheetH = $state(SNAP_MID);
	let dragging = $state(false);
	let snap = $state<SnapState>('mid');
	let dragState = $state({ active: false, startY: 0, startH: SNAP_MID, velY: 0, prevY: 0, prevT: 0 });

	function snapTo(target: SnapState) {
		snap = target;
		sheetH = SNAP_HEIGHTS[target];
	}

	function resolveSnap(h: number, vel: number): SnapState {
		const proj = h + vel * 90;
		if (proj < (SNAP_MIN + SNAP_MID) / 2) return 'min';
		if (proj < (SNAP_MID + SNAP_MAX) / 2) return 'mid';
		return 'max';
	}

	function onHandlePointerDown(e: PointerEvent) {
		e.preventDefault();
		const now = Date.now();
		dragging = true;
		dragState = { active: true, startY: e.clientY, startH: sheetH, velY: 0, prevY: e.clientY, prevT: now };

		function onMove(ev: PointerEvent) {
			if (!dragState.active) return;
			const t = Date.now();
			const dt = t - dragState.prevT || 1;
			dragState.velY = (dragState.prevY - ev.clientY) / dt;
			dragState.prevY = ev.clientY;
			dragState.prevT = t;
			const delta = dragState.startY - ev.clientY;
			sheetH = Math.max(SNAP_MIN - 10, Math.min(SNAP_MAX + 10, dragState.startH + delta));
		}

		function onUp() {
			dragState.active = false;
			dragging = false;
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
			snapTo(resolveSnap(sheetH, dragState.velY));
		}

		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
	}

	/** 键盘调整面板高度（拖拽的键盘等价操作）。 */
	function onHandleKeydown(e: KeyboardEvent) {
		if (e.target !== e.currentTarget) return;
		const steps: SnapState[] = ['min', 'mid', 'max'];
		const idx = steps.indexOf(snap);
		if (e.key === 'ArrowUp') {
			e.preventDefault();
			snapTo(steps[Math.min(idx + 1, steps.length - 1)]);
		} else if (e.key === 'ArrowDown') {
			e.preventDefault();
			snapTo(steps[Math.max(idx - 1, 0)]);
		}
	}

	/* ── Content state ── */
	let showFilter = $state(false);
	let filterAnchorEl = $state<HTMLElement | null>(null);
	let activeFilters = $state<Set<PinType>>(new Set(['user', 'group', 'post']));
	let searchQuery = $state('');
	let selectedPin = $state<Pin | null>(null);
	let selectedLoc = $state<LocationResult | null>(null);
	const zoom = 0.045;
	let mapCenter = $state(XUZHOU_CENTER);
	let recentItems = $state(RECENT_LOCATIONS);
	let activeTab = $state<'recent' | 'hot'>('recent');

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

	const { url: embedUrl, bounds } = $derived(buildMapData(mapCenter, zoom));
	const filteredPins = $derived(mapPins);
	const searchResults = $derived(searchQuery.trim() ? SEARCH_DB.filter((s) => s.displayName.includes(searchQuery)).slice(0, 5) : []);

	function handleMapTap() {
		selectedPin = null;
		showFilter = false;
		snapTo('min');
	}

	function handleSelectLoc(loc: LocationResult) {
		selectedLoc = loc;
		searchQuery = loc.displayName;
		mapCenter = { lat: loc.lat, lng: loc.lng };
		selectedPin = null;
		if (snap === 'max') snapTo('mid');
	}

	function handleConfirm() {
		const loc = selectedLoc ?? { displayName: '当前位置', lat: XUZHOU_CENTER.lat, lng: XUZHOU_CENTER.lng };
		try {
			localStorage.setItem('hotspot_last_location', JSON.stringify([loc.lat, loc.lng]));
		} catch {}
		history.back();
	}

	function toggleFilter(type: PinType) {
		const next = new Set(activeFilters);
		next.has(type) ? next.delete(type) : next.add(type);
		activeFilters = next;
	}

	// 图钉详情跳转：user/group/post 分别路由到对应详情页
	function openPinDetail(pin: Pin) {
		if (pin.type === 'user') void goto(ROUTES.user(pin.id));
		else if (pin.type === 'group') void goto(ROUTES.group(pin.id));
		else void goto(ROUTES.moment(pin.id));
	}

	const sheetStyle = $derived(
		`position:absolute;bottom:0;left:0;right:0;height:${sheetH}px;z-index:20;border-radius:22px 22px 0 0;border:1px solid var(--neu-border);border-bottom:none;box-shadow:0 -6px 28px var(--neu-shadow-dark);display:flex;flex-direction:column;overflow:hidden;${dragging ? '' : 'transition:height 0.3s cubic-bezier(0.34,1.56,0.64,1);'}`
	);
	const rightCtrlBottom = $derived(sheetH + 10);
</script>

<div class="bg-[var(--neu-bg)]" style="position:relative;width:100%;height:100%;overflow:hidden;">
	<!-- ══ MAP (full screen, tappable) ══ -->
	<div
		style="position:absolute;inset:0;z-index:0;"
		role="button"
		tabindex="0"
		aria-label="地图（点击空白处收起面板）"
		onclick={handleMapTap}
		onkeydown={(e) => {
			if (e.target !== e.currentTarget) return;
			if (e.key === 'Enter' || e.key === ' ' || e.key === 'Escape') {
				e.preventDefault();
				handleMapTap();
			}
		}}
	>
		<iframe src={embedUrl} title="地图" style="width:100%;height:100%;border:0;pointer-events:none;"></iframe>

		<!-- Current location pulsing dot -->
		<div style="position:absolute;left:50%;top:46%;transform:translate(-50%,-50%);z-index:5;pointer-events:none;">
			<div style="width:18px;height:18px;border-radius:50%;background:{PIN_COLORS.user};border:3px solid var(--neu-bg);box-shadow:0 2px 10px {pinTone('user', 55)};position:relative;">
				<span class="absolute rounded-full" style="inset:-7px;background:{pinTone('user', 20)};border:1px solid {pinTone('user', 30)};"></span>
			</div>
		</div>

		<!-- Heatmap buckets（大视口 / 高密度时服务端返回热力统计） -->
		{#each heatBuckets as bucket (bucket.cell_token)}
			<MapHeatDot {bucket} {bounds} />
		{/each}

		<!-- Marker pins -->
		{#each filteredPins as pin (pin.id)}
			<MapPinMarker
				pin={pin}
				{bounds}
				isSelected={selectedPin?.id === pin.id}
				onSelect={(e) => {
					e.stopPropagation();
					selectedPin = selectedPin?.id === pin.id ? null : pin;
					if (snap === 'min') snapTo('mid');
				}}
			/>
		{/each}

		{#if mapLoading}
			<div class="absolute top-16 left-1/2 -translate-x-1/2 z-20 rounded-full bg-[var(--neu-bg)] px-3 py-1 text-[11px] text-[var(--neu-text-muted)]" style="box-shadow:0 2px 12px var(--neu-shadow-dark);border:1px solid var(--neu-border);">
				加载地图数据…
			</div>
		{/if}

		<!-- Attribution -->
		<div class="text-[var(--neu-text-muted)]" style="position:absolute;bottom:4px;left:6px;font-size:9px;opacity:0.4;pointer-events:none;z-index:1;">
			© OpenStreetMap contributors
		</div>
	</div>

	<!-- ══ TOP BAR ══ -->
	<div style="position:absolute;top:0;left:0;right:0;z-index:50;">
		<PageHeader title="地图探索" onBack={() => history.back()}>
			{#snippet actions()}
				<Popover.Root bind:open={showFilter}>
					<div bind:this={filterAnchorEl}>
						<button
							type="button"
							class="neu-hover-btn flex items-center gap-2 rounded-full outline-none {showFilter ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-[var(--neu-bg)] text-[var(--neu-text)]'}"
							style="padding:9px 16px;font-size:14px;border:{showFilter ? '1px solid transparent' : '1px solid var(--neu-border)'};box-shadow:{showFilter ? '0 4px 16px var(--neu-glow)' : '0 2px 12px var(--neu-shadow-dark), 0 1px 3px var(--neu-shadow-dark)'};"
							onclick={() => (showFilter = !showFilter)}
						>
							<SlidersHorizontal style="width:14px;height:14px;" strokeWidth={showFilter ? 2.5 : 1.75} />
							筛选
						</button>
					</div>

					<!-- Filter legend panel -->
					<Popover.Content
						customAnchor={filterAnchorEl}
						side="bottom"
						align="end"
						sideOffset={8}
						class="w-[136px] overflow-hidden rounded-[16px] p-0"
					>
						{#each [
							['user', '用户', markerUserUrl, PIN_COLORS.user],
							['group', '群组', markerGroupUrl, PIN_COLORS.group],
							['post', '动态', markerMomentUrl, PIN_COLORS.post]
						] as [type, label, icon, color], i}
							{@const active = activeFilters.has(type as PinType)}
							{#if i > 0}
								<div class="bg-[var(--neu-border)]" style="height:1px;"></div>
							{/if}
							<button
								type="button"
								class="neu-hover-btn w-full flex items-center gap-2 px-3 py-3 outline-none"
								style="opacity:{active ? 1 : 0.38};"
								onclick={() => toggleFilter(type as PinType)}
							>
								<img src={icon} alt={label} style="width:18px;height:24px;flex-shrink:0;" />
								<span class="text-[var(--neu-text)]" style="flex:1;font-size:13px;text-align:left;">{label}</span>
								<div style="width:13px;height:13px;border-radius:50%;flex-shrink:0;background:{active ? color : 'transparent'};border:2px solid {active ? color : 'var(--neu-border)'};box-shadow:{active ? `0 0 6px ${color}55` : 'none'};transition:all 0.18s;"></div>
							</button>
						{/each}
					</Popover.Content>
				</Popover.Root>
			{/snippet}
		</PageHeader>
	</div>

	<!-- ══ RIGHT CONTROLS — bottom tracks sheet ══ -->
	<div style="position:absolute;right:12px;bottom:{rightCtrlBottom}px;z-index:35;display:flex;flex-direction:column;gap:8px;align-items:center;transition:bottom 0.3s cubic-bezier(0.34,1.56,0.64,1);">
		<!-- GPS re-center -->
		<button
			type="button"
			class="neu-hover-btn flex items-center justify-center rounded-full outline-none neu-circle-hover"
			style="width:42px;height:42px;{floatBtnStyle}"
			onclick={() => {
				mapCenter = XUZHOU_CENTER;
				selectedLoc = null;
				searchQuery = '';
			}}
		>
			<Crosshair style="width:18px;height:18px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
		</button>
	</div>

	<!-- ══ BOTTOM SHEET (draggable) ══ -->
	<div class="bg-[var(--neu-bg)]" style={sheetStyle}>
		<!-- Drag handle -->
		<div
			role="slider"
			aria-label="面板高度"
			aria-orientation="vertical"
			aria-valuemin={SNAP_MIN}
			aria-valuemax={SNAP_MAX}
			aria-valuenow={Math.min(SNAP_MAX, Math.max(SNAP_MIN, sheetH))}
			tabindex="0"
			style="flex-shrink:0;display:flex;flex-direction:column;align-items:center;padding-top:10px;padding-bottom:6px;cursor:grab;touch-action:none;user-select:none;"
			onpointerdown={onHandlePointerDown}
			onkeydown={onHandleKeydown}
		>
			<div class="bg-[var(--neu-border)]" style="width:38px;height:4px;border-radius:2px;"></div>
		</div>

		<!-- Search bar -->
		<div style="flex-shrink:0;padding:0 14px 8px;">
			<div class="flex items-center gap-2.5 px-3.5 rounded-[15px] neu-input-surface" style="height:44px;">
				<Search style="width:15px;height:15px;color:var(--neu-text-muted);flex-shrink:0;" strokeWidth={1.5} />
				<input
					bind:value={searchQuery}
					oninput={(e) => {
						searchQuery = e.currentTarget.value;
						selectedLoc = null;
						if (snap === 'min') snapTo('mid');
					}}
					onfocus={() => {
						if (snap === 'min') snapTo('mid');
					}}
					placeholder="搜索地点、地标或坐标"
					class="flex-1 bg-transparent outline-none text-[var(--neu-text)] text-sm"
				/>
				{#if searchQuery}
					<button
						type="button"
						class="neu-hover-btn outline-none"
						onclick={() => {
							searchQuery = '';
							selectedLoc = null;
						}}
					>
						<X style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={2} />
					</button>
				{:else if selectedLoc}
					<Check style="width:14px;height:14px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={2.5} />
				{/if}
			</div>
		</div>

		<!-- Scrollable content — ONLY shown when snap is not "min" -->
		{#if snap !== 'min'}
			<div style="flex:1;min-height:0;overflow-y:auto;" class="no-scrollbar">
				{#if searchQuery && searchResults.length > 0}
					<div class="animate-fade-in">
						{#each searchResults as r, i}
							<button
								type="button"
								class="neu-hover-bg app-list-row app-list-row--standard neu-row-hover px-5 outline-none"
								style="background:var(--neu-bg);border-bottom:{i < searchResults.length - 1 ? '1px solid var(--neu-border)' : 'none'};"
								aria-pressed={selectedLoc?.displayName === r.displayName}
								onclick={() => handleSelectLoc(r)}
							>
								<div class="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-accent-primary)]" style="width:34px;height:34px;opacity:0.12;">
									<MapPin style="width:14px;height:14px;color:var(--neu-accent-primary);" strokeWidth={1.75} />
								</div>
								<span class="text-[var(--neu-text)]" style="font-size:14px;flex:1;">{r.displayName}</span>
								{#if selectedLoc?.displayName === r.displayName}
									<Check style="width:14px;height:14px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={2.5} />
								{/if}
							</button>
						{/each}
					</div>
				{:else if searchQuery && searchResults.length === 0}
					<div class="flex flex-col items-center py-6">
						<MapPin style="width:26px;height:26px;color:var(--neu-text-muted);opacity:0.28;" strokeWidth={1} />
						<p class="text-[var(--neu-text-muted)]" style="font-size:13px;margin-top:8px;">未找到匹配地点</p>
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
											<span class="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]" style="width:10px;height:10px;border:2px solid var(--neu-bg);"></span>
										{/if}
									</div>
								{:else if selectedPin.type === 'post' && selectedPin.image}
									<img src={selectedPin.image} alt={selectedPin.name || selectedPin.title || ''} class="rounded-[10px] object-cover flex-shrink-0" style="width:42px;height:42px;" />
								{:else}
									<img src={getMarkerUrl(selectedPin.type, selectedPin.count ?? 1)} alt="" style="width:22px;height:29px;flex-shrink:0;" />
								{/if}
								<div class="flex-1 min-w-0">
									<p class="text-[var(--neu-text)] text-[13px]">{selectedPin.name || selectedPin.title}</p>
									<p class="text-[var(--neu-text-muted)]" style="font-size:11px;margin-top:1px;">
										{PIN_LABELS[selectedPin.type]}
										{selectedPin.distance ? ` · ${selectedPin.distance}` : ''}
										{selectedPin.tag ? ` · #${selectedPin.tag}` : ''}
									</p>
								</div>
								<button type="button" class="neu-hover-btn px-2.5 py-1.5 rounded-full outline-none flex-shrink-0" style="background:{PIN_COLORS[selectedPin.type]};color:white;font-size:11px;" onclick={() => selectedPin && openPinDetail(selectedPin)}>
									查看
								</button>
								<button type="button" class="neu-hover-btn outline-none flex-shrink-0 ml-0.5" onclick={() => (selectedPin = null)}>
									<X style="width:13px;height:13px;color:var(--neu-text-muted);" strokeWidth={2} />
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
									class="neu-hover-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none {activeTab === tab.id ? 'bg-[var(--neu-accent-primary)] text-white' : 'bg-[var(--neu-bg)] text-[var(--neu-text-muted)]'}"
									style={activeTab === tab.id ? 'font-size:12px;box-shadow:0 3px 10px var(--neu-glow);' : 'font-size:12px;border:1px solid var(--neu-border);'}
									onclick={() => (activeTab = tab.id as 'recent' | 'hot')}
								>
									<tab.icon style="width:11px;height:11px;" strokeWidth={activeTab === tab.id ? 2.5 : 1.5} />
									{tab.label}
								</button>
							{/each}
							{#if activeTab === 'recent' && recentItems.length > 0}
								<button type="button" class="neu-hover-btn ml-auto outline-none text-[var(--neu-text-muted)] text-xs" onclick={() => (recentItems = [])}>
									清空
								</button>
							{/if}
						</div>

						<!-- List items -->
						{#if activeTab === 'recent'}
							{#if recentItems.length > 0}
								{#each recentItems as loc, i}
									<button
										type="button"
										class="neu-hover-bg app-list-row app-list-row--standard neu-row-hover px-4 outline-none"
										style="background:var(--neu-bg);border-bottom:{i < recentItems.length - 1 ? '1px solid var(--neu-border)' : 'none'};"
										aria-pressed={selectedLoc?.displayName === loc.displayName}
										onclick={() => handleSelectLoc(loc)}
									>
										<div class="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-bg)]" style="width:36px;height:36px;border:1px solid var(--neu-border);box-shadow:3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light);font-size:18px;">
											{loc.emoji}
										</div>
										<div class="flex-1 min-w-0">
											<p class="truncate text-[var(--neu-text)] text-sm">{loc.displayName}</p>
											<p class="text-[var(--neu-text-muted)]" style="font-size:11px;margin-top:1px;">{loc.desc}</p>
										</div>
										{#if selectedLoc?.displayName === loc.displayName}
											<Check style="width:14px;height:14px;color:var(--neu-accent-primary);flex-shrink:0;" strokeWidth={2.5} />
										{:else}
											<ChevronRight style="width:13px;height:13px;color:var(--neu-border);flex-shrink:0;" strokeWidth={1.5} />
										{/if}
									</button>
								{/each}
							{:else}
								<p class="text-[var(--neu-text-muted)]" style="font-size:13px;padding:12px 20px;">暂无最近记录</p>
							{/if}
						{:else}
							{#if HOT_LOCATIONS.length}
								{#each HOT_LOCATIONS as loc, i}
									<button
										type="button"
										class="neu-hover-bg app-list-row app-list-row--standard neu-row-hover px-4 outline-none"
										style="background:var(--neu-bg);border-bottom:{i < HOT_LOCATIONS.length - 1 ? '1px solid var(--neu-border)' : 'none'};"
										aria-pressed={selectedLoc?.displayName === loc.displayName}
										onclick={() => handleSelectLoc(loc)}
									>
										<div class="flex items-center justify-center rounded-[13px] flex-shrink-0 bg-[var(--neu-accent-primary)]" style="width:36px;height:36px;font-size:18px;opacity:0.88;">
											{loc.emoji}
										</div>
										<div class="flex-1 min-w-0">
											<p class="truncate text-[var(--neu-text)] text-sm">{loc.displayName}</p>
											<p class="text-[var(--neu-text-muted)]" style="font-size:11px;margin-top:1px;">{loc.desc}</p>
										</div>
										<div class="flex items-center gap-1 flex-shrink-0">
											<Users style="width:11px;height:11px;color:var(--neu-accent-primary);" strokeWidth={1.5} />
											<span class="text-[var(--neu-accent-primary)] text-xs">{loc.users}</span>
										</div>
									</button>
								{/each}
							{:else}
								<div class="flex flex-col items-center gap-2 py-8">
									<Star style="width:20px;height:20px;color:var(--neu-text-muted);opacity:0.3;" strokeWidth={1} />
									<p class="text-[var(--neu-text-muted)] text-xs">暂无热门地点</p>
								</div>
							{/if}
						{/if}
					</div>
				{/if}
			</div>
		{/if}

		<!-- Confirm button — always at bottom of sheet -->
		<div style="flex-shrink:0;padding:8px 14px 18px;">
			<button
				type="button"
				class="neu-hover-btn w-full flex items-center justify-center gap-2 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white"
				style="height:50px;font-size:15px;box-shadow:0 6px 20px var(--neu-glow);"
				onclick={handleConfirm}
			>
				<MapPin style="width:15px;height:15px;" strokeWidth={2} />
				{selectedLoc ? `确认：${selectedLoc.displayName.slice(0, 11)}` : '设为浏览锚点'}
			</button>
		</div>
	</div>
</div>
