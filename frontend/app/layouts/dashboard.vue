<template>
  <!-- 使用 v-model 控制侧边栏折叠状态，由 settings store 持久化 -->
  <UDashboardGroup v-model:collapsed="sidebarCollapsed" :persistent="false">
    <!-- 侧边栏 -->
    <UDashboardSidebar
      resizable
      collapsible
      side="left"
      mode="drawer"
      :ui="{ footer: 'border-t border-default' }"
    >
      <!-- Logo 区域 -->
      <template #header="{ collapsed }">
        <NuxtLink
          to="/"
          class="flex items-center gap-3 hover:opacity-90 transition-opacity"
          aria-label="返回首页"
        >
          <div
            class="w-8 h-8 rounded-xl bg-linear-to-br from-primary-500 to-red-500 flex items-center justify-center shadow-lg shrink-0"
          >
            <UIcon
              name="i-heroicons-fire"
              class="w-5 h-5 text-white"
            />
          </div>
          <span
            :class="collapsed ? 'hidden' : 'text-lg font-bold text-emphasis whitespace-nowrap'"
          >
            HotSpot
          </span>
        </NuxtLink>
      </template>

      <!-- 搜索和导航 -->
      <template #default="{ collapsed }">
        <!-- 导航菜单搜索按钮（搜索侧边栏导航项） -->

        <!-- 主导航 -->
        <UNavigationMenu
          :items="navigationItems"
          :collapsed="collapsed"
          variant="pill"
          orientation="vertical"
          highlight
        />
      </template>

      <!-- 底部工具栏 -->
      <template #footer="{ collapsed }">
        <div class="space-y-3">
          <!-- 当前位置显示（可点击修改） -->
          <UTooltip :text="collapsed ? currentLocationDisplay : '点击修改位置'" :placement="collapsed ? 'right' : 'top'">
            <UButton
              variant="ghost"
              :icon="collapsed ? 'i-heroicons-map-pin' : undefined"
              size="sm"
              :square="collapsed"
              :block="!collapsed"
              :class="collapsed ? '' : 'justify-start text-left'"
              @click="openLocationPicker"
            >
              <template v-if="!collapsed">
                <UIcon name="i-heroicons-map-pin" class="w-4 h-4 text-primary shrink-0" />
                <span class="truncate text-xs">{{ currentLocationDisplay }}</span>
              </template>
            </UButton>
          </UTooltip>

          <!-- 工具按钮行 -->
          <div :class="collapsed ? 'flex flex-col items-center gap-2' : 'flex items-center justify-between'">
            <!-- 全局搜索按钮 -->
            <UTooltip text="全局搜索" :kbds="['meta', 'K']">
              <UButton
                variant="ghost"
                icon="i-heroicons-magnifying-glass"
                size="sm"
                square
                @click="openGlobalSearch"
              />
            </UTooltip>

            <!-- 通知按钮 -->
            <UTooltip text="通知中心">
              <UButton
                variant="ghost"
                icon="i-heroicons-bell"
                size="sm"
                square
                @click="navigateTo('/notifications')"
              />
            </UTooltip>

            <!-- 颜色模式切换 -->
            <UColorModeSwitch />

            <!-- 折叠/展开按钮（仅桌面端显示，移动端用顶部 toggle） -->
            <UDashboardSidebarCollapse class="hidden lg:flex" />
          </div>
        </div>
      </template>
    </UDashboardSidebar>

    <!-- 主内容区域 -->
    <UDashboardPanel>
      <!-- 顶部导航栏 -->
      <template #header>
        <UDashboardNavbar>
          <!-- 右侧工具栏 -->
          <template #right>
            <div class="flex items-center gap-2">
              <!-- 语言切换 - 桌面端显示完整下拉框 -->
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
                    :name="switchingLocale ? 'i-heroicons-arrow-path' : 'i-heroicons-language'"
                    class="icon-sm"
                    :class="{ 'loading-spin': switchingLocale }"
                  />
                </template>
              </USelectMenu>

              <!-- 语言切换 - 移动端显示简洁按钮 -->
              <UPopover class="sm:hidden">
                <UButton
                  variant="ghost"
                  size="sm"
                  :icon="switchingLocale ? 'i-heroicons-arrow-path' : 'i-heroicons-language'"
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
                      block
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
      </template>

      <!-- 页面内容 -->
      <template #body>
        <!-- 全高度模式（聊天页面等） -->
        <div v-if="isFullHeight" class="h-full flex flex-col overflow-hidden">
          <slot />
        </div>
        <!-- 普通页面模式 -->
        <div v-else class="w-full px-4 sm:px-6 lg:px-8 py-4 max-w-[1600px] mx-auto">
          <!-- 全局返回按钮 -->
          <div v-if="canGoBack" class="mb-4">
            <UButton
              variant="ghost"
              color="neutral"
              icon="i-heroicons-arrow-left"
              @click="goBack"
            >
              返回
            </UButton>
          </div>
          <slot />
        </div>
      </template>
    </UDashboardPanel>
  </UDashboardGroup>
</template>

<script setup lang="ts">
const { t } = useI18n();
const { currentLocale, locales, switchLocale } = useLocaleSwitcher();
const route = useRoute();
const router = useRouter();
const toast = useToast();

// 返回按钮逻辑

// 主要导航页面（不显示返回按钮）
const mainNavPages = ['/', '/map', '/moments', '/users/nearby', '/groups/discover', '/chat'];

// 检查是否是全高度模式（聊天页面等）
const isFullHeight = computed(() => route.meta.fullHeight === true);

// 是否可以返回（只要不是主导航页面就显示返回按钮）
const canGoBack = computed(() => {
  return !mainNavPages.includes(route.path);
});

// 返回上一页
const goBack = () => {
  // 如果有历史记录则返回，否则回到首页
  if (window.history.length > 1) {
    router.back();
  } else {
    router.push('/');
  }
};

// 使用水合安全的 store 访问
const authStore = useAuthStore();
const messagesStore = useMessagesStore();
const settingsStore = useSettingsStore();
const locationStore = useLocationStore();
const unreadCount = computed(() => messagesStore.totalUnreadCount);

// WebSocket 连接
const { connect: wsConnect, isConnected: wsConnected, status: wsStatus } = useGlobalWebSocket();

// 侧边栏折叠状态（双向绑定到 store）
const sidebarCollapsed = computed({
  get: () => settingsStore.sidebarCollapsed,
  set: (value: boolean) => settingsStore.setSidebarCollapsed(value)
});

// 当前位置显示：SSR 时使用默认值，onMounted 后同步 store 值
const currentLocationDisplay = ref('未设置位置');

onMounted(() => {
  // 立即同步位置
  currentLocationDisplay.value = locationStore.location?.locationName || '未设置位置';
  // 监听变化
  watch(() => locationStore.location?.locationName, (name) => {
    currentLocationDisplay.value = name || '未设置位置';
  });
  
  // 初始化 WebSocket 连接（用户已登录时）
  if (authStore.isAuthenticated) {
    wsConnect();
  }
});

// 模态框管理
const modal = useModal();

// 打开位置选择器（使用高德地图）
const openLocationPicker = () => {
  modal.showLocationPicker((location) => {
    locationStore.setLocation(location.latitude, location.longitude, location.locationName);
    toast.add({ title: '位置已更新', description: location.locationName, color: 'success' });
  });
};

// 导航菜单项 - 按功能逻辑分组
const navigationItems = computed(() => [
  // 第一组：核心功能
  [
    {
      label: t('nav.home'),
      icon: 'i-heroicons-home',
      to: '/',
      active: route.path === '/',
    },
    {
      label: t('nav.map'),
      icon: 'i-heroicons-map',
      to: '/map',
      active: route.path === '/map',
    },
  ],
  // 第二组：发现内容
  [
    {
      label: t('nav.moments'),
      icon: 'i-heroicons-photo',
      to: '/moments',
      active: route.path.startsWith('/moments'),
    },
    {
      label: t('nav.nearbyUsers'),
      icon: 'i-heroicons-users',
      to: '/users/nearby',
      active: route.path === '/users/nearby',
    },
    {
      label: t('nav.groups'),
      icon: 'i-heroicons-user-group',
      to: '/groups/discover',
      active: route.path.startsWith('/groups'),
    },
  ],
  // 第三组：社交通讯
  [
    {
      label: t('nav.messages'),
      icon: 'i-heroicons-chat-bubble-left-right',
      to: '/chat',
      active: route.path.startsWith('/chat'),
      badge: unreadCount.value > 0
        ? (unreadCount.value > 99 ? '99+' : unreadCount.value.toString())
        : undefined,
    },
  ],
]);

// 语言切换
const switchingLocale = ref(false);

const localeOptions = computed(() =>
  locales.value.map((locale) => ({
    value: locale.code,
    label: locale.name,
  }))
);

const handleLocaleChange = async (newLocale: string | undefined) => {
  if (switchingLocale.value || !newLocale) return;
  // 验证 locale 是否有效
  if (newLocale !== 'zh-CN' && newLocale !== 'en-US') return;

  switchingLocale.value = true;
  try {
    const success = await switchLocale(newLocale);
    if (success) {
      toast.add({
        title: '语言切换成功',
        description: `已切换到${newLocale === 'zh-CN' ? '中文' : 'English'}`,
        color: 'success',
      });
    }
  } catch (error) {
    console.error('语言切换错误:', error);
    toast.add({
      title: '语言切换失败',
      description: '请刷新页面后重试',
      color: 'error',
    });
  } finally {
    switchingLocale.value = false;
  }
};

// 全局搜索
const openGlobalSearch = () => {
  modal.showGlobalSearch();
};

// 快捷键
defineShortcuts({
  meta_k: { usingInput: true, handler: openGlobalSearch },
  ctrl_k: { usingInput: true, handler: openGlobalSearch },
  q: { usingInput: false, handler: () => navigateTo('/chat') },
});
</script>
