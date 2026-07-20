<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { loadAMap } from '$lib/utils/amap';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { t } from '$lib/i18n';
  import Navigation2 from 'lucide-svelte/icons/navigation-2';
  import Compass from 'lucide-svelte/icons/compass';

  let { class: className = '' }: { class?: string } = $props();

  let container: HTMLDivElement | undefined = $state();
  let map: AMap.Map | null = null;
  let marker: AMap.Marker | null = null;
  let mapError = $state(false);
  let mapLoading = $state(true);

  onMount(() => {
    initMap();
  });

  onDestroy(() => {
    map?.destroy();
    map = null;
  });

  $effect(() => {
    // 锚点变化时更新地图中心
    if (!map) return;
    const anchorUpdatedAt = mapAnchorStore.anchor.updatedAt;
    if (!anchorUpdatedAt) return;
    map.setCenter([mapAnchorStore.longitude, mapAnchorStore.latitude]);
    marker?.setPosition([mapAnchorStore.longitude, mapAnchorStore.latitude]);
  });

  async function initMap() {
    if (!container) return;
    mapLoading = true;
    mapError = false;
    try {
      await loadAMap();
      if (!container) return;
      const amap = window.AMap!;
      map = new amap.Map(container, {
        zoom: 14,
        center: [mapAnchorStore.longitude, mapAnchorStore.latitude],
        mapStyle: 'amap://styles/light',
        resizeEnable: true,
        showIndoorMap: false
      } as AMap.MapOptions);

      marker = new amap.Marker({
        position: [mapAnchorStore.longitude, mapAnchorStore.latitude]
      });
      marker.setMap(map);
    } catch {
      mapError = true;
    } finally {
      mapLoading = false;
    }
  }

  function zoomToCurrentLocation() {
    if (!map) return;
    map.setZoomAndCenter(16, [mapAnchorStore.longitude, mapAnchorStore.latitude]);
  }
</script>

<div
  class="overflow-hidden rounded-[var(--radius-xl)] {className}"
  style="aspect-ratio: 2.4/1; position: relative;"
>
  {#if mapLoading}
    <div
      class="absolute inset-0 flex items-center justify-center"
      style="background: linear-gradient(135deg, var(--surface-secondary), var(--surface-bg));"
    >
      <div
        class="flex items-center justify-center rounded-full"
        style="width: 44px; height: 44px; background: var(--surface-bg); opacity: 0.88;"
      >
        <span
          class="rounded-full animate-pulse"
          style="width: 14px; height: 14px; background: var(--brand); box-shadow: 0 0 0 4px var(--glow);"
        ></span>
      </div>
    </div>
  {:else if mapError}
    <div
      class="absolute inset-0 flex flex-col items-center justify-center gap-2"
      style="background: linear-gradient(135deg, var(--surface-secondary), var(--surface-bg));"
    >
      <Compass size={24} strokeWidth={1.5} color="var(--text-secondary)" style="opacity: 0.5;" />
      <span class="text-meta" style="color: var(--text-secondary)">{t('mapCard.loadFailed')}</span>
    </div>
  {/if}

  <div bind:this={container} class="h-full w-full" class:invisible={mapLoading || mapError}></div>

  <div class="absolute bottom-3 right-3 flex flex-col gap-2">
    <button
      class="map-card__zoom-btn"
      onclick={zoomToCurrentLocation}
      aria-label={t('mapCard.locateAria')}
    >
      <Navigation2 size={14} strokeWidth={1.5} />
    </button>
  </div>

  <div
    class="absolute bottom-3 left-3 flex items-center gap-1 rounded-full px-2.5 py-1"
    style="background: var(--surface-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1px solid var(--glass-border);"
  >
    <span class="text-caption" style="color: var(--brand); font-weight: 600">{t('mapCard.located')}</span>
    <span class="text-xs" style="color: var(--text-secondary)">{t('mapCard.dragHint')}</span>
  </div>
</div>

<style>
  .map-card__zoom-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border-radius: var(--radius-md);
    background: var(--surface-bg);
    opacity: 0.88;
    backdrop-filter: blur(8px);
    border: 1px solid var(--glass-border);
    cursor: pointer;
    outline: none;
    color: var(--text-primary);
    transition:
      color 0.18s ease,
      background 0.18s ease,
      border-color 0.18s ease,
      transform 0.15s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .map-card__zoom-btn:hover,
  .map-card__zoom-btn:focus-visible {
    color: var(--brand);
    border-color: color-mix(in srgb, var(--brand) 30%, var(--border-theme));
    background: color-mix(in srgb, var(--brand) 12%, transparent);
  }

  .map-card__zoom-btn:active {
    transform: scale(0.9);
    background: color-mix(in srgb, var(--brand) 18%, transparent);
  }
</style>
