<script lang="ts">
  import type { MapHotspotItem } from '$lib/types/api';
  import type { AnchorHistoryItem } from '$lib/utils/anchor-history';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import Crosshair from 'lucide-svelte/icons/crosshair';
  import Search from 'lucide-svelte/icons/search';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import X from 'lucide-svelte/icons/x';
  import Check from 'lucide-svelte/icons/check';
  import Clock from 'lucide-svelte/icons/clock';
  import Star from 'lucide-svelte/icons/star';
  import TrendingUp from 'lucide-svelte/icons/trending-up';
  import Users from 'lucide-svelte/icons/users';
  import Heart from 'lucide-svelte/icons/heart';
  import Navigation from 'lucide-svelte/icons/navigation';
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
    filterState: { showUsers: boolean; showGroups: boolean; showMoments: boolean };
    totalCounts: { users: number; groups: number; moments: number };
    onback: () => void;
    onconfirm: () => void;
    onsearch: (text: string) => void;
    onselectlocation: (loc: { latitude: number; longitude: number; name: string }) => void;
    ontabchange: (tab: 'recent' | 'hot' | 'nearby') => void;
    onpinselect: (item: MapHotspotItem | null) => void;
    onfilterchange: (filter: { showUsers: boolean; showGroups: boolean; showMoments: boolean }) => void;
    onrecenter: () => void;
    regionLabel?: string;
  }

  let {
    selectedLocation,
    selectedPin,
    activeTab,
    searchText,
    searchResults,
    anchorHistory,
    nearbyPins,
    hotLocations,
    filterState,
    totalCounts,
    onback,
    onconfirm,
    onsearch,
    onselectlocation,
    ontabchange,
    onpinselect,
    onfilterchange,
    onrecenter,
    regionLabel = '',
  }: Props = $props();

  const TYPE_CONFIG = [
    { key: 'showUsers' as const, color: '#2563EB', marker: '/map/marker_user.svg', label: '用户', countKey: 'users' as const },
    { key: 'showGroups' as const, color: '#059669', marker: '/map/marker_group.svg', label: '群组', countKey: 'groups' as const },
    { key: 'showMoments' as const, color: '#EA580C', marker: '/map/marker_moment.svg', label: '动态', countKey: 'moments' as const },
  ];

  const TAB_CONFIG = [
    { key: 'nearby' as const, label: '附近', icon: TrendingUp },
    { key: 'hot' as const, label: '热门', icon: Star },
    { key: 'recent' as const, label: '最近', icon: Clock },
  ];

  function formatDistance(meters?: number): string {
    if (meters == null) return '';
    if (meters < 1000) return `${Math.round(meters)}m`;
    return `${(meters / 1000).toFixed(1)}km`;
  }

  function formatTimestamp(ts: number): string {
    const diff = Date.now() - ts;
    const minutes = Math.floor(diff / 60000);
    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}小时前`;
    const days = Math.floor(hours / 24);
    return `${days}天前`;
  }

  function getTypeBadge(type: string): { label: string; color: string } {
    switch (type) {
      case 'user': return { label: '用户', color: '#2563EB' };
      case 'group': return { label: '群组', color: '#059669' };
      case 'moment': return { label: '动态', color: '#EA580C' };
      default: return { label: type, color: '#8A9190' };
    }
  }

  let localSearchText = $state(searchText);
  let searchInputEl: HTMLInputElement | undefined = $state();

  $effect(() => {
    localSearchText = searchText;
  });

  function handleSearchInput() {
    onsearch(localSearchText);
  }

  function clearSearch() {
    localSearchText = '';
    onsearch('');
    searchInputEl?.focus();
  }

  function toggleFilter(key: 'showUsers' | 'showGroups' | 'showMoments') {
    onfilterchange({ ...filterState, [key]: !filterState[key] });
  }
</script>

<div
  class="flex h-full flex-shrink-0 flex-col select-none"
  style="width: 340px; background: var(--neu-bg); border-right: 1px solid var(--neu-border);"
>
  <!-- Header -->
  <div class="px-5 pt-4 pb-3" style="border-bottom: 1px solid var(--neu-border);">
    <div class="flex items-center justify-between">
      <button
        onclick={onback}
        class="flex h-9 w-9 items-center justify-center rounded-full transition-transform active:scale-[0.92]"
        style="box-shadow: 4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light);"
        aria-label="返回"
      >
        <ArrowLeft class="h-4 w-4" style="color: var(--neu-text);" />
      </button>
      <span class="text-base font-semibold" style="color: var(--neu-text);">位置选择</span>
      <button
        onclick={onrecenter}
        class="flex h-9 w-9 items-center justify-center rounded-full transition-transform active:scale-[0.92]"
        style="box-shadow: 4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light);"
        aria-label="GPS定位"
      >
        <Crosshair class="h-4 w-4" style="color: var(--neu-accent-primary);" />
      </button>
    </div>
    {#if regionLabel}
      <p class="mt-1.5 text-center text-[11px]" style="color: var(--neu-text-muted);">
        {regionLabel}
      </p>
    {/if}
  </div>

  <!-- Type Stat Chips -->
  <div class="flex gap-2 px-5 py-3">
    {#each TYPE_CONFIG as cfg}
      {@const isActive = filterState[cfg.key]}
      {@const count = totalCounts[cfg.countKey]}
      <button
        onclick={() => toggleFilter(cfg.key)}
        class="flex flex-1 flex-col items-center gap-0.5 rounded-[14px] px-2 py-2.5 transition-all active:scale-[0.92]"
        style={
          isActive
            ? `background: ${cfg.color}14; border: 1px solid ${cfg.color}40; box-shadow: 0 2px 8px ${cfg.color}25;`
            : `background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-sm);`
        }
      >
        <img src={cfg.marker} alt="" class="h-5 w-5" />
        <span
          class="text-[15px] font-bold leading-tight"
          style={isActive ? `color: ${cfg.color};` : 'color: var(--neu-text);'}
        >
          {count}
        </span>
        <span
          class="text-[10px]"
          style={isActive ? `color: ${cfg.color};` : 'color: var(--neu-text-muted);'}
        >
          {cfg.label}
        </span>
      </button>
    {/each}
  </div>

  <!-- Search Bar -->
  <div class="relative px-5 pb-2">
    <div
      class="neu-input-surface flex h-[42px] items-center gap-2 rounded-[14px] px-3"
    >
      <Search class="h-3.5 w-3.5 shrink-0" style="color: var(--neu-text-muted);" />
      <input
        bind:this={searchInputEl}
        type="text"
        bind:value={localSearchText}
        placeholder="搜索地点或地标..."
        class="flex-1 bg-transparent text-[13px] outline-none"
        style="color: var(--neu-text);"
        oninput={handleSearchInput}
      />
      {#if localSearchText.length > 0}
        <button
          onclick={clearSearch}
          class="shrink-0 rounded-full p-0.5 transition-colors hover:bg-black/5"
          aria-label="清除搜索"
        >
          <X class="h-3.5 w-3.5" style="color: var(--neu-text-muted);" />
        </button>
      {/if}
    </div>

    <!-- Search Results Dropdown -->
    {#if searchResults.length > 0}
      <div
        class="absolute left-5 right-5 top-full z-20 mt-1 max-h-[240px] overflow-y-auto rounded-[16px] p-1.5"
        style="background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-lg);"
      >
        {#each searchResults as result, i (`sr_${result.latitude}_${result.longitude}_${i}`)}
          <button
            class="flex w-full items-start gap-3 rounded-[12px] px-3 py-2.5 text-left transition-all active:scale-[0.97]"
            style="background: transparent;"
            onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background = 'var(--neu-fallback-bg)'; }}
            onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
            onclick={() => onselectlocation({ latitude: result.latitude, longitude: result.longitude, name: result.name })}
          >
            <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full" style="background: var(--neu-fallback-bg);">
              <MapPin class="h-4 w-4" style="color: var(--neu-accent-primary);" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="truncate text-[13px] font-medium" style="color: var(--neu-text);">{result.name}</p>
              {#if result.address}
                <p class="mt-0.5 truncate text-[11px]" style="color: var(--neu-text-muted);">{result.address}</p>
              {/if}
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Tab Pills -->
  <div class="flex gap-2 px-5 pb-3">
    {#each TAB_CONFIG as tab}
      {@const isActive = activeTab === tab.key}
      {@const Icon = tab.icon}
      <button
        onclick={() => ontabchange(tab.key)}
        class="flex flex-1 items-center justify-center gap-1.5 rounded-full py-2 text-[13px] font-medium transition-all active:scale-[0.92]"
        style={
          isActive
            ? 'background: var(--neu-accent-primary); color: #ffffff; box-shadow: 0 4px 14px var(--neu-glow);'
            : 'background: var(--neu-bg); color: var(--neu-text-muted); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-sm);'
        }
      >
        <Icon class="h-3.5 w-3.5" />
        <span>{tab.label}</span>
      </button>
    {/each}
  </div>

  <!-- Scrollable List Content -->
  <div class="min-h-0 flex-1 overflow-y-auto no-scrollbar px-5">
    {#if activeTab === 'nearby'}
      <!-- Current Location Card -->
      <div
        class="mb-3 rounded-[16px] p-3"
        style="background: linear-gradient(135deg, var(--neu-accent-primary), #7C3AED);"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-9 w-9 items-center justify-center rounded-full"
            style="background: rgba(255,255,255,0.2);"
          >
            <Navigation class="h-4 w-4 text-white" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <p class="text-[13px] font-semibold text-white">当前位置</p>
              <span class="relative flex h-2 w-2">
                <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75"></span>
                <span class="relative inline-flex h-2 w-2 rounded-full bg-green-400"></span>
              </span>
            </div>
            {#if regionLabel}
              <p class="mt-0.5 truncate text-[11px] text-white/70">{regionLabel}</p>
            {/if}
          </div>
        </div>
      </div>

      <!-- Nearby Pin List -->
      {#if nearbyPins.length > 0}
        <div class="space-y-2 pb-3">
          {#each nearbyPins as pin, i (`nearby_${pin.object_id}_${i}`)}
            {@const badge = getTypeBadge(pin.object_type)}
            {@const isSelected = selectedPin?.object_id === pin.object_id}
            <button
              class="flex w-full items-start gap-3 rounded-[15px] p-3 text-left transition-all active:scale-[0.97]"
              style={
                isSelected
                  ? `background: var(--neu-bg); border: 1.5px solid ${badge.color}; box-shadow: 0 2px 10px ${badge.color}20;`
                  : 'background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-sm);'
              }
              onclick={() => onpinselect(isSelected ? null : pin)}
            >
              <!-- Avatar/Image -->
              <div class="h-10 w-10 shrink-0 overflow-hidden rounded-[12px]"
                style={pin.cover_url ? '' : `background: ${badge.color}18;`}
              >
                {#if pin.cover_url}
                  <img src={pin.cover_url} alt="" class="h-full w-full object-cover" />
                {:else}
                  <div class="flex h-full w-full items-center justify-center">
                    {#if pin.object_type === 'user'}
                      <Users class="h-4 w-4" style="color: {badge.color};" />
                    {:else if pin.object_type === 'group'}
                      <Heart class="h-4 w-4" style="color: {badge.color};" />
                    {:else}
                      <Star class="h-4 w-4" style="color: {badge.color};" />
                    {/if}
                  </div>
                {/if}
              </div>
              <!-- Info -->
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-1.5">
                  <span class="truncate text-[13px] font-semibold" style="color: var(--neu-text);">{pin.title}</span>
                  <span
                    class="shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-bold uppercase leading-none"
                    style="background: ${badge.color}14; color: {badge.color};"
                  >
                    {badge.label}
                  </span>
                </div>
                {#if pin.subtitle}
                  <p class="mt-0.5 truncate text-[11px]" style="color: var(--neu-text-muted);">{pin.subtitle}</p>
                {/if}
                <div class="mt-1 flex items-center gap-2 text-[11px]" style="color: var(--neu-text-muted);">
                  {#if pin.distance_meters != null}
                    <span>{formatDistance(pin.distance_meters)}</span>
                  {/if}
                  {#if pin.distance_meters != null && pin.subtitle}
                    <span style="color: var(--neu-border);">·</span>
                  {/if}
                  <ChevronRight class="h-3 w-3" style="color: var(--neu-text-muted);" />
                </div>
              </div>
            </button>
          {/each}
        </div>
      {:else}
        <div class="flex flex-col items-center py-8 text-center">
          <TrendingUp class="mb-2 h-8 w-8" style="color: var(--neu-text-muted); opacity: 0.4;" />
          <p class="text-[12px]" style="color: var(--neu-text-muted);">附近暂无热点</p>
        </div>
      {/if}

    {:else if activeTab === 'hot'}
      <!-- Hot Locations -->
      {#if hotLocations.length > 0}
        <div class="space-y-2 pb-3">
          {#each hotLocations as loc, i (`hot_${loc.name}_${i}`)}
            {@const isLocSelected = selectedLocation?.name === loc.name}
            <button
              class="flex w-full items-center gap-3 rounded-[15px] p-3 text-left transition-all active:scale-[0.97]"
              style={
                isLocSelected
                  ? 'background: var(--neu-bg); border: 1.5px solid var(--neu-accent-primary); box-shadow: 0 2px 10px var(--neu-glow);'
                  : 'background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-sm);'
              }
              onclick={() => {
                if (isLocSelected) return;
                onselectlocation({ latitude: 0, longitude: 0, name: loc.name });
              }}
            >
              <!-- Emoji Square -->
              <div
                class="flex h-[46px] w-[46px] shrink-0 items-center justify-center rounded-[14px] text-xl"
                style="background: var(--neu-fallback-bg);"
              >
                {loc.emoji}
              </div>
              <!-- Info -->
              <div class="min-w-0 flex-1">
                <p class="truncate text-[13px] font-semibold" style="color: var(--neu-text);">{loc.name}</p>
                {#if loc.desc}
                  <p class="mt-0.5 truncate text-[11px]" style="color: var(--neu-text-muted);">{loc.desc}</p>
                {/if}
                {#if loc.users != null}
                  <div class="mt-1 flex items-center gap-1 text-[11px]" style="color: var(--neu-text-muted);">
                    <Users class="h-3 w-3" />
                    <span>{loc.users} 人</span>
                  </div>
                {/if}
              </div>
              {#if isLocSelected}
                <Check class="h-4 w-4 shrink-0" style="color: var(--neu-accent-primary);" />
              {/if}
            </button>
          {/each}
        </div>
      {:else}
        <div class="flex flex-col items-center py-8 text-center">
          <Star class="mb-2 h-8 w-8" style="color: var(--neu-text-muted); opacity: 0.4;" />
          <p class="text-[12px]" style="color: var(--neu-text-muted);">暂无热门地点</p>
        </div>
      {/if}

    {:else if activeTab === 'recent'}
      <!-- Recent Anchor History -->
      {#if anchorHistory.length > 0}
        <div class="space-y-2 pb-3">
          {#each anchorHistory as item, i (`recent_${item.timestamp}_${i}`)}
            {@const isItemSelected = selectedLocation?.name === item.locationName}
            <button
              class="flex w-full items-center gap-3 rounded-[15px] p-3 text-left transition-all active:scale-[0.97]"
              style={
                isItemSelected
                  ? 'background: var(--neu-bg); border: 1.5px solid var(--neu-accent-primary); box-shadow: 0 2px 10px var(--neu-glow);'
                  : 'background: var(--neu-bg); border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-sm);'
              }
              onclick={() => onselectlocation({ latitude: item.latitude, longitude: item.longitude, name: item.locationName })}
            >
              <!-- Emoji Circle -->
              <div
                class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-lg"
                style="background: var(--neu-fallback-bg);"
              >
                <Clock class="h-5 w-5" style="color: var(--neu-accent-primary);" />
              </div>
              <!-- Info -->
              <div class="min-w-0 flex-1">
                <p class="truncate text-[13px] font-semibold" style="color: var(--neu-text);">{item.locationName}</p>
                <p class="mt-0.5 text-[11px]" style="color: var(--neu-text-muted);">{formatTimestamp(item.timestamp)}</p>
              </div>
              {#if isItemSelected}
                <Check class="h-4 w-4 shrink-0" style="color: var(--neu-accent-primary);" />
              {/if}
            </button>
          {/each}
        </div>
      {:else}
        <div class="flex flex-col items-center py-8 text-center">
          <Clock class="mb-2 h-8 w-8" style="color: var(--neu-text-muted); opacity: 0.4;" />
          <p class="text-[12px]" style="color: var(--neu-text-muted);">暂无浏览历史</p>
        </div>
      {/if}
    {/if}
  </div>

  <!-- Confirm Section -->
  <div class="shrink-0 px-5 py-4" style="border-top: 1px solid var(--neu-border);">
    <!-- Selected Location Preview -->
    {#if selectedLocation}
      <div
        class="mb-3 flex items-center gap-2.5 rounded-[14px] px-3 py-2.5"
        style="background: var(--neu-fallback-bg); box-shadow: inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light);"
      >
        <Check class="h-4 w-4 shrink-0" style="color: var(--neu-accent-primary);" />
        <span class="min-w-0 flex-1 truncate text-[13px] font-semibold" style="color: var(--neu-accent-primary);">
          {selectedLocation.name}
        </span>
        <button
          onclick={() => onpinselect(null)}
          class="shrink-0 rounded-full p-1 transition-colors hover:bg-black/5"
          aria-label="清除选择"
        >
          <X class="h-3.5 w-3.5" style="color: var(--neu-text-muted);" />
        </button>
      </div>
    {:else}
      <div
        class="mb-3 flex items-center gap-2.5 rounded-[14px] px-3 py-2.5"
        style="background: var(--neu-fallback-bg); box-shadow: inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light);"
      >
        <MapPin class="h-4 w-4 shrink-0" style="color: var(--neu-text-muted);" />
        <span class="text-[13px]" style="color: var(--neu-text-muted);">拖动地图或搜索选择位置</span>
      </div>
    {/if}

    <!-- Confirm Button -->
    <button
      onclick={onconfirm}
      disabled={!selectedLocation}
      class="flex h-12 w-full items-center justify-center gap-2 rounded-full text-[14px] font-semibold text-white transition-all active:scale-[0.97]"
      style={
        selectedLocation
          ? 'background: var(--neu-accent-primary); box-shadow: 0 4px 14px var(--neu-glow);'
          : 'background: var(--neu-border); cursor: not-allowed; opacity: 0.6;'
      }
    >
      <MapPin class="h-4 w-4" />
      <span>{selectedLocation ? '确认此位置 / 设为浏览锚点' : '请先选择位置'}</span>
    </button>
  </div>
</div>
