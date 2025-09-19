<template>
    <UModal
        title="搜索"
        :overlay="false"
        :transition="false"
        class="search-modal"
    >
        <template #default></template>

        <template #body>
            <UCommandPalette
                v-model="searchQuery"
                :groups="searchGroups"
                :loading="isSearching"
                placeholder="搜索用户、群组、动态..."
                :empty-state="emptyState"
                @update:search-term="handleSearch"
                @close="handleClose"
            >
                <template #default="{ item }">
                    <div class="flex items-center space-x-3">
                        <UAvatar
                            v-if="item.type === 'user'"
                            :alt="item.label"
                            size="sm"
                        >
                            {{ item.label?.charAt(0) || "U" }}
                        </UAvatar>
                        <div
                            v-else-if="item.type === 'group'"
                            class="w-8 h-8 bg-blue-100 dark:bg-blue-900/50 rounded-lg flex items-center justify-center"
                        >
                            <UIcon
                                name="i-heroicons-user-group"
                                class="w-4 h-4 text-blue-600 dark:text-blue-400"
                            />
                        </div>
                        <div
                            v-else-if="item.type === 'moment'"
                            class="w-8 h-8 bg-purple-100 dark:bg-purple-900/50 rounded-lg flex items-center justify-center"
                        >
                            <UIcon
                                name="i-heroicons-photo"
                                class="w-4 h-4 text-purple-600 dark:text-purple-400"
                            />
                        </div>
                        <div
                            v-else
                            class="w-8 h-8 bg-neutral-100 dark:bg-neutral-800 rounded-lg flex items-center justify-center"
                        >
                            <UIcon
                                :name="
                                    item.icon || 'i-heroicons-magnifying-glass'
                                "
                                class="w-4 h-4"
                            />
                        </div>

                        <div class="flex-1">
                            <p class="font-medium text-sm">{{ item.label }}</p>
                            <p
                                v-if="item.description"
                                class="text-xs text-neutral-500"
                            >
                                {{ item.description }}
                            </p>
                        </div>

                        <UBadge
                            v-if="item.badge"
                            size="xs"
                            :color="item.badgeColor || 'neutral'"
                        >
                            {{ item.badge }}
                        </UBadge>
                    </div>
                </template>
            </UCommandPalette>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import { useDebounceFn } from "@vueuse/core";

// 导入服务

const userService = useUserService();
const groupService = useGroupService();
const momentService = useMomentService();

// 导入统一的格式化工具函数

interface SearchItem {
    id: string;
    label: string;
    description?: string;
    type: "user" | "group" | "moment" | "action";
    icon?: string;
    badge?: string;
    badgeColor?: string;
    to?: string;
    click?: () => void;
}

interface SearchGroup {
    id: string;
    key: string;
    label: string;
    commands: SearchItem[];
}

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

const searchQuery = ref("");
const isSearching = ref(false);

// 空状态配置
const emptyState = computed(() => ({
    icon: "i-heroicons-magnifying-glass",
    label: "没有找到结果",
    queryLabel: searchQuery.value
        ? `没有找到与 "${searchQuery.value}" 相关的结果`
        : "开始搜索",
}));

// 搜索结果分组
const searchGroups = ref<SearchGroup[]>([
    {
        id: "actions",
        key: "actions",
        label: "快速操作",
        commands: [
            {
                id: "create-group",
                label: "创建群组",
                description: "创建一个新的群组",
                type: "action",
                icon: "i-heroicons-plus",
                click: () => {
                    handleClose();
                    navigateTo("/groups/create");
                },
            },
            {
                id: "create-moment",
                label: "发布动态",
                description: "分享你的精彩瞬间",
                type: "action",
                icon: "i-heroicons-camera",
                click: () => {
                    handleClose();
                    // 发出事件让父组件处理创建动态
                    emit('create-moment');
                },
            },
            {
                id: "users-nearby",
                label: "附近用户",
                description: "发现周围的朋友",
                type: "action",
                icon: "i-heroicons-users",
                click: () => {
                    handleClose();
                    navigateTo("/users/nearby");
                },
            },
        ],
    },
    {
        id: "users",
        key: "users",
        label: "用户",
        commands: [],
    },
    {
        id: "groups",
        key: "groups",
        label: "群组",
        commands: [],
    },
    {
        id: "moments",
        key: "moments",
        label: "动态",
        commands: [],
    },
]);

// 搜索处理
const handleSearch = useDebounceFn(async (query: string) => {
    if (!query.trim()) {
        // 重置为默认快速操作
        if (searchGroups.value[1]) searchGroups.value[1].commands = [];
        if (searchGroups.value[2]) searchGroups.value[2].commands = [];
        if (searchGroups.value[3]) searchGroups.value[3].commands = [];
        return;
    }

    isSearching.value = true;

    try {
        // 执行真实搜索

        // 真实的用户搜索
        const userResults: SearchItem[] = [];
        try {
            const userResponse = await userService.searchUsers(query, undefined);
            if (userResponse.items) {
                userResults.push(...userResponse.items.map((user: any) => ({
                    id: `user-${user.user_id}`,
                    label: user.nickname,
                    description: user.distance ? `附近 ${Math.round(user.distance)}m` : '位置未知',
                    type: "user" as const,
                    badge: user.last_active_at ? "最近活跃" : undefined,
                    badgeColor: "green" as const,
                    click: () => {
                        handleClose();
                        navigateTo(`/users/${user.user_id}`);
                    },
                })));
            }
        } catch (error) {
            console.warn('用户搜索失败:', error);
        }

        // 真实的群组搜索
        const groupResults: SearchItem[] = [];
        try {
            const groupResponse = await groupService.searchGroupsByName(query, undefined);
            if (groupResponse.items) {
                groupResults.push(...groupResponse.items.map((group: any) => ({
                    id: `group-${group.group_id}`,
                    label: group.name,
                    description: `${group.member_count} 名成员`,
                    type: "group" as const,
                    badge: group.is_member ? "已加入" : "可加入",
                    badgeColor: group.is_member ? "blue" : "gray" as const,
                    click: () => {
                        handleClose();
                        navigateTo(`/groups/${group.group_id}`);
                    },
                })));
            }
        } catch (error) {
            console.warn('群组搜索失败:', error);
        }

        // 真实的生活瞬间搜索
        const momentResults: SearchItem[] = [];
        try {
            const momentResponse = await momentService.searchMoments({
                type: "tags",
                tags: [query]
            }, undefined, 5);
            if (momentResponse.items) {
                momentResults.push(...momentResponse.items.map((moment: any) => ({
                    id: `moment-${moment.moment_id}`,
                    label: moment.description || '无标题',
                    description: `${moment.nickname} · ${formatTime(moment.created_at)}`,
                    type: "moment" as const,
                    badge: moment.likes_count > 10 ? "热门" : undefined,
                    badgeColor: "orange" as const,
                    click: () => {
                        handleClose();
                        navigateTo(`/moments/${moment.moment_id}`);
                    },
                })));
            }
        } catch (error) {
            console.warn('生活瞬间搜索失败:', error);
        }

        // 更新搜索结果
        if (searchGroups.value[1]) searchGroups.value[1].commands = userResults;
        if (searchGroups.value[2]) searchGroups.value[2].commands = groupResults;
        if (searchGroups.value[3]) searchGroups.value[3].commands = momentResults;
    } catch (error) {
        console.error("搜索失败:", error);
    } finally {
        isSearching.value = false;
    }
}, 300);

// 定义 emit
const emit = defineEmits<{
    close: [];
    'create-moment': [];
}>();

// 关闭弹窗 - useOverlay 会自动管理关闭状态
const handleClose = () => {
    // 重置搜索状态
    searchQuery.value = "";
    if (searchGroups.value[1]) searchGroups.value[1].commands = [];
    if (searchGroups.value[2]) searchGroups.value[2].commands = [];
    if (searchGroups.value[3]) searchGroups.value[3].commands = [];
    
    // 通过 emit 通知父组件关闭
    emit('close');
};

// 监听搜索查询变化
watch(searchQuery, handleSearch);
</script>

<style scoped>
.search-modal :deep(.command-palette) {
    border: none;
    box-shadow: none;
}
</style>
