<template>
  <div class="relative w-full h-full bg-slate-base rounded-lg overflow-hidden">
    <!-- 地图容器 -->
    <div ref="mapContainer" class="w-full h-full" />

    <!-- 加载中 -->
    <div
      v-if="loading"
      class="absolute inset-0 flex items-center justify-center bg-glass-white-deep"
    >
      <div class="text-center">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-strong mx-auto mb-2" />
        <p class="text-sm text-secondary-content">加载地图中...</p>
      </div>
    </div>

    <!-- 控制按钮 -->
    <div v-if="mapReady" class="absolute top-4 right-4 flex flex-col gap-2">
      <UButton icon="i-heroicons-plus" variant="solid" color="neutral" size="sm" @click="zoomIn" />
      <UButton icon="i-heroicons-minus" variant="solid" color="neutral" size="sm" @click="zoomOut" />
      <UButton icon="i-heroicons-map-pin" variant="solid" color="primary" size="sm" :loading="locating" @click="locateMe" />
    </div>

    <!-- 选中的标记信息 -->
    <div
      v-if="selectedMarker"
      class="absolute bottom-4 left-4 right-4 bg-card-elevated rounded-lg p-4 shadow-lg"
    >
      <div class="flex items-center gap-3">
        <UAvatar :src="selectedMarker.avatar" size="sm" />
        <div class="flex-1 min-w-0">
          <p class="font-medium truncate">{{ selectedMarker.title }}</p>
          <p class="text-sm text-secondary-content">{{ selectedMarker.description }}</p>
        </div>
        <UButton icon="i-heroicons-x-mark" variant="ghost" size="sm" @click="selectedMarker = null" />
      </div>
    </div>

    <!-- 未配置提示 -->
    <div
      v-if="!isConfigured"
      class="absolute inset-0 flex items-center justify-center bg-surface"
    >
      <div class="text-center p-4">
        <UIcon name="i-heroicons-map" class="w-12 h-12 text-muted mx-auto mb-3" />
        <p class="text-secondary-content">请配置 NUXT_PUBLIC_AMAP_KEY</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
export interface MapMarker {
  id: string
  type: 'user' | 'group' | 'moment' | 'location'
  title: string
  description: string
  latitude: number
  longitude: number
  avatar?: string
  distance?: number
  data?: Record<string, unknown>
}

const props = withDefaults(defineProps<{
  markers?: MapMarker[]
  center?: { latitude: number; longitude: number }
  zoom?: number
}>(), {
  markers: () => [],
  center: () => ({ latitude: 39.9042, longitude: 116.4074 }),
  zoom: 13
})

const emit = defineEmits<{
  markerClick: [marker: MapMarker]
  mapClick: [coords: { latitude: number; longitude: number }]
}>()

const config = useRuntimeConfig()
const toast = useToast()

const mapContainer = ref<HTMLElement>()
const mapInstance = ref<AMap.Map | null>(null)
const markerInstances = ref<Map<string, AMap.Marker>>(new Map())
const loading = ref(true)
const locating = ref(false)
const mapReady = ref(false)
const selectedMarker = ref<MapMarker | null>(null)

const isConfigured = computed(() => !!config.public.amapKey)

// 等待 AMap 加载
const waitForAMap = (): Promise<void> => {
  return new Promise((resolve, reject) => {
    if (window.AMap) {
      resolve()
      return
    }
    let attempts = 0
    const check = setInterval(() => {
      attempts++
      if (window.AMap) {
        clearInterval(check)
        resolve()
      } else if (attempts > 50) {
        clearInterval(check)
        reject(new Error('AMap 加载超时'))
      }
    }, 100)
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
      viewMode: '2D'
    })

    mapInstance.value.on('click', (e: unknown) => {
      const event = e as { lnglat: AMap.LngLat }
      emit('mapClick', {
        latitude: event.lnglat.getLat(),
        longitude: event.lnglat.getLng()
      })
    })

    mapReady.value = true
    updateMarkers()
  } catch (error) {
    console.error('[Map] 初始化失败:', error)
    toast.add({ title: '地图加载失败', color: 'error' })
  } finally {
    loading.value = false
  }
}

// 更新标记
const updateMarkers = () => {
  if (!mapInstance.value) return

  // 清除旧标记
  markerInstances.value.forEach(m => mapInstance.value?.remove(m))
  markerInstances.value.clear()

  // 添加新标记
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
    const pos = await new Promise<GeolocationPosition>((resolve, reject) => {
      navigator.geolocation.getCurrentPosition(resolve, reject, {
        enableHighAccuracy: false,
        timeout: 5000,
        maximumAge: 60000
      })
    })
    mapInstance.value.setCenter([pos.coords.longitude, pos.coords.latitude])
    toast.add({ title: '定位成功', color: 'success' })
  } catch {
    toast.add({ title: '定位失败', color: 'error' })
  } finally {
    locating.value = false
  }
}

// 监听
watch(() => props.markers, updateMarkers, { deep: true })
watch(() => props.center, (c) => {
  mapInstance.value?.setCenter([c.longitude, c.latitude])
})

onMounted(initMap)
onUnmounted(() => {
  mapInstance.value?.destroy()
  mapInstance.value = null
})
</script>
