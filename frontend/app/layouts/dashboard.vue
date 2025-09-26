<template>
  <UDashboardGroup>
    <!-- 侧边栏 -->
    <UDashboardSidebar 
      collapsible
      resizable
      :min-size="15"
      :default-size="20"
      :max-size="30"
      :collapsed-size="0"
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
          color="primary"
          variant="pill"
          orientation="vertical"
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
                {{ (currentUser as any)?.is_temp ? "临时用户" : "正式用户" }}
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

            <ThemeToggle size="sm" />
            
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
        <UDashboardNavbar>
          <!-- 页面标题 -->
          <template #center>
            <h1 class="text-lg font-semibold text-neutral-900 dark:text-white">
              {{ pageTitle }}
            </h1>
          </template>

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
              <USelectMenu
                v-model="selectedLocaleObject"
                :items="localeOptions"
                size="sm"
                class="min-w-24 hidden sm:block"
                :loading="switchingLocale"
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
              </USelectMenu>

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
// 使用现有的 composables
const { openSearchModal } = useModals();
const { currentLocale, switchLocale } = useLocaleSwitcher();
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
const pageTitle = computed(() => {
  return route.meta.title || "HotSpot";
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
      badge: authStore.user ? undefined : { label: "新", color: "primary" },
    },
    {
      label: t("nav.messages"),
      icon: "i-heroicons-chat-bubble-left-right",
      to: "/chat",
      active: route.path.startsWith("/chat"),
      badge: unreadCount.value > 0 ? { 
        label: unreadCount.value > 99 ? "99+" : unreadCount.value.toString(),
        color: "error" 
      } : undefined,
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
          to: "/users/search",
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
const { availableLocales } = useI18n();
const localeNames: Record<string, string> = {
  "zh-CN": "中文",
  "en-US": "English",
};

const localeOptions = computed(() => 
  availableLocales.map((code: string) => ({
    label: localeNames[code] || code,
    value: code,
  }))
);

const selectedLocaleObject = computed(() => {
  const currentCode = currentLocale.value;
  return {
    label: localeNames[currentCode] || currentCode,
    value: currentCode,
  };
});

const handleLocaleChange = async (selectedItem: { label: string; value: string }) => {
  if (switchingLocale.value) return;

  switchingLocale.value = true;
  try {
    const success = await switchLocale(selectedItem.value);
    if (success) {
      toast.add({
        title: "语言切换成功",
        description: `已切换到${selectedItem.label}`,
        color: "success",
      });
    } else {
      throw new Error("切换失败");
    }
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
const openSearch = () => {
  openSearchModal({
    onCreateMoment: () => {
      navigateTo('/moments');
    }
  });
};

// 通知功能
const toggleNotifications = () => {
  showNotifications.value = !showNotifications.value
};

// 快捷键
onMounted(() => {
  const shortcuts = useShortcuts();
  shortcuts.register('meta+k', openSearch, '全局搜索', '导航');
  shortcuts.register('alt+n', toggleNotifications, '查看通知', '导航');
  shortcuts.register('meta+m', () => navigateTo('/chat'), '消息中心', '导航');
});
</script>
