<template>
  <!-- 现代化的导航栏设计 - 使用 Nuxt UI v3 组件 -->
  <UContainer>
    <!-- 桌面端导航 -->
    <nav class="hidden lg:block">
      <div class="flex items-center justify-between py-4">
        <!-- 左侧 Logo -->
        <NuxtLink to="/" class="flex items-center gap-3 hover:opacity-90 transition-opacity">
          <div
            class="w-10 h-10 rounded-xl bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center shadow-lg">
            <UIcon name="i-heroicons-fire" class="w-6 h-6 text-white" />
          </div>
          <span class="text-2xl font-bold bg-gradient-to-r from-orange-600 to-red-600 bg-clip-text text-transparent">
            HotSpot
          </span>
        </NuxtLink>

        <!-- 中央主导航 -->
        <div class="flex items-center">
          <UNavigationMenu 
            :items="navigationItems" 
            color="primary" 
            variant="pill" 
            highlight
            orientation="horizontal"
            class="mx-8 flex items-center gap-1"
          />
        </div>

        <!-- 右侧工具栏 -->
        <div class="flex items-center gap-2">
          <!-- 搜索按钮 -->
          <UTooltip text="全局搜索 (⌘K)" :kbds="['meta', 'k']">
            <UButton variant="ghost" color="neutral" icon="i-heroicons-magnifying-glass" size="lg" square
              class="hover:bg-neutral-100 dark:hover:bg-neutral-800" @click="openSearch" />
          </UTooltip>

          <!-- 消息按钮 -->
          <UButton to="/chat" variant="ghost" color="neutral" size="lg" square
            class="relative hover:bg-neutral-100 dark:hover:bg-neutral-800 transition-all duration-200">
            <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5" />
            <UBadge v-if="unreadCount > 0" size="xs" color="error"
              class="absolute -top-1 -right-1 min-w-5 h-5 flex items-center justify-center animate-pulse">
              {{ unreadCount > 99 ? '99+' : unreadCount }}
            </UBadge>
          </UButton>

          <!-- 通知按钮 -->
          <UTooltip text="查看通知" :kbds="['alt', 'n']">
            <UButton variant="ghost" color="neutral" size="lg" square
              class="relative hover:bg-neutral-100 dark:hover:bg-neutral-800" @click="toggleNotifications">
              <UIcon name="i-heroicons-bell" class="w-5 h-5" />
              <UBadge v-if="hasSystemAlerts" size="xs" color="error" class="absolute -top-1 -right-1 w-3 h-3" />
            </UButton>
          </UTooltip>

          <!-- 语言切换 -->
          <UTooltip text="切换语言 (Alt+L)" :kbds="['alt', 'l']">
            <USelectMenu 
              v-model="selectedLocaleObject" 
              :items="localeOptions" 
              size="lg" 
              class="min-w-32" 
              :loading="switchingLocale" 
              :disabled="switchingLocale"
              placeholder="选择语言"
              @update:model-value="handleLocaleChange">
              <template #leading>
                <UIcon :name="switchingLocale ? 'i-heroicons-arrow-path' : 'i-heroicons-language'" class="w-4 h-4"
                  :class="{ 'animate-spin': switchingLocale }" />
              </template>
            </USelectMenu>
          </UTooltip>

          <!-- 主题切换 -->
          <ThemeToggle size="lg" />

          <!-- 用户菜单 -->
          <UserMenu />
        </div>
      </div>
    </nav>

    <!-- 平板端导航 -->
    <nav class="hidden md:block lg:hidden">
      <div class="flex items-center justify-between py-3">
        <!-- Logo -->
        <NuxtLink to="/" class="flex items-center gap-2">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center">
            <UIcon name="i-heroicons-fire" class="w-5 h-5 text-white" />
          </div>
          <span class="text-lg font-bold bg-gradient-to-r from-orange-600 to-red-600 bg-clip-text text-transparent">
            HotSpot
          </span>
        </NuxtLink>

        <!-- 右侧按钮组 -->
        <div class="flex items-center gap-1">
          <!-- 搜索按钮 -->
          <UTooltip text="搜索 (⌘K)" :kbds="['meta', 'k']">
            <UButton variant="ghost" icon="i-heroicons-magnifying-glass" size="md" square @click="openSearch" />
          </UTooltip>

          <!-- 消息按钮 -->
          <UTooltip text="消息">
            <UButton to="/chat" variant="ghost" size="md" square class="relative">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="w-4 h-4" />
              <UBadge v-if="unreadCount > 0" size="xs" color="error" class="absolute -top-0.5 -right-0.5">
                {{ unreadCount > 9 ? '9+' : unreadCount }}
              </UBadge>
            </UButton>
          </UTooltip>

          <!-- 导航菜单 - 使用 UPopover 处理 hover -->
          <UPopover mode="hover" :popper="{ placement: 'bottom-end' }">
            <UTooltip text="导航菜单">
              <UButton 
                variant="ghost"
                icon="i-heroicons-bars-3" 
                size="md" 
                square 
              />
            </UTooltip>
            
            <template #panel>
              <UCard class="w-72">
                <template #body>
                  <UNavigationMenu 
                    :items="navigationItems" 
                    color="primary" 
                    variant="pill" 
                    orientation="vertical"
                    class="w-full flex flex-col gap-2"
                  />
                </template>
              </UCard>
            </template>
          </UPopover>
        </div>
      </div>


    </nav>

    <!-- 移动端无顶部导航，使用底部导航栏 -->
  </UContainer>

  <!-- 移动端底部导航栏 - 整合所有功能 -->
  <div
    class="fixed bottom-0 left-0 right-0 md:hidden bg-white/95 dark:bg-neutral-900/95 backdrop-blur-lg border-t border-neutral-200/50 dark:border-neutral-700/50 z-40">
    <UContainer>
      <div class="flex items-center justify-around py-2">
        <!-- 首页 -->
        <UTooltip text="首页 (⌘H)" :kbds="['meta', 'h']" :content="{ side: 'top' }">
          <UButton to="/" variant="ghost" size="lg" square :color="$route.path === '/' ? 'primary' : 'neutral'"
            class="flex flex-col items-center gap-1 py-2">
            <UIcon name="i-heroicons-home" class="w-5 h-5" />
            <span class="text-xs">{{ t('nav.home') }}</span>
          </UButton>
        </UTooltip>

        <!-- 我的群组 -->
        <UTooltip text="我的群组 (⌘G)" :kbds="['meta', 'g']" :content="{ side: 'top' }">
          <UButton to="/groups" variant="ghost" size="lg" square
            :color="$route.path.startsWith('/groups') ? 'primary' : 'neutral'"
            class="flex flex-col items-center gap-1 py-2">
            <UIcon name="i-heroicons-user-group" class="w-5 h-5" />
            <span class="text-xs">{{ t('nav.groups') }}</span>
          </UButton>
        </UTooltip>

        <!-- 地图探索 -->
        <UTooltip text="地图探索 (⌘M)" :kbds="['meta', 'm']" :content="{ side: 'top' }">
          <UButton to="/map" variant="ghost" size="lg" square :color="$route.path === '/map' ? 'primary' : 'neutral'"
            class="flex flex-col items-center gap-1 py-2">
            <UIcon name="i-heroicons-map" class="w-5 h-5" />
            <span class="text-xs">{{ t('nav.map') }}</span>
          </UButton>
        </UTooltip>

        <!-- 消息聊天 -->
        <UTooltip text="消息聊天 (⌘C)" :kbds="['meta', 'c']" :content="{ side: 'top' }">
          <UButton to="/chat" variant="ghost" size="lg" square
            :color="$route.path.startsWith('/chat') ? 'primary' : 'neutral'"
            class="flex flex-col items-center gap-1 py-2 relative">
            <UIcon name="i-heroicons-chat-bubble-left-right" class="w-5 h-5" />
            <span class="text-xs">{{ t('nav.messages') }}</span>
            <UBadge v-if="unreadCount > 0" size="xs" color="error"
              class="absolute top-1 right-2 min-w-4 h-4 flex items-center justify-center">
              {{ unreadCount > 9 ? '9+' : unreadCount }}
            </UBadge>
          </UButton>
        </UTooltip>

        <!-- 更多菜单 -->
        <UTooltip text="更多功能" :content="{ side: 'top' }">
          <UButton variant="ghost" size="lg" square color="neutral"
            class="flex flex-col items-center gap-1 py-2 relative" @click="mobileDrawerOpen = true">
            <UIcon name="i-heroicons-ellipsis-horizontal" class="w-5 h-5" />
            <span class="text-xs">更多</span>
            <UBadge v-if="hasSystemAlerts" size="xs" color="error" class="absolute top-1 right-2 w-2 h-2" />
          </UButton>
        </UTooltip>
      </div>
    </UContainer>
  </div>

  <!-- 移动端导航抽屉 -->
  <MobileDrawer v-model:open="mobileDrawerOpen" @toggle-notifications="toggleNotifications" />

  <!-- 通知面板 -->
  <div 
    v-if="notificationPanelOpen"
    class="fixed inset-0 z-50 lg:inset-auto lg:top-16 lg:right-4 lg:w-80"
    @click.self="notificationPanelOpen = false"
  >
    <UCard class="h-full lg:h-auto lg:max-h-96 lg:shadow-lg">
      <template #header>
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-semibold">通知</h3>
          <UButton 
            variant="ghost" 
            size="sm" 
            icon="i-heroicons-x-mark" 
            square 
            @click="notificationPanelOpen = false"
          />
        </div>
      </template>

      <template #body>
        <div class="space-y-3 max-h-64 overflow-y-auto">
          <div 
            v-for="notification in notifications" 
            :key="notification.id"
            class="flex items-start space-x-3 p-3 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
            :class="{ 'bg-primary-50 dark:bg-primary-900/20': !notification.read }"
          >
            <div 
              class="w-2 h-2 rounded-full mt-2 flex-shrink-0"
              :class="{
                'bg-blue-500': notification.type === 'info',
                'bg-green-500': notification.type === 'success',
                'bg-yellow-500': notification.type === 'warning',
                'bg-red-500': notification.type === 'error'
              }"
            />
            <div class="flex-1 min-w-0">
              <p class="font-medium text-sm">{{ notification.title }}</p>
              <p class="text-sm text-neutral-600 dark:text-neutral-400 mt-1">{{ notification.message }}</p>
              <p class="text-xs text-neutral-500 mt-2">{{ notification.time }}</p>
            </div>
          </div>

          <div v-if="notifications.length === 0" class="text-center py-8">
            <UIcon name="i-heroicons-bell-slash" class="w-12 h-12 text-neutral-400 mx-auto mb-3" />
            <p class="text-neutral-500">暂无通知</p>
          </div>
        </div>
      </template>

      <template #footer v-if="notifications.length > 0">
        <div class="flex justify-between">
          <UButton variant="ghost" size="sm" @click="notifications.forEach(n => n.read = true)">
            全部标记为已读
          </UButton>
          <UButton variant="ghost" size="sm" to="/notifications">
            查看全部
          </UButton>
        </div>
      </template>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { NavigationMenuItem } from '@nuxt/ui'

// 导入组件
const SearchModal = defineAsyncComponent(() => import('~/components/SearchModal.vue'));
const UserMenu = defineAsyncComponent(() => import('~/components/UserMenu.vue'));

// 响应式状态
const mobileDrawerOpen = ref(false)

// 路由相关
const route = useRoute()
const { t } = useI18n()
const overlay = useOverlay()

// 创建搜索 Modal 实例
const searchModal = overlay.create(SearchModal)

// 多语言切换 - 使用 useLocaleSwitcher
const { currentLocale, locales, switchLocale } = useLocaleSwitcher()
const toast = useToast()
const switchingLocale = ref(false)

// 语言选项 - 使用对象数组格式，符合 USelectMenu 的要求
const { availableLocales } = useI18n()
const localeNames: Record<string, string> = {
  'zh-CN': '中文',
  'en-US': 'English'
}

const localeOptions = computed(() => {
  return availableLocales.map((code: string) => ({
    label: localeNames[code] || code,
    value: code
  }))
})

// 当前选中的语言对象
const selectedLocaleObject = computed(() => {
  const currentCode = currentLocale.value
  return {
    label: localeNames[currentCode] || currentCode,
    value: currentCode
  }
})

// 处理语言切换
const handleLocaleChange = async (selectedItem: { label: string; value: string }) => {
  if (switchingLocale.value) return
  
  switchingLocale.value = true
  try {
    const success = await switchLocale(selectedItem.value)
    if (success) {
      toast.add({
        title: '语言切换成功',
        description: `已切换到${selectedItem.label}`,
        color: 'success'
      })
    } else {
      throw new Error('切换失败')
    }
  } catch (error) {
    console.error('语言切换错误:', error)
    toast.add({
      title: '语言切换失败',
      description: '请刷新页面后重试',
      color: 'error'
    })
  } finally {
    switchingLocale.value = false
  }
}

// 用户状态
const authStore = useAuthStore()
const currentUser = computed(() => authStore.user)

// 消息相关计算属性
const unreadCount = computed(() => {
  // TODO: 从 store 获取未读消息数量
  return 0
})

// 系统通知
const hasSystemAlerts = computed(() => {
  // TODO: 从 store 获取系统通知状态
  return false
})

// 用户菜单项逻辑已移至 UserMenu 组件

// 桌面端导航菜单项
const navigationItems = computed((): NavigationMenuItem[][] => [
  [
    {
      label: t('nav.home'),
      icon: 'i-heroicons-home',
      to: '/',
      active: route.path === '/'
    },
    {
      label: t('nav.map'),
      icon: 'i-heroicons-map',
      to: '/map',
      active: route.path === '/map'
    }
  ],
  [
    {
      label: t('nav.groups'),
      icon: 'i-heroicons-user-group',
      children: [
        {
          label: t('nav.myGroups'),
          icon: 'i-heroicons-user-group',
          to: '/groups'
        },
        {
          label: t('nav.discoverGroups'),
          icon: 'i-heroicons-globe-alt',
          to: '/groups/discover'
        }
      ]
    },
    {
      label: t('nav.users'),
      icon: 'i-heroicons-users',
      children: [
        {
          label: t('nav.nearbyUsers'),
          icon: 'i-heroicons-users',
          to: '/nearby-users'
        },
        {
          label: t('nav.searchUsers'),
          icon: 'i-heroicons-magnifying-glass',
          to: '/users/search'
        }
      ]
    }
  ]
])

// 移动端导航现在主要通过底部导航栏处理，这里保留备用

// 平板端 Accordion 菜单项
const tabletAccordionItems = computed(() => [
  {
    label: t('nav.home'),
    icon: 'i-heroicons-bars-3',
    defaultOpen: true,
    slot: 'navigation-content'
  }
])

// 移动端设置下拉菜单项
const mobileSettingsItems = computed(() => [
  [
    {
      label: t('nav.profile'),
      icon: 'i-heroicons-user-circle',
      to: '/profile'
    },
    {
      label: t('nav.settings'),
      icon: 'i-heroicons-cog-6-tooth',
      to: '/settings'
    }
  ],
  [
    {
      label: '语言切换',
      icon: switchingLocale.value ? 'i-heroicons-arrow-path' : 'i-heroicons-language',
      children: localeOptions.value.map((option: any) => ({
        label: option.label,
        click: async () => {
          await handleLocaleChange(option)
        }
      }))
    },
    {
      label: '主题切换',
      icon: 'i-heroicons-sun',
      click: () => {
        // 这里可以添加主题切换逻辑
        const colorMode = useColorMode()
        colorMode.preference = colorMode.preference === 'dark' ? 'light' : 'dark'
      }
    }
  ]
])

// 搜索功能
const openSearch = () => {
  searchModal.open()
}

// 通知面板状态
const notificationPanelOpen = ref(false)

// 通知数据
const notifications = ref([
  {
    id: 1,
    title: '欢迎使用 HotSpot',
    message: '开始探索周围的精彩世界吧！',
    time: '刚刚',
    type: 'info',
    read: false
  }
])

// 通知切换
const toggleNotifications = () => {
  notificationPanelOpen.value = !notificationPanelOpen.value
}

// 全局快捷键管理
const { registerShortcut, setupGlobalShortcuts } = useGlobalShortcuts()

// 注册导航相关快捷键
onMounted(() => {
  // 注册快捷键
  registerShortcut({
    key: 'meta_k',
    description: '全局搜索',
    handler: openSearch,
    category: '导航',
    visible: true
  })

  registerShortcut({
    key: 'alt_n',
    description: '查看通知',
    handler: toggleNotifications,
    category: '导航',
    visible: true
  })

  registerShortcut({
    key: 'alt_l',
    description: '切换语言',
    handler: () => {
      // 触发语言选择菜单
      const selectElement = document.querySelector('[data-headlessui-state]')
      if (selectElement) {
        (selectElement as HTMLElement).click()
      }
    },
    category: '导航',
    visible: true
  })

  // 设置全局快捷键监听
  const disposeGlobal = setupGlobalShortcuts()
  
  // 设置导航快捷键
  const disposeNav = defineShortcuts({
    meta_k: {
      usingInput: true,
      handler: () => openSearch(),
    },
    alt_n: {
      usingInput: true,
      handler: () => toggleNotifications(),
    },
  })

  onUnmounted(() => {
    try {
      disposeGlobal()
      disposeNav()
    } catch {
      /* noop */
    }
  })
})

// 监听路由变化（UPopover会自动处理关闭）
watch(() => route.path, () => {
  // UPopover会自动处理菜单关闭，无需手动控制
})
</script>