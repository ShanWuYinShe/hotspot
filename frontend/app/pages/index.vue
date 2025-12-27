<template>
    <UPage>
        <UPageHeader
            title="HotSpot"
            :description="headerDescription"
            :links="headerLinks"
        />

        <UPageBody>
            <div class="space-y-6">
                <UCard variant="subtle">
                    <template #header>
                        <div>
                            <div class="font-bold tracking-tight">探索</div>
                            <UBadge color="neutral" variant="soft" size="xs" class="mt-1">
                              从附近开始，快速进入你想看的内容
                            </UBadge>
                        </div>
                    </template>

                    <UPageGrid>
                        <UPageCard
                            title="附近用户"
                            description="发现周围的朋友"
                            icon="i-lucide-users"
                            to="/users/nearby"
                            variant="soft"
                        />
                        <UPageCard
                            title="附近群组"
                            description="发现可加入的群组"
                            icon="i-lucide-users-round"
                            to="/groups/nearby"
                            variant="soft"
                        />
                        <UPageCard
                            title="附近瞬间"
                            description="浏览周边动态"
                            icon="i-lucide-image"
                            to="/moments/nearby"
                            variant="soft"
                        />
                    </UPageGrid>
                </UCard>

                <UCard variant="subtle">
                    <template #header>
                        <div class="flex items-center justify-between gap-3">
                            <div>
                                <div class="font-bold tracking-tight">我的</div>
                                <UBadge color="neutral" variant="soft" size="xs" class="mt-1">
                                  登录后可查看统计、通知与快捷入口
                                </UBadge>
                            </div>
                            <UBadge
                                v-if="isHydrated && isLoggedIn && unreadCount > 0"
                                color="info"
                                variant="soft"
                                size="sm"
                            >
                                {{ unreadCount > 99 ? '99+' : unreadCount }} 未读
                            </UBadge>
                        </div>
                    </template>

                    <div
                        v-if="!isHydrated"
                        class="grid grid-cols-2 md:grid-cols-4 gap-4"
                    >
                        <USkeleton
                            v-for="i in 4"
                            :key="i"
                            class="h-24"
                        />
                    </div>

                    <StatsGrid
                        v-else-if="isLoggedIn"
                        :stats="statsItems"
                        :columns="4"
                    />

                    <UPageGrid v-else>
                        <UPageCard
                            title="快速体验"
                            description="一键创建访客账号，随时升级"
                            icon="i-lucide-zap"
                            variant="soft"
                            :to="undefined"
                            @click="createTempUser"
                        />
                        <UPageCard
                            title="隐私设置"
                            description="登录后可配置位置、在线状态与可见性"
                            icon="i-lucide-shield-check"
                            variant="soft"
                            :to="undefined"
                            @click="openSettings"
                        />
                        <UPageCard
                            title="通知中心"
                            description="登录后可查看与你相关的实时事件"
                            icon="i-lucide-bell"
                            variant="soft"
                            :to="undefined"
                            @click="openNotifications"
                        />
                    </UPageGrid>
                </UCard>
            </div>
        </UPageBody>
    </UPage>
</template>

<script setup lang="ts">
import { getPageSeoConfig, buildSeoMeta } from "~/config/seo";
import { useAuthService } from "~/services/auth.service";
import { useUsersService } from "~/services/users.service";
import type { UserStatsData } from "~/types/api";
import type { AuthState } from "~/stores/auth.store";

// 从数据源获取 SEO 配置
const seoConfig = getPageSeoConfig("home");

// 使用 Nuxt v4 的 definePageMeta
definePageMeta({
    title: seoConfig.title,
    description: seoConfig.description,
    // 首页与全站保持一致：同一套壳（Nginx 直进 / 站内跳转观感一致）
    layout: 'dashboard',
});

// 直接使用 Nuxt v4 的 useSeoMeta
useSeoMeta(buildSeoMeta("home"));

const modal = useModal();
const { requireAuth } = useRequireAuth()
const { withErrorHandling } = useErrorHandler()
const authService = useAuthService()
const userService = useUsersService()
const feedback = useFeedbackStore()
const isHydrated = ref(false)
onMounted(() => { isHydrated.value = true })

// 水合安全：持久化 store 必须通过 useHydratedStore 访问（避免 SSR hydration mismatch）
const authStore = useAuthStore()
const messagesStore = useMessagesStore()
const locationStore = useLocationStore()

const DEFAULT_HYDRATED_AUTH_STATE: AuthState = {
    currentUserId: null,
    accounts: {},
    accountOrder: []
}

const hydratedAuthState = useHydratedStore(authStore, 'authState', DEFAULT_HYDRATED_AUTH_STATE)
const hydratedIsAuthed = useHydratedStore(authStore, 'isAuthenticated', false)

const unreadCount = computed(() => messagesStore.totalUnreadCount)
const currentUser = computed(() => {
    const state = hydratedAuthState.value
    if (!state.currentUserId) return null
    const session = state.accounts[state.currentUserId]
    return session?.user ?? null
})
const isLoggedIn = computed(() => hydratedIsAuthed.value)

const openSettings = () => {
    if (!requireAuth('进入隐私设置')) return
    navigateTo('/settings')
}

const openNotifications = () => {
    if (!requireAuth('查看通知')) return
    navigateTo('/notifications')
}

const currentLocationName = computed(() => locationStore.locationName || '未设置位置')

const headerDescription = computed(() => {
    if (!isLoggedIn.value) return '基于位置的社交平台，发现身边有趣的人和事'
    if (!currentUser.value) return '欢迎回来'
    return `你好，${currentUser.value.nickname} · ${currentLocationName.value}`
})

const openCreateMoment = async () => {
    const id = await modal.showCreateMoment()
    if (id) {
        await navigateTo(`/moments/${id}`)
    }
}

const openCreateGroup = async () => {
    const id = await modal.showCreateGroup()
    if (id) {
        await navigateTo(`/groups/${id}`)
    }
}

const headerLinks = computed(() => {
    // 首页不重复展示登录/注册入口：dashboard layout 已提供统一入口
    // 同时保持 SSR/CSR 结构稳定，避免直进与站内跳转观感割裂
    if (!isHydrated.value) return []
    if (!isLoggedIn.value) return []

    return [
        { label: '发瞬间', onClick: openCreateMoment, color: 'primary', variant: 'solid', icon: 'i-lucide-square-pen' },
        { label: '建群', onClick: openCreateGroup, color: 'neutral', variant: 'outline', icon: 'i-lucide-users-round' },
    ]
})

const createTempUser = async () => {
    await withErrorHandling(async () => {
        const authData = await authService.guest({
            nickname: `访客_${Date.now().toString(36).slice(-4)}`,
            client_type: 'app',
        })
        authStore.setAccountSession(
            { user_id: authData.user_id, nickname: authData.nickname },
            authData.access_token,
            authData.refresh_token ?? null,
            new Date(authData.access_token_expires_at).getTime(),
            new Date(authData.refresh_token_expires_at).getTime()
        )
        feedback.success('欢迎体验 HotSpot', '临时账户已创建，开始探索吧！')
    }, {
        title: '创建失败',
        fallbackMessage: '请稍后重试'
    })
}

const loadingStats = ref(false)
const myStats = ref<UserStatsData | null>(null)

const loadMyStats = async () => {
    if (!isLoggedIn.value) return
    if (loadingStats.value) return

    loadingStats.value = true
    try {
        myStats.value = await userService.getMyStats()
    } catch {
        myStats.value = null
    } finally {
        loadingStats.value = false
    }
}

watch(isLoggedIn, (v) => {
    if (v) void loadMyStats()
    else myStats.value = null
}, { immediate: true })

const statsItems = computed(() => {
    const s = myStats.value
    return [
        { key: 'moments', label: '瞬间', value: s?.moments_count ?? 0, icon: 'i-lucide-image', color: 'primary' as const, onClick: () => { void navigateTo('/moments') } },
        { key: 'friends', label: '好友', value: s?.friends_count ?? 0, icon: 'i-lucide-users-round', color: 'green' as const, onClick: () => { void navigateTo('/friends') } },
        { key: 'followers', label: '粉丝', value: s?.followers_count ?? 0, icon: 'i-lucide-users', color: 'blue' as const, onClick: () => { void navigateTo('/followers') } },
        { key: 'groups', label: '群组', value: s?.groups_count ?? 0, icon: 'i-lucide-users-round', color: 'purple' as const, onClick: () => { void navigateTo('/groups') } }
    ]
})

// 快捷键管理 - 使用 Nuxt UI v4 原生支持
defineShortcuts({
    h: {
        usingInput: false,
        handler: () => navigateTo("/"),
    },
    g: {
        usingInput: false,
        handler: () => navigateTo("/groups"),
    },
    c: {
        usingInput: false,
        handler: () => navigateTo("/chat"),
    },
    u: {
        usingInput: false,
        handler: () => navigateTo("/users/nearby"),
    },
    p: {
        usingInput: false,
        handler: () => navigateTo("/moments"),
    },
});
</script>
