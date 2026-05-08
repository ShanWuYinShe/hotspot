<script lang="ts">
	import type { MapHotspotItem } from '$lib/types/api';
	import type { AnchorHistoryItem } from '$lib/utils/anchor-history';
	import Search from 'lucide-svelte/icons/search';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import X from 'lucide-svelte/icons/x';
	import Check from 'lucide-svelte/icons/check';
	import Clock from 'lucide-svelte/icons/clock';
	import Star from 'lucide-svelte/icons/star';
	import TrendingUp from 'lucide-svelte/icons/trending-up';
	import Users from 'lucide-svelte/icons/users';
	import Heart from 'lucide-svelte/icons/heart';
	import ChevronRight from 'lucide-svelte/icons/chevron-right';

	interface SearchResultItem {
		name: string;
		address: string;
		latitude: number;
		longitude: number;
	}

	interface Props {
		selectedLocation: { latitude: number; longitude: number; name: string } | null;
		selectedPin: MapHotspotItem | null;
		activeTab: 'recent' | 'hot' | 'nearby';
		searchText: string;
		searchResults: SearchResultItem[];
		anchorHistory: AnchorHistoryItem[];
		nearbyPins: MapHotspotItem[];
		hotLocations: { name: string; desc: string; emoji: string; users?: number }[];
		onconfirm: () => void;
		onsearch: (text: string) => void;
		onselectlocation: (loc: { latitude: number; longitude: number; name: string }) => void;
		ontabchange: (tab: 'recent' | 'hot' | 'nearby') => void;
		onpindismiss: () => void;
		onbottomsheetheight: (height: number) => void;
	}

	let {
		selectedLocation = $bindable(null),
		selectedPin = $bindable(null),
		activeTab = $bindable('nearby'),
		searchText = $bindable(''),
		searchResults = [],
		anchorHistory = [],
		nearbyPins = [],
		hotLocations = [],
		onconfirm,
		onsearch,
		onselectlocation,
		ontabchange,
		onpindismiss,
		onbottomsheetheight
	}: Props = $props();

	// ---- Snap points ----
	const SNAP_MIN = 148;
	const SNAP_MID = 318;
	const SNAP_MAX = 540;

	// ---- Pin type colors ----
	const PIN_COLORS: Record<string, string> = {
		user: '#2563EB',
		group: '#059669',
		moment: '#EA580C'
	};
	const PIN_LABELS: Record<string, string> = {
		user: '用户',
		group: '群组',
		moment: '动态'
	};

	// ---- Drag state ----
	let sheetHeight = $state(SNAP_MID);
	let isDragging = false;
	let dragStartY = 0;
	let dragStartHeight = 0;
	let velocity = 0;
	let lastDragY = 0;
	let lastDragTime = 0;
	let transitionEnabled = $state(true);

	// ---- Internal search state ----
	let localSearchText = $state(searchText);

	$effect(() => {
		localSearchText = searchText;
	});

	// ---- Drag handlers ----
	function handlePointerDown(e: PointerEvent) {
		isDragging = true;
		transitionEnabled = false;
		dragStartY = e.clientY;
		dragStartHeight = sheetHeight;
		velocity = 0;
		lastDragY = e.clientY;
		lastDragTime = Date.now();
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
	}

	function handlePointerMove(e: PointerEvent) {
		if (!isDragging) return;
		const now = Date.now();
		const dt = now - lastDragTime || 1;
		velocity = (lastDragY - e.clientY) / dt;
		lastDragY = e.clientY;
		lastDragTime = now;
		const delta = dragStartY - e.clientY;
		sheetHeight = Math.max(SNAP_MIN - 10, Math.min(SNAP_MAX + 10, dragStartHeight + delta));
		onbottomsheetheight?.(sheetHeight);
	}

	function handlePointerUp() {
		if (!isDragging) return;
		isDragging = false;
		transitionEnabled = true;

		const projected = sheetHeight + velocity * 90;
		if (projected < (SNAP_MIN + SNAP_MID) / 2) sheetHeight = SNAP_MIN;
		else if (projected < (SNAP_MID + SNAP_MAX) / 2) sheetHeight = SNAP_MID;
		else sheetHeight = SNAP_MAX;

		onbottomsheetheight?.(sheetHeight);
	}

	// ---- Search ----
	function handleSearchInput(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		localSearchText = value;
		onsearch?.(value);
	}

	function clearSearch() {
		localSearchText = '';
		onsearch?.('');
	}

	function selectSearchResult(result: SearchResultItem) {
		onselectlocation?.({
			latitude: result.latitude,
			longitude: result.longitude,
			name: result.name
		});
		localSearchText = '';
		onsearch?.('');
	}

	// ---- Tab helpers ----
	const TAB_CONFIG = [
		{ key: 'nearby' as const, label: '附近', icon: TrendingUp },
		{ key: 'hot' as const, label: '热门', icon: Star },
		{ key: 'recent' as const, label: '最近', icon: Clock }
	];

	function switchTab(tab: 'recent' | 'hot' | 'nearby') {
		activeTab = tab;
		ontabchange?.(tab);
	}

	// ---- Format distance ----
	function formatDistance(meters?: number): string {
		if (meters == null) return '';
		if (meters < 1000) return `${Math.round(meters)}m`;
		return `${(meters / 1000).toFixed(1)}km`;
	}

	// ---- Pin icon by type ----
	function pinIcon(type: string) {
		switch (type) {
			case 'user':
				return Users;
			case 'group':
				return Heart;
			default:
				return MapPin;
		}
	}

	// ---- Helper: hex color to rgba ----
	function hexToRgba(hex: string, alpha: number): string {
		const r = parseInt(hex.slice(1, 3), 16);
		const g = parseInt(hex.slice(3, 5), 16);
		const b = parseInt(hex.slice(5, 7), 16);
		return `rgba(${r}, ${g}, ${b}, ${alpha})`;
	}

	// ---- Derived state ----
	const isSearchMode = $derived(localSearchText.trim().length > 0);
	const showContent = $derived(sheetHeight > SNAP_MIN);
</script>

<div
	class="mobile-bottom-sheet"
	style="
		height: {sheetHeight}px;
		transition: {transitionEnabled ? 'height 0.3s cubic-bezier(0.32, 0.72, 0, 1)' : 'none'};
	"
>
	<!-- Drag handle -->
	<div
		class="drag-handle"
		onpointerdown={handlePointerDown}
		onpointermove={handlePointerMove}
		onpointerup={handlePointerUp}
		onpointercancel={handlePointerUp}
	>
		<div class="handle-bar"></div>
	</div>

	<!-- Search bar -->
	<div class="search-bar-wrapper">
		<div class="search-bar neu-input-surface">
			<Search size={16} style="color: var(--neu-text-muted); flex-shrink: 0;" />
			<input
				type="text"
				placeholder="搜索地点、地标或坐标"
				value={localSearchText}
				oninput={handleSearchInput}
			/>
			{#if localSearchText.length > 0}
				<button class="clear-btn" onclick={clearSearch}>
					<X size={14} />
				</button>
			{/if}
		</div>
	</div>

	<!-- Selected pin mini card -->
	{#if selectedPin && !isSearchMode}
		{@const pinColor = PIN_COLORS[selectedPin.object_type] || '#6B7280'}
		<div
			class="selected-pin-card"
			style="background: {hexToRgba(pinColor, 0.1)}; border: 1px solid {hexToRgba(pinColor, 0.3)};"
		>
			<div class="pin-card-left">
				<div class="pin-type-dot" style="background: {pinColor};"></div>
				<div class="pin-card-info">
					<span class="pin-card-name">{selectedPin.title}</span>
					<span class="pin-card-meta">
						{PIN_LABELS[selectedPin.object_type] || '未知'}
						{#if selectedPin.distance_meters != null}
							· {formatDistance(selectedPin.distance_meters)}
						{/if}
					</span>
				</div>
			</div>
			<button class="pin-dismiss-btn" onclick={onpindismiss}>
				<X size={14} />
			</button>
		</div>
	{/if}

	<!-- Scrollable content area -->
	{#if showContent}
		<div class="content-area no-scrollbar">
			{#if isSearchMode}
				<!-- Search results -->
				<div class="search-results">
					{#if searchResults.length === 0}
						<div class="empty-state">
							<Search size={24} style="color: var(--neu-text-muted);" />
							<span>搜索地点</span>
						</div>
					{:else}
						{#each searchResults as result, i (`search_${i}`)}
							{@const isSelected =
								selectedLocation &&
								Math.abs(selectedLocation.latitude - result.latitude) < 0.0001 &&
								Math.abs(selectedLocation.longitude - result.longitude) < 0.0001}
							<button
								class="list-item"
								onclick={() => selectSearchResult(result)}
							>
								<div class="list-item-icon">
									<MapPin size={16} style="color: var(--neu-text-muted);" />
								</div>
								<div class="list-item-content">
									<span class="list-item-name">{result.name}</span>
									<span class="list-item-desc">{result.address}</span>
								</div>
								{#if isSelected}
									<Check size={16} style="color: var(--neu-accent-primary); flex-shrink: 0;" />
								{/if}
							</button>
						{/each}
					{/if}
				</div>
			{:else}
				<!-- Tab pills -->
				<div class="tab-bar">
					<div class="tab-pills">
						{#each TAB_CONFIG as tab}
							<button
								class="tab-pill"
								class:active={activeTab === tab.key}
								onclick={() => switchTab(tab.key)}
							>
								<tab.icon size={13} />
								{tab.label}
							</button>
						{/each}
					</div>
					{#if activeTab === 'recent' && anchorHistory.length > 0}
						<button class="clear-link" onclick={() => ontabchange?.('recent')}>
							清空
						</button>
					{/if}
				</div>

				<!-- Tab content -->
				{#if activeTab === 'nearby'}
					<div class="tab-content">
						{#if nearbyPins.length === 0}
							<div class="empty-state">
								<TrendingUp size={24} style="color: var(--neu-text-muted);" />
								<span>附近暂无热点</span>
							</div>
						{:else}
							{#each nearbyPins as pin (`nearby_${pin.object_type}_${pin.object_id}`)}
								{@const nearbyColor = PIN_COLORS[pin.object_type] || '#6B7280'}
								{@const Icon = pinIcon(pin.object_type)}
								<button
									class="list-item"
									onclick={() =>
										onselectlocation?.({
											latitude: pin.location.latitude,
											longitude: pin.location.longitude,
											name: pin.title
										})}
								>
									<div
										class="list-item-icon pin-avatar"
										style="background: {hexToRgba(nearbyColor, 0.1)};"
									>
										<Icon size={14} style="color: {nearbyColor};" />
									</div>
									<div class="list-item-content">
										<span class="list-item-name">{pin.title}</span>
										<span class="list-item-desc">
											<span class="type-badge" style="color: {nearbyColor};">
												{PIN_LABELS[pin.object_type] || '未知'}
											</span>
											{#if pin.subtitle}
												· {pin.subtitle}
											{/if}
											{#if pin.distance_meters != null}
												· {formatDistance(pin.distance_meters)}
											{/if}
										</span>
									</div>
									<ChevronRight size={14} style="color: var(--neu-text-muted); flex-shrink: 0;" />
								</button>
							{/each}
						{/if}
					</div>
				{:else if activeTab === 'hot'}
					<div class="tab-content">
						{#if hotLocations.length === 0}
							<div class="empty-state">
								<Star size={24} style="color: var(--neu-text-muted);" />
								<span>暂无热门地点</span>
							</div>
						{:else}
							{#each hotLocations as loc, i (`hot_${i}`)}
								<button
									class="list-item"
									onclick={() =>
										onselectlocation?.({
											latitude: 0,
											longitude: 0,
											name: loc.name
										})}
								>
									<div class="list-item-icon emoji-icon">
										<span>{loc.emoji}</span>
									</div>
									<div class="list-item-content">
										<span class="list-item-name">{loc.name}</span>
										<span class="list-item-desc">
											{loc.desc}
											{#if loc.users}
												· {loc.users}人
											{/if}
										</span>
									</div>
									<ChevronRight size={14} style="color: var(--neu-text-muted); flex-shrink: 0;" />
								</button>
							{/each}
						{/if}
					</div>
				{:else if activeTab === 'recent'}
					<div class="tab-content">
						{#if anchorHistory.length === 0}
							<div class="empty-state">
								<Clock size={24} style="color: var(--neu-text-muted);" />
								<span>暂无浏览历史</span>
							</div>
						{:else}
							{#each anchorHistory as item, i (`recent_${i}`)}
								{@const isSelected =
									selectedLocation &&
									Math.abs(selectedLocation.latitude - item.latitude) < 0.001 &&
									Math.abs(selectedLocation.longitude - item.longitude) < 0.001}
								<button
									class="list-item"
									class:selected={isSelected}
									onclick={() =>
										onselectlocation?.({
											latitude: item.latitude,
											longitude: item.longitude,
											name: item.locationName
										})}
								>
									<div class="list-item-icon">
										<MapPin size={16} style="color: var(--neu-text-muted);" />
									</div>
									<div class="list-item-content">
										<span class="list-item-name">{item.locationName}</span>
										<span class="list-item-desc">
											{new Date(item.timestamp).toLocaleDateString('zh-CN')}
										</span>
									</div>
									{#if isSelected}
										<Check size={16} style="color: var(--neu-accent-primary); flex-shrink: 0;" />
									{/if}
								</button>
							{/each}
						{/if}
					</div>
				{/if}
			{/if}
		</div>
	{/if}

	<!-- Confirm button -->
	<div class="confirm-wrapper">
		<button class="confirm-btn" onclick={onconfirm}>
			<MapPin size={16} />
			设为浏览锚点
		</button>
	</div>
</div>

<style>
	.mobile-bottom-sheet {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		background: var(--neu-bg);
		border: 1px solid var(--neu-border);
		border-bottom: none;
		border-radius: 22px 22px 0 0;
		box-shadow: 0 -6px 28px var(--neu-shadow-dark);
		display: flex;
		flex-direction: column;
		overflow: hidden;
		touch-action: none;
		user-select: none;
	}

	/* Drag handle */
	.drag-handle {
		display: flex;
		justify-content: center;
		align-items: center;
		padding: 10px 0 6px;
		cursor: grab;
		touch-action: none;
	}

	.drag-handle:active {
		cursor: grabbing;
	}

	.handle-bar {
		width: 38px;
		height: 4px;
		border-radius: 2px;
		background: var(--neu-border);
	}

	/* Search bar */
	.search-bar-wrapper {
		padding: 0 14px 8px;
	}

	.search-bar {
		display: flex;
		align-items: center;
		gap: 8px;
		height: 44px;
		padding: 0 12px;
		border-radius: 15px;
		background: var(--neu-input-surface, var(--neu-bg));
		border: 1px solid var(--neu-border);
	}

	.search-bar input {
		flex: 1;
		background: none;
		border: none;
		outline: none;
		font-size: 14px;
		color: var(--neu-text);
		min-width: 0;
	}

	.search-bar input::placeholder {
		color: var(--neu-text-muted);
	}

	.clear-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: var(--neu-border);
		color: var(--neu-text-muted);
		border: none;
		cursor: pointer;
		flex-shrink: 0;
		transition: transform 0.15s;
	}

	.clear-btn:active {
		transform: scale(0.9);
	}

	/* Selected pin card */
	.selected-pin-card {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin: 0 14px 8px;
		padding: 10px 12px;
		border-radius: 12px;
	}

	.pin-card-left {
		display: flex;
		align-items: center;
		gap: 10px;
		min-width: 0;
	}

	.pin-type-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}

	.pin-card-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.pin-card-name {
		font-size: 14px;
		font-weight: 500;
		color: var(--neu-text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.pin-card-meta {
		font-size: 12px;
		color: var(--neu-text-muted);
	}

	.pin-dismiss-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		border: none;
		background: var(--neu-border);
		color: var(--neu-text-muted);
		cursor: pointer;
		flex-shrink: 0;
		transition: transform 0.15s;
	}

	.pin-dismiss-btn:active {
		transform: scale(0.9);
	}

	/* Tab bar */
	.tab-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 14px 8px;
	}

	.tab-pills {
		display: flex;
		gap: 6px;
	}

	.tab-pill {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 6px 12px;
		border-radius: 9999px;
		border: 1px solid var(--neu-border);
		background: var(--neu-bg);
		color: var(--neu-text-muted);
		font-size: 13px;
		cursor: pointer;
		transition: all 0.15s;
	}

	.tab-pill:active {
		transform: scale(0.93);
	}

	.tab-pill.active {
		background: var(--neu-accent-primary);
		border-color: var(--neu-accent-primary);
		color: white;
		box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
	}

	.clear-link {
		font-size: 12px;
		color: var(--neu-text-muted);
		background: none;
		border: none;
		cursor: pointer;
		transition: transform 0.15s;
	}

	.clear-link:active {
		transform: scale(0.93);
	}

	/* Content area */
	.content-area {
		flex: 1;
		overflow-y: auto;
		overflow-x: hidden;
		min-height: 0;
	}

	.search-results,
	.tab-content {
		padding: 0 6px;
	}

	/* List item */
	.list-item {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 10px 8px;
		border: none;
		background: none;
		cursor: pointer;
		text-align: left;
		border-radius: 10px;
		transition: transform 0.15s;
	}

	.list-item:active {
		transform: scale(0.97);
	}

	.list-item:hover {
		background: var(--neu-border);
	}

	.list-item.selected {
		background: color-mix(in srgb, var(--neu-accent-primary) 8%, transparent);
	}

	.list-item-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: var(--neu-border);
		flex-shrink: 0;
	}

	.list-item-icon.pin-avatar {
		border-radius: 50%;
	}

	.list-item-icon.emoji-icon span {
		font-size: 18px;
		line-height: 1;
	}

	.list-item-content {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
		flex: 1;
	}

	.list-item-name {
		font-size: 14px;
		font-weight: 500;
		color: var(--neu-text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.list-item-desc {
		font-size: 12px;
		color: var(--neu-text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.type-badge {
		font-weight: 500;
	}

	/* Empty state */
	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding: 28px 0;
		color: var(--neu-text-muted);
		font-size: 13px;
	}

	/* Confirm button */
	.confirm-wrapper {
		padding: 8px 14px 12px;
	}

	.confirm-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 6px;
		width: 100%;
		height: 50px;
		border-radius: 9999px;
		border: none;
		background: var(--neu-accent-primary);
		color: white;
		font-size: 15px;
		font-weight: 500;
		cursor: pointer;
		box-shadow: 0 2px 12px rgba(59, 130, 246, 0.3);
		transition: transform 0.15s;
	}

	.confirm-btn:active {
		transform: scale(0.93);
	}
</style>
