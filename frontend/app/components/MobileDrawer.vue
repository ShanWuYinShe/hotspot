<template>
  <!-- 移动端导航抽屉 - 使用 UDrawer -->
  <UDrawer v-model:open="isOpen" side="left" :overlay="true">
    <template #header>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center">
            <UIcon name="i-heroicons-fire" class="w-5 h-5 text-white" />
          </div>
          <span class="text-lg font-bold bg-gradient-to-r from-orange-600 to-red-600 bg-clip-text text-transparent">
            HotSpot
          </span>
        </div>
        <UButton variant="ghost" icon="i-heroicons-x-mark" size="sm" square @click="isOpen = false" />
      </div>
    </template>

    <template #body>
      <div class="space-y-6 p-4">
        <!-- 用户信息卡片 -->
        <UCard v-if="authStore.user" class="bg-gradient-to-r from-primary-50 to-blue-50 dark:from-primary-900/20 dark:to-blue-900/20">
          <div class="flex items-center gap-3">
            <UAvatar 
              :alt="authStore.user.nickname" 
              size="lg"
              class="ring-2 ring-white dark:ring-neutral-800"
            >
              {{ authStore.user.nickname?.charAt(0) || 'U' }}
            </UAvatar>
            <div class="flex-1">
              <p class="font-semibold text-neutral-900 dark:text-white">
                {{ authStore.user.nickname }}
              </p>
              <p class="text-sm text-neutral-600 dark:text-neutral-400">
                {{ authStore.user.email || '临时用户' }}
              </p>
              <UBadge 
                :color="authStore.user.is_temporary ? 'warning' : 'success'" 
                size="xs" 
                class="mt-1"
              >
                {{ authStore.user.is_temporary ? '临时账户' : '正式用户' }}
              </UBadge>
            </div>
          </div>
        </UCard>

        <!-- 未登录状态 -->
        <UCard v-else class="bg-gradient-to-r from-neutral-50 to-neutral-100 dark:from-neutral-800 dark:to-neutral-900">
          <div class="text-center py-4">
            <UIcon name="i-heroicons-user-circle" class="w-12 h-12 text-neutral-400 mx-auto mb-2" />
            <p class="font-medium text-neutral-900 dark:text-white mb-2">未登录</p>
            <UButton to="/auth" size="sm" @click="isOpen = false">
              登录 / 注册
            </UButton>
          </div>
        </UCard>

        <!-- 主要导航 -->
        <div class="space-y-2">
          <h3 class="text-sm font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider px-2">
            主要功能
          </h3>
          <UNavigationMenu 
            :items="mainNavItems" 
            orientation="vertical"
            class="space-y-1"
            @click="handleNavClick"
          />
        </div>

        <!-- 发现功能 -->
        <div class="space-y-2">
          <h3 class="text-sm font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider px-2">
            发现
          </h3>
          <UNavigationMenu 
            :items="discoverNavItems" 
            orientation="vertical"
            class="space-y-1"
            @click="handleNavClick"
          />
        </div>

        <!-- 设置和工具 -->
        <div class="space-y-2">
          <h3 class="text-sm font-semibold text-neutral-500 dark:text-neutral-400 uppercase tracking-wider px-2">
            设置
          </h3>
          <UNavigationMenu 
            :items="settingsNavItems" 
            orientation="vertical"
            class="space-y-1"
            @click="handleNavClick"
          />
        </div>
      </div>
    </template>

    <template #footer>
      <div class="p-4 border-t border-neutral-200 dark:border-neutral-700">
        <div class="flex items-center justify-between">
          <!-- 主题切换 -->
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-sun" class="w-4 h-4 text-neutral-500" />
            <span class="text-sm text-neutral-600 dark:text-neutral-400">主题</span>
          </div>
          <ThemeToggle size="sm" />
        </div>
        
        <!-- 语言切换 -->
        <div class="flex items-center justify-between mt-3">
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-language" class="w-4 h-4 text-neutral-500" />
            <span class="text-sm text-neutral-600 dark:text-neutral-400">语言</span>
          </div>
          <USelectMenu 
            v-model="selectedLocale" 
            :options="localeOptions" 
            size="sm"
            class="w-20"
          />
        </div>

        <!-- 版本信息 -->
        <div class="text-center mt-4 pt-3 border-t border-neutral-100 dark:border-neutral-800">
          <p class="text-xs text-neutral-400">HotSpot v1.0.0</p>
        </div>
      </div>
    </template>
  </UDrawer>
</template>

<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

// Props
interface Props {
  open?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  open: false
})

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean]
  'toggle-notifications': []
}>()

// 响应式状态
const isOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

// 路由和国际化
const route = useRoute()
const { t } = useI18n()
const authStore = useAuthStore()

// 语言切换
const { currentLocale, switchLocale, locales } = useLocaleSwitcher()
const selectedLocale = computed({
  get: () => currentLocale.value,
  set: (value) => switchLocale(value)
})

const localeOptions = computed(() => {
  if (!locales.value || !Array.isArray(locales.value)) {
    return []
  }
  return locales.value.map((locale: any) => ({
    label: locale.name || locale.code,
    value: locale.code
  }))
})

// 主要导航项
const mainNavItems = computed((): NavigationMenuItem[][] => [
  [
    {
      label: '搜索',
      icon: 'i-heroicons-magnifying-glass',
      click: () => {
        // 触发全局搜索
        // TODO: 实现搜索功能
        isOpen.value = false
      }
    },
    {
      label: t('nav.moments'),
      icon: 'i-heroicons-photo',
      to: '/moments',
      active: route.path.startsWith('/moments')
    },
    {
      label: '附近用户',
      icon: 'i-heroicons-users',
      to: '/nearby-users',
      active: route.path === '/nearby-users'
    }
  ]
])

// 发现导航项
const discoverNavItems = computed((): NavigationMenuItem[][] => [
  [
    {
      label: t('nav.discoverGroups'),
      icon: 'i-heroicons-globe-alt',
      to: '/groups/discover',
      active: route.path === '/groups/discover'
    },
    {
      label: '用户搜索',
      icon: 'i-heroicons-magnifying-glass-circle',
      to: '/users/search',
      active: route.path === '/users/search'
    }
  ]
])

// 设置导航项
const settingsNavItems = computed((): NavigationMenuItem[][] => [
  [
    {
      label: t('nav.profile'),
      icon: 'i-heroicons-user-circle',
      to: '/profile',
      active: route.path === '/profile'
    },
    {
      label: t('nav.settings'),
      icon: 'i-heroicons-cog-6-tooth',
      to: '/settings',
      active: route.path === '/settings'
    },
    {
      label: '通知',
      icon: 'i-heroicons-bell',
      click: () => {
        // 触发通知面板
        emit('toggle-notifications')
        isOpen.value = false
      }
    }
  ]
])

// 处理导航点击
const handleNavClick = () => {
  // 导航后关闭抽屉
  nextTick(() => {
    isOpen.value = false
  })
}

// 监听路由变化，自动关闭抽屉
watch(() => route.path, () => {
  isOpen.value = false
})
</script>