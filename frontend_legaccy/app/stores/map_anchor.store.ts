import { defineStore } from 'pinia'
import { DEFAULTS } from '~/config/defaults'

export interface MapAnchor {
  latitude: number
  longitude: number
  /** 浏览锚点名称（可为空，展示时可回退到经纬度） */
  name?: string
  updated_at: number
}

export const useMapAnchorStore = defineStore('map_anchor', () => {
  const anchor = ref<MapAnchor>({
    latitude: DEFAULTS.latitude,
    longitude: DEFAULTS.longitude,
    name: DEFAULTS.locationName,
    updated_at: Date.now()
  })

  const anchorName = computed(() => anchor.value.name || '')

  const setAnchor = (latitude: number, longitude: number, name?: string) => {
    anchor.value = {
      latitude,
      longitude,
      name,
      updated_at: Date.now()
    }
  }

  const setAnchorName = (name: string) => {
    anchor.value = {
      ...anchor.value,
      name,
      updated_at: Date.now()
    }
  }

  return {
    anchor,
    anchorName,
    setAnchor,
    setAnchorName
  }
}, {
  // 浏览锚点属于“偏好”，需要持久化（避免刷新丢失）
  persist: true
})

