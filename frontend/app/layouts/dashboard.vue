<template>
  <UDashboardGroup :persistent="false">
    <UDashboardSidebar
      side="left"
      mode="drawer"
      :menu="{ portal: false }"
      :collapsed="sidebarCollapsed"
      :collapsible="true"
      class="hidden sm:flex"
      @update:collapsed="(value: boolean) => sidebarCollapsed = value"
    >
      <template #header="{ collapsed }">
        <div
          class="p-2 flex flex-col gap-2 overflow-x-hidden"
          :class="{ 'items-center': collapsed }"
        >
          <NuxtLink to="/" class="flex items-center gap-2 group focus:outline-none">
            <div class="relative flex items-center justify-center shrink-0">
              <UAvatar
                icon="i-lucide-flame"
                size="md"
                variant="logo"
              />
            </div>
            
            <span
              v-if="!collapsed"
              class="hs-logo-text"
            >
              HotSpot
            </span>
          </NuxtLink>
        </div>
      </template>

      <template #default="{ collapsed }">
        <div
          class="py-4"
          :class="collapsed ? 'px-2 flex justify-center' : 'px-3'"
        >
          <UNavigationMenu
            :items="navigationItems"
            :collapsed="collapsed"
            :variant="collapsed ? 'link' : 'pill'"
            orientation="vertical"
            :highlight="!collapsed"
            :class="collapsed ? 'w-auto' : 'w-full'"
          />
        </div>
      </template>

      <template #footer="{ collapsed }">
        <div
          class="p-2 w-full flex flex-col gap-2 overflow-x-hidden"
          :class="{ 'items-center': collapsed }"
        >
          <UTooltip
            text="设置浏览锚点"
            :content="{ side: collapsed ? 'right' : 'top' }"
            :portal="false"
          >
            <UButton
              variant="soft"
              size="md"
              :square="collapsed"
              icon="i-lucide-map-pin"
              @click="openLocationPicker"
            >
              <div
                v-if="!collapsed"
                class="truncate flex-1 text-left"
              >
                <UBadge>浏览锚点</UBadge>
                <div class="hs-text-strong">
                  {{ currentLocationDisplay }}
                </div>
              </div>
            </UButton>
          </UTooltip>
          <USeparator class="my-1 w-full" />

          <div
            class="w-full"
            :class="collapsed ? 'flex flex-col gap-2 items-center justify-center' : 'flex items-center justify-between'"
          >
            <div :class="collapsed ? 'flex flex-col gap-2 items-center' : 'flex items-center gap-1.5 flex-nowrap'">
              <UTooltip
                text="搜索 (⌘K)"
                :content="{ side: collapsed ? 'right' : 'top' }"
                :portal="false"
              >
                <UButton
                  variant="ghost"
                  color="neutral"
                  :size="collapsed ? 'md' : 'sm'"
                  icon="i-lucide-search"
                  :square="collapsed"
                  @click="openGlobalSearch"
                />
              </UTooltip>

              <UTooltip
                text="通知"
                :content="{ side: collapsed ? 'right' : 'top' }"
                :portal="false"
              >
                <UChip
                  :show="unreadCount > 0"
                  :text="unreadCount > 99 ? '99+' : unreadCount"
                  color="error"
                  position="top-right"
                  size="sm"
                  :inset="true"
                >
                  <UButton
                    variant="ghost"
                    color="neutral"
                    :size="collapsed ? 'md' : 'sm'"
                    icon="i-lucide-bell"
                    :square="collapsed"
                    @click="onNotificationsClick"
                  />
                </UChip>
              </UTooltip>

              <UTooltip
                text="主题切换"
                :content="{ side: collapsed ? 'right' : 'top' }"
                :portal="false"
              >
                <UColorModeButton
                  variant="ghost"
                  color="neutral"
                  :size="collapsed ? 'md' : 'sm'"
                  :square="collapsed"
                />
              </UTooltip>
            </div>

            <div class="flex justify-center">
              <UTooltip
                :text="collapsed ? '展开侧边栏' : '收起侧边栏'"
                :content="{ side: collapsed ? 'right' : 'top' }"
                :portal="false"
              >
                <UButton
                  variant="ghost"
                  color="neutral"
                  :size="collapsed ? 'md' : 'sm'"
                  :square="collapsed"
                  :icon="collapsed ? 'i-lucide-chevrons-right' : 'i-lucide-chevrons-left'"
                  @click="sidebarCollapsed = !sidebarCollapsed"
                />
              </UTooltip>
            </div>
          </div>

        </div>
      </template>
    </UDashboardSidebar>

    <UDashboardPanel>
      <template #header>
        <div class="sticky top-0 z-30">
          <UDashboardNavbar :toggle="false">
            <template #left>
              <UTooltip
                v-if="showNavbarBack"
                text="返回"
                :portal="false"
              >
                <UButton
                  variant="ghost"
                  color="neutral"
                  icon="i-lucide-arrow-left"
                  class="hidden sm:inline-flex"
                  @click="goBack"
                />
              </UTooltip>
              <UButton
                class="sm:hidden"
                variant="ghost"
                color="neutral"
                icon="i-lucide-menu"
                @click="mobileNavOpen = true"
              />
            </template>
            <template #right>
              <div class="flex items-center gap-3">
                <USelectMenu
                  :model-value="currentLocale"
                  :items="localeOptions"
                  value-key="value"
                  size="sm"
                  class="w-24 hidden sm:block"
                  :disabled="switchingLocale"
                  @update:model-value="handleLocaleChange"
                >
                  <template #leading>
                    <UIcon
                      :name="switchingLocale ? 'i-lucide-refresh-cw' : 'i-lucide-languages'"
                      :size="16"
                      :class="switchingLocale && 'animate-spin'"
                    />
                  </template>
                </USelectMenu>

                <UPopover
                  class="sm:hidden"
                  :portal="false"
                >
                  <UButton
                    variant="ghost"
                    size="md"
                    :icon="switchingLocale ? 'i-lucide-refresh-cw' : 'i-lucide-languages'"
                    :loading="switchingLocale"
                  />
                  <template #content>
                    <div class="p-2 space-y-1">
                      <UButton
                        v-for="locale in localeOptions"
                        :key="locale.value"
                        :color="currentLocale === locale.value ? 'primary' : 'neutral'"
                        :variant="currentLocale === locale.value ? 'soft' : 'ghost'"
                        size="sm"
                        :block="true"
                        class="justify-start"
                        @click="handleLocaleChange(locale.value)"
                      >
                        {{ locale.label }}
                      </UButton>
                    </div>
                  </template>
                </UPopover>

                <!-- 用户菜单 -->
                <UserMenu />
              </div>
            </template>
          </UDashboardNavbar>

          <div v-if="dashboardHeaderEnabled">
            <UPageHeader
              :title="dashboardTitle"
              :description="dashboardDescription"
            >
              <template #links>
                <div class="flex items-center gap-2">
                  <UButton
                    v-for="action in mergedDashboardHeaderActions"
                    :key="action.key"
                    :color="action.color"
                    :variant="action.variant || 'outline'"
                    :size="action.size || 'sm'"
                    :leading-icon="action.icon"
                    :loading="action.loading"
                    :disabled="action.disabled"
                    :class="action.class"
                    @click="action.action"
                  >
                    {{ action.label }}
                  </UButton>
                </div>
              </template>
            </UPageHeader>
          </div>

          <div
            v-if="dashboardStats && dashboardStats.length"
            class="w-full px-3 sm:px-4 lg:px-6 pb-4"
          >
            <StatsGrid
              :stats="dashboardStats"
              :columns="dashboardStatsColumns"
            />
          </div>
        </div>
      </template>

      <template #body>
        <div
          v-if="isFullHeight"
          class="h-full min-h-0"
        >
          <slot />
        </div>

        <!-- 普通页面模式：内容只负责 padding，不再承担 header/sticky，避免滚动穿透 -->
        <div
          v-else
          :class="bodyContainerClass"
        >
          <slot />
        </div>
      </template>
    </UDashboardPanel>
  </UDashboardGroup>

  <UDrawer
    v-model:open="mobileNavOpen"
    direction="bottom"
    title="导航"
    description="快速跳转到主要功能"
  >
    <template #body>
      <div class="space-y-3 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
        <UNavigationMenu
          :items="navigationItems"
          orientation="vertical"
          :highlight="true"
          variant="pill"
        />

        <USeparator class="my-2" />

        <!-- 移动端功能区 -->
        <div class="grid grid-cols-4 gap-2 px-2">
          <div class="flex flex-col items-center gap-1">
            <UButton
              variant="soft"
              color="neutral"
              icon="i-lucide-search"
              size="lg"
              :square="true"
              @click="openGlobalSearch"
            />
            <UBadge variant="subtle" color="neutral" size="xs">搜索</UBadge>
          </div>

          <div class="flex flex-col items-center gap-1">
            <div class="relative">
              <UButton
                variant="soft"
                color="neutral"
                icon="i-lucide-bell"
                size="lg"
                :square="true"
                @click="onNotificationsClick"
              />
              <UChip
                v-if="unreadCount > 0"
                :text="unreadCount > 99 ? '99+' : unreadCount"
                color="error"
                position="top-right"
                size="sm"
                :inset="false"
                class="absolute top-0 right-0 translate-x-1/3 -translate-y-1/3"
              />
            </div>
            <UBadge variant="subtle" color="neutral" size="xs">通知</UBadge>
          </div>

          <div class="flex flex-col items-center gap-1">
            <UColorModeButton
              variant="soft"
              color="neutral"
              size="lg"
              :square="true"
            />
            <UBadge variant="subtle" color="neutral" size="xs">主题</UBadge>
          </div>

          <div class="flex flex-col items-center gap-1">
             <UButton
              variant="soft"
              color="neutral"
              icon="i-lucide-map-pin"
              size="lg"
              :square="true"
              @click="openLocationPicker"
            />
            <UBadge variant="subtle" color="neutral" size="xs">定位</UBadge>
          </div>
        </div>
      </div>
    </template>
  </UDrawer>
</template>

<script setup lang="ts">
import type { LanguageCode } from '~/stores/settings.store'
import { DASHBOARD_HEADER_CTX_KEY, type DashboardHeaderAction } from '~/composables/useDashboardHeader'
import StatsGrid from '~/components/data/StatsGrid.vue'
import type { StatItem } from '~/components/data/StatsGrid.vue'

const { t, locale, availableLocales } = useI18n();
const nuxtApp = useNuxtApp()

// 语言切换（只在本 layout 用，避免额外 composable）
const localeNames: Record<LanguageCode, string> = {
  'zh-CN': '中文',
  'en-US': 'English'
}

const currentLocale = computed(() => (locale.value || 'zh-CN') as LanguageCode)

const locales = computed(() =>
  (availableLocales || []).map((code: string) => ({
    code: code as LanguageCode,
    name: localeNames[code as LanguageCode] || code
  }))
)

const switchLocale = async (code: LanguageCode): Promise<boolean> => {
  try {
    await (nuxtApp.$i18n as { setLocale: (c: LanguageCode) => Promise<void> }).setLocale(code)
    settingsStore.setLanguage(code)
    return true
  } catch (error) {
    console.error('语言切换失败:', error)
    return false
  }
}
const route = useRoute();
const router = useRouter();
const mainNavPages = ['/', '/map', '/moments/nearby', '/users/nearby', '/groups/nearby', '/chat'];

const isFullHeight = computed(() => route.meta.fullHeight === true);

const dashboardHeaderEnabled = computed(() => {
  if (isFullHeight.value) return false
  return route.meta.dashboardHeader === true
})

type ContentWidth = 'container' | 'full'
const contentWidth = computed<ContentWidth>(() => {
  // 默认：桌面端阅读更舒服（container），需要“铺满”的页面再显式 opt-in full
  const raw = route.meta.contentWidth
  return raw === 'full' ? 'full' : 'container'
})

const bodyContainerClass = computed(() => {
  const base = 'w-full px-3 sm:px-4 lg:px-6 py-5'
  if (contentWidth.value === 'full') return base
  return `${base} max-w-(--ui-container) mx-auto`
})

const dashboardTitle = ref<string>(String(route.meta.title || ''))
const dashboardDescription = ref<string>(String(route.meta.description || ''))

const dashboardStats = ref<StatItem[] | null>(null)
const dashboardStatsColumns = ref<1 | 2 | 3 | 4 | 5 | 6>(4)

const dashboardHeaderActions = ref<DashboardHeaderAction[]>([])

type DashboardHeaderMetaAction = {
  key: string
  label: string
  icon?: string
  color?: DashboardHeaderAction['color']
  variant?: DashboardHeaderAction['variant']
  size?: DashboardHeaderAction['size']
  disabled?: boolean
  to: string
}

const dashboardMetaActions = computed<DashboardHeaderAction[]>(() => {
  const raw = route.meta.dashboardActions as unknown
  if (!Array.isArray(raw)) return []

  const actions: DashboardHeaderAction[] = []
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue
    const a = item as Partial<DashboardHeaderMetaAction>
    if (!a.key || typeof a.key !== 'string') continue
    if (!a.label || typeof a.label !== 'string') continue
    if (!a.to || typeof a.to !== 'string') continue

    actions.push({
      key: a.key,
      label: a.label,
      icon: a.icon,
      color: a.color,
      variant: a.variant,
      size: a.size,
      disabled: a.disabled,
      action: () => navigateTo(a.to!)
    })
  }
  return actions
})

const mergedDashboardHeaderActions = computed<DashboardHeaderAction[]>(() => {
  const map = new Map<string, DashboardHeaderAction>()
  for (const a of dashboardMetaActions.value) map.set(a.key, a)
  for (const a of dashboardHeaderActions.value) map.set(a.key, a)
  return [...map.values()]
})

provide(DASHBOARD_HEADER_CTX_KEY, {
  setHeader: (h: { title?: string; description?: string }) => {
    if (h.title !== undefined) dashboardTitle.value = h.title
    if (h.description !== undefined) dashboardDescription.value = h.description
  },
  clearHeader: () => {
    dashboardTitle.value = String(route.meta.title || '')
    dashboardDescription.value = String(route.meta.description || '')
  },
  setStats: (stats: StatItem[] | null) => {
    dashboardStats.value = stats || null
  },
  clearStats: () => {
    dashboardStats.value = null
  },
  setActions: (actions: DashboardHeaderAction[]) => {
    dashboardHeaderActions.value = actions
  },
  clearActions: () => {
    dashboardHeaderActions.value = []
  }
})

watch(() => route.fullPath, () => {
  dashboardHeaderActions.value = []
  dashboardStats.value = null
  dashboardTitle.value = String(route.meta.title || '')
  dashboardDescription.value = String(route.meta.description || '')
})

const canGoBack = computed(() => {
  return !mainNavPages.includes(route.path);
});

const showNavbarBack = computed(() => {
  return canGoBack.value && route.meta.showNavbarBack !== false
})

const goBack = () => {
  if (router.options.history.state.back) {
    router.back();
  } else {
    router.push('/');
  }
};

const authStore = useAuthStore();
const messagesStore = useMessagesStore();
const settingsStore = useSettingsStore();
const locationStore = useLocationStore();
const anchorStore = useMapAnchorStore()

const unreadCount = messagesStore.totalUnreadCount;

const sidebarCollapsedValue = useHydratedStore(settingsStore, 'sidebarCollapsed', false);
const sidebarCollapsed = computed({
  get: () => sidebarCollapsedValue.value,
  set: (value: boolean) => settingsStore.setSidebarCollapsed(value)
});

const anchorName = useHydratedStore(anchorStore, 'anchorName', '')
const currentLocationDisplay = computed(() => anchorName.value || '未设置锚点')

const modal = useModal();
const feedback = useFeedbackStore()
const { requireAuth } = useRequireAuth()
const adminStore = useAdminStore()

const hydratedMe = useHydratedStore(authStore, 'user', null) as Ref<{ user_type?: string } | null>
const hydratedIsAdmin = computed(() => {
  return hydratedMe.value?.user_type === 'admin' || adminStore.hasAccess
})

const mobileNavOpen = ref(false)

const onNotificationsClick = () => {
  if (!requireAuth('查看通知')) return
  navigateTo('/notifications')
}

const openLocationPicker = () => {
  modal.showLocationPicker((location) => {
    anchorStore.setAnchor(location.latitude, location.longitude, location.locationName)
    feedback.success('浏览锚点已更新', location.locationName)
  });
};

const navigationItems = computed(() => [
  [
    {
      label: t('nav.home'),
      icon: 'i-lucide-house',
      to: '/',
      active: route.path === '/',
    }
  ],
  [
    {
      label: t('nav.moments'),
      icon: 'i-lucide-image',
      to: '/moments/nearby',
      active: route.path.startsWith('/moments'),
    },
    {
      label: t('nav.nearbyUsers'),
      icon: 'i-lucide-users',
      to: '/users/nearby',
      active: route.path.startsWith('/users/nearby'),
    },
    {
      label: t('nav.groups'),
      icon: 'i-lucide-users-round',
      to: '/groups/nearby',
      active: route.path.startsWith('/groups'),
    },
  ],
  [
    {
      label: t('nav.messages'),
      icon: 'i-lucide-messages-square',
      to: '/chat',
      active: route.path.startsWith('/chat'),
      badge: unreadCount > 0
        ? (unreadCount > 99 ? '99+' : unreadCount.toString())
        : undefined,
    },
  ],

  ...(hydratedIsAdmin.value ? [[
    {
      label: '管理',
      icon: 'i-lucide-shield-check',
      to: '/admin',
      active: route.path.startsWith('/admin')
    }
  ]] : [])
]);

const switchingLocale = ref(false);

const localeOptions = computed(() =>
  locales.value.map((locale) => ({
    value: locale.code,
    label: locale.name,
  }))
);

const handleLocaleChange = async (newLocale: string | undefined) => {
  if (switchingLocale.value || !newLocale) return;
  if (newLocale !== 'zh-CN' && newLocale !== 'en-US') return;

  switchingLocale.value = true;
  try {
    const success = await switchLocale(newLocale);
    if (success) {
      feedback.success('语言切换成功', `已切换到${newLocale === 'zh-CN' ? '中文' : 'English'}`)
    }
  } catch (error) {
    console.error('语言切换错误:', error);
    feedback.error('语言切换失败', '请刷新页面后重试')
  } finally {
    switchingLocale.value = false;
  }
};

const openGlobalSearch = () => {
  modal.showGlobalSearch();
};

watch(() => route.fullPath, () => {
  mobileNavOpen.value = false
})

defineShortcuts({
  meta_k: { usingInput: true, handler: openGlobalSearch },
  ctrl_k: { usingInput: true, handler: openGlobalSearch },
  q: { usingInput: false, handler: () => navigateTo('/chat') },
});
</script>
