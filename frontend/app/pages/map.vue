<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <div class="mb-8 sm:mb-10">
            <h1 class="text-3xl font-bold text-neutral-900 mb-1">地图</h1>
            <p class="text-neutral-600 dark:text-neutral-300">发现附近的用户、群组和动态</p>
        </div>

        <!-- 筛选标签页 -->
        <div class="mb-6">
            <div class="flex flex-wrap gap-2">
                <UButton 
                    variant="ghost" 
                    size="sm"
                    :class="{ 'bg-primary/10 text-primary': activeFilters.includes('users') }"
                    @click="toggleFilter('users')"
                >
                    <UIcon name="i-heroicons-users" class="w-4 h-4 mr-1" />
                    用户 ({{ nearbyUsers.length }})
                </UButton>
                <UButton 
                    variant="ghost" 
                    size="sm"
                    :class="{ 'bg-primary/10 text-primary': activeFilters.includes('groups') }"
                    @click="toggleFilter('groups')"
                >
                    <UIcon name="i-heroicons-user-group" class="w-4 h-4 mr-1" />
                    群组 ({{ nearbyGroups.length }})
                </UButton>
                <UButton 
                    variant="ghost" 
                    size="sm"
                    :class="{ 'bg-primary/10 text-primary': activeFilters.includes('moments') }"
                    @click="toggleFilter('moments')"
                >
                    <UIcon name="i-heroicons-photo" class="w-4 h-4 mr-1" />
                    动态 ({{ nearbyMoments.length }})
                </UButton>
            </div>
        </div>

        <!-- 地图容器（SSR/CSR统一渲染，onMounted 初始化真实地图） -->
        <UCard class="h-96 mb-6 overflow-hidden" aria-label="地图容器">
            <div
                v-if="!amapEnabled"
                class="h-full flex items-center justify-center"
            >
                <EmptyState
                    icon="i-heroicons-map"
                    title="地图未启用"
                    description="未检测到高德地图 Key，请在环境变量中配置 NUXT_PUBLIC_AMAP_KEY 后刷新。"
                >
                    <template #illustration>
                        <EmptyNearby size="lg" />
                    </template>
                </EmptyState>
            </div>
            <div v-else ref="mapEl" class="h-full w-full"></div>
        </UCard>

        <!-- 操作按钮与状态提示 -->
        <div class="flex flex-wrap items-center gap-3 mb-6">
            <UButton size="lg" :loading="isLoading" @click="refreshLocation">
                <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 mr-2" />
                刷新位置
            </UButton>
            <UButton size="lg" variant="outline" @click="toggleLocationSharing">
                <UIcon name="i-heroicons-map-pin" class="w-4 h-4 mr-2" />
                {{ isLocationSharing ? "关闭共享" : "开启共享" }}
            </UButton>
            <UAlert v-if="permissionDenied" color="warning" variant="soft" class="text-sm">
                已拒绝位置权限，请在浏览器地址栏位置图标中重新授权
            </UAlert>
        </div>

        <!-- 内容列表区域 -->
        <UCard>
            <template #header>
                <div class="flex items-center justify-between">
                    <h3 class="text-lg font-semibold">{{ currentTabTitle }}</h3>
                    <div class="flex items-center gap-2">
                        <UButton
                            size="sm"
                            variant="ghost"
                            @click="refreshCurrentContent"
                            :loading="loadingCurrent"
                        >
                            <UIcon
                                name="i-heroicons-arrow-path"
                                class="w-4 h-4"
                            />
                        </UButton>
                    </div>
                </div>
            </template>

            <!-- 用户列表 -->
            <div v-if="currentTab === 'users'" class="space-y-3">
                <USkeleton v-if="loadingUsers" class="h-24" />
                <template v-else>
                    <EmptyState
                        v-if="nearbyUsers.length === 0"
                        icon="i-heroicons-users"
                        title="暂无附近用户"
                        description="尝试扩大搜索范围或稍后再试"
                        action-text="刷新"
                        :action-handler="() => refreshUsers()"
                        secondary-action-text="开启共享"
                        :secondary-action-handler="toggleLocationSharing"
                    >
                        <template #illustration>
                            <EmptyNearby size="lg" />
                        </template>
                    </EmptyState>
                    <div v-else class="divide-y">
                        <div
                            v-for="u in nearbyUsers"
                            :key="u.user_id"
                            class="py-3 flex items-center gap-3 cursor-pointer hover:bg-neutral-50 dark:hover:bg-neutral-900 rounded-lg px-2"
                            @click="showUserDetail(u)"
                        >
                            <UAvatar :alt="u.nickname">{{ u.nickname?.charAt(0) }}</UAvatar>
                            <div class="flex-1 min-w-0">
                                <div class="flex items-center gap-2">
                                    <span class="font-medium truncate">{{ u.nickname }}</span>
                                </div>
                                <div class="text-xs text-neutral-500 flex items-center gap-2">
                                    <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
                                    <span>{{ formatDistanceMeters(u.distance) }}</span>
                                    <span v-if="u.location_name" class="truncate">· {{ u.location_name }}</span>
                                </div>
                            </div>
                            <UButton
                                size="sm"
                                variant="outline"
                                @click.stop="goChatUser(u.user_id)"
                            >
                                聊天
                            </UButton>
                        </div>
                        <div v-if="hasMoreUsers" class="pt-3">
                            <UButton
                                block
                                :loading="loadingMoreUsers"
                                @click="loadMoreUsers"
                            >
                                加载更多
                            </UButton>
                        </div>
                    </div>
                </template>
            </div>

            <!-- 群组列表 -->
            <div v-else-if="currentTab === 'groups'" class="space-y-3">
                <USkeleton v-if="loadingGroups" class="h-24" />
                <template v-else>
                    <EmptyState
                        v-if="nearbyGroups.length === 0"
                        icon="i-heroicons-user-group"
                        title="暂无附近群组"
                        description="试试扩大搜索范围或查看发现页"
                        action-text="刷新"
                        :action-handler="() => refreshGroups()"
                        secondary-action-text="发现群组"
                        :secondary-action-handler="() => navigateTo('/groups/discover')"
                    >
                        <template #illustration>
                            <EmptyNearby size="lg" />
                        </template>
                    </EmptyState>
                    <div v-else class="divide-y">
                        <div
                            v-for="g in nearbyGroups"
                            :key="g.group_id"
                            class="py-3 flex items-center gap-3 cursor-pointer hover:bg-neutral-50 dark:hover:bg-neutral-900 rounded-lg px-2"
                            @click="showGroupDetail(g)"
                        >
                            <UAvatar :alt="g.name">{{ g.name?.charAt(0) }}</UAvatar>
                            <div class="flex-1 min-w-0">
                                <div class="flex items-center gap-2">
                                    <span class="font-medium truncate">{{ g.name }}</span>
                                    <UBadge
                                        v-if="g.is_password_required"
                                        variant="soft"
                                        size="xs"
                                    >私有</UBadge>
                                </div>
                                <div class="text-xs text-neutral-500 flex items-center gap-2">
                                    <UIcon name="i-heroicons-users" class="w-3 h-3" />
                                    <span>{{ g.member_count }} 成员</span>
                                    <UIcon name="i-heroicons-map-pin" class="w-3 h-3 ml-2" />
                                    <span>{{ formatDistanceKm(g.distance) }}</span>
                                </div>
                            </div>
                            <UButton
                                size="sm"
                                :disabled="g.is_member"
                                @click.stop="enterGroupOrJoin(g)"
                            >
                                {{ g.is_member ? "已加入" : "加入" }}
                            </UButton>
                        </div>
                        <div v-if="hasMoreGroups" class="pt-3">
                            <UButton
                                block
                                :loading="loadingMoreGroups"
                                @click="loadMoreGroups"
                            >
                                加载更多
                            </UButton>
                        </div>
                    </div>
                </template>
            </div>

            <!-- 动态列表 -->
            <div v-else-if="currentTab === 'moments'" class="space-y-3">
                <USkeleton v-if="loadingMoments" class="h-24" />
                <template v-else>
                    <EmptyState
                        v-if="nearbyMoments.length === 0"
                        icon="i-heroicons-photo"
                        title="暂无附近动态"
                        description="尝试扩大搜索范围或发布第一条动态"
                        action-text="刷新"
                        :action-handler="() => refreshMoments()"
                        secondary-action-text="发布动态"
                        :secondary-action-handler="() => navigateTo('/moments/create')"
                    >
                        <template #illustration>
                            <EmptyNearby size="lg" />
                        </template>
                    </EmptyState>
                    <div v-else class="divide-y">
                        <div
                            v-for="m in nearbyMoments"
                            :key="m.moment_id"
                            class="py-4 cursor-pointer hover:bg-neutral-50 dark:hover:bg-neutral-900 rounded-lg px-2"
                            @click="showMomentDetail(m)"
                        >
                            <div class="flex items-start gap-3 mb-2">
                                <UAvatar :alt="m.nickname" size="sm">{{ m.nickname?.charAt(0) }}</UAvatar>
                                <div class="flex-1 min-w-0">
                                    <div class="flex items-center gap-2">
                                        <span class="font-medium text-sm">{{ m.nickname }}</span>
                                        <span class="text-xs text-neutral-500">{{ formatTime(m.created_at) }}</span>
                                    </div>
                                    <div class="text-xs text-neutral-500 flex items-center gap-1">
                                        <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
                                        <span>{{ formatDistanceMeters(m.distance) }}</span>
                                        <span v-if="m.location_name">· {{ m.location_name }}</span>
                                    </div>
                                </div>
                            </div>
                            <p class="text-sm text-neutral-700 dark:text-neutral-300 mb-2 line-clamp-2">
                                {{ m.description }}
                            </p>
                            <div class="flex items-center gap-4 text-xs text-neutral-500">
                                <span class="flex items-center gap-1">
                                    <UIcon name="i-heroicons-heart" class="w-3 h-3" />
                                    {{ m.likes_count }}
                                </span>
                                <span class="flex items-center gap-1">
                                    <UIcon name="i-heroicons-chat-bubble-left-right" class="w-3 h-3" />
                                    {{ m.comments_count }}
                                </span>
                            </div>
                        </div>
                        <div v-if="hasMoreMoments" class="pt-3">
                            <UButton
                                block
                                :loading="loadingMoreMoments"
                                @click="loadMoreMoments"
                            >
                                加载更多
                            </UButton>
                        </div>
                    </div>
                </template>
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
definePageMeta({ title: "地图", requiresAuth: true });

import { DEFAULTS as defaults } from "~/config/defaults";
import { useUserService } from "~/services/user.service";
import type { UserInfo, GroupInfo, MomentInfo } from "~/types";
const EmptyNearby = defineAsyncComponent(() => import("~/components/illustrations/EmptyNearby.vue"));

// stores & services
const groupsStore = useGroupsStore();
const momentsStore = useMomentsStore();
const userService = useUserService();

// geolocation & permissions
const {
    coordinates,
    getCurrentPosition,
    reverseGeocode,
    formatDistance,
    watchPosition,
    clearWatch,
} = useGeolocation();
const { checkGeolocation, requestGeolocation } = usePermissions();

// UI state
const toast = useSmartToast();
const isLoading = ref(false);
const mapEl = ref<HTMLDivElement | null>(null);
let map: any = null;
let markers: any[] = [];
const isLocationSharing = ref(false);
const permissionDenied = ref(false);
const loadingUsers = ref(false);
const loadingMoreUsers = ref(false);
const loadingGroups = ref(false);
const loadingMoreGroups = ref(false);
const loadingMoments = ref(false);
const loadingMoreMoments = ref(false);

// 筛选状态
const activeFilters = ref<('users' | 'groups' | 'moments')[]>(['users']);
const currentTab = ref<'users' | 'groups' | 'moments'>('users');

// derived
const coords = computed(() => coordinates.value);
// 地图 SDK 按需启用
const config = useRuntimeConfig();
const amapEnabled = computed(() => !!config.public.amapKey);

async function ensureMap() {
    if (!amapEnabled.value || !mapEl.value) return;
    if (map) return;
    // 动态加载 AMap JS SDK（不引入额外依赖，简单 script 注入）
    await new Promise<void>((resolve, reject) => {
        const id = "amap-sdk";
        if (document.getElementById(id)) return resolve();
        const s = document.createElement("script");
        s.id = id;
        s.src = `https://webapi.amap.com/maps?v=2.0&key=${encodeURIComponent(config.public.amapKey as string)}`;
        s.async = true;
        s.onload = () => resolve();
        s.onerror = () => reject(new Error("AMap SDK 加载失败"));
        document.head.appendChild(s);
    });
    // @ts-ignore 全局 AMap
    map = new (window as any).AMap.Map(mapEl.value, {
        zoom: 14,
        center: coords.value
            ? [coords.value.longitude, coords.value.latitude]
            : undefined,
        viewMode: "2D",
    });
}

function clearMarkers() {
    if (!map || !markers.length) return;
    // @ts-ignore
    map.remove(markers);
    markers = [];
}

function renderMarkers() {
    if (!map) return;
    clearMarkers();
    const points: Array<{
        lng: number;
        lat: number;
        label: string;
        color: string;
        type: string;
        data: any;
    }> = [];
    
    // 用户点 - 仅在筛选器激活时显示
    if (activeFilters.value.includes('users')) {
        for (const u of nearbyUsers.value) {
            if (typeof u.longitude === "number" && typeof u.latitude === "number") {
                points.push({
                    lng: u.longitude,
                    lat: u.latitude,
                    label: u.nickname || String(u.user_id),
                    color: "#10B981",
                    type: 'user',
                    data: u
                });
            }
        }
    }
    
    // 群组点 - 仅在筛选器激活时显示
    if (activeFilters.value.includes('groups')) {
        for (const g of nearbyGroups.value) {
            if (
                typeof (g as any).longitude === "number" &&
                typeof (g as any).latitude === "number"
            ) {
                points.push({
                    lng: (g as any).longitude,
                    lat: (g as any).latitude,
                    label: g.name,
                    color: "#6366F1",
                    type: 'group',
                    data: g
                });
            }
        }
    }
    
    // 动态点 - 仅在筛选器激活时显示
    if (activeFilters.value.includes('moments')) {
        for (const m of nearbyMoments.value) {
            if (typeof m.longitude === "number" && typeof m.latitude === "number") {
                points.push({
                    lng: m.longitude,
                    lat: m.latitude,
                    label: m.nickname || `动态`,
                    color: "#F59E0B",
                    type: 'moment',
                    data: m
                });
            }
        }
    }
    
    // @ts-ignore
    const AMap = (window as any).AMap;
    for (const p of points) {
        const m = new AMap.Marker({
            position: [p.lng, p.lat],
            anchor: "bottom-center",
            content: `<div style="background:${p.color};color:white;padding:2px 6px;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,.25);font-size:12px;white-space:nowrap;transform:translateY(-4px)">${p.label}</div>`,
        });
        
        // 添加点击事件
        m.on('click', () => {
            handleMarkerClick(p);
        });
        
        // @ts-ignore
        map.add(m);
        markers.push(m);
    }
}
const nearbyUsers = ref<UserInfo[]>([]);
const nearbyGroups = computed<GroupInfo[]>(() => groupsStore.nearbyGroups);
const nearbyMoments = computed<MomentInfo[]>(() => momentsStore.nearbyMoments);

const hasMoreUsers = ref(false);
const nextUsersCursor = ref<number | undefined>(undefined);
const hasMoreGroups = computed(() => groupsStore.pagination.nearbyGroups.hasMore);
const hasMoreMoments = computed(() => momentsStore.pagination.nearbyMoments.hasMore);

// 计算属性
const currentTabTitle = computed(() => {
    switch (currentTab.value) {
        case 'users':
            return `附近用户 (${nearbyUsers.value.length})`;
        case 'groups':
            return `附近群组 (${nearbyGroups.value.length})`;
        case 'moments':
            return `附近动态 (${nearbyMoments.value.length})`;
        default:
            return '附近内容';
    }
});

const loadingCurrent = computed(() => {
    switch (currentTab.value) {
        case 'users':
            return loadingUsers.value;
        case 'groups':
            return loadingGroups.value;
        case 'moments':
            return loadingMoments.value;
        default:
            return false;
    }
});

// helpers
const ensureLocation = async () => {
    const state = await checkGeolocation().catch(() => "prompt");
    if (state !== "granted") {
        const granted = await requestGeolocation();
        if (!granted) {
            permissionDenied.value = true;
            throw new Error("未授权位置权限");
        }
    }
    const pos = await getCurrentPosition();
    try {
        const name = await reverseGeocode(pos.latitude, pos.longitude);
        await userService.updateLocation(pos.latitude, pos.longitude, name);
    } catch {
        /* 后端更新失败不阻断 */
    }
    return pos;
};

const refreshLocation = async () => {
    isLoading.value = true;
    try {
        const pos = await ensureLocation();
        await ensureMap();
        if (map && pos) {
            // @ts-ignore
            map.setCenter([pos.longitude, pos.latitude]);
        }
        await Promise.all([refreshUsers(pos), refreshGroups(pos), refreshMoments(pos)]);
        toast.success("位置已更新");
        renderMarkers();
    } catch (e: any) {
        toast.error({
            title: "无法刷新位置",
            description: e?.message
        });
    } finally {
        isLoading.value = false;
    }
};

const radiusMeters = ref<number>(defaults.nearbyRadiusMeters);

const refreshUsers = async (pos?: { latitude: number; longitude: number }) => {
    loadingUsers.value = true;
    try {
        const p = pos || (await getCurrentPosition());
        const response = await userService.findNearbyUsers({
            latitude: p.latitude,
            longitude: p.longitude,
            radius: radiusMeters.value,
        });
        nearbyUsers.value = response.items;
        hasMoreUsers.value = response.has_more;
        nextUsersCursor.value = response.next_cursor;
    } catch (e: any) {
        toast.add({
            title: "拉取附近用户失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        loadingUsers.value = false;
    }
};

const loadMoreUsers = async () => {
    loadingMoreUsers.value = true;
    try {
        const p = coords.value || (await getCurrentPosition());
        const response = await userService.findNearbyUsers({
            latitude: p.latitude,
            longitude: p.longitude,
            radius: radiusMeters.value,
            cursor: nextUsersCursor.value,
        });
        nearbyUsers.value.push(...response.items);
        hasMoreUsers.value = response.has_more;
        nextUsersCursor.value = response.next_cursor;
    } catch (e: any) {
        toast.add({
            title: "加载更多用户失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        loadingMoreUsers.value = false;
    }
};

const refreshGroups = async (pos?: { latitude: number; longitude: number }) => {
    loadingGroups.value = true;
    try {
        const p = pos || (await getCurrentPosition());
        await groupsStore.searchNearbyGroups({
            latitude: p.latitude,
            longitude: p.longitude,
            radius: radiusMeters.value,
        });
    } catch (e: any) {
        toast.add({
            title: "拉取附近群组失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        loadingGroups.value = false;
    }
};

const loadMoreGroups = async () => {
    loadingMoreGroups.value = true;
    try {
        const p = coords.value || (await getCurrentPosition());
        await groupsStore.searchNearbyGroups({
            latitude: p.latitude,
            longitude: p.longitude,
            radius: radiusMeters.value,
            loadMore: true,
        });
    } catch (e: any) {
        toast.add({
            title: "加载更多群组失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        loadingMoreGroups.value = false;
    }
};

const refreshMoments = async (pos?: { latitude: number; longitude: number }) => {
    loadingMoments.value = true;
    try {
        const p = pos || (await getCurrentPosition());
        await momentsStore.fetchNearbyMoments({
            latitude: p.latitude,
            longitude: p.longitude,
            radius: radiusMeters.value,
        });
    } catch (e: any) {
        toast.add({
            title: "拉取附近动态失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        loadingMoments.value = false;
    }
};

const loadMoreMoments = async () => {
    loadingMoreMoments.value = true;
    try {
        const p = coords.value || (await getCurrentPosition());
        await momentsStore.fetchNearbyMoments({
            latitude: p.latitude,
            longitude: p.longitude,
            radius: radiusMeters.value,
            loadMore: true,
        });
    } catch (e: any) {
        toast.add({
            title: "加载更多动态失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        loadingMoreMoments.value = false;
    }
};

// 交互
const goChatUser = (userId: number) => navigateTo(`/chat/user/${userId}`);
const enterGroupOrJoin = async (g: GroupInfo) => {
    try {
        if (!g.is_member) {
            await groupsStore.joinGroup(g.group_id);
            toast.success("加入成功");
            await refreshGroups();
        }
        navigateTo(`/chat/group/${g.group_id}`);
    } catch (e: any) {
        toast.add({
            title: "操作失败",
            description: e?.message,
            color: "error",
        });
    }
};

// 位置共享：开始/停止监听
let watchId: number | null = null;
const toggleLocationSharing = async () => {
    if (isLocationSharing.value) {
        if (watchId != null) {
            clearWatch(watchId);
            watchId = null;
        }
        isLocationSharing.value = false;
        toast.info("位置共享已关闭");
        return;
    }

    try {
        const pos = await ensureLocation();
        await ensureMap();
        watchId = watchPosition(async (coords) => {
            try {
                await userService.updateLocation(
                    coords.latitude,
                    coords.longitude,
                    await reverseGeocode(coords.latitude, coords.longitude),
                );
            } catch {}
            // 增量刷新附近数据（可按需节流，这里简单触发刷新）
            const [usersResponse] = await Promise.all([
                userService.findNearbyUsers({
                    latitude: coords.latitude,
                    longitude: coords.longitude,
                    radius: radiusMeters.value,
                }),
                groupsStore.searchNearbyGroups({
                    latitude: coords.latitude,
                    longitude: coords.longitude,
                    radius: radiusMeters.value,
                }),
                momentsStore.fetchNearbyMoments({
                    latitude: coords.latitude,
                    longitude: coords.longitude,
                    radius: radiusMeters.value,
                })
            ]);
            nearbyUsers.value = usersResponse.items;
            hasMoreUsers.value = usersResponse.has_more;
            nextUsersCursor.value = usersResponse.next_cursor;
            if (map) {
                // @ts-ignore
                map.setCenter([coords.longitude, coords.latitude]);
                renderMarkers();
            }
        });
        isLocationSharing.value = true;
        toast.info("位置共享已开启");
    } catch (e: any) {
        permissionDenied.value = true;
        toast.error({
            title: "无法开启共享",
            description: e?.message
        });
    }
};

// 页面挂载：不自动请求权限，仅已授权时静默加载一次
onMounted(async () => {
    try {
        const state = await checkGeolocation().catch(() => "prompt");
        if (state === "granted") {
            try {
                const pos = await getCurrentPosition();
                try {
                    await userService.updateLocation(
                        pos.latitude,
                        pos.longitude,
                        await reverseGeocode(pos.latitude, pos.longitude),
                    );
                } catch {}
                await ensureMap();
                if (map && pos) {
                    // @ts-ignore
                    map.setCenter([pos.longitude, pos.latitude]);
                }
                const [usersResponse] = await Promise.all([
                    userService.findNearbyUsers({
                        latitude: pos.latitude,
                        longitude: pos.longitude,
                        radius: radiusMeters.value,
                    }),
                    groupsStore.searchNearbyGroups({
                        latitude: pos.latitude,
                        longitude: pos.longitude,
                        radius: radiusMeters.value,
                    }),
                ]);
                nearbyUsers.value = usersResponse.items;
                hasMoreUsers.value = usersResponse.has_more;
                nextUsersCursor.value = usersResponse.next_cursor;
                renderMarkers();
            } catch (error: any) {
                console.warn('[Map] Failed to load nearby data:', error);
                // 静默失败，用户可以手动刷新
            }
        }
    } catch (error: any) {
        console.error('[Map] Failed to initialize map:', error);
        toast.add({
            title: '地图初始化失败',
            description: error?.message || '无法初始化地图，请稍后重试',
            color: 'error'
        });
    }
});

// 筛选管理
function toggleFilter(filterType: 'users' | 'groups' | 'moments') {
    const index = activeFilters.value.indexOf(filterType);
    if (index > -1) {
        activeFilters.value.splice(index, 1);
        // 如果当前tab被取消筛选，切换到其他可用tab
        if (currentTab.value === filterType && activeFilters.value.length > 0) {
            const firstFilter = activeFilters.value[0];
            if (firstFilter) {
                currentTab.value = firstFilter;
            }
        } else if (activeFilters.value.length === 0) {
            // 如果没有筛选项，默认显示用户
            activeFilters.value.push('users');
            currentTab.value = 'users';
        }
    } else {
        activeFilters.value.push(filterType);
        currentTab.value = filterType;
    }
    
    // 重新渲染地图标记
    renderMarkers();
}

// 刷新当前内容
async function refreshCurrentContent() {
    switch (currentTab.value) {
        case 'users':
            await refreshUsers();
            break;
        case 'groups':
            await refreshGroups();
            break;
        case 'moments':
            await refreshMoments();
            break;
    }
    renderMarkers();
}

// 标记点击处理
function handleMarkerClick(point: { type: string; data: any }) {
    switch (point.type) {
        case 'user':
            showUserDetail(point.data);
            break;
        case 'group':
            showGroupDetail(point.data);
            break;
        case 'moment':
            showMomentDetail(point.data);
            break;
    }
}

// 详情显示函数
function showUserDetail(user: UserInfo) {
    // 可以显示用户详情弹窗，这里简单导航到聊天
    navigateTo(`/chat/user/${user.user_id}`);
}

function showGroupDetail(group: GroupInfo) {
    // 可以显示群组详情弹窗，这里简单导航到群组页面
    navigateTo(`/groups/${group.group_id}`);
}

function showMomentDetail(moment: MomentInfo) {
    // 导航到动态详情页面
    navigateTo(`/moments/${moment.moment_id}`);
}

// 时间格式化
function formatTime(s: string) {
    const t = new Date(s);
    const d = Date.now() - t.getTime();
    const m = Math.floor(d / 60000);
    const h = Math.floor(d / 3600000);
    const day = Math.floor(d / 86400000);
    
    if (m < 60) return `${m}分钟前`;
    if (h < 24) return `${h}小时前`;
    return `${day}天前`;
}

// 监听筛选变化，自动切换tab到第一个激活的筛选项
watch(activeFilters, (newFilters) => {
    if (newFilters.length > 0 && !newFilters.includes(currentTab.value)) {
        const firstFilter = newFilters[0];
        if (firstFilter) {
            currentTab.value = firstFilter;
        }
    }
}, { deep: true });

// 距离显示：米/千米统一走 geolocation.formatDistance
const formatDistanceMeters = (d?: number) =>
    typeof d === "number" ? formatDistance(d) : "-";
const formatDistanceKm = (d?: number) =>
    typeof d === "number" ? formatDistance(d) : "-";
</script>
