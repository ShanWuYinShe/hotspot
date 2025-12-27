<template>
  <div
    ref="rootEl"
    class="relative w-full h-full hs-map-frame"
    @click.capture="handleContainerClick"
  >
    <!-- 地图容器 -->
    <div
      ref="mapContainer"
      class="w-full h-full"
      :data-hs-map-interactive="isInteractive ? 'true' : 'false'"
    />

    <!-- 浏览锚点：中心覆盖（“我想看这里”），不是“我的真实位置” -->
    <div
      v-if="mapReady && showAnchor"
      class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-full z-20 pointer-events-none"
    >
      <div class="flex flex-col items-center">
        <UBadge
          color="primary"
          variant="solid"
          size="lg"
          square
          icon="i-lucide-telescope"
        />
        <div class="mt-1">
          <UCard variant="subtle">
            <UBadge size="sm:lg">浏览锚点</UBadge>
          </UCard>
        </div>
      </div>
    </div>

    <!-- 聚焦提示：默认不吃滚轮/拖拽，避免用户滚动列表时误操作地图 -->
    <div
      v-if="mapReady && !isInteractive"
      class="absolute top-4 left-4 z-20 pointer-events-none"
    >
      <UCard variant="subtle">
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-mouse-pointer-click" :size="16" />
          <UBadge color="primary" variant="soft" size="sm:lg">点击地图以启用缩放/拖拽</UBadge>
        </div>
      </UCard>
    </div>

    <!-- 加载中 -->
    <div
      v-if="loading"
      class="absolute inset-0 z-50 flex items-center justify-center hs-overlay-scrim"
    >
      <div class="text-center">
        <UButton loading variant="ghost" color="neutral" />
        <div class="mt-2 flex justify-center">
          <UBadge>加载地图中...</UBadge>
        </div>
      </div>
    </div>

    <!-- 控制按钮 -->
    <div
      v-if="mapReady"
      class="absolute top-4 right-4 flex flex-col gap-2"
    >
      <UButton
        icon="i-lucide-plus"
        variant="solid"
        color="neutral"
        size="sm"
        @click="zoomIn"
      />
      <UButton
        icon="i-lucide-minus"
        variant="solid"
        color="neutral"
        size="sm"
        @click="zoomOut"
      />
      <UButton
        icon="i-lucide-map-pin"
        size="sm"
        :loading="locating"
        @click="locateMe"
      />
    </div>

    <!-- 选中的标记信息 -->
    <UCard
      v-if="selectedMarker"
      variant="subtle"
      class="absolute bottom-4 left-4 z-10 w-64"
    >
      <div class="flex items-center gap-3">
        <UAvatar
          :src="selectedMarker.avatar"
          size="sm"
        />
        <div class="flex-1 min-w-0">
          <p class="font-medium truncate">{{ selectedMarker.title }}</p>
          <UBadge class="mt-1 w-fit max-w-full truncate">
            {{ selectedMarker.description }}
          </UBadge>
        </div>
        <UButton
          icon="i-lucide-x"
          variant="ghost"
          size="sm"
          @click="selectedMarker = null"
        />
      </div>
    </UCard>

    <!-- 未配置提示 -->
    <div
      v-if="!isConfigured"
      class="absolute inset-0 z-10 flex items-center justify-center pointer-events-none"
    >
      <UEmpty
        icon="i-lucide-map"
        title="地图未配置"
        description="请配置 NUXT_PUBLIC_AMAP_KEY"
        variant="outline"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useGeolocation } from '~/composables/useGeolocation'
import { waitForAMap } from '~/plugins/01.amap.client'

export interface MapMarker {
  id: string
  type: 'user' | 'group' | 'moment' | 'location' | 'cluster'
  title: string
  description: string
  latitude: number
  longitude: number
  avatar?: string
  distance?: number
  data?: Record<string, unknown>
}

const props = withDefaults(defineProps<{
  markers: MapMarker[]
  center: { latitude: number; longitude: number }
  zoom: number
  /**
   * 默认不允许滚轮/拖拽等交互（避免用户滚动列表时误操作地图）
   * 点击地图后进入聚焦态才启用；点击外部失焦后自动禁用
   */
  interactive?: boolean
  /**
   * 地图中心锚点（“我想看这里”），默认显示。
   * 注意：这不是“用户真实位置”，仅用于浏览/探索。
   */
  showAnchor?: boolean
}>(), {
  markers: () => [],
  center: () => ({ latitude: 39.9042, longitude: 116.4074 }),
  zoom: 13,
  interactive: false,
  showAnchor: true
})

const emit = defineEmits<{
  markerClick: [marker: MapMarker]
  mapClick: [coords: { latitude: number; longitude: number }]
  viewportChange: [payload: { center: { latitude: number; longitude: number }; zoom: number; bounds: { north: number; south: number; east: number; west: number } }]
}>()

const config = useRuntimeConfig()
const feedback = useFeedbackStore()
const colorMode = useColorMode()
const geolocation = useGeolocation()

const mapContainer = ref<HTMLElement>()
const rootEl = ref<HTMLElement>()
const mapInstance = ref<AMap.Map | null>(null)
const markerInstances = ref<Map<string, AMap.Marker>>(new Map())
const loading = ref(true)
const locating = ref(false)
const mapReady = ref(false)
const selectedMarker = ref<MapMarker | null>(null)

const isConfigured = computed(() => !!config.public.amapKey)

// 交互开关：默认 false，点击地图后启用，点击外部失焦后禁用
const isInteractive = ref<boolean>(props.interactive)

const setInteractiveStatus = (enabled: boolean) => {
  const map = mapInstance.value as unknown as { setStatus?: (s: Record<string, boolean>) => void } | null
  // AMap 支持通过 setStatus 控制大部分交互开关
  map?.setStatus?.({
    scrollWheel: enabled,
    dragEnable: enabled,
    zoomEnable: enabled,
    doubleClickZoom: enabled,
    keyboardEnable: enabled,
    touchZoom: enabled
  })
}

// 初始化地图
const initMap = async () => {
  if (!mapContainer.value || !isConfigured.value) {
    loading.value = false
    return
  }

  try {
    await waitForAMap()

    mapInstance.value = new AMap.Map(mapContainer.value, {
      center: [props.center.longitude, props.center.latitude],
      zoom: props.zoom,
      viewMode: '2D',
      mapStyle: colorMode.value === 'dark' ? 'amap://styles/dark' : 'amap://styles/normal'
    })
    
    // Apply initial interactive state
    setInteractiveStatus(isInteractive.value)

    mapInstance.value.on('click', (e: unknown) => {
      const event = e as { lnglat: AMap.LngLat }
      emit('mapClick', {
        latitude: event.lnglat.getLat(),
        longitude: event.lnglat.getLng()
      })
    })

    const emitViewport = () => {
      if (!mapInstance.value) return
      // Even if not "interactive" (e.g. scroll disabled), we might still want to emit viewport changes 
      // if they happen programmatically or via allowed gestures. 
      // But usually viewport changes ONLY happen if interactive or programmatically.
      // If programmatically (setCenter), we definitely want to emit.
      // So removing !isInteractive check might be safer if we want 2-way binding to work even if map is "locked" from user.
      // However, the original intent was probably to prevent noise.
      // Let's allow emitting if map is ready.
      
      const center = mapInstance.value.getCenter()
      const zoom = mapInstance.value.getZoom()
      const b = (
        mapInstance.value as unknown as {
          getBounds: () => { getNorthEast: () => AMap.LngLat; getSouthWest: () => AMap.LngLat }
        }
      ).getBounds()
      emit('viewportChange', {
        center: { latitude: center.getLat(), longitude: center.getLng() },
        zoom,
        bounds: {
          north: b.getNorthEast().getLat(),
          east: b.getNorthEast().getLng(),
          south: b.getSouthWest().getLat(),
          west: b.getSouthWest().getLng()
        }
      })
    }
    mapInstance.value.on('moveend', emitViewport)
    mapInstance.value.on('zoomend', emitViewport)

    mapReady.value = true
    updateMarkers()
  } catch (error) {
    console.error('[Map] 初始化失败:', error)
    feedback.error('地图加载失败')
  } finally {
    loading.value = false
  }
}

const clusterPrecisionByZoom = (zoom: number) => {
  // 越小的 zoom（越大范围）=> 颗粒更粗，减少 marker 密度
  if (zoom <= 6) return 1.0   // ~省级/大区：按 1 度聚合
  if (zoom <= 8) return 0.5
  if (zoom <= 10) return 0.2
  if (zoom <= 12) return 0.1
  return 0 // 不聚合
}

// 更新标记
const updateMarkers = () => {
  if (!mapInstance.value) return

  // 清除旧标记
  markerInstances.value.forEach(m => mapInstance.value?.remove(m))
  markerInstances.value.clear()

  const zoom = mapInstance.value.getZoom()
  // 当外部已经提供了“服务端聚合点”时，不再在组件内二次聚合（否则会把 cluster 再 cluster，密度与 count 都会失真）
  const hasServerClusters = props.markers.some((m) => m.type === 'cluster')
  const precision = hasServerClusters ? 0 : clusterPrecisionByZoom(zoom)

  // 简单聚合：仅基于当前已加载数据做“抽稀/计数”，不依赖后端额外接口
  // 注意：要做到“真实省级全量统计”必须后端提供聚合端点；这里是 UI 侧的可用优化。
  if (precision > 0) {
    const buckets = new Map<string, { latitude: number; longitude: number; count: number; sample: MapMarker }>()
    for (const m of props.markers) {
      const lat = Math.round(m.latitude / precision) * precision
      const lng = Math.round(m.longitude / precision) * precision
      const key = `${lat.toFixed(4)},${lng.toFixed(4)}`
      const item = buckets.get(key)
      if (item) {
        item.count += 1
      } else {
        buckets.set(key, { latitude: lat, longitude: lng, count: 1, sample: m })
      }
    }

    for (const [key, b] of buckets.entries()) {
      // 单个则用原 marker
      if (b.count === 1) {
        const data = b.sample
        const marker = new AMap.Marker({
          position: [data.longitude, data.latitude],
          title: data.title,
          extData: data
        })
        marker.on('click', () => {
          selectedMarker.value = data
          emit('markerClick', data)
        })
        mapInstance.value?.add(marker)
        markerInstances.value.set(data.id, marker)
        continue
      }

      // 多个则用“数量气泡”
      const markerOptions = {
        position: [b.longitude, b.latitude],
        title: `该区域约 ${b.count} 条结果`,
        extData: b.sample,
        // AMap runtime 支持 Marker content，但 TS 类型定义未覆盖；这里用 unknown 收敛到官方类型
        content: `
          <div class="hs-map-cluster">${b.count}</div>
        `
      } as unknown as AMap.MarkerOptions

      const marker = new AMap.Marker(markerOptions)
      marker.on('click', () => {
        // 点击聚合点：仅提示，不自动跳转（避免误触）
        const sample = b.sample
        selectedMarker.value = { ...sample, title: `该区域`, description: `约 ${b.count} 条结果` }
      })
      mapInstance.value?.add(marker)
      markerInstances.value.set(`cluster:${key}`, marker)
    }
    return
  }

  // zoom 较大：直接展示所有 marker
  props.markers.forEach(data => {
    const marker = new AMap.Marker({
      position: [data.longitude, data.latitude],
      title: data.title,
      extData: data
    })
    marker.on('click', () => {
      selectedMarker.value = data
      emit('markerClick', data)
    })
    mapInstance.value?.add(marker)
    markerInstances.value.set(data.id, marker)
  })
}

// 控制方法
const zoomIn = () => {
  if (mapInstance.value) {
    mapInstance.value.setZoom(mapInstance.value.getZoom() + 1)
  }
}

const zoomOut = () => {
  if (mapInstance.value) {
    mapInstance.value.setZoom(mapInstance.value.getZoom() - 1)
  }
}

const locateMe = async () => {
  if (!mapInstance.value) return
  locating.value = true

  try {
    const pos = await geolocation.getPosition({ silent: false })
    if (pos) {
      mapInstance.value.setCenter([pos.coords.longitude, pos.coords.latitude])
      feedback.success('定位成功')
    }
  } catch {
    feedback.error('定位失败')
  } finally {
    locating.value = false
  }
}

// 监听
watch(() => props.markers, updateMarkers, { deep: true })
watch(
  () => [props.center.latitude, props.center.longitude] as const,
  ([lat, lng]) => {
    mapInstance.value?.setCenter([lng, lat])
  }
)
watch(() => props.zoom, (z) => {
  if (!mapInstance.value) return
  mapInstance.value.setZoom(z)
})
watch(() => colorMode.value, (mode) => {
  const map = mapInstance.value as unknown as { setMapStyle?: (style: string) => void } | null
  map?.setMapStyle?.(mode === 'dark' ? 'amap://styles/dark' : 'amap://styles/normal')
})

onMounted(initMap)
onUnmounted(() => {
  document.removeEventListener('pointerdown', onOutsidePointerDown, true)
  mapInstance.value?.destroy()
  mapInstance.value = null
})

// 点击容器：进入交互态
const handleContainerClick = () => {
  if (!mapReady.value) return
  if (isInteractive.value) return
  isInteractive.value = true
  setInteractiveStatus(true)
  // 进入交互态时，避免“滚动锁定”导致误操作：立即给一次提示已开启由 overlay 控制
}

const onOutsidePointerDown = (e: PointerEvent) => {
  if (!isInteractive.value) return
  const root = rootEl.value
  const target = e.target as Node | null
  if (!root || !target) return
  if (root.contains(target)) return
  isInteractive.value = false
  selectedMarker.value = null
  setInteractiveStatus(false)
}

watch(
  () => isInteractive.value,
  (enabled) => {
    // 只要地图 ready，就保证状态一致
    if (!mapReady.value) return
    setInteractiveStatus(enabled)
    document.removeEventListener('pointerdown', onOutsidePointerDown, true)
    if (enabled) {
      document.addEventListener('pointerdown', onOutsidePointerDown, true)
    }
  },
  { immediate: true }
)
</script>
