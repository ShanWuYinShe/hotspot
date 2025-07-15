<!-- 位置分享组件 -->
<template>
  <div class="location-share">
    <!-- 触发按钮 -->
    <NDialog v-model="visible" title="分享位置" width="90%" :style="{ maxWidth: '500px' }" @close="handleClose">
      <div class="location-content">
        <!-- 当前位置信息 -->
        <div v-if="currentLocation" class="current-location">
          <div class="location-header">
            <NIcon class="location-icon">
              <Location />
            </NIcon>
            <div class="location-info">
              <h4 class="location-name">{{ currentLocation.name || '当前位置' }}</h4>
              <p class="location-address">{{ currentLocation.address || '获取地址中...' }}</p>
            </div>
            <NButton size="small" @click="refreshLocation" :loading="isRefreshing" :icon="Refresh">
              刷新
            </NButton>
          </div>

          <!-- 地图预览 -->
          <div class="map-preview">
            <div v-if="!mapLoaded" class="map-placeholder">
              <NSkeleton animated height="200px" width="100%" />
            </div>
            <div v-else class="map-container" ref="mapContainer">
              <!-- 这里可以集成地图组件 -->
              <div class="map-mock">
                <NIcon class="map-pin">
                  <LocationFilled />
                </NIcon>
                <p>{{ currentLocation.name }}</p>
                <small>精确度: {{ currentLocation.accuracy?.toFixed(0) }}米</small>
              </div>
            </div>
          </div>

          <!-- 位置详情 -->
          <div class="location-details">
            <div class="detail-item">
              <span class="label">经度:</span>
              <span class="value">{{ currentLocation.longitude?.toFixed(6) }}</span>
            </div>
            <div class="detail-item">
              <span class="label">纬度:</span>
              <span class="value">{{ currentLocation.latitude?.toFixed(6) }}</span>
            </div>
            <div v-if="currentLocation.accuracy" class="detail-item">
              <span class="label">精确度:</span>
              <span class="value">{{ currentLocation.accuracy.toFixed(0) }}米</span>
            </div>
          </div>
        </div>

        <!-- 位置搜索 -->
        <div class="location-search">
          <NInput v-model="searchKeyword" placeholder="搜索其他位置..." @keyup.enter="searchLocation" class="search-input">
            <template #suffix>
              <NButton @click="searchLocation" :loading="isSearching" :icon="Search" />
            </template>
          </NInput>

          <!-- 搜索结果 -->
          <div v-if="searchResults.length > 0" class="search-results">
            <div v-for="(result, index) in searchResults" :key="index" class="search-result-item"
              @click="selectSearchResult(result)">
              <NIcon class="result-icon">
                <Location />
              </NIcon>
              <div class="result-info">
                <div class="result-name">{{ result.name }}</div>
                <div class="result-address">{{ result.address }}</div>
              </div>
            </div>
          </div>
        </div>

        <!-- 最近位置 -->
        <div v-if="recentLocations.length > 0" class="recent-locations">
          <h5>最近位置</h5>
          <div class="recent-list">
            <div v-for="(location, index) in recentLocations" :key="index" class="recent-item"
              @click="selectLocation(location)">
              <NIcon class="recent-icon">
                <Clock />
              </NIcon>
              <div class="recent-info">
                <div class="recent-name">{{ location.name }}</div>
                <div class="recent-address">{{ location.address }}</div>
              </div>
            </div>
          </div>
        </div>

        <!-- 错误状态 -->
        <div v-if="error" class="error-state">
          <NAlert :title="error" type="error" :closable="false" show-icon />
          <div class="error-actions">
            <NButton @click="getCurrentLocation" :loading="isGettingLocation">
              重新获取位置
            </NButton>
          </div>
        </div>

        <!-- 无位置状态 -->
        <div v-if="!currentLocation && !error && !isGettingLocation" class="no-location">
          <div class="no-location-icon">📍</div>
          <h4>获取您的位置</h4>
          <p>点击下方按钮获取当前位置信息</p>
          <NButton type="primary" @click="getCurrentLocation" :loading="isGettingLocation" :icon="Location">
            获取位置
          </NButton>
        </div>
      </div>

      <template #action>
        <div class="dialog-footer">
          <NButton @click="handleClose">取消</NButton>
          <NButton type="primary" @click="shareLocation" :disabled="!currentLocation" :loading="isSharing"
            :icon="Share">
            分享位置
          </NButton>
        </div>
      </template>
    </NDialog>
  </div>
</template>

<script setup lang="ts">

import { NDialog, NIcon, NButton, NSkeleton, NInput, useMessage, NAlert } from 'naive-ui'
import { Location, LocationOutline as LocationFilled, Refresh, Search, Time as Clock, Share } from '@vicons/ionicons5'

// API实例
const message = useMessage()

interface LocationInfo {
  name?: string
  address?: string
  latitude: number
  longitude: number
  accuracy?: number
}

interface Props {
  modelValue?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [visible: boolean]
  'share': [location: LocationInfo]
  'close': []
}>()

// 状态
const visible = ref(props.modelValue || false)
const currentLocation = ref<LocationInfo | null>(null)
const searchKeyword = ref('')
const searchResults = ref<LocationInfo[]>([])
const recentLocations = ref<LocationInfo[]>([])
const error = ref('')

// 加载状态
const isGettingLocation = ref(false)
const isRefreshing = ref(false)
const isSearching = ref(false)
const isSharing = ref(false)
const mapLoaded = ref(false)

// 地图容器引用
const mapContainer = ref()

// 监听外部状态变化
watch(() => props.modelValue, (newVal) => {
  visible.value = newVal || false
  if (newVal && !currentLocation.value) {
    getCurrentLocation()
  }
})

// 监听内部状态变化
watch(visible, (newVal) => {
  emit('update:modelValue', newVal)
  if (newVal && !currentLocation.value) {
    getCurrentLocation()
  }
})

// 获取当前位置
const getCurrentLocation = async () => {
  if (!navigator.geolocation) {
    error.value = '浏览器不支持地理定位'
    return
  }

  isGettingLocation.value = true
  error.value = ''

  try {
    const position = await new Promise<GeolocationPosition>((resolve, reject) => {
      navigator.geolocation.getCurrentPosition(
        resolve,
        reject,
        {
          enableHighAccuracy: true,
          timeout: 15000,
          maximumAge: 300000 // 5分钟缓存
        }
      )
    })

    const { latitude, longitude, accuracy } = position.coords

    // 获取地址信息
    const locationInfo: LocationInfo = {
      latitude,
      longitude,
      accuracy
    }

    // 反向地理编码获取地址
    await reverseGeocode(locationInfo)

    currentLocation.value = locationInfo

    // 地图加载
    setTimeout(() => {
      mapLoaded.value = true
    }, 1000)

  } catch (err) {
    console.error('获取位置失败:', err)

    if (err instanceof GeolocationPositionError) {
      switch (err.code) {
        case err.PERMISSION_DENIED:
          error.value = '位置访问被拒绝，请在浏览器设置中允许位置访问'
          break
        case err.POSITION_UNAVAILABLE:
          error.value = '位置信息不可用'
          break
        case err.TIMEOUT:
          error.value = '获取位置超时，请重试'
          break
        default:
          error.value = '获取位置失败'
      }
    } else {
      error.value = '获取位置失败'
    }
  } finally {
    isGettingLocation.value = false
  }
}

// 反向地理编码
const reverseGeocode = async (location: LocationInfo) => {
  try {
    // 这里应该调用真实的地理编码API
    // 从API获取真实数据
    await new Promise(resolve => setTimeout(resolve, 500))

    location.name = '当前位置'
    location.address = `纬度 ${location.latitude.toFixed(4)}, 经度 ${location.longitude.toFixed(4)}`

    // 地理编码服务待集成
    // 这里将集成真实的地理编码API
    // location.address = data.address

  } catch (err) {
    console.warn('反向地理编码失败:', err)
    location.name = '未知位置'
    location.address = `${location.latitude.toFixed(4)}, ${location.longitude.toFixed(4)}`
  }
}

// 刷新位置
const refreshLocation = async () => {
  isRefreshing.value = true
  await getCurrentLocation()
  isRefreshing.value = false
}

// 搜索位置
const searchLocation = async () => {
  if (!searchKeyword.value.trim()) return

  isSearching.value = true
  try {
    // 这里应该调用真实的搜索API
    await new Promise(resolve => setTimeout(resolve, 500))

    // 搜索结果
    searchResults.value = [
      {
        name: searchKeyword.value,
        address: '搜索结果地址',
        latitude: 39.9042,
        longitude: 116.4074
      }
    ]

    // 位置搜索服务待集成
    // 这里将集成真实的位置搜索API

  } catch (err) {
    console.error('搜索位置失败:', err)
    message.error('搜索位置失败')
  } finally {
    isSearching.value = false
  }
}

// 选择搜索结果
const selectSearchResult = (result: LocationInfo) => {
  selectLocation(result)
  searchResults.value = []
  searchKeyword.value = ''
}

// 选择位置
const selectLocation = (location: LocationInfo) => {
  currentLocation.value = { ...location }

  // 添加到最近位置
  addToRecent(location)
}

// 添加到最近位置
const addToRecent = (location: LocationInfo) => {
  // 移除已存在的
  const index = recentLocations.value.findIndex(
    l => Math.abs(l.latitude - location.latitude) < 0.001 &&
      Math.abs(l.longitude - location.longitude) < 0.001
  )
  if (index > -1) {
    recentLocations.value.splice(index, 1)
  }

  // 添加到开头
  recentLocations.value.unshift({ ...location })

  // 限制数量
  if (recentLocations.value.length > 5) {
    recentLocations.value = recentLocations.value.slice(0, 5)
  }

  // 保存到本地存储
  saveRecentLocations()
}

// 保存最近位置
const saveRecentLocations = () => {
  try {
    localStorage.setItem('recent-locations', JSON.stringify(recentLocations.value))
  } catch (e) {
    console.warn('保存最近位置失败:', e)
  }
}

// 加载最近位置
const loadRecentLocations = () => {
  try {
    const saved = localStorage.getItem('recent-locations')
    if (saved) {
      recentLocations.value = JSON.parse(saved)
    }
  } catch (e) {
    console.warn('加载最近位置失败:', e)
  }
}

// 分享位置
const shareLocation = async () => {
  if (!currentLocation.value) return

  isSharing.value = true
  try {
    emit('share', { ...currentLocation.value })
    addToRecent(currentLocation.value)
    handleClose()
  } catch (err) {
    console.error('分享位置失败:', err)
    message.error('分享位置失败')
  } finally {
    isSharing.value = false
  }
}

// 关闭对话框
const handleClose = () => {
  visible.value = false
  emit('close')

  // 重置状态
  searchKeyword.value = ''
  searchResults.value = []
  error.value = ''
  mapLoaded.value = false
}

// 组件挂载时加载最近位置
onMounted(() => {
  loadRecentLocations()
})

// 暴露方法
defineExpose({
  show: () => { visible.value = true },
  hide: () => { visible.value = false },
  refresh: refreshLocation
})
</script>

<style scoped lang="scss">
.location-share {
  display: inline-block;
}

.location-content {
  .current-location {
    margin-bottom: 1.5rem;

    .location-header {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      margin-bottom: 1rem;

      .location-icon {
        color: var(--el-color-primary);
        font-size: 1.25rem;
      }

      .location-info {
        flex: 1;
        min-width: 0;

        .location-name {
          margin: 0 0 0.25rem 0;
          font-size: 1rem;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        .location-address {
          margin: 0;
          font-size: 0.875rem;
          color: var(--el-text-color-secondary);
          line-height: 1.4;
        }
      }
    }

    .map-preview {
      border-radius: 8px;
      overflow: hidden;
      margin-bottom: 1rem;
      border: 1px solid var(--el-border-color-light);

      .map-placeholder,
      .map-container {
        height: 200px;
      }

      .map-mock {
        height: 100%;
        background: linear-gradient(135deg, #e8f4fd 0%, #c8e6f5 100%);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        color: var(--el-color-primary);

        .map-pin {
          font-size: 2rem;
          margin-bottom: 0.5rem;
        }

        p {
          margin: 0.25rem 0;
          font-weight: 600;
        }

        small {
          color: var(--el-text-color-secondary);
        }
      }
    }

    .location-details {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
      gap: 0.5rem;

      .detail-item {
        display: flex;
        justify-content: space-between;
        padding: 0.5rem;
        background: var(--el-fill-color-lighter);
        border-radius: 6px;
        font-size: 0.875rem;

        .label {
          color: var(--el-text-color-secondary);
        }

        .value {
          font-family: monospace;
          color: var(--el-text-color-primary);
          font-weight: 500;
        }
      }
    }
  }

  .location-search {
    margin-bottom: 1.5rem;

    .search-input {
      margin-bottom: 0.75rem;
    }

    .search-results {
      max-height: 200px;
      overflow-y: auto;
      border: 1px solid var(--el-border-color-light);
      border-radius: 6px;

      .search-result-item {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        padding: 0.75rem;
        cursor: pointer;
        border-bottom: 1px solid var(--el-border-color-lighter);
        transition: background-color 0.2s ease;

        &:hover {
          background: var(--el-fill-color-light);
        }

        &:last-child {
          border-bottom: none;
        }

        .result-icon {
          color: var(--el-color-primary);
          flex-shrink: 0;
        }

        .result-info {
          flex: 1;
          min-width: 0;

          .result-name {
            font-weight: 500;
            margin-bottom: 0.25rem;
          }

          .result-address {
            font-size: 0.875rem;
            color: var(--el-text-color-secondary);
          }
        }
      }
    }
  }

  .recent-locations {
    h5 {
      margin: 0 0 0.75rem 0;
      font-size: 0.875rem;
      color: var(--el-text-color-secondary);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }

    .recent-list {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;

      .recent-item {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        padding: 0.75rem;
        background: var(--el-fill-color-lighter);
        border-radius: 6px;
        cursor: pointer;
        transition: all 0.2s ease;

        &:hover {
          background: var(--el-fill-color);
          transform: translateY(-1px);
        }

        .recent-icon {
          color: var(--el-color-info);
          flex-shrink: 0;
        }

        .recent-info {
          flex: 1;
          min-width: 0;

          .recent-name {
            font-weight: 500;
            margin-bottom: 0.25rem;
          }

          .recent-address {
            font-size: 0.875rem;
            color: var(--el-text-color-secondary);
          }
        }
      }
    }
  }

  .error-state,
  .no-location {
    text-align: center;
    padding: 2rem 1rem;

    .error-actions {
      margin-top: 1rem;
    }

    .no-location-icon {
      font-size: 3rem;
      margin-bottom: 1rem;
    }

    h4 {
      margin: 0 0 0.5rem 0;
      color: var(--el-text-color-primary);
    }

    p {
      margin: 0 0 1.5rem 0;
      color: var(--el-text-color-secondary);
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

@media (max-width: 768px) {
  .location-content {
    .location-details {
      grid-template-columns: 1fr;
    }
  }
}
</style>
