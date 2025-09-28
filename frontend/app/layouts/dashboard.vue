<template>
  <UDashboardGroup>
    <!-- 侧边栏 -->
    <UDashboardSidebar 
      collapsible
      resizable
      :min-size="15"
      :default-size="20"
      :max-size="30"
      :collapsed-size="4"
      side="left"
      :toggle="{
        color: 'neutral',
        variant: 'ghost',
        class: 'rounded-full'
      }"
    >
      <!-- Logo 区域 -->
      <template #header="{ collapsed }">
        <NuxtLink
          to="/"
          class="flex items-center gap-3 hover:opacity-90 transition-opacity p-3"
          aria-label="返回首页"
        >
          <div
            class="w-8 h-8 rounded-xl bg-gradient-to-br from-primary-500 to-error-600 flex items-center justify-center shadow-lg"
          >
            <UIcon
              name="i-heroicons-fire"
              class="w-5 h-5 text-white"
            />
          </div>
          <span
            v-if="!collapsed"
            class="text-lg font-bold bg-gradient-to-r from-primary-600 to-error-600 bg-clip-text text-transparent"
          >
            HotSpot
          </span>
        </NuxtLink>
      </template>

      <!-- 主导航 -->
      <template #default="{ collapsed }">
        <UNavigationMenu
          :items="navigationItems"
          :collapsed="collapsed"
          variant="pill"
          orientation="vertical"
          highlight
          :tooltip="{
            delayDuration: 300,
            content: { side: 'right', sideOffset: 12 }
          }"
          :popover="{
            mode: 'hover',
            content: { side: 'right', align: 'start', sideOffset: 12 }
          }"
          class="p-3"
        />
      </template>

      <!-- 底部工具栏 -->
      <template #footer="{ collapsed }">
        <div class="p-3 space-y-2">
          <!-- 用户信息 -->
          <div v-if="currentUser && !collapsed" class="flex items-center gap-3 p-2 rounded-lg bg-neutral-100 dark:bg-neutral-800">
            <UAvatar
              :alt="currentUser.nickname"
              size="sm"
            >
              {{ currentUser.nickname?.charAt(0) || "U" }}
            </UAvatar>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">{{ currentUser.nickname || "用户" }}</p>
              <p class="text-xs text-neutral-500 truncate">
                {{ isTempUser(currentUser.user_id) ? "临时用户" : "正式用户" }}
              </p>
            </div>
          </div>

          <!-- 工具按钮 -->
          <div class="flex gap-1">
            <UTooltip text="搜索 (⌘K)">
              <UButton
                variant="ghost"
                icon="i-heroicons-magnifying-glass"
                size="sm"
                square
                @click="openSearch"
              />
            </UTooltip>
            
            <UTooltip text="通知">
              <UPopover v-model:open="showNotifications" :popper="{ placement: 'bottom-end' }">
                <UButton
                  variant="ghost"
                  icon="i-heroicons-bell"
                  size="sm"
                  square
                  class="relative"
                >
                  <UBadge
                    v-if="notificationPanelRef?.unreadCount > 0"
                    size="xs"
                    color="error"
                    class="absolute -top-1 -right-1 w-2 h-2"
                  />
                </UButton>
                
                <template #panel>
                  <Suspense>
                    <NotificationPanel ref="notificationPanelRef" />
                    <template #fallback>
                      <div class="w-96 h-64 flex items-center justify-center">
                        <USkeleton class="w-full h-full" />
                      </div>
                    </template>
                  </Suspense>
                </template>
              </UPopover>
            </UTooltip>

            <UColorModeSwitch/>
            
            <UTooltip text="设置">
              <UButton
                to="/settings"
                variant="ghost"
                icon="i-heroicons-cog-6-tooth"
                size="sm"
                square
              />
            </UTooltip>
          </div>
        </div>
      </template>
    </UDashboardSidebar>

    <!-- 主内容区域 -->
    <UDashboardPanel>
      <!-- 顶部导航栏 -->
      <template #header>
        <UDashboardNavbar
          title="HotSpot"
          :toggle="{
            color: 'neutral',
            variant: 'ghost',
            class: 'rounded-full'
          }"
          toggle-side="left"
        >

          <!-- 右侧工具栏 -->
          <template #right>
            <div class="flex items-center gap-1 sm:gap-2">
              <!-- 消息按钮 -->
              <UTooltip text="消息中心">
                <UButton
                  to="/chat"
                  variant="ghost"
                  size="sm"
                  square
                  class="relative"
                >
                  <UIcon
                    name="i-heroicons-chat-bubble-left-right"
                    class="w-4 h-4"
                  />
                  <UBadge
                    v-if="unreadCount > 0"
                    size="xs"
                    color="error"
                    class="absolute -top-1 -right-1 min-w-4 h-4 flex items-center justify-center"
                  >
                    {{ unreadCount > 9 ? "9+" : unreadCount }}
                  </UBadge>
                </UButton>
              </UTooltip>

              <!-- 语言切换 - 移动端隐藏 -->
              <ULocaleSelect
                v-model="currentLocale"
                :locales="nuxtLocales"
                size="sm"
                class="min-w-24 hidden sm:block"
                :disabled="switchingLocale"
                @update:model-value="handleLocaleChange"
              >
                <template #leading>
                  <UIcon
                    :name="switchingLocale ? 'i-heroicons-arrow-path' : 'i-heroicons-language'"
                    class="w-3 h-3"
                    :class="{ 'animate-spin': switchingLocale }"
                  />
                </template>
              </ULocaleSelect>

              <!-- 用户菜单 -->
              <UserMenu />
            </div>
          </template>
        </UDashboardNavbar>
      </template>

      <!-- 页面内容 -->
      <template #body>
        <div class="p-3 sm:p-6">
          <slot />
        </div>
      </template>
    </UDashboardPanel>
  </UDashboardGroup>
</template>

<script setup lang="ts">
import { isTempUser } from '~/utils/userTypes'

// 使用 composables
const { showCommandPalette } = useModal();
const { locale: currentLocale, availableLocales } = useI18n();
const route = useRoute();
const { t } = useI18n();
const toast = useToast();


// 用户状态
const authStore = useAuthStore();
const currentUser = computed(() => authStore.user);

// 消息相关
const messagesStore = useMessagesStore();
const unreadCount = computed(() => messagesStore.totalUnreadCount);

// 通知系统
const showNotifications = ref(false)
const notificationPanelRef = ref()

// 系统通知
const hasSystemAlerts = computed(() => {
  return notificationPanelRef.value?.unreadCount > 0
});

// 页面标题 - 从 route.meta 获取
const pageTitle = computed<string>(() => {
  const title = route.meta.title;
  return typeof title === 'string' ? title : "HotSpot";
});

// 导航菜单项
const navigationItems = computed(() => [
  [
    {
      label: t("nav.home"),
      icon: "i-heroicons-home",
      to: "/",
      active: route.path === "/",
    },
    {
      label: t("nav.map"),
      icon: "i-heroicons-map",
      to: "/map",
      active: route.path === "/map",
    },
  ],
  [
    {
      label: t("nav.groups"),
      icon: "i-heroicons-user-group",
      to: "/groups",
      active: route.path.startsWith("/groups"),
      badge: authStore.user ? undefined : "新",
    },
    {
      label: t("nav.messages"),
      icon: "i-heroicons-chat-bubble-left-right",
      to: "/chat",
      active: route.path.startsWith("/chat"),
      badge: unreadCount.value > 0 ? 
        (unreadCount.value > 99 ? "99+" : unreadCount.value.toString()) : undefined,
    },
  ],
  [
    {
      label: t("nav.users"),
      icon: "i-heroicons-users",
      children: [
        {
          label: t("nav.nearbyUsers"),
          icon: "i-heroicons-users",
          to: "/users/nearby",
        },
        {
          label: t("nav.searchUsers"),
          icon: "i-heroicons-magnifying-glass",
          to: "/search",
        },
      ],
    },
    {
      label: t("nav.moments"),
      icon: "i-heroicons-photo",
      to: "/moments",
      active: route.path.startsWith("/moments"),
    },
  ],
]);

// 语言切换相关
const switchingLocale = ref(false);

// 创建 Nuxt UI LocaleSelect 需要的 locales 格式
const nuxtLocales = computed(() => 
  availableLocales.map((code: string) => ({
    code,
    name: code === 'zh-CN' ? '中文' : 'English',
    dir: 'ltr' as const,
    messages: {} // 这里可以传入对应的翻译消息
  }))
);

const handleLocaleChange = async (newLocale: string | undefined) => {
  if (switchingLocale.value || !newLocale) return;

  switchingLocale.value = true;
  try {
    // 使用 Nuxt i18n 的 setLocale
    const nuxtApp = useNuxtApp();
    await (nuxtApp.$i18n as any).setLocale(newLocale);
    
    // 保存到 settings store
    const settingsStore = useSettingsStore();
    settingsStore.setLanguage(newLocale);
    
    const localeName = newLocale === 'zh-CN' ? '中文' : 'English';
    toast.add({
      title: "语言切换成功",
      description: `已切换到${localeName}`,
      color: "success",
    });
  } catch (error) {
    console.error("语言切换错误:", error);
    toast.add({
      title: "语言切换失败",
      description: "请刷新页面后重试",
      color: "error",
    });
  } finally {
    switchingLocale.value = false;
  }
};

// 搜索功能
const openSearch = async () => {
  await showCommandPalette({
    onCreateMoment: () => {
      navigateTo('/moments');
    }
  });
};

// 通知功能
const toggleNotifications = () => {
  showNotifications.value = !showNotifications.value
};

// 快捷键管理 - 使用 Nuxt UI v4 原生支持
defineShortcuts({
  's': {
    usingInput: false,
    handler: openSearch,
  },
  'n': {
    usingInput: false,
    handler: toggleNotifications,
  },
  'q': {
    usingInput: false,
    handler: () => navigateTo('/chat'),
  },
});
</script>
