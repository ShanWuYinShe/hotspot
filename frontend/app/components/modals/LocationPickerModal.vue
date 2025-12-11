<template>
  <UModal
    :title="title"
    description="选择位置信息"
    :ui="{ width: 'sm:max-w-2xl' }"
  >
    <template #body="{ close }">
      <div class="space-y-4">
        <!-- 地图容器 -->
        <div class="relative rounded-xl overflow-hidden border border-default">
          <div ref="mapContainer" class="w-full h-64 sm:h-80 bg-interactive" />
          <!-- 中心标记 -->
          <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-full pointer-events-none z-10">
            <UIcon name="i-heroicons-map-pin-solid" class="w-8 h-8 text-primary drop-shadow-lg" />
          </div>
          <!-- 加载状态 -->
          <div v-if="mapLoading" class="absolute inset-0 flex items-center justify-center bg-overlay">
            <UIcon name="i-heroicons-arrow-path" class="w-8 h-8 text-primary animate-spin" />
          </div>
        </div>

        <!-- 选中位置信息 -->
        <div class="p-3 bg-surface/50 rounded-lg">
          <div class="flex items-start gap-3">
            <UIcon name="i-heroicons-map-pin" class="w-5 h-5 text-primary mt-0.5 shrink-0" />
            <div class="min-w-0 flex-1">
              <p class="font-medium text-sm truncate">{{ selectedLocationName || '拖动地图选择位置' }}</p>
              <p v-if="selectedLatitude && selectedLongitude" class="text-xs text-secondary-content mt-1">
                {{ selectedLatitude.toFixed(6) }}, {{ selectedLongitude.toFixed(6) }}
              </p>
            </div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex gap-3">
          <UButton
            variant="soft"
            icon="i-heroicons-arrow-path"
            :loading="gettingLocation"
            @click="locateMe"
          >
            定位到当前位置
          </UButton>
          <UButton
            class="flex-1"
            :disabled="!selectedLocationName"
            @click="() => confirmLocation(close)"
          >
            确认选择
          </UButton>
        </div>
      </div>
    </template>

    <template #footer="{ close }">
      <UButton
        variant="outline"
        @click="close"
      >
        取消
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { DEFAULTS as defaults } from '~/config/defaults'
import { geocoding } from '~/utils/geocoding'
import { waitForAMap, loadAMapPlugin } from '~/plugins/amap.client'

export interface LocationResult {
  latitude: number
  longitude: number
  locationName: string
}

interface Props {
  title?: string
  onSelect?: (location: LocationResult) => void
}

const props = withDefaults(defineProps<Props>(), {
  title: '选择位置'
})

const toast = useToast()
const locationStore = useLocationStore()

// 状态
const mapContainer = ref<HTMLElement | null>(null)
const mapLoading = ref(true)
const gettingLocation = ref(false)
const selectedLatitude = ref<number>(locationStore.location?.latitude || defaults.latitude)
const selectedLongitude = ref<number>(locationStore.location?.longitude || defaults.longitude)
const selectedLocationName = ref<string>(locationStore.location?.locationName || '')

// 高德地图实例
let mapInstance: AMap.Map | null = null
let geolocationInstance: AMap.Geolocation | null = null

// 初始化地图 - 使用 nextTick 确保 DOM 已渲染
onMounted(async () => {
  await nextTick()
  await initMap()
})

// 清理地图
onUnmounted(() => {
  if (mapInstance) {
    mapInstance.destroy()
    mapInstance = null
  }
  geolocationInstance = null
})

async function initMap() {
  if (!mapContainer.value) {
    mapLoading.value = false
    return
  }

  mapLoading.value = true
  try {
    // 等待高德地图 SDK 加载完成（由 amap.client.ts 插件负责加载）
    await waitForAMap()
    
    // 确保 Geolocation 插件已加载
    await loadAMapPlugin('AMap.Geolocation')
    
    // 创建地图实例
    mapInstance = new AMap.Map(mapContainer.value, {
      zoom: 15,
      center: [selectedLongitude.value, selectedLatitude.value],
      mapStyle: document.documentElement.classList.contains('dark') ? 'amap://styles/dark' : 'amap://styles/normal'
    })

    // 创建定位实例
    geolocationInstance = new AMap.Geolocation({
      enableHighAccuracy: true,
      timeout: 10000,
      convert: true,
      showButton: false,
      showMarker: false,
      showCircle: false,
      extensions: 'all'
    })

    // 监听地图移动结束事件
    mapInstance.on('moveend', handleMapMoveEnd)
    
    // 初始获取中心点位置名称
    await updateLocationName()
  } catch (error) {
    console.error('[LocationPicker] 地图初始化失败:', error)
    toast.add({ title: '地图加载失败', description: '请刷新页面重试', color: 'error' })
  } finally {
    mapLoading.value = false
  }
}

// 处理地图移动结束
async function handleMapMoveEnd() {
  if (!mapInstance) return
  
  const center = mapInstance.getCenter()
  selectedLatitude.value = center.getLat()
  selectedLongitude.value = center.getLng()
  
  await updateLocationName()
}

// 更新位置名称
async function updateLocationName() {
  try {
    selectedLocationName.value = await geocoding.getLocationName(
      selectedLatitude.value,
      selectedLongitude.value
    )
  } catch {
    selectedLocationName.value = `${selectedLatitude.value.toFixed(4)}, ${selectedLongitude.value.toFixed(4)}`
  }
}

// 定位到当前位置（使用高德地图定位 API）
async function locateMe() {
  if (!geolocationInstance || !mapInstance) {
    toast.add({ title: '定位服务未就绪', color: 'warning' })
    return
  }
  
  gettingLocation.value = true
  try {
    const result = await new Promise<AMap.GeolocationResult>((resolve, reject) => {
      geolocationInstance!.getCurrentPosition((status, result) => {
        if (status === 'complete') {
          resolve(result)
        } else {
          reject(new Error(result.message || '定位失败'))
        }
      })
    })
    
    // 使用高德地图定位结果
    const lng = result.position.getLng()
    const lat = result.position.getLat()
    
    selectedLatitude.value = lat
    selectedLongitude.value = lng
    mapInstance.setCenter([lng, lat])
    
    // 如果定位结果包含地址信息，直接使用
    if (result.formattedAddress) {
      selectedLocationName.value = formatLocationName(result)
    } else {
      await updateLocationName()
    }
  } catch {
    toast.add({ 
      title: '获取位置失败', 
      description: '请检查位置权限设置或稍后重试', 
      color: 'error' 
    })
  } finally {
    gettingLocation.value = false
  }
}

// 格式化位置名称
function formatLocationName(result: AMap.GeolocationResult): string {
  const addr = result.addressComponent
  if (addr) {
    // 优先返回：区 + 街道
    if (addr.district && addr.street) {
      return `${addr.district}${addr.street}`
    }
    if (addr.district && addr.township) {
      return `${addr.district}${addr.township}`
    }
    if (addr.district) {
      return addr.district
    }
    if (addr.city) {
      return addr.city
    }
  }
  return result.formattedAddress || '未知位置'
}

// 确认选择
function confirmLocation(close: () => void) {
  if (!selectedLocationName.value) return

  props.onSelect?.({
    latitude: selectedLatitude.value,
    longitude: selectedLongitude.value,
    locationName: selectedLocationName.value
  })
  close()
}

</script>
