<!--
高德地图组件
支持用户位置标记、附近用户显示、群组位置等功能
-->
<template>
  <div class="amap-container">
    <div ref="mapContainer" class="amap-wrapper" :style="{ height: mapHeight }"></div>

    <!-- 地图控制面板 -->
    <div class="map-controls" v-if="showControls">
      <div class="control-group">
        <NButton @click="centerToCurrentLocation" :icon="Location" size="small" type="primary" :loading="isLocating"
          round>
          {{ isLocating ? '定位中...' : '回到我的位置' }}
        </NButton>

        <NButton @click="toggleUserLayer" :icon="UserFilled" size="small" :type="showUsers ? 'primary' : 'default'"
          round>
          {{ showUsers ? '隐藏用户' : '显示用户' }}
        </NButton>

        <NButton @click="toggleGroupLayer" :icon="OfficeBuilding" size="small"
          :type="showGroups ? 'primary' : 'default'" round>
          {{ showGroups ? '隐藏群组' : '显示群组' }}
        </NButton>
      </div>

      <div class="control-group" v-if="isFullscreen">
        <NButton @click="$emit('close')" :icon="Close" size="small" type="info" round>
          关闭地图
        </NButton>
      </div>
    </div>

    <!-- 位置信息面板 -->
    <div class="location-info" v-if="currentLocation && showLocationInfo">
      <div class="location-text">
        <NIcon>
          <Location />
        </NIcon>
        <span>{{ currentLocation.locationName || '当前位置' }}</span>
      </div>
      <div class="coordinates">
        {{ currentLocation.latitude?.toFixed(6) }}, {{ currentLocation.longitude?.toFixed(6) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { NButton, NIcon, useMessage } from 'naive-ui'
import { Location, Person as UserFilled, Business as OfficeBuilding, Close } from '@vicons/ionicons5'
import type { UserLocationPin } from '~/types/models/user'
import type { GroupInfo } from '~/types/models/group'
import { defaultImages } from '@/utils/assets'

// API实例
const message = useMessage()

interface Props {
  // 地图高度
  mapHeight?: string
  // 是否显示控制面板
  showControls?: boolean
  // 是否显示位置信息
  showLocationInfo?: boolean
  // 是否全屏模式
  isFullscreen?: boolean
  // 初始中心点
  center?: [number, number]
  // 初始缩放级别
  zoom?: number
  // 附近用户数据
  nearbyUsers?: UserLocationPin[]
  // 附近群组数据
  nearbyGroups?: GroupInfo[]
}

interface Emits {
  (e: 'close'): void
  (e: 'locationUpdate', location: { latitude: number; longitude: number; locationName?: string }): void
  (e: 'userClick', user: UserLocationPin): void
  (e: 'groupClick', group: GroupInfo): void
}

const props = withDefaults(defineProps<Props>(), {
  mapHeight: '400px',
  showControls: true,
  showLocationInfo: true,
  isFullscreen: false,
  zoom: 15,
  nearbyUsers: () => [],
  nearbyGroups: () => []
})

const emit = defineEmits<Emits>()

// 地图相关
const mapContainer = ref<HTMLDivElement>()
let mapInstance: any = null
let currentLocationMarker: any = null
let userMarkers: any[] = []
let groupMarkers: any[] = []

// 状态
const isMapReady = ref(false)
const isLocating = ref(false)
const showUsers = ref(true)
const showGroups = ref(true)
const currentLocation = ref<{ latitude: number; longitude: number; locationName?: string } | null>(null)

// Store
const userStore = useUserStore()
const authStore = useAuthStore()

let AMapLoader: any = null

// 初始化地图
const initMap = async () => {
  if (typeof window === 'undefined') return; // SSR 阶段直接跳过
  try {
    if (!AMapLoader) {
      AMapLoader = (await import('@amap/amap-jsapi-loader')).default
    }
    const AMap = await AMapLoader.load({
      key: '58e6f10e4de9cf60e04d7b8e72b70a53', // 请替换为您的高德地图API Key
      version: '2.0',
      plugins: ['AMap.Geolocation', 'AMap.Geocoder']
    });

    if (!mapContainer.value) return;

    // 创建地图实例
    mapInstance = new AMap.Map(mapContainer.value, {
      zoom: props.zoom,
      center: props.center || [116.397428, 39.90923], // 默认北京
      mapStyle: 'amap://styles/darkblue', // 深色主题
      showLabel: true,
      showBuildingBlock: true
    });

    // 添加地图控件
    mapInstance.addControl(new AMap.Scale());
    mapInstance.addControl(new AMap.ToolBar({
      position: {
        top: '10px',
        right: '10px'
      }
    }));

    isMapReady.value = true;

    // 如果有用户位置，设置当前位置
    if (userStore.currentLocation) {
      await setCurrentLocation(
        userStore.currentLocation.latitude,
        userStore.currentLocation.longitude,
        userStore.currentLocation.locationName
      );
    } else {
      // 尝试获取当前位置
      await getCurrentLocation();
    }

    // 加载附近用户和群组
    updateUserMarkers();
    updateGroupMarkers();

  } catch (error) {
    console.error('地图初始化失败:', error);
    message.error('地图加载失败，请检查网络连接');
  }
}

// 获取当前位置
const getCurrentLocation = async () => {
  if (!mapInstance) return

  isLocating.value = true

  try {
    const geolocation = new (window as any).AMap.Geolocation({
      enableHighAccuracy: true,
      timeout: 10000,
      maximumAge: 0
    })

    geolocation.getCurrentPosition((status: string, result: any) => {
      if (status === 'complete') {
        const { lng, lat } = result.position
        setCurrentLocation(lat, lng, result.formattedAddress)

        // 通知父组件位置更新
        emit('locationUpdate', {
          latitude: lat,
          longitude: lng,
          locationName: result.formattedAddress
        })
      } else {
        console.error('定位失败:', result)
        message.warning('定位失败，使用默认位置')
      }
      isLocating.value = false
    })
  } catch (error) {
    console.error('获取位置失败:', error)
    isLocating.value = false
  }
}

// 设置当前位置
const setCurrentLocation = async (lat: number, lng: number, locationName?: string) => {
  if (!mapInstance) return

  currentLocation.value = { latitude: lat, longitude: lng, locationName }

  // 移除旧的位置标记
  if (currentLocationMarker) {
    mapInstance.remove(currentLocationMarker)
  }

  // 创建当前位置标记
  currentLocationMarker = new (window as any).AMap.Marker({
    position: [lng, lat],
    icon: new (window as any).AMap.Icon({
      size: new (window as any).AMap.Size(32, 32),
      image: '/images/markers/current-location.png',
      imageSize: new (window as any).AMap.Size(32, 32)
    }),
    title: `我的位置: ${authStore.user?.nickname || '我'}`,
    zIndex: 100
  })

  mapInstance.add(currentLocationMarker)
  mapInstance.setCenter([lng, lat])
}

// 回到当前位置
const centerToCurrentLocation = async () => {
  if (currentLocation.value && mapInstance) {
    mapInstance.setCenter([currentLocation.value.longitude, currentLocation.value.latitude])
    mapInstance.setZoom(16)
  } else {
    await getCurrentLocation()
  }
}

// 更新用户标记
const updateUserMarkers = () => {
  if (!mapInstance || !showUsers.value) return

  // 清除旧标记
  if (userMarkers.length > 0) {
    mapInstance.remove(userMarkers)
    userMarkers = []
  }

  // 添加新标记
  props.nearbyUsers.forEach(user => {
    const marker = new (window as any).AMap.Marker({
      position: [user.longitude, user.latitude],
      icon: new (window as any).AMap.Icon({
        size: new (window as any).AMap.Size(24, 24),
        image: user.avatar_url || defaultImages.userAvatar,
        imageSize: new (window as any).AMap.Size(24, 24)
      }),
      title: `用户: ${user.nickname}`,
      zIndex: 50
    })

    // 点击事件
    marker.on('click', () => {
      emit('userClick', user)
    })

    userMarkers.push(marker)
  })

  if (userMarkers.length > 0) {
    mapInstance.add(userMarkers)
  }
}

// 更新群组标记
const updateGroupMarkers = () => {
  if (!mapInstance || !showGroups.value) return

  // 清除旧标记
  if (groupMarkers.length > 0) {
    mapInstance.remove(groupMarkers)
    groupMarkers = []
  }

  // 添加新标记
  props.nearbyGroups.forEach(group => {
    if (group.latitude && group.longitude) {
      const marker = new (window as any).AMap.Marker({
        position: [group.longitude, group.latitude],
        icon: new (window as any).AMap.Icon({
          size: new (window as any).AMap.Size(28, 28),
          image: '/images/markers/group-marker.png',
          imageSize: new (window as any).AMap.Size(28, 28)
        }),
        title: `群组: ${group.name}`,
        zIndex: 60
      })

      // 点击事件
      marker.on('click', () => {
        emit('groupClick', group)
      })

      groupMarkers.push(marker)
    }
  })

  if (groupMarkers.length > 0) {
    mapInstance.add(groupMarkers)
  }
}

// 切换用户图层
const toggleUserLayer = () => {
  showUsers.value = !showUsers.value
  if (showUsers.value) {
    updateUserMarkers()
  } else {
    if (userMarkers.length > 0) {
      mapInstance.remove(userMarkers)
      userMarkers = []
    }
  }
}

// 切换群组图层
const toggleGroupLayer = () => {
  showGroups.value = !showGroups.value
  if (showGroups.value) {
    updateGroupMarkers()
  } else {
    if (groupMarkers.length > 0) {
      mapInstance.remove(groupMarkers)
      groupMarkers = []
    }
  }
}

// 监听数据变化
watch(() => props.nearbyUsers, updateUserMarkers, { deep: true })
watch(() => props.nearbyGroups, updateGroupMarkers, { deep: true })

// 组件挂载
onMounted(() => {
  if (typeof window !== 'undefined') {
    nextTick(() => {
      initMap();
    });
  }
})

// 组件卸载
onUnmounted(() => {
  if (mapInstance) {
    mapInstance.destroy()
  }
})
</script>

<style scoped lang="scss">
.amap-container {
  position: relative;
  width: 100%;
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-background-secondary);
}

.amap-wrapper {
  width: 100%;
  border-radius: 8px;
}

.map-controls {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: var(--z-modal);
  display: flex;
  flex-direction: column;
  gap: 12px;

  .control-group {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
}

.location-info {
  position: absolute;
  bottom: 16px;
  left: 16px;
  z-index: var(--z-modal);
  background: rgba(0, 0, 0, 0.8);
  color: white;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 12px;
  backdrop-filter: blur(10px);

  .location-text {
    display: flex;
    align-items: center;
    gap: 4px;
    font-weight: 500;
    margin-bottom: 2px;
  }

  .coordinates {
    color: rgba(255, 255, 255, 0.7);
    font-size: 10px;
  }
}

// 响应式设计
@media (max-width: 768px) {
  .map-controls {
    top: 12px;
    left: 12px;

    .control-group {
      flex-direction: column;
      width: max-content;
    }
  }

  .location-info {
    bottom: 12px;
    left: 12px;
    right: 12px;
  }
}
</style>
