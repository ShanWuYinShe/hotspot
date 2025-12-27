<template>
  <UModal
    :fullscreen="isMobile"
    width="xl"
    :padding="false"
    :title="title"
    :ui="{ 
      content: 'h-[85vh] sm:h-[80vh] flex flex-col', 
      body: 'flex-1 p-0 overflow-hidden relative flex flex-col',
      close: 'sm:flex'
    }"
  >
    <template #body="{ close }">
      <!-- 地图容器 -->
      <div class="relative flex-1 min-h-0 hs-map-background">
        <MapContainer
          :center="center"
          :zoom="zoom"
          :markers="markers"
          :interactive="true"
          :show-anchor="true"
          @viewport-change="handleViewportChange"
          @marker-click="handleMarkerClick"
        />

        <!-- 顶部控制栏 (悬浮) -->
        <div class="absolute top-4 left-4 right-4 z-20 flex justify-between items-start pointer-events-none">
          <!-- 模式切换 -->
          <div class="pointer-events-auto flex flex-col gap-2">
            <UCard variant="outline" class="hs-floating-control" :ui="{ body: 'p-1 flex items-center gap-1' }">
              <UTooltip text="显示附近的人">
                <UButton
                  :variant="modes.includes('users') ? 'soft' : 'ghost'"
                  :color="modes.includes('users') ? 'primary' : 'neutral'"
                  size="sm"
                  icon="i-lucide-users"
                  @click="toggleMode('users')"
                />
              </UTooltip>
              <UTooltip text="显示附近群组">
                <UButton
                  :variant="modes.includes('groups') ? 'soft' : 'ghost'"
                  :color="modes.includes('groups') ? 'primary' : 'neutral'"
                  size="sm"
                  icon="i-lucide-users-round"
                  @click="toggleMode('groups')"
                />
              </UTooltip>
              <UTooltip text="显示附近动态">
                <UButton
                  :variant="modes.includes('moments') ? 'soft' : 'ghost'"
                  :color="modes.includes('moments') ? 'primary' : 'neutral'"
                  size="sm"
                  icon="i-lucide-image"
                  @click="toggleMode('moments')"
                />
              </UTooltip>
            </UCard>
          </div>

          <!-- 右侧工具 -->
          <div class="pointer-events-auto flex flex-col gap-2">
             <UButton
              color="white"
              variant="solid"
              icon="i-lucide-refresh-cw"
              :loading="loading"
              size="sm"
              class="hs-floating-control"
              @click="refresh"
            />
            <UButton
              color="white"
              variant="solid"
              icon="i-lucide-crosshair"
              :loading="locating"
              size="sm"
              class="hs-floating-control"
              @click="locateMe"
            />
          </div>
        </div>

        <!-- 底部位置确认栏 (悬浮) -->
        <div class="absolute bottom-6 left-4 right-4 z-20 pointer-events-none">
          <div class="pointer-events-auto hs-floating-panel p-3 flex flex-col sm:flex-row items-center gap-3 max-w-2xl mx-auto">
            <UChip color="primary" size="xl" class="shrink-0">
              <UIcon name="i-lucide-map-pin" />
            </UChip>
            
            <div class="flex-1 min-w-0 text-center sm:text-left">
              <UBadge variant="subtle" color="neutral" size="md" class="w-full truncate">
                {{ selectedLocationName || '拖动地图选择位置' }}
              </UBadge>
              <UBadge variant="subtle" color="neutral" size="xs" class="font-mono mt-1">
                {{ center.latitude.toFixed(6) }}, {{ center.longitude.toFixed(6) }}
              </UBadge>
            </div>

            <UButton
              size="md"
              class="w-full sm:w-auto min-w-[100px]"
              :disabled="!selectedLocationName"
              @click="confirmLocation(close)"
            >
              设为锚点
            </UButton>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import MapContainer, { type MapMarker } from '~/components/map/MapContainer.vue'
import { DEFAULTS as defaults } from '~/config/defaults'
import { geocoding } from '~/utils/geocoding'
import { useMapService } from '~/services/map.service'
import { useGeolocation } from '~/composables/useGeolocation'
import { useBreakpoints, breakpointsTailwind } from '@vueuse/core'
import type { Bounds, GroupSearchItem, MomentSearchItem, UserSearchItem } from '~/types/api'

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
  title: '地图探索'
})

// 响应式断点
const breakpoints = useBreakpoints(breakpointsTailwind)
const isMobile = breakpoints.smaller('sm')

// 依赖
const anchorStore = useMapAnchorStore()
const mapService = useMapService()
const geolocation = useGeolocation()
const { withErrorHandling } = useErrorHandler()
const feedback = useFeedbackStore()

// 状态
const loading = ref(false)
const locating = ref(false)
const zoom = ref(15)
const center = reactive({
  latitude: defaults.latitude,
  longitude: defaults.longitude
})
const selectedLocationName = ref('')

// 搜索模式（支持多选）
type Mode = 'users' | 'groups' | 'moments'
const modes = ref<Mode[]>(['users', 'groups', 'moments'])

// 搜索结果
const usersResults = ref<UserSearchItem[]>([])
const groupsResults = ref<GroupSearchItem[]>([])
const momentsResults = ref<MomentSearchItem[]>([])
const lastBounds = ref<Bounds | null>(null)

// 计算属性：标记点
const markers = computed<MapMarker[]>(() => {
  const list: MapMarker[] = []

  if (modes.value.includes('users')) {
    list.push(...usersResults.value
      .filter(u => !!u.geo)
      .map(u => ({
        id: `u:${u.user_id}`,
        type: 'user' as const,
        title: u.nickname,
        description: u.geo?.location_name || '未知位置',
        latitude: u.geo!.latitude,
        longitude: u.geo!.longitude,
        avatar: u.avatar_url
      }))
    )
  }

  if (modes.value.includes('groups')) {
    list.push(...groupsResults.value
      .filter(g => !!g.geo)
      .map(g => ({
        id: `g:${g.id}`,
        type: 'group' as const,
        title: g.name,
        description: g.geo?.location_name || '未知位置',
        latitude: g.geo!.latitude,
        longitude: g.geo!.longitude,
        avatar: g.avatar_url
      }))
    )
  }

  if (modes.value.includes('moments')) {
    list.push(...momentsResults.value
      .filter(m => !!m.geo)
      .map(m => ({
        id: `m:${m.id}`,
        type: 'moment' as const,
        title: m.user?.nickname || '动态',
        description: m.description || '查看详情',
        latitude: m.geo!.latitude,
        longitude: m.geo!.longitude,
        avatar: m.user?.avatar_url
      }))
    )
  }

  return list
})

// 初始化：从 Store 恢复位置
const hydratedAnchor = useHydratedStore(anchorStore, 'anchor', {
  latitude: defaults.latitude,
  longitude: defaults.longitude,
  name: ''
})

watch(
  hydratedAnchor,
  (val) => {
    // 只有当距离变化显著时才更新 center（避免拖拽时的微小抖动循环）
    if (
      Math.abs(val.latitude - center.latitude) > 0.0001 ||
      Math.abs(val.longitude - center.longitude) > 0.0001
    ) {
      center.latitude = val.latitude
      center.longitude = val.longitude
      selectedLocationName.value = val.name || ''
    }
  },
  { deep: true, immediate: true }
)

// 模式切换
const toggleMode = (mode: Mode) => {
  const index = modes.value.indexOf(mode)
  if (index > -1) {
    modes.value.splice(index, 1)
  } else {
    modes.value.push(mode)
  }
  refresh()
}

// 刷新数据
const refresh = async () => {
  if (!lastBounds.value) return
  
  loading.value = true
  await withErrorHandling(async () => {
    const req = {
      bounds: lastBounds.value!,
      latitude: center.latitude,
      longitude: center.longitude,
      limit: 50 // 限制数量防止地图过载
    }

    const promises = []
    
    if (modes.value.includes('users')) {
      promises.push(mapService.searchUsersByBounds(req).then(res => {
        if (res.type === 'users') usersResults.value = res.items
      }))
    } else {
      usersResults.value = []
    }

    if (modes.value.includes('groups')) {
      promises.push(mapService.searchGroupsByBounds(req).then(res => {
        if (res.type === 'groups') groupsResults.value = res.items
      }))
    } else {
      groupsResults.value = []
    }

    if (modes.value.includes('moments')) {
      promises.push(mapService.searchMomentsByBounds(req).then(res => {
        if (res.type === 'moments') momentsResults.value = res.items
      }))
    } else {
      momentsResults.value = []
    }

    await Promise.all(promises)
  }, {
    loading,
    showToast: false // 静默刷新
  })
  loading.value = false
}

const refreshDebounced = useDebounceFn(refresh, 500)

// 视口变化处理（拖拽/缩放）
const handleViewportChange = async (payload: { center: { latitude: number; longitude: number }; zoom: number; bounds: Bounds }) => {
  // 1. 更新本地状态
  center.latitude = payload.center.latitude
  center.longitude = payload.center.longitude
  zoom.value = payload.zoom
  lastBounds.value = payload.bounds

  // 2. 逆地理编码更新名称
  selectedLocationName.value = '' // 清空表示正在加载
  try {
    const name = await geocoding.getLocationName(center.latitude, center.longitude)
    selectedLocationName.value = name
  } catch {
    selectedLocationName.value = `${center.latitude.toFixed(4)}, ${center.longitude.toFixed(4)}`
  }

  // 3. 刷新附近数据
  refreshDebounced()
}

// 标记点击
const handleMarkerClick = (marker: MapMarker) => {
  // 根据类型跳转详情
  const [type, id] = marker.id.split(':')
  if (type === 'u') navigateTo(`/users/${id}`)
  if (type === 'g') navigateTo(`/groups/${id}`)
  if (type === 'm') navigateTo(`/moments/${id}`)
}

// 定位到我
const locateMe = async () => {
  locating.value = true
  try {
    const pos = await geolocation.getPosition({ silent: false })
    if (pos) {
      center.latitude = pos.coords.latitude
      center.longitude = pos.coords.longitude
      // 触发一次视口更新会联动刷新数据
    }
  } catch {
    feedback.error('定位失败')
  } finally {
    locating.value = false
  }
}

// 确认选择
const confirmLocation = (close: () => void) => {
  if (!selectedLocationName.value) return

  // 更新全局锚点
  anchorStore.setAnchor(center.latitude, center.longitude, selectedLocationName.value)
  
  // 回调
  props.onSelect?.({
    latitude: center.latitude,
    longitude: center.longitude,
    locationName: selectedLocationName.value
  })
  
  feedback.success('浏览锚点已更新', selectedLocationName.value)
  close()
}
</script>
