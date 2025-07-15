<template>
  <div class="app-page settings-page">
    <div class="page-header">
      <h1 class="page-title">设置</h1>
      <p class="page-subtitle">个性化您的应用体验</p>
    </div>

    <div class="settings-container">
      <!-- 侧边栏 -->
      <div class="settings-sidebar">
        <div class="sidebar-nav">
          <div v-for="tab in tabs" :key="tab.key" class="nav-item" :class="{ active: activeTab === tab.key }"
            @click="activeTab = tab.key">
            <NIcon class="nav-icon" :size="20">
              <component :is="tab.icon" />
            </NIcon>
            <span class="nav-label">{{ tab.label }}</span>
          </div>
        </div>
      </div>

      <!-- 主内容区 -->
      <div class="settings-content">
        <!-- 通用设置 -->
        <div v-if="activeTab === 'general'" class="settings-section">
          <h2 class="section-title">通用设置</h2>
          <div class="settings-grid">
            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Monitor />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">导航栏位置</h3>
                  <p class="setting-description">选择导航栏的显示位置</p>
                </div>
              </div>
              <div class="setting-control">
                <NSelect 
                  v-model:value="settings.navigation.position" 
                  @update:value="handleNavigationPositionChange"
                  class="setting-select" 
                  placement="bottom-start"
                  :options="[
                    { label: '顶部', value: 'top' },
                    { label: '底部', value: 'bottom' }
                  ]" 
                />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Navigate />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">自动隐藏导航栏</h3>
                  <p class="setting-description">滚动时是否自动隐藏导航栏</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch 
                  v-model:value="settings.navigation.alwaysVisible" 
                  @update:value="handleNavigationVisibilityChange" 
                  size="large" 
                />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Moon />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">主题模式</h3>
                  <p class="setting-description">选择应用的外观主题</p>
                </div>
              </div>
              <div class="setting-control">
                <NSelect v-model:value="settings.theme" @update:value="handleThemeChange" class="setting-select"
                  placement="bottom-start" :options="[
                    { label: '跟随系统', value: 'auto' },
                    { label: '浅色', value: 'light' },
                    { label: '深色', value: 'dark' }
                  ]" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Document />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">语言</h3>
                  <p class="setting-description">选择应用显示语言</p>
                </div>
              </div>
              <div class="setting-control">
                <NSelect v-model:value="settings.language" @update:value="handleLanguageChange" class="setting-select"
                  placement="bottom-start" :options="[
                    { label: '简体中文', value: 'zh-CN' },
                    { label: 'English', value: 'en-US' }
                  ]" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <ZoomIn />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">字体大小</h3>
                  <p class="setting-description">调整应用内文字大小</p>
                </div>
              </div>
              <div class="setting-control">
                <NSelect v-model:value="settings.fontSize" @update:value="handleFontSizeChange" class="setting-select"
                  placement="bottom-start" :options="[
                    { label: '小', value: 'small' },
                    { label: '中', value: 'medium' },
                    { label: '大', value: 'large' }
                  ]" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <InformationCircle />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">显示工具提示</h3>
                  <p class="setting-description">在界面元素上显示帮助提示</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.showTooltips" @update:value="handleTooltipsChange" size="large" />
              </div>
            </div>
          </div>
        </div>

        <!-- 通知设置 -->
        <div v-if="activeTab === 'notifications'" class="settings-section">
          <h2 class="section-title">通知设置</h2>
          <div class="settings-grid">
            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <ChatbubbleEllipses />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">私信通知</h3>
                  <p class="setting-description">接收私人消息通知</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.notifications.message_notifications"
                  @update:value="handleMessageNotificationsChange" size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <People />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">群组通知</h3>
                  <p class="setting-description">接收群组消息通知</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.notifications.group_notifications"
                  @update:value="handleGroupNotificationsChange" size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Notifications />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">系统通知</h3>
                  <p class="setting-description">接收系统重要通知</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.notifications.system_notifications"
                  @update:value="handleSystemNotificationsChange" size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Location />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">位置相关通知</h3>
                  <p class="setting-description">接收基于位置的通知</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.notifications.location_notifications"
                  @update:value="handleLocationNotificationsChange" size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <VolumeHigh />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">声音提醒</h3>
                  <p class="setting-description">播放通知提示音</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.notifications.sound_enabled" @update:value="handleSoundChange"
                  size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <PhonePortrait />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">振动提醒</h3>
                  <p class="setting-description">设备振动提醒</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.notifications.vibration_enabled" @update:value="handleVibrationChange"
                  size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Navigate />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">通知半径</h3>
                  <p class="setting-description">设置接收通知的距离范围（公里）</p>
                </div>
              </div>
              <div class="setting-control">
                <NInputNumber v-model:value="settings.notifications.notification_radius_km" size="large" :min="1"
                  :max="100" @update:value="handleNotificationRadiusChange" class="setting-number" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <MoonOutline />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">免打扰时段</h3>
                  <p class="setting-description">设置免打扰时间段</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.notifications.quiet_hours.enabled"
                  @update:value="handleQuietHoursEnable" size="large" />
              </div>
            </div>

            <div v-if="settings.notifications.quiet_hours.enabled" class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Time />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">免打扰时间</h3>
                  <p class="setting-description">选择免打扰的具体时间段</p>
                </div>
              </div>
              <div class="setting-control">
                <div class="time-range-picker">
                  <NTimePicker v-model:value="quietHoursStart" format="HH:mm" class="setting-time-picker" placement="bottom-start"
                    @update:value="handleQuietHoursStartChange" />
                  <span>至</span>
                  <NTimePicker v-model:value="quietHoursEnd" format="HH:mm" class="setting-time-picker" placement="bottom-start"
                    @update:value="handleQuietHoursEndChange" />
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 隐私设置 -->
        <div v-if="activeTab === 'privacy'" class="settings-section">
          <h2 class="section-title">隐私设置</h2>
          <div class="settings-grid">
            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <LocationOutline />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">分享我的位置</h3>
                  <p class="setting-description">允许其他用户看到您的位置信息</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.privacy.location" @update:value="handleLocationPrivacyChange"
                  size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Wifi />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">显示在线状态</h3>
                  <p class="setting-description">让其他用户看到您的在线状态</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.privacy.online" @update:value="handleOnlinePrivacyChange"
                  size="large" />
              </div>
            </div>
          </div>
        </div>

        <!-- 自动刷新 -->
        <div v-if="activeTab === 'autoRefresh'" class="settings-section">
          <h2 class="section-title">自动刷新</h2>
          <div class="settings-grid">
            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Refresh />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">启用自动刷新</h3>
                  <p class="setting-description">自动更新页面内容</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.autoRefresh.enabled" @update:value="handleAutoRefreshEnabledChange"
                  size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Wifi />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">启用实时更新</h3>
                  <p class="setting-description">使用WebSocket进行实时数据更新</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.autoRefresh.websocketEnabled"
                  @update:value="handleWebsocketEnabledChange" size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Timer />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">刷新间隔</h3>
                  <p class="setting-description">设置自动刷新的时间间隔（秒）</p>
                </div>
              </div>
              <div class="setting-control">
                <NInputNumber v-model:value="settings.autoRefresh.interval" size="large" :min="5" :max="300"
                  @update:value="handleIntervalChange" class="setting-number" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Location />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">位置变化时刷新</h3>
                  <p class="setting-description">当位置发生变化时自动刷新</p>
                </div>
              </div>
              <div class="setting-control">
                <NSwitch v-model:value="settings.autoRefresh.locationCheck" @update:value="handleLocationCheckChange"
                  size="large" />
              </div>
            </div>

            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <Navigate />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">位置变化阈值</h3>
                  <p class="setting-description">触发刷新的位置变化距离（米）</p>
                </div>
              </div>
              <div class="setting-control">
                <NInputNumber v-model:value="settings.autoRefresh.locationThreshold" size="large" :min="10" :max="1000"
                  @update:value="handleLocationThresholdChange" class="setting-number" />
              </div>
            </div>
          </div>
        </div>

        <!-- 操作 -->
        <div v-if="activeTab === 'actions'" class="settings-section">
          <h2 class="section-title">操作</h2>
          <div class="settings-grid">
            <div class="setting-card">
              <div class="setting-header">
                <NIcon class="setting-icon" size="20">
                  <RefreshOutline />
                </NIcon>
                <div class="setting-info">
                  <h3 class="setting-title">恢复默认设置</h3>
                  <p class="setting-description">将所有设置恢复为默认值</p>
                </div>
              </div>
              <div class="setting-control">
                <NButton type="error" @click="handleResetToDefaults" size="large">
                  重置设置
                </NButton>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// Vue 3 Composition API
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'

// Naive UI 组件
import { NSelect, NSwitch, NButton, NInputNumber, NTimePicker, NIcon, useMessage } from 'naive-ui'

// 图标
import {
  Settings, Notifications, LockClosed, Refresh, Build,
  Desktop as Monitor, Moon, Document, Search as ZoomIn, InformationCircle,
  ChatbubbleEllipses, People, Location, VolumeHigh, PhonePortrait,
  Navigate, MoonOutline, Time, LocationOutline, Wifi, Timer, RefreshOutline
} from '@vicons/ionicons5'

// 项目 composables 和 stores
import { useAppSettings, type AppSettings } from '@/composables/useAppSettings'

// 页面元数据
definePageMeta({
  layout: 'default',
  title: '设置',
  middleware: ['auth'],
  ssr: false // 禁用 SSR，避免 Naive UI 组件的 SSR 问题
})

// 侧边栏标签配置
const tabs = [
  { key: 'general', label: '通用', icon: Settings },
  { key: 'notifications', label: '通知', icon: Notifications },
  { key: 'privacy', label: '隐私', icon: LockClosed },
  { key: 'autoRefresh', label: '自动刷新', icon: Refresh },
  { key: 'actions', label: '操作', icon: Build }
]

// 响应式状态
const activeTab = ref('general')
const { 
  settings: globalSettings, 
  updateSetting, 
  resetSettings, 
  isLoaded, 
  loadSettings,
  initializeSettings,
  setupSettingsWatchers,
  playNotificationSound,
  triggerVibration
} = useAppSettings()

// 直接使用全局设置，不创建副本，确保响应式
const settings = globalSettings

// Naive UI 消息实例
const message = useMessage()

// 免打扰时段选择器的状态
const quietHoursStart = ref<number | null>(null)
const quietHoursEnd = ref<number | null>(null)

// 当设置加载或变更时，同步免打扰时段选择器
const syncQuietHoursRange = () => {
  const { start_time, end_time } = settings.value.notifications.quiet_hours
  if (start_time && end_time) {
    const startDate = new Date()
    const [startHour, startMinute] = start_time.split(':')
    startDate.setHours(Number(startHour), Number(startMinute), 0)
    quietHoursStart.value = startDate.getTime()

    const endDate = new Date()
    const [endHour, endMinute] = end_time.split(':')
    endDate.setHours(Number(endHour), Number(endMinute), 0)
    quietHoursEnd.value = endDate.getTime()
  } else {
    // 设置默认值：22:00 - 08:00
    const defaultStart = new Date()
    defaultStart.setHours(22, 0, 0)
    quietHoursStart.value = defaultStart.getTime()

    const defaultEnd = new Date()
    defaultEnd.setHours(8, 0, 0)
    quietHoursEnd.value = defaultEnd.getTime()
  }
}

// 监听全局设置变化
const stopWatch = watch(globalSettings, () => {
  syncQuietHoursRange()
}, { deep: true })

// 生命周期钩子
onMounted(async () => {
  // 确保设置已加载
  if (!isLoaded.value) {
    await loadSettings()
  }
  syncQuietHoursRange()

  // 初始化设置
  initializeSettings()
  setupSettingsWatchers()
})

onUnmounted(() => {
  stopWatch()
})

// 设置处理方法
const handleNavigationPositionChange = (val: 'top' | 'bottom') => {
  if (process.dev) {
    console.log('导航位置变更:', val)
  }
  updateSetting('navigation', { ...settings.value.navigation, position: val })
  message.success('导航位置已更新')
}

const handleNavigationVisibilityChange = (val: boolean) => {
  if (process.dev) {
    console.log('导航可见性变更:', val)
  }
  updateSetting('navigation', { ...settings.value.navigation, alwaysVisible: val })
  message.success('导航显示设置已更新')
}

const handleThemeChange = (val: 'auto' | 'light' | 'dark') => {
  if (process.dev) {
    console.log('主题变更:', val)
  }
  updateSetting('theme', val)
  message.success('主题已更新')
}

const handleLanguageChange = (val: 'zh-CN' | 'en-US') => {
  if (process.dev) {
    console.log('语言变更:', val)
  }
  updateSetting('language', val)
  message.success('语言已更新')
}

const handleFontSizeChange = (val: 'small' | 'medium' | 'large') => {
  if (process.dev) {
    console.log('字体大小变更:', val)
  }
  updateSetting('fontSize', val)
  message.success('字体大小已更新')
}

const handleTooltipsChange = (val: boolean) => {
  if (process.dev) {
    console.log('工具提示变更:', val)
  }
  updateSetting('showTooltips', val)
  message.success('工具提示设置已更新')
}

const handleMessageNotificationsChange = (val: boolean) => {
  if (process.dev) {
    console.log('私信通知变更:', val)
  }
  const newNotificationSettings = { ...settings.value.notifications, message_notifications: val }
  updateSetting('notifications', newNotificationSettings)
  message.success('私信通知设置已更新')
}

const handleGroupNotificationsChange = (val: boolean) => {
  if (process.dev) {
    console.log('群组通知变更:', val)
  }
  const newNotificationSettings = { ...settings.value.notifications, group_notifications: val }
  updateSetting('notifications', newNotificationSettings)
  message.success('群组通知设置已更新')
}

const handleSystemNotificationsChange = (val: boolean) => {
  if (process.dev) {
    console.log('系统通知变更:', val)
  }
  const newNotificationSettings = { ...settings.value.notifications, system_notifications: val }
  updateSetting('notifications', newNotificationSettings)
  message.success('系统通知设置已更新')
}

const handleLocationNotificationsChange = (val: boolean) => {
  if (process.dev) {
    console.log('位置通知变更:', val)
  }
  const newNotificationSettings = { ...settings.value.notifications, location_notifications: val }
  updateSetting('notifications', newNotificationSettings)
  message.success('位置通知设置已更新')
}

const handleSoundChange = (val: boolean) => {
  if (process.dev) {
    console.log('声音提醒变更:', val)
  }
  const newNotificationSettings = { ...settings.value.notifications, sound_enabled: val }
  updateSetting('notifications', newNotificationSettings)
  message.success('声音提醒设置已更新')
  
  // 播放测试音效
  if (val) {
    playNotificationSound()
  }
}

const handleVibrationChange = (val: boolean) => {
  if (process.dev) {
    console.log('振动提醒变更:', val)
  }
  const newNotificationSettings = { ...settings.value.notifications, vibration_enabled: val }
  updateSetting('notifications', newNotificationSettings)
  message.success('振动提醒设置已更新')
  
  // 测试振动
  if (val) {
    triggerVibration()
  }
}

const handleNotificationRadiusChange = (val: number | null) => {
  console.log('通知半径变更:', val)
  const newNotificationSettings = { ...settings.value.notifications, notification_radius_km: val || 1 }
  updateSetting('notifications', newNotificationSettings)
  message.success('通知半径已更新')
}

const handleLocationPrivacyChange = (val: boolean) => {
  console.log('位置隐私变更:', val)
  updateSetting('privacy', { ...settings.value.privacy, location: val })
  message.success('位置隐私设置已更新')
}

const handleOnlinePrivacyChange = (val: boolean) => {
  console.log('在线状态隐私变更:', val)
  updateSetting('privacy', { ...settings.value.privacy, online: val })
  message.success('在线状态设置已更新')
}

const handleAutoRefreshEnabledChange = (val: boolean) => {
  console.log('自动刷新变更:', val)
  updateSetting('autoRefresh', { ...settings.value.autoRefresh, enabled: val })
  message.success('自动刷新设置已更新')
}

const handleWebsocketEnabledChange = (val: boolean) => {
  console.log('WebSocket变更:', val)
  updateSetting('autoRefresh', { ...settings.value.autoRefresh, websocketEnabled: val })
  message.success('实时更新设置已更新')
}

const handleIntervalChange = (val: number | null) => {
  console.log('刷新间隔变更:', val)
  updateSetting('autoRefresh', { ...settings.value.autoRefresh, interval: val || 30 })
  message.success('刷新间隔已更新')
}

const handleLocationCheckChange = (val: boolean) => {
  console.log('位置变化刷新变更:', val)
  updateSetting('autoRefresh', { ...settings.value.autoRefresh, locationCheck: val })
  message.success('位置变化刷新设置已更新')
}

const handleLocationThresholdChange = (val: number | null) => {
  console.log('位置变化阈值变更:', val)
  updateSetting('autoRefresh', { ...settings.value.autoRefresh, locationThreshold: val || 100 })
  message.success('位置变化阈值已更新')
}

const handleQuietHoursEnable = (val: boolean) => {
  console.log('免打扰时段变更:', val)
  const newQuietHours = { ...settings.value.notifications.quiet_hours, enabled: val }
  const newNotificationSettings = { ...settings.value.notifications, quiet_hours: newQuietHours }
  updateSetting('notifications', newNotificationSettings)
  message.success('免打扰时段设置已更新')
}

const handleQuietHoursTime = (newRange: [number, number] | null) => {
  if (newRange && newRange.length === 2) {
    const formatTime = (timestamp: number) => {
      const date = new Date(timestamp)
      return date.toTimeString().slice(0, 5)
    }
    const newQuietHours = {
      ...settings.value.notifications.quiet_hours,
      start_time: formatTime(newRange[0]),
      end_time: formatTime(newRange[1]),
    }
    console.log('免打扰时间变更:', newQuietHours)
    const newNotificationSettings = { ...settings.value.notifications, quiet_hours: newQuietHours }
    updateSetting('notifications', newNotificationSettings)
    message.success('免打扰时间已更新')
  }
}

// 处理单个时间选择器的变化
const handleQuietHoursStartChange = (value: number | null) => {
  if (value && quietHoursEnd.value) {
    const formatTime = (timestamp: number) => {
      const date = new Date(timestamp)
      return date.toTimeString().slice(0, 5)
    }
    const newQuietHours = {
      ...settings.value.notifications.quiet_hours,
      start_time: formatTime(value),
      end_time: formatTime(quietHoursEnd.value),
    }
    console.log('免打扰开始时间变更:', newQuietHours)
    const newNotificationSettings = { ...settings.value.notifications, quiet_hours: newQuietHours }
    updateSetting('notifications', newNotificationSettings)
    message.success('免打扰时间已更新')
  }
}

const handleQuietHoursEndChange = (value: number | null) => {
  if (value && quietHoursStart.value) {
    const formatTime = (timestamp: number) => {
      const date = new Date(timestamp)
      return date.toTimeString().slice(0, 5)
    }
    const newQuietHours = {
      ...settings.value.notifications.quiet_hours,
      start_time: formatTime(quietHoursStart.value),
      end_time: formatTime(value),
    }
    console.log('免打扰结束时间变更:', newQuietHours)
    const newNotificationSettings = { ...settings.value.notifications, quiet_hours: newQuietHours }
    updateSetting('notifications', newNotificationSettings)
    message.success('免打扰时间已更新')
  }
}

const handleResetToDefaults = () => {
  console.log('重置设置')
  resetSettings()
  message.success('已恢复为默认设置')
}
</script>

<style lang="scss" scoped>
// 使用项目统一的设计系统
.settings-page {
  background: transparent;
  min-height: 100vh;
  padding-bottom: 2rem;
}

.settings-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 1rem;
  display: flex;
  gap: 2rem;

  @media (max-width: 768px) {
    flex-direction: column;
    gap: 1rem;
    padding: 0 0.5rem;
  }
}

.settings-sidebar {
  width: 280px;
  background: var(--bg-color, white);
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 12px;
  padding: 1.5rem 0;
  height: fit-content;
  position: sticky;
  top: 2rem;
  box-shadow: var(--shadow-sm, 0 4px 6px -1px rgba(0, 0, 0, 0.1));

  @media (max-width: 768px) {
    width: 100%;
    position: static;
    padding: 1rem 0;
  }

  .sidebar-nav {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;

    @media (max-width: 768px) {
      flex-direction: row;
      overflow-x: auto;
      padding: 0 1rem;
      gap: 0;

      &::-webkit-scrollbar {
        display: none;
      }

      scrollbar-width: none;
    }

    .nav-item {
      display: flex;
      align-items: center;
      padding: 1rem 1.5rem;
      cursor: pointer;
      transition: all 0.2s ease;
      border-radius: 0;
      position: relative;
      color: var(--text-color-primary, #0f172a);

      @media (max-width: 768px) {
        flex-shrink: 0;
        padding: 0.75rem 1rem;
        min-width: 120px;
        justify-content: center;
        border-radius: 8px;
        margin: 0 0.25rem;
      }

      &:hover {
        background: var(--color-primary-light-9, #fff7ed);
        color: var(--color-primary, #f97316);
      }

      &.active {
        background: var(--color-primary-light-9, #fff7ed);
        color: var(--color-primary, #f97316);
        font-weight: 500;

        &::before {
          content: '';
          position: absolute;
          left: 0;
          top: 0;
          bottom: 0;
          width: 3px;
          background: var(--color-primary, #f97316);

          @media (max-width: 768px) {
            left: 0;
            right: 0;
            top: auto;
            bottom: 0;
            width: auto;
            height: 3px;
          }
        }
      }

      .nav-icon {
        margin-right: 0.75rem;
        flex-shrink: 0;

        @media (max-width: 768px) {
          margin-right: 0.5rem;
        }
      }

      .nav-label {
        font-size: 0.95rem;

        @media (max-width: 768px) {
          font-size: 0.875rem;
        }
      }
    }
  }
}

.settings-content {
  flex: 1;
  min-width: 0;
}

.settings-section {
  .section-title {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text-color-primary, #0f172a);
    margin: 0 0 1.5rem 0;
  }
}

.settings-grid {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.setting-card {
  background: var(--bg-color, white);
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 12px;
  padding: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  transition: all 0.2s ease;
  box-shadow: var(--shadow-xs, 0 2px 4px rgba(0, 0, 0, 0.05));
  position: relative;
  z-index: var(--z-content);

  &:hover {
    border-color: var(--color-primary-light-7, #fdba74);
    box-shadow: var(--shadow-sm, 0 4px 8px rgba(0, 0, 0, 0.1));
    transform: translateY(-2px);
    z-index: var(--z-elevated);
  }

  // 当设置卡片包含打开的下拉框时，提高z-index
  &:has(.n-base-select--active),
  &:has(.n-base-select--focused) {
    z-index: var(--z-dropdown);
  }

  @media (max-width: 768px) {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
    padding: 1rem;
  }

  .setting-header {
    display: flex;
    align-items: center;
    flex: 1;
    min-width: 0;

    @media (max-width: 768px) {
      width: 100%;
    }

    .setting-icon {
      color: var(--color-primary, #f97316);
      margin-right: 1rem;
      flex-shrink: 0;
    }

    .setting-info {
      flex: 1;
      min-width: 0;

      .setting-title {
        font-size: 1rem;
        font-weight: 500;
        color: var(--text-color-primary, #0f172a);
        margin: 0 0 0.25rem 0;
      }

      .setting-description {
        font-size: 0.875rem;
        color: var(--text-color-secondary, #475569);
        margin: 0;
        line-height: 1.4;
      }
    }
  }

  .setting-control {
    flex-shrink: 0;
    margin-left: 1rem;

    @media (max-width: 768px) {
      width: 100%;
      margin-left: 0;
      display: flex;
      justify-content: flex-end;
    }
  }
}

// 设置控件样式
.setting-select {
  min-width: 140px;

  @media (max-width: 768px) {
    width: 100%;
    max-width: 200px;
  }

  // 确保下拉菜单不被遮挡
  :deep(.n-base-select-menu) {
    z-index: var(--z-emergency) !important;
  }

  :deep(.n-popover) {
    z-index: var(--z-emergency) !important;
  }
}

.setting-number {
  width: 120px;

  @media (max-width: 768px) {
    width: 100%;
    max-width: 200px;
  }
}

.setting-time-picker {
  width: 120px;

  @media (max-width: 768px) {
    width: 100%;
    max-width: 140px;
  }
}

.time-range-picker {
  display: flex;
  align-items: center;
  gap: 0.75rem;

  @media (max-width: 768px) {
    width: 100%;
    justify-content: flex-end;
  }

  span {
    color: var(--text-color-secondary, #475569);
    font-size: 0.875rem;
    white-space: nowrap;
  }
}

// 深色模式适配
@media (prefers-color-scheme: dark) {

  .settings-sidebar,
  .setting-card {
    background: var(--color-bg-elevated, #1e293b);
    border-color: var(--color-border-primary, #334155);
  }

  .setting-card {
    box-shadow: var(--shadow-xs, 0 2px 4px rgba(0, 0, 0, 0.2));

    &:hover {
      box-shadow: var(--shadow-sm, 0 4px 8px rgba(0, 0, 0, 0.3));
    }
  }
}
</style>