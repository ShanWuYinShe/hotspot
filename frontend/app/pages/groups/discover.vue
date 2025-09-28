<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <UCard>
            <template #header>
                <div class="flex items-center justify-between">
                    <div>
                        <h1 class="text-3xl font-bold text-neutral-900">
                            发现群组
                        </h1>
                        <p class="text-neutral-600 dark:text-neutral-300">
                            基于你的位置展示附近可加入的群组
                        </p>
                    </div>
                    <UButton
                        variant="outline"
                        size="lg"
                        icon="i-heroicons-arrow-path"
                        :loading="loading"
                        @click="loadNearbyGroups"
                        >重新搜索</UButton
                    >
                </div>
            </template>

            <!-- 搜索过滤器 -->
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                <UInput
                    v-model="searchQuery"
                    placeholder="搜索群组名称..."
                    icon="i-heroicons-magnifying-glass"
                    @input="searchGroups"
                />
                <USelect
                    v-model="categoryFilter"
                    :items="categoryOptions"
                    placeholder="选择分类"
                    @change="filterGroups"
                />
                <USelect
                    v-model="sortBy"
                    :items="sortOptions"
                    placeholder="排序方式"
                    @change="sortGroups"
                />
            </div>

            <div class="space-y-4">
                <USkeleton v-if="loading" class="h-32" />
            </div>
            <div v-if="!loading">
                <UAlert
                    v-if="permissionDenied"
                    color="warning"
                    variant="soft"
                    title="已拒绝位置权限"
                    class="max-w-2xl mb-4"
                >
                    <template #description>
                        <div class="space-y-2">
                            <p class="text-sm text-neutral-600">
                                你可以在浏览器地址栏的位置图标中重新允许权限，或点击下方按钮再次尝试。
                            </p>
                            <UButton
                                size="xs"
                                variant="outline"
                                @click="authorizeAndSearch"
                                >再次尝试</UButton
                            >
                            <div class="text-xs text-neutral-500 mt-2">
                                提示：为避免每次弹窗，请在浏览器权限提示中勾选“记住此决定”。
                            </div>
                        </div>
                    </template>
                </UAlert>

                <div v-if="items.length > 0">
                    <UTable :rows="items" :columns="columns">
                        <template #group-cell="{ row }">
                            <div class="flex items-center gap-3 min-w-[240px]">
                                <UAvatar :alt="(row as any).name">{{
                                    (row as any).name?.charAt(0)
                                }}</UAvatar>
                                <div class="min-w-0">
                                    <div class="flex items-center gap-2">
                                        <span class="font-medium truncate">{{
                                            (row as any).name
                                        }}</span>
                                        <UTooltip
                                            v-if="(row as any).is_password_required"
                                            text="私有群组，需要密码或邀请才能加入"
                                        >
                                            <UBadge
                                                variant="soft"
                                                size="xs"
                                            >私有</UBadge>
                                        </UTooltip>
                                    </div>
                                    <p
                                        class="text-xs text-neutral-500 truncate"
                                    >
                                        {{ (row as any).description || "—" }}
                                    </p>
                                </div>
                            </div>
                        </template>
                        <template #members-cell="{ row }">
                            <span
                                class="text-sm text-neutral-600 dark:text-neutral-400 flex items-center gap-1"
                            >
                                <UIcon
                                    name="i-heroicons-users"
                                    class="w-4 h-4"
                                />
                                {{ (row as any).member_count }} 成员
                            </span>
                        </template>
                        <template #distance-cell="{ row }">
                            <span
                                class="text-sm text-neutral-600 dark:text-neutral-400"
                                >{{
                                    (
                                        ((row as any).distance || 0) / 1000
                                    ).toFixed(1)
                                }}
                                km</span
                            >
                        </template>
                        <template #actions-cell="{ row }">
                            <div class="flex items-center justify-end gap-2">
                                <UButton
                                    size="sm"
                                    :disabled="(row as any).is_member"
                                    @click="joinGroup(row as any)"
                                    >{{
                                        (row as any).is_member
                                            ? "已加入"
                                            : "加入"
                                    }}</UButton
                                >
                            </div>
                        </template>
                        <template #empty>
                            <div class="py-10">
                                <EmptyState
                                    v-if="!loadError"
                                    icon="i-heroicons-map-pin"
                                    title="暂无附近群组"
                                    description="授权位置后可展示附近可加入的群组"
                                    action-text="允许位置访问并搜索"
                                    :action-handler="authorizeAndSearch"
                                >
                                    <template #illustration>
                                        <EmptyGroups size="lg" />
                                    </template>
                                </EmptyState>
                                <UAlert
                                    v-else
                                    color="warning"
                                    variant="soft"
                                    title="搜索失败"
                                >
                                    <template #description>
                                        <div class="space-y-2">
                                            <p class="text-sm text-neutral-600">
                                                请检查网络或位置权限后重试。
                                            </p>
                                            <UButton
                                                size="xs"
                                                variant="outline"
                                                @click="authorizeAndSearch"
                                                >再次尝试</UButton
                                            >
                                        </div>
                                    </template>
                                </UAlert>
                            </div>
                        </template>
                    </UTable>
                </div>

                <div v-else>
                    <EmptyState
                        v-if="!loadError"
                        icon="i-heroicons-map-pin"
                        title="暂无附近群组"
                        description="授权位置后可展示附近可加入的群组"
                        action-text="允许位置访问并搜索"
                        :action-handler="authorizeAndSearch"
                    >
                        <template #illustration>
                            <EmptyGroups size="lg" />
                        </template>
                        <template #actions>
                            <div class="flex flex-col items-center gap-3">
                                <UButton
                                    color="primary"
                                    @click="authorizeAndSearch"
                                    >允许位置访问并搜索</UButton
                                >
                                <UButton
                                    variant="ghost"
                                    size="xs"
                                    @click="showWhyPermission"
                                    >为什么需要？</UButton
                                >
                                <div class="text-xs text-neutral-500">
                                    提示：为避免每次弹窗，请在浏览器权限提示中勾选“记住此决定”。
                                </div>
                            </div>
                        </template>
                    </EmptyState>
                </div>
            </div>
        </UCard>
    </div>
    <!-- 权限说明对话框改为程序化 useOverlay 管理（见脚本中的 showWhyPermission） -->
</template>

<script setup lang="ts">
import { defineAsyncComponent } from "vue";
// TableColumn 类型已在组件内部定义
import type { GroupResponse } from "~/types/api";
definePageMeta({ title: "发现群组", requiresAuth: true, layout: 'dashboard' });

const EmptyGroups = defineAsyncComponent(
    () => import("~/components/illustrations/EmptyGroups.vue"),
);
const groupsStore = useGroupsStore();
// 本地UI配置，不需要从设置中获取
const preferCardOnDenied = ref(true);

        // 移除已删除的 modal 函数
const showWhyPermission = async () => {
    if (confirm("根据您的当前位置展示附近的群组。仅在你同意后请求权限，可随时在浏览器设置中更改。\n\n点击确定允许位置访问并搜索")) {
        await authorizeAndSearch();
    }
};
const { success, error, info } = useNotifications()
const toast = { success, error, info };
const locationManager = useLocationManager();
const { checkGeolocation, requestGeolocation, onGeolocationChange } =
    useDeviceAccess();

const items = ref<GroupResponse[]>([]);
const loading = ref(false);
const loadError = ref(false);
const whyOpen = ref(false);
const permissionDenied = ref(false);

// 搜索和过滤功能
const searchQuery = ref("");
const categoryFilter = ref("all");
const sortBy = ref("distance");

// 选项数据
const categoryOptions = [
    { label: "全部分类", value: "all" },
    { label: "学习交流", value: "study" },
    { label: "兴趣爱好", value: "hobby" },
    { label: "生活社区", value: "life" },
    { label: "工作职场", value: "work" },
    { label: "运动健身", value: "sport" },
    { label: "其他", value: "other" },
];

const sortOptions = [
    { label: "按距离排序", value: "distance" },
    { label: "按成员数排序", value: "members" },
    { label: "按创建时间排序", value: "created" },
    { label: "按活跃度排序", value: "activity" },
];

// UTable 列
const columns = [
    { id: "group", label: "群组" },
    {
        id: "members",
        label: "成员数",
        class: "hidden sm:table-cell w-32",
    },
    {
        id: "distance",
        label: "距离",
        class: "hidden md:table-cell w-32",
    },
    { id: "actions", label: "操作", class: "w-28 text-right" },
];

// 虚拟化窗口逻辑移除，统一表格渲染

const loadNearbyGroups = async () => {
    loading.value = true;
    loadError.value = false;
    try {
        // 获取当前位置（优先使用位置管理器的缓存位置）
        let location = locationManager.getLocation();

        // 如果没有缓存位置，尝试获取当前位置
        if (!location) {
            try {
                location = await locationManager.getLocationWithFallback();
            } catch (error) {
                // 如果获取失败，使用默认位置
                location = locationManager.getLocationOrDefault();
                toast.add({
                    title: "位置获取失败",
                    description: "使用默认位置进行搜索",
                    color: "warning",
                });
            }
        }

        const resp = await groupsStore.searchNearbyGroups({
            latitude: location.latitude,
            longitude: location.longitude,
            radius: 5000,
        });
        items.value = resp;
    } catch (e: any) {
        loadError.value = true;
        toast.add({
            title: "搜索失败",
            description: e?.message || "搜索群组时发生错误",
            color: "error",
        });
    } finally {
        loading.value = false;
    }
};

// 搜索、过滤和排序方法
const searchGroups = () => {
    filterAndSortGroups();
};

const filterGroups = () => {
    filterAndSortGroups();
};

const sortGroups = () => {
    filterAndSortGroups();
};

const filterAndSortGroups = () => {
    let filtered = [...items.value];

    if (searchQuery.value.trim()) {
        const query = searchQuery.value.toLowerCase();
        filtered = filtered.filter(
            (group) =>
                group.name.toLowerCase().includes(query) ||
                (group.description &&
                    group.description.toLowerCase().includes(query)),
        );
    }

    if (categoryFilter.value && categoryFilter.value !== "all") {
        // 预留分类过滤
    }

    switch (sortBy.value) {
        case "distance":
            filtered.sort((a, b) => (a.distance || 0) - (b.distance || 0));
            break;
        case "members":
            filtered.sort(
                (a, b) => (b.member_count || 0) - (a.member_count || 0),
            );
            break;
        case "created":
            filtered.sort(
                (a, b) =>
                    new Date(b.created_at || 0).getTime() -
                    new Date(a.created_at || 0).getTime(),
            );
            break;
        case "activity":
            filtered.sort(
                (a, b) => (b.member_count || 0) - (a.member_count || 0),
            );
            break;
    }
    items.value = filtered;
};

onMounted(async () => {
    const state = await checkGeolocation().catch(() => "prompt");
    if (state === "granted") {
        await loadNearbyGroups();
    }
});

const joinGroup = async (g: GroupResponse) => {
    try {
        const full = await groupsStore.joinGroup(g.group_id);
        await groupsStore.fetchMyGroups();
        toast.add({
            title: "加入成功",
            description: `已加入群组 ${g.name}`,
            color: "success",
        });
        navigateTo(`/chat/groups/${g.group_id}`);
    } catch (e: unknown) {
        toast.add({
            title: "加入失败",
            description: (e as any)?.message,
            color: "error",
        });
    }
};

const authorizeAndSearch = async () => {
    const state = await checkGeolocation();
    if (state === "granted") {
        whyOpen.value = false;
        await loadNearbyGroups();
        return;
    }
    const granted = await requestGeolocation();
    if (!granted) {
        permissionDenied.value = true;
        return;
    }
    whyOpen.value = false;
    await loadNearbyGroups();
};

onMounted(() => {
    const handler = async () => {
        if (document.visibilityState === "visible") {
            const state = await checkGeolocation().catch(() => "prompt");
            if (state !== "granted") {
                permissionDenied.value = state === "denied";
            }
        }
    };
    document.addEventListener("visibilitychange", handler);
    onUnmounted(() =>
        document.removeEventListener("visibilitychange", handler),
    );
    let stopPerm: (() => void) | null = null;
    onGeolocationChange((state) => {
        if (state !== "granted") {
            permissionDenied.value = state === "denied";
        }
    })
        .then((stop) => {
            stopPerm = stop;
        })
        .catch(() => {});
    onUnmounted(() => {
        try {
            stopPerm?.();
        } catch {}
    });
});
</script>
