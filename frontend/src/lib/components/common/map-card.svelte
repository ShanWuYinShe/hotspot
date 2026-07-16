<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { loadAMap } from '$lib/utils/amap';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
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
      <span class="text-meta" style="color: var(--text-secondary)">地图加载失败</span>
    </div>
  {/if}

  <div bind:this={container} class="h-full w-full" class:invisible={mapLoading || mapError}></div>

  <div class="absolute bottom-3 right-3 flex flex-col gap-2">
    <button
      class="flex items-center justify-center rounded-[var(--radius-md)] outline-none"
      style="width: 34px; height: 34px; background: var(--surface-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1px solid var(--glass-border); box-shadow: var(--shadow-md);"
      onclick={zoomToCurrentLocation}
      aria-label="定位到当前位置"
    >
      <Navigation2 size={14} strokeWidth={1.5} color="var(--text-primary)" />
    </button>
  </div>

  <div
    class="absolute bottom-3 left-3 flex items-center gap-1 rounded-full px-2.5 py-1"
    style="background: var(--surface-bg); opacity: 0.88; backdrop-filter: blur(8px); border: 1px solid var(--glass-border);"
  >
    <span class="text-caption" style="color: var(--brand); font-weight: 600">已定位</span>
    <span class="text-xs" style="color: var(--text-secondary)">拖拽浏览周边</span>
  </div>
</div>
