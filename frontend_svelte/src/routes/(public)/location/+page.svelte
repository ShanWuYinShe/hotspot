<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import Search from 'lucide-svelte/icons/search';
  import X from 'lucide-svelte/icons/x';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Crosshair from 'lucide-svelte/icons/crosshair';
  import SlidersHorizontal from 'lucide-svelte/icons/sliders-horizontal';
  import type { Map as LeafletMap } from 'leaflet';

  /* ── Types & Constants ─────────────────────────────────────────── */
  type PinType = 'user' | 'group' | 'post';
  type SnapState = 'min' | 'mid' | 'max';

  interface Pin {
    id: number;
    type: PinType;
    lat: number;
    lng: number;
    count?: number;
    name?: string;
    avatar?: string;
    distance?: string;
    online?: boolean;
    members?: number;
    tag?: string;
    title?: string;
    likes?: number;
    image?: string;
  }

  interface LocationResult {
    displayName: string;
    lat: number;
    lng: number;
    emoji?: string;
    desc?: string;
    users?: number;
  }

  const SNAP_MIN = 148;
  const SNAP_MID = 318;
  const SNAP_MAX = 540;
  const SNAP_HEIGHTS: Record<SnapState, number> = { min: SNAP_MIN, mid: SNAP_MID, max: SNAP_MAX };
  const XUZHOU_CENTER = { lat: 34.2604, lng: 117.1855 };
  const DEFAULT_ZOOM = 15;

  const PIN_COLORS: Record<PinType, string> = {
    user: '#2563EB',
    group: '#059669',
    post: '#EA580C'
  };
  const PIN_LABELS: Record<PinType, string> = { user: '用户', group: '群组', post: '动态' };

  /* ── Mock Data ─────────────────────────────────────────────────── */
  const MOCK_PINS: Pin[] = [
    {
      id: 1,
      type: 'user',
      lat: 34.268,
      lng: 117.19,
      name: 'Liam',
      distance: '246m',
      online: true,
      avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80',
      tag: '摄影'
    },
    {
      id: 2,
      type: 'user',
      lat: 34.255,
      lng: 117.18,
      name: 'Sophia',
      distance: '380m',
      online: true,
      avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80',
      tag: '旅行'
    },
    {
      id: 3,
      type: 'user',
      lat: 34.272,
      lng: 117.175,
      name: 'Marco',
      distance: '520m',
      online: false,
      avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80',
      tag: '音乐'
    },
    { id: 10, type: 'user', lat: 34.248, lng: 117.195, count: 5, name: '5人聚集' },
    { id: 11, type: 'user', lat: 34.275, lng: 117.2, count: 3, name: '3人聚集' },
    {
      id: 20,
      type: 'group',
      lat: 34.262,
      lng: 117.185,
      name: '摄影爱好者',
      members: 28,
      tag: '摄影'
    },
    {
      id: 21,
      type: 'group',
      lat: 34.253,
      lng: 117.2,
      name: '美食打卡群',
      members: 89,
      tag: '美食'
    },
    { id: 25, type: 'group', lat: 34.265, lng: 117.165, count: 4, name: '4个群组' },
    {
      id: 40,
      type: 'post',
      lat: 34.27,
      lng: 117.195,
      title: '植物园晨曦',
      likes: 124,
      image: 'https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=200',
      tag: '自然'
    },
    {
      id: 41,
      type: 'post',
      lat: 34.258,
      lng: 117.172,
      title: '云龙湖日落',
      likes: 201,
      image: 'https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=200',
      tag: '风景'
    },
    {
      id: 42,
      type: 'post',
      lat: 34.245,
      lng: 117.188,
      title: '老街咖啡',
      likes: 56,
      image: 'https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=200',
      tag: '生活'
    },
    { id: 45, type: 'post', lat: 34.26, lng: 117.208, count: 7, title: '7条动态' }
  ];

  const RECENT_LOCATIONS: LocationResult[] = [
    { displayName: '云龙湖风景区', lat: 34.2444, lng: 117.1989, desc: '今天 10:24', emoji: '🏞️' },
    { displayName: '彭城广场', lat: 34.2568, lng: 117.1952, desc: '昨天 18:55', emoji: '🏙️' },
    { displayName: '徐州植物园', lat: 34.275, lng: 117.17, desc: '3天前', emoji: '🌿' }
  ];

  const HOT_LOCATIONS: LocationResult[] = [
    {
      displayName: '云龙湖风景区',
      lat: 34.2444,
      lng: 117.1989,
      emoji: '🏞️',
      desc: '国家5A级旅游景区',
      users: 42
    },
    {
      displayName: '彭城广场',
      lat: 34.2568,
      lng: 117.1952,
      emoji: '🏙️',
      desc: '徐州市中心地标',
      users: 38
    },
    {
      displayName: '汉文化景区',
      lat: 34.252,
      lng: 117.2045,
      emoji: '🏯',
      desc: '汉代遗址公园',
      users: 23
    },
    {
      displayName: '徐州博物馆',
      lat: 34.2682,
      lng: 117.1874,
      emoji: '🏛️',
      desc: '历史文化展馆',
      users: 15
    },
    {
      displayName: '徐州植物园',
      lat: 34.275,
      lng: 117.17,
      emoji: '🌿',
      desc: '城市绿肺·天然氧吧',
      users: 12
    }
  ];

  const SEARCH_DB: LocationResult[] = [
    { displayName: '云龙湖风景区', lat: 34.2444, lng: 117.1989 },
    { displayName: '徐州市中心广场', lat: 34.2604, lng: 117.1855 },
    { displayName: '徐州植物园', lat: 34.275, lng: 117.17 },
    { displayName: '彭城广场', lat: 34.2568, lng: 117.1952 },
    { displayName: '汉文化景区', lat: 34.252, lng: 117.2045 },
    { displayName: '徐州博物馆', lat: 34.2682, lng: 117.1874 },
    { displayName: '泉山森林公园', lat: 34.23, lng: 117.16 }
  ];

  /* ── State ─────────────────────────────────────────────────────── */
  let mapEl: HTMLElement;
  let map: LeafletMap | null = null;

  let snap = $state<SnapState>('mid');
  let sheetHeight = $state(SNAP_MID);
  let showFilter = $state(false);
  let activeFilters = $state<Set<PinType>>(new Set(['user', 'group', 'post']));
  let searchQuery = $state('');
  let selectedPin = $state<Pin | null>(null);
  let selectedLoc = $state<LocationResult | null>(null);
  let activeTab = $state<'recent' | 'hot'>('recent');

  // Drag state
  let dragging = $state(false);
  let dragStart = { y: 0, startH: 0 };

  const searchResults = $derived(
    searchQuery.trim()
      ? SEARCH_DB.filter((s) => s.displayName.includes(searchQuery)).slice(0, 5)
      : []
  );

  /* ── Snap logic ────────────────────────────────────────────────── */
  function snapTo(target: SnapState) {
    snap = target;
    sheetHeight = SNAP_HEIGHTS[target];
  }

  function resolveSnap(h: number, vel: number): SnapState {
    const proj = h + vel * 90;
    if (proj < (SNAP_MIN + SNAP_MID) / 2) return 'min';
    if (proj < (SNAP_MID + SNAP_MAX) / 2) return 'mid';
    return 'max';
  }

  function onPointerDown(e: PointerEvent) {
    dragging = true;
    dragStart = { y: e.clientY, startH: sheetHeight };
  }

  function onPointerMove(e: PointerEvent) {
    if (!dragging) return;
    const delta = dragStart.y - e.clientY;
    sheetHeight = Math.max(SNAP_MIN - 10, Math.min(SNAP_MAX + 10, dragStart.startH + delta));
  }

  function onPointerUp(e: PointerEvent) {
    if (!dragging) return;
    dragging = false;
    const vel = dragStart.y - e.clientY;
    snapTo(resolveSnap(sheetHeight, vel));
  }

  function handleMapClick() {
    selectedPin = null;
    showFilter = false;
    snapTo('min');
  }

  function handleSelectLoc(loc: LocationResult) {
    selectedLoc = loc;
    searchQuery = loc.displayName;
    if (map) map.setView([loc.lat, loc.lng], DEFAULT_ZOOM);
    if (snap === 'max') snapTo('mid');
  }

  function handleConfirm() {
    goto(-1);
  }

  function toggleFilter(type: PinType) {
    activeFilters = new Set(
      activeFilters.has(type)
        ? [...activeFilters].filter((f) => f !== type)
        : [...activeFilters, type]
    );
  }

  /* ── Leaflet lifecycle ─────────────────────────────────────────── */
  onMount(async () => {
    const L = await import('leaflet');

    const centerLat = mapAnchorStore.latitude || XUZHOU_CENTER.lat;
    const centerLng = mapAnchorStore.longitude || XUZHOU_CENTER.lng;

    map = L.map(mapEl, {
      center: [centerLat, centerLng],
      zoom: DEFAULT_ZOOM,
      zoomControl: false,
      attributionControl: false
    });

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; OpenStreetMap',
      maxZoom: 19
    }).addTo(map);

    // Add pins
    markers = MOCK_PINS.map((pin) => {
      const color = PIN_COLORS[pin.type];
      const isCluster = (pin.count ?? 1) > 1;
      const html = isCluster
        ? `<div style="position:relative;width:40px;height:48px;"><svg width="40" height="48" viewBox="0 0 32 42"><ellipse cx="16" cy="36" rx="12" ry="5" fill="rgba(0,0,0,0.15)"/><path d="M16 2C8.8 2 3 7.8 3 14.5c0 9.3 13 23.5 13 23.5s13-14.2 13-23.5C29 7.8 23.2 2 16 2z" fill="${color}"/><circle cx="16" cy="14" r="8" fill="white"/></svg><span style="position:absolute;top:10px;left:50%;transform:translateX(-50%);color:${color};font-weight:700;font-size:12px;">${pin.count}</span></div>`
        : `<div style="width:32px;height:42px;"><svg width="32" height="42" viewBox="0 0 32 42"><ellipse cx="16" cy="36" rx="10" ry="4" fill="rgba(0,0,0,0.15)"/><path d="M16 2C8.8 2 3 7.8 3 14.5c0 9.3 13 23.5 13 23.5s13-14.2 13-23.5C29 7.8 23.2 2 16 2z" fill="${color}"/><circle cx="16" cy="14" r="8" fill="white"/></svg></div>`;

      const icon = L.divIcon({
        html,
        className: '',
        iconSize: isCluster ? [40, 48] : [32, 42],
        iconAnchor: isCluster ? [20, 42] : [16, 42]
      });
      const m = L.marker([pin.lat, pin.lng], { icon }).addTo(map!);
      m.on('click', () => {
        selectedPin = pin;
        snapTo('mid');
      });
      return m;
    });

    return () => {
      map?.remove();
    };
  });
</script>

<!-- ════════════════════════════════════════════════════ MOBILE / TABLET ══════ -->
{#if !platformStore.isDesktop}
  <div class="relative w-full h-[100dvh] overflow-hidden" style="background: var(--neu-bg)">
    <!-- Map -->
    <div class="absolute inset-0 z-0" onclick={handleMapClick}>
      <div bind:this={mapEl} class="w-full h-full"></div>
    </div>

    <!-- Floating right controls -->
    <div
      class="absolute z-20 flex flex-col gap-2"
      style="right: 12px; bottom: {sheetHeight + 10}px; transition: bottom 0.25s ease-out"
    >
      <button
        class="flex items-center justify-center rounded-full"
        style="width: 40px; height: 40px; background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 0 2px 12px var(--neu-shadow-dark);"
        onclick={() => map?.setView([XUZHOU_CENTER.lat, XUZHOU_CENTER.lng], DEFAULT_ZOOM)}
      >
        <Crosshair size={18} style="color: var(--neu-text)" />
      </button>
    </div>

    <!-- Bottom sheet -->
    <div
      class="absolute bottom-0 left-0 right-0 z-30 rounded-t-[28px] overflow-hidden"
      style="height: {sheetHeight}px; background: var(--neu-bg); box-shadow: 0 -8px 40px rgba(0,0,0,0.18), 0 -2px 0 var(--neu-border); transition: height 0.25s ease-out;"
    >
      <!-- Handle -->
      <div
        class="flex justify-center pt-2 pb-1 cursor-grab active:cursor-grabbing touch-none"
        onpointerdown={onPointerDown}
        onpointermove={onPointerMove}
        onpointerup={onPointerUp}
      >
        <div class="w-8 h-1 rounded-full" style="background: var(--neu-border-pressed)"></div>
      </div>

      <!-- Search + Confirm -->
      <div class="flex items-center gap-2 px-4 pt-1 pb-3">
        <button
          onclick={() => goto(-1)}
          class="flex items-center justify-center rounded-full flex-shrink-0"
          style="width: 36px; height: 36px; background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 2px 2px 6px var(--neu-shadow-dark), -2px -2px 6px var(--neu-shadow-light);"
        >
          <ArrowLeft size={18} style="color: var(--neu-text)" />
        </button>
        <div
          class="flex-1 flex items-center gap-2 rounded-full px-3 py-2"
          style="background: var(--neu-bg); box-shadow: inset 3px 3px 6px var(--neu-shadow-dark), inset -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
        >
          <Search size={14} style="color: var(--neu-text-muted)" />
          <input
            type="text"
            bind:value={searchQuery}
            placeholder="搜索地点"
            class="flex-1 bg-transparent outline-none text-sm"
            style="color: var(--neu-text);"
          />
          {#if searchQuery}
            <button onclick={() => (searchQuery = '')}
              ><X size={14} style="color: var(--neu-text-muted)" /></button
            >
          {/if}
        </div>
        <button
          onclick={handleConfirm}
          class="px-4 py-2 rounded-full flex-shrink-0 text-white text-sm font-medium"
          style="background: var(--neu-accent-primary); box-shadow: 0 4px 12px var(--neu-glow);"
          >确认</button
        >
      </div>

      <!-- Filter pills -->
      <div class="flex gap-2 px-4 pb-2 overflow-x-auto no-scrollbar">
        {#each ['user', 'group', 'post'] as PinType[] as type}
          <button
            onclick={() => toggleFilter(type)}
            class="flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium transition-all duration-200 flex-shrink-0"
            class:bg-opacity-15
            style="background: {activeFilters.has(type)
              ? PIN_COLORS[type] + '18'
              : 'var(--neu-border)'}; color: {activeFilters.has(type)
              ? PIN_COLORS[type]
              : 'var(--neu-text-muted)'}; border: 1px solid {activeFilters.has(type)
              ? PIN_COLORS[type] + '40'
              : 'transparent'};"
          >
            <span class="w-2 h-2 rounded-full" style="background: {PIN_COLORS[type]}"></span>
            {PIN_LABELS[type]}
          </button>
        {/each}
        <button
          onclick={() => (showFilter = !showFilter)}
          class="flex items-center justify-center rounded-full flex-shrink-0"
          style="width: 28px; height: 28px; background: showFilter ? 'var(--neu-accent-soft)' : 'transparent'; border: 1px solid var(--neu-border);"
        >
          <SlidersHorizontal size={13} style="color: var(--neu-text-muted)" />
        </button>
      </div>

      <!-- Content area -->
      <div class="overflow-y-auto no-scrollbar px-4" style="height: calc(100% - 120px);">
        {#if searchQuery.trim() && searchResults.length > 0}
          <!-- Search results -->
          {#each searchResults as loc}
            <button
              onclick={() => handleSelectLoc(loc)}
              class="flex items-center gap-3 w-full py-3 text-left border-b"
              style="border-color: var(--neu-border)"
            >
              <span class="text-lg">{loc.emoji || '📍'}</span>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium truncate" style="color: var(--neu-text)">
                  {loc.displayName}
                </p>
                {#if loc.desc}<p class="text-xs" style="color: var(--neu-text-muted)">
                    {loc.desc}
                  </p>{/if}
              </div>
              <ChevronRight size={14} style="color: var(--neu-text-muted)" />
            </button>
          {/each}
        {:else if !searchQuery.trim() && activeTab === 'recent'}
          <!-- Recent locations -->
          <p
            class="text-xs font-medium uppercase tracking-wider mb-2 mt-1"
            style="color: var(--neu-text-muted); opacity: 0.75;"
          >
            最近去过
          </p>
          {#each RECENT_LOCATIONS as loc}
            <button
              onclick={() => handleSelectLoc(loc)}
              class="flex items-center gap-3 w-full py-3 text-left border-b"
              style="border-color: var(--neu-border)"
            >
              <span class="text-lg">{loc.emoji}</span>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium truncate" style="color: var(--neu-text)">
                  {loc.displayName}
                </p>
                {#if loc.desc}<p class="text-xs" style="color: var(--neu-text-muted)">
                    {loc.desc}
                  </p>{/if}
              </div>
              <ChevronRight size={14} style="color: var(--neu-text-muted)" />
            </button>
          {/each}
        {:else if !searchQuery.trim()}
          <!-- Hot locations -->
          <p
            class="text-xs font-medium uppercase tracking-wider mb-2 mt-1"
            style="color: var(--neu-text-muted); opacity: 0.75;"
          >
            热门地点
          </p>
          {#each HOT_LOCATIONS as loc}
            <button
              onclick={() => handleSelectLoc(loc)}
              class="flex items-center gap-3 w-full py-3 text-left border-b"
              style="border-color: var(--neu-border)"
            >
              <span class="text-lg">{loc.emoji}</span>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium truncate" style="color: var(--neu-text)">
                  {loc.displayName}
                </p>
                {#if loc.desc}<p class="text-xs" style="color: var(--neu-text-muted)">
                    {loc.desc}
                  </p>{/if}
              </div>
              {#if loc.users}<span
                  class="text-xs flex-shrink-0"
                  style="color: var(--neu-text-muted)">{loc.users}人在附近</span
                >{/if}
            </button>
          {/each}
        {:else}
          <div class="flex flex-col items-center justify-center py-12">
            <p class="text-sm" style="color: var(--neu-text-muted)">未找到地点</p>
          </div>
        {/if}
      </div>
    </div>
  </div>
{:else}
  <!-- ════════════════════════════════════════════════════ DESKTOP ═══════ -->
  <div class="relative w-full h-[100dvh] overflow-hidden" style="background: var(--neu-bg)">
    <!-- Map -->
    <div class="absolute inset-0 z-0" onclick={handleMapClick}>
      <div bind:this={mapEl} class="w-full h-full"></div>
    </div>

    <!-- Side panel -->
    <div
      class="absolute right-0 top-0 bottom-0 z-20 flex flex-col overflow-hidden"
      style="width: 380px; background: var(--neu-bg); border-left: 1px solid var(--neu-border); box-shadow: -4px 0 24px rgba(0,0,0,0.08);"
    >
      <!-- Panel header -->
      <div class="flex items-center gap-3 px-4 pt-4 pb-3">
        <button
          onclick={() => goto(-1)}
          class="flex items-center justify-center rounded-full flex-shrink-0"
          style="width: 36px; height: 36px; background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: 2px 2px 6px var(--neu-shadow-dark), -2px -2px 6px var(--neu-shadow-light);"
        >
          <ArrowLeft size={18} style="color: var(--neu-text)" />
        </button>
        <div
          class="flex-1 flex items-center gap-2 rounded-full px-3 py-2"
          style="background: var(--neu-bg); box-shadow: inset 3px 3px 6px var(--neu-shadow-dark), inset -3px -3px 6px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
        >
          <Search size={14} style="color: var(--neu-text-muted)" />
          <input
            type="text"
            bind:value={searchQuery}
            placeholder="搜索地点"
            class="flex-1 bg-transparent outline-none text-sm"
            style="color: var(--neu-text);"
          />
          {#if searchQuery}
            <button onclick={() => (searchQuery = '')}
              ><X size={14} style="color: var(--neu-text-muted)" /></button
            >
          {/if}
        </div>
      </div>

      <!-- Filter pills -->
      <div class="flex gap-2 px-4 pb-3">
        {#each ['user', 'group', 'post'] as PinType[] as type}
          <button
            onclick={() => toggleFilter(type)}
            class="flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium transition-all duration-200"
            style="background: {activeFilters.has(type)
              ? PIN_COLORS[type] + '18'
              : 'var(--neu-border)'}; color: {activeFilters.has(type)
              ? PIN_COLORS[type]
              : 'var(--neu-text-muted)'}; border: 1px solid {activeFilters.has(type)
              ? PIN_COLORS[type] + '40'
              : 'transparent'};"
          >
            <span class="w-2 h-2 rounded-full" style="background: {PIN_COLORS[type]}"></span>
            {PIN_LABELS[type]}
          </button>
        {/each}
      </div>

      <!-- Panel content -->
      <div class="flex-1 overflow-y-auto no-scrollbar px-4">
        {#if searchQuery.trim() && searchResults.length > 0}
          {#each searchResults as loc}
            <button
              onclick={() => handleSelectLoc(loc)}
              class="flex items-center gap-3 w-full py-3 text-left border-b"
              style="border-color: var(--neu-border)"
            >
              <span class="text-lg">{loc.emoji || '📍'}</span>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium truncate" style="color: var(--neu-text)">
                  {loc.displayName}
                </p>
                {#if loc.desc}<p class="text-xs" style="color: var(--neu-text-muted)">
                    {loc.desc}
                  </p>{/if}
              </div>
              <ChevronRight size={14} style="color: var(--neu-text-muted)" />
            </button>
          {/each}
        {:else}
          <!-- Tab toggle -->
          <div class="flex gap-1 mb-3 p-0.5 rounded-full" style="background: var(--neu-border)">
            <button
              onclick={() => (activeTab = 'recent')}
              class="flex-1 rounded-full py-1.5 text-xs font-medium transition-all"
              class:bg-white={activeTab === 'recent'}
              style="background: {activeTab === 'recent'
                ? 'var(--neu-bg)'
                : 'transparent'}; color: {activeTab === 'recent'
                ? 'var(--neu-text)'
                : 'var(--neu-text-muted)'}; box-shadow: {activeTab === 'recent'
                ? '2px 2px 4px var(--neu-shadow-dark), -2px -2px 4px var(--neu-shadow-light)'
                : 'none'};">最近</button
            >
            <button
              onclick={() => (activeTab = 'hot')}
              class="flex-1 rounded-full py-1.5 text-xs font-medium transition-all"
              style="background: {activeTab === 'hot'
                ? 'var(--neu-bg)'
                : 'transparent'}; color: {activeTab === 'hot'
                ? 'var(--neu-text)'
                : 'var(--neu-text-muted)'}; box-shadow: {activeTab === 'hot'
                ? '2px 2px 4px var(--neu-shadow-dark), -2px -2px 4px var(--neu-shadow-light)'
                : 'none'};">热门</button
            >
          </div>

          {#if activeTab === 'recent'}
            <p
              class="text-xs font-medium uppercase tracking-wider mb-2"
              style="color: var(--neu-text-muted); opacity: 0.75;"
            >
              最近去过
            </p>
            {#each RECENT_LOCATIONS as loc}
              <button
                onclick={() => handleSelectLoc(loc)}
                class="flex items-center gap-3 w-full py-3 text-left border-b"
                style="border-color: var(--neu-border)"
              >
                <span class="text-lg">{loc.emoji}</span>
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium truncate" style="color: var(--neu-text)">
                    {loc.displayName}
                  </p>
                  {#if loc.desc}<p class="text-xs" style="color: var(--neu-text-muted)">
                      {loc.desc}
                    </p>{/if}
                </div>
                <ChevronRight size={14} style="color: var(--neu-text-muted)" />
              </button>
            {/each}
          {:else}
            <p
              class="text-xs font-medium uppercase tracking-wider mb-2"
              style="color: var(--neu-text-muted); opacity: 0.75;"
            >
              热门地点
            </p>
            {#each HOT_LOCATIONS as loc}
              <button
                onclick={() => handleSelectLoc(loc)}
                class="flex items-center gap-3 w-full py-3 text-left border-b"
                style="border-color: var(--neu-border)"
              >
                <span class="text-lg">{loc.emoji}</span>
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium truncate" style="color: var(--neu-text)">
                    {loc.displayName}
                  </p>
                  {#if loc.desc}<p class="text-xs" style="color: var(--neu-text-muted)">
                      {loc.desc}
                    </p>{/if}
                </div>
                {#if loc.users}<span
                    class="text-xs flex-shrink-0"
                    style="color: var(--neu-text-muted)">{loc.users}人在附近</span
                  >{/if}
              </button>
            {/each}
          {/if}
        {/if}
      </div>
    </div>
  </div>
{/if}
