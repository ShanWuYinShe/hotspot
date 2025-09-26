<template>
  <div class="relative w-full h-full bg-slate-100 dark:bg-slate-800 rounded-lg overflow-hidden">
    <!-- 地图容器 -->
    <div
      ref="mapContainer"
      class="w-full h-full"
      :class="{ 'opacity-50': loading }"
    />
    
    <!-- 加载状态 -->
    <div
      v-if="loading"
      class="absolute inset-0 flex items-center justify-center bg-white/80 dark:bg-slate-900/80"
    >
      <div class="text-center">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600 mx-auto mb-2"></div>
        <p class="text-sm text-slate-600 dark:text-slate-400">加载地图中...</p>
      </div>
    </div>
    
    <!-- 地图控制按钮 -->
    <div class="absolute top-4 right-4 flex flex-col gap-2">
      <UButton
        icon="i-heroicons-magnifying-glass-plus"
        variant="solid"
        color="neutral"
        size="sm"
        @click="zoomIn"
      />
      <UButton
        icon="i-heroicons-magnifying-glass-minus"
        variant="solid"
        color="neutral"
        size="sm"
        @click="zoomOut"
      />
      <UButton
        icon="i-heroicons-arrow-path"
        variant="solid"
        color="neutral"
        size="sm"
        @click="resetView"
      />
      <UButton
        icon="i-heroicons-map-pin"
        variant="solid"
        color="primary"
        size="sm"
        @click="locateMe"
        :loading="locating"
      />
    </div>
    
    <!-- 地图图例 -->
    <div
      v-if="showLegend && markers.length > 0"
      class="absolute bottom-4 left-4 bg-white dark:bg-slate-800 rounded-lg p-3 shadow-lg border border-slate-200 dark:border-slate-700"
    >
      <h4 class="text-sm font-medium text-slate-900 dark:text-white mb-2">图例</h4>
      <div class="space-y-1">
        <div
          v-for="type in markerTypes"
          :key="type.key"
          class="flex items-center gap-2 text-xs"
        >
          <div
            class="w-3 h-3 rounded-full"
            :style="{ backgroundColor: type.color }"
          />
          <span class="text-slate-600 dark:text-slate-400">{{ type.label }}</span>
        </div>
      </div>
    </div>
    
    <!-- 地图信息面板 -->
    <div
      v-if="selectedMarker"
      class="absolute bottom-4 right-4 bg-white dark:bg-slate-800 rounded-lg p-4 shadow-lg border border-slate-200 dark:border-slate-700 max-w-xs"
    >
      <div class="flex items-start gap-3">
        <UAvatar
          :src="selectedMarker.avatar"
          :alt="selectedMarker.title"
          size="sm"
        />
        <div class="flex-1 min-w-0">
          <h4 class="font-medium text-slate-900 dark:text-white truncate">
            {{ selectedMarker.title }}
          </h4>
          <p class="text-sm text-slate-600 dark:text-slate-400 mt-1">
            {{ selectedMarker.description }}
          </p>
          <div class="flex items-center gap-2 mt-2 text-xs text-slate-500">
            <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
            <span>距离 {{ formatDistance(selectedMarker.distance || 0) }}</span>
          </div>
        </div>
        <UButton
          icon="i-heroicons-x-mark"
          variant="ghost"
          size="xs"
          @click="selectedMarker = null"
        />
      </div>
      
      <div class="flex gap-2 mt-3">
        <UButton
          size="xs"
          variant="outline"
          @click="navigateToMarker(selectedMarker)"
        >
          导航
        </UButton>
        <UButton
          size="xs"
          @click="viewMarkerDetails(selectedMarker)"
        >
          查看详情
        </UButton>
      </div>
    </div>
    
    <!-- 开发中提示 -->
    <div
      v-if="!mapLoaded"
      class="absolute inset-0 flex items-center justify-center bg-slate-50 dark:bg-slate-800"
    >
      <div class="text-center">
        <UIcon name="i-heroicons-map" class="w-16 h-16 text-slate-400 mx-auto mb-4" />
        <h3 class="text-lg font-medium text-slate-900 dark:text-white mb-2">地图功能</h3>
        <p class="text-slate-600 dark:text-slate-400 mb-4">即将支持高德地图集成</p>
        <div class="space-y-2 text-sm text-slate-500">
          <p>• 实时位置定位</p>
          <p>• 附近用户和群组</p>
          <p>• 路线导航</p>
          <p>• 地点标记</p>
        </div>
        <UButton
          class="mt-4"
          variant="outline"
          @click="initializeMockMap"
        >
          查看演示
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { formatDistance } from '~/utils/formatters'

export interface MapMarker {
  id: string
  type: 'user' | 'group' | 'moment' | 'location'
  title: string
  description: string
  latitude: number
  longitude: number
  avatar?: string
  distance?: number
  data?: any
}

interface Props {
  markers?: MapMarker[]
  center?: { latitude: number; longitude: number }
  zoom?: number
  showLegend?: boolean
  interactive?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  markers: () => [],
  center: () => ({ latitude: 39.9042, longitude: 116.4074 }), // 北京天安门
  zoom: 13,
  showLegend: true,
  interactive: true
})

const emit = defineEmits<{
  markerClick: [marker: MapMarker]
  mapClick: [coordinates: { latitude: number; longitude: number }]
  locationUpdate: [location: { latitude: number; longitude: number }]
}>()

// 状态
const mapContainer = ref<HTMLElement>()
const loading = ref(false)
const locating = ref(false)
const mapLoaded = ref(false)
const selectedMarker = ref<MapMarker | null>(null)
const currentLocation = ref(props.center)

// 地图实例（预留给真实地图SDK）
const mapInstance = ref<any>(null)

// 标记类型配置
const markerTypes = computed(() => [
  { key: 'user', label: '用户', color: '#3b82f6' },
  { key: 'group', label: '群组', color: '#10b981' },
  { key: 'moment', label: '动态', color: '#f59e0b' },
  { key: 'location', label: '地点', color: '#ef4444' }
])

// 方法
const initializeMockMap = () => {
  loading.value = true
  
  // 模拟地图加载
  setTimeout(() => {
    mapLoaded.value = true
    loading.value = false
    
    // 如果有标记，显示第一个
    if (props.markers.length > 0) {
      selectedMarker.value = props.markers[0]
    }
  }, 1500)
}

const zoomIn = () => {
  // 预留给真实地图SDK
  console.log('Zoom in')
}

const zoomOut = () => {
  // 预留给真实地图SDK
  console.log('Zoom out')
}

const resetView = () => {
  currentLocation.value = props.center
  selectedMarker.value = null
  // 预留给真实地图SDK
  console.log('Reset view to center')
}

const locateMe = async () => {
  if (!navigator.geolocation) {
    useSmartToast().error('您的浏览器不支持定位功能')
    return
  }
  
  locating.value = true
  
  try {
    const position = await new Promise<GeolocationPosition>((resolve, reject) => {
      navigator.geolocation.getCurrentPosition(resolve, reject, {
        enableHighAccuracy: true,
        timeout: 10000,
        maximumAge: 60000
      })
    })
    
    const location = {
      latitude: position.coords.latitude,
      longitude: position.coords.longitude
    }
    
    currentLocation.value = location
    emit('locationUpdate', location)
    
    useSmartToast().success('定位成功')
  } catch (error) {
    console.error('定位失败:', error)
    useSmartToast().error('定位失败，请检查位置权限')
  } finally {
    locating.value = false
  }
}

const navigateToMarker = (marker: MapMarker) => {
  // 预留给导航功能
  const url = `https://uri.amap.com/navigation?to=${marker.longitude},${marker.latitude}&name=${encodeURIComponent(marker.title)}`
  window.open(url, '_blank')
}

const viewMarkerDetails = (marker: MapMarker) => {
  emit('markerClick', marker)
}

// 监听标记变化
watch(() => props.markers, (newMarkers) => {
  if (mapLoaded.value && newMarkers.length > 0) {
    // 预留给真实地图SDK更新标记
    console.log('Update markers:', newMarkers)
  }
}, { deep: true })

// 组件挂载时初始化
onMounted(() => {
  // 这里可以初始化真实的地图SDK
  // 目前显示开发中状态
})

// 组件卸载时清理
onUnmounted(() => {
  if (mapInstance.value) {
    // 清理地图实例
    mapInstance.value = null
  }
})
</script>

<style scoped>
/* 地图容器样式 */
.map-container {
  position: relative;
  width: 100%;
  height: 100%;
}

/* 预留给地图SDK的样式 */
:deep(.amap-container) {
  border-radius: 0.5rem;
}
</style>
