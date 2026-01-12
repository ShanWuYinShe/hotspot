<template>
    <UDashboardGroup>
        <UDashboardSidebar
            collapsible
            v-model:collapsed="sidebarCollapsed"
            mode="drawer"
        >
            <template #header="{ collapsed }">
                <div
                    class="w-full flex items-center"
                    :class="collapsed ? 'justify-center' : 'justify-start'"
                >
                    <UButton
                        to="/"
                        variant="link"
                        color="neutral"
                        size="md"
                        :square="collapsed"
                        :block="!collapsed"
                        :aria-label="collapsed ? t('app.title') : undefined"
                    >
                        <div class="size-8 shrink-0 rounded-full overflow-hidden">
                            <NuxtImg
                                src="/logo_256.png"
                                :alt="t('app.title')"
                                class="size-full object-cover"
                            />
                        </div>
                        <span v-if="!collapsed">{{ t('app.title') }}</span>
                    </UButton>
                </div>
            </template>
            <template #default="{ collapsed }">
                <UNavigationMenu
                    orientation="vertical"
                    :items="navigationItems"
                    :collapsed="collapsed"
                    :variant="collapsed ? 'link' : 'pill'"
                />
            </template>

            <template #footer="{ collapsed }">
                <div
                    class="w-full flex flex-col gap-2"
                    :class="collapsed ? 'items-center' : ''"
                >
                    <!-- 第一行：更新浏览锚点（展开时显示当前位置名） -->
                    <UTooltip
                        :text="t('dashboard.anchor.update')"
                        :content="{ side: collapsed ? 'right' : 'top' }"
                        :portal="true"
                    >
                        <UButton
                            variant="soft"
                            color="neutral"
                            size="md"
                            icon="i-lucide-map-pin"
                            trailing
                            :square="collapsed"
                            :block="!collapsed"
                            @click="openLocationPicker"
                        >
                            <template v-if="!collapsed">
                                <div class="min-w-0 text-left">
                                    <div class="truncate">{{ t('dashboard.anchor.label') }}</div>
                                    <div class="truncate">
                                        {{ currentLocationDisplay }}
                                    </div>
                                </div>
                            </template>
                        </UButton>
                    </UTooltip>

                    <USeparator :class="collapsed ? 'w-8' : 'w-full'" />

                    <!-- 第二行：操作按钮（配置驱动） -->
                    <div
                        class="w-full"
                        :class="collapsed
                            ? 'flex flex-col items-center gap-2'
                            : 'flex items-center justify-between'
                            "
                    >
                        <!-- 左侧按钮组：搜索、通知、主题 -->
                        <div
                            class="flex items-center gap-2"
                            :class="collapsed ? 'flex-col' : ''"
                        >
                            <template
                                v-for="action in leftActions"
                                :key="action.key"
                            >
                                <UTooltip
                                    :text="action.tooltip"
                                    :content="{
                                        side: collapsed ? 'right' : 'top',
                                    }"
                                    :portal="true"
                                >
                                    <template v-if="action.type === 'button'">
                                        <UButton
                                            variant="ghost"
                                            color="neutral"
                                            size="md"
                                            :icon="action.icon"
                                            :square="true"
                                            @click="action.onClick"
                                        />
                                    </template>

                                    <template v-else-if="action.type === 'chip'">
                                        <UChip
                                            :show="action.show"
                                            :text="action.text"
                                            color="error"
                                            position="top-right"
                                            size="sm"
                                            :inset="true"
                                        >
                                            <UButton
                                                variant="ghost"
                                                color="neutral"
                                                size="md"
                                                :icon="action.icon"
                                                :square="true"
                                                @click="action.onClick"
                                            />
                                        </UChip>
                                    </template>

                                    <template v-else-if="action.type === 'colorMode'">
                                        <UColorModeButton
                                            variant="ghost"
                                            color="neutral"
                                            size="md"
                                            :square="true"
                                        />
                                    </template>
                                </UTooltip>
                            </template>
                        </div>

                        <!-- 右侧按钮：折叠（仅桌面端显示） -->
                        <div class="hidden lg:block">
                            <UTooltip
                                :text="collapseAction.tooltip"
                                :content="{
                                    side: collapsed ? 'right' : 'top',
                                }"
                                :portal="true"
                            >
                                <UButton
                                    variant="ghost"
                                    color="neutral"
                                    size="md"
                                    :icon="collapseAction.icon"
                                    :square="true"
                                    @click="collapseAction.onClick"
                                />
                            </UTooltip>
                        </div>
                    </div>
                </div>
            </template>
        </UDashboardSidebar>

        <UDashboardPanel>
            <template #header>
                <UDashboardNavbar>
                    <template #right>
                        <div class="flex items-center gap-3">
                            <USelectMenu
                                :model-value="currentLocale"
                                :items="localeOptions"
                                value-key="value"
                                size="sm"
                                :disabled="switchingLocale"
                                @update:model-value="handleLocaleChange"
                            >
                                <template #leading>
                                    <UIcon
                                        :name="switchingLocale
                                            ? 'i-lucide-refresh-cw'
                                            : 'i-lucide-languages'
                                            "
                                        class="size-4"
                                        :class="{ 'animate-spin': switchingLocale }"
                                    />
                                </template>
                            </USelectMenu>

                            <UserMenu />
                        </div>
                    </template>
                </UDashboardNavbar>
            </template>

            <template #body>
                <slot />
            </template>
        </UDashboardPanel>
    </UDashboardGroup>
</template>

<script setup lang="ts">
import type { LanguageCode } from "~/stores/settings.store";
import { useSettingsStore } from "~/stores/settings.store";
import { useMessagesStore } from "~/stores/messages.store";
import { useAuthStore } from "~/stores/auth.store";
import { useAdminStore } from "~/stores/admin.store";
import { useMapAnchorStore } from "~/stores/map_anchor.store";
import { useFeedbackStore } from "~/stores/feedback.store";

// @nuxtjs/i18n auto-imports useI18n, but TypeScript may need explicit type hint

const { t, locale, availableLocales } = useI18n();
const nuxtApp = useNuxtApp();
const { requireAuth } = useRequireAuth();
const route = useRoute();
const modal = useModal();
const settingsStore = useSettingsStore();
const messagesStore = useMessagesStore();
const authStore = useAuthStore();
const adminStore = useAdminStore();
const anchorStore = useMapAnchorStore();
const feedback = useFeedbackStore();

const switchingLocale = ref(false);

// Sidebar state
const sidebarCollapsedValue = useHydratedStore(
    settingsStore,
    "sidebarCollapsed",
    false,
);
const sidebarCollapsed = computed({
    get: () => sidebarCollapsedValue.value,
    set: (value: boolean) => settingsStore.setSidebarCollapsed(value),
});

// Unread count
const unreadCount = computed(() => messagesStore.totalUnreadCount);

const hydratedAnchorName = useHydratedStore(anchorStore, "anchorName", "");
const currentLocationDisplay = computed(
    () => hydratedAnchorName.value || t('dashboard.anchor.notSet'),
);

// 注意：AuthData 中没有 user_type 字段，所以不能依赖 user_type 来判断是否是 admin
// 只能依赖 adminStore.hasAccess，它通过调用 POST /api/v1/admin/rbac/me 获取
// 使用 useHydratedStore 避免 SSR hydration mismatch (在非 Admin 页面，服务端为 false，客户端持久化可能为 true)
const isAdmin = useHydratedStore(adminStore, 'hasAccess', false);

// Navigation Items
// 注意：不手动设置 active，让 UNavigationMenu 根据路由自动判断，避免 SSR/CSR 不一致
const navigationItems = computed(() => [
    [
        {
            label: t("nav.home"),
            icon: "i-lucide-house",
            to: "/",
        },
    ],
    [
        {
            label: t("nav.moments"),
            icon: "i-lucide-image",
            to: "/moments/nearby",
        },
        {
            label: t("nav.nearbyUsers"),
            icon: "i-lucide-users",
            to: "/users/nearby",
        },
        {
            label: t("nav.groups"),
            icon: "i-lucide-users-round",
            to: "/groups/nearby",
        },
    ],
    [
        {
            label: t("nav.messages"),
            icon: "i-lucide-messages-square",
            to: "/chat",
            badge:
                unreadCount.value > 0
                    ? unreadCount.value > 99
                        ? "99+"
                        : unreadCount.value.toString()
                    : undefined,
        },
    ],
    ...(isAdmin.value
        ? [
            [
                {
                    label: t('nav.admin'),
                    icon: "i-lucide-shield-check",
                    to: "/admin",
                },
            ],
        ]
        : []),
]);

function onNotificationsClick() {
    if (!requireAuth(t('dashboard.tooltips.notifications'))) return;
    navigateTo("/notifications");
}

async function openGlobalSearch() {
    await modal.showGlobalSearch();
}

async function openLocationPicker() {
    await modal.showLocationPicker((loc) => {
        anchorStore.setAnchor(loc.latitude, loc.longitude, loc.locationName);
        feedback.success(t('dashboard.anchor.updated'), loc.locationName);
    });
}

type FooterAction =
    | {
        key: string;
        type: "button";
        tooltip: string;
        icon: string;
        onClick: () => void;
    }
    | {
        key: string;
        type: "chip";
        tooltip: string;
        icon: string;
        show: boolean;
        text: string;
        onClick: () => void;
    }
    | {
        key: string;
        type: "colorMode";
        tooltip: string;
    };

// 左侧按钮组：搜索、通知、主题
const leftActions = computed<FooterAction[]>(() => [
    {
        key: "search",
        type: "button",
        tooltip: t('dashboard.tooltips.search'),
        icon: "i-lucide-search",
        onClick: () => {
            openGlobalSearch();
        },
    },
    {
        key: "notifications",
        type: "chip",
        tooltip: t('dashboard.tooltips.notifications'),
        icon: "i-lucide-bell",
        show: unreadCount.value > 0,
        text: unreadCount.value > 99 ? "99+" : unreadCount.value.toString(),
        onClick: () => {
            onNotificationsClick();
        },
    },
    {
        key: "theme",
        type: "colorMode",
        tooltip: t('dashboard.tooltips.theme'),
    },
]);

// 右侧按钮：折叠
const collapseAction = computed<FooterAction & { type: "button" }>(() => ({
    key: "collapse",
    type: "button",
    tooltip: sidebarCollapsed.value ? t('dashboard.tooltips.expandSidebar') : t('dashboard.tooltips.collapseSidebar'),
    icon: sidebarCollapsed.value
        ? "i-lucide-chevrons-right"
        : "i-lucide-chevrons-left",
    onClick: () => {
        sidebarCollapsed.value = !sidebarCollapsed.value;
    },
}));

// Locale Logic
const currentLocale = computed(() => (locale.value || "zh-CN") as LanguageCode);
const localeOptions = computed(() =>
    (availableLocales || []).map((code: string) => ({
        value: code as LanguageCode,
        label: t(`languages.${code}`),
    })),
);

async function handleLocaleChange(newLocale: string | undefined) {
    if (switchingLocale.value || !newLocale) return;
    if (newLocale !== "zh-CN" && newLocale !== "en-US") return;

    switchingLocale.value = true;
    try {
        locale.value = newLocale as LanguageCode;
        settingsStore.setLanguage(newLocale as LanguageCode);
    } finally {
        switchingLocale.value = false;
    }
}
</script>
