<template>
    <UModal
        title="搜索"
        class="search-modal"
    >
        <template #body>
            <UCommandPalette
                v-model="searchQuery"
                :groups="searchGroups"
                :loading="isSearching"
                placeholder="搜索用户、群组、动态... (输入 '/' 查看高级选项)"
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
            {
                id: "search-users",
                label: "搜索用户",
                description: "按用户类型、在线状态筛选",
                type: "action",
                icon: "i-heroicons-user-circle",
                click: () => {
                    handleClose();
                    navigateTo("/users/search");
                },
            },
            {
                id: "search-groups",
                label: "搜索群组",
                description: "按成员数、类型筛选群组",
                type: "action",
                icon: "i-heroicons-magnifying-glass",
                click: () => {
                    handleClose();
                    navigateTo("/groups");
                },
            },
            {
                id: "map-explore",
                label: "地图探索",
                description: "在地图上发现附近内容",
                type: "action",
                icon: "i-heroicons-map",
                click: () => {
                    handleClose();
                    navigateTo("/map");
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

    // 处理高级搜索命令
    if (query.startsWith('/')) {
        handleAdvancedCommands(query);
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

// 高级搜索命令处理
const handleAdvancedCommands = (query: string) => {
    const command = query.toLowerCase();
    const advancedCommands: SearchItem[] = [];

    // 用户相关命令
    if (command.includes('user') || command.includes('用户') || command === '/') {
        advancedCommands.push(
            {
                id: "users-online",
                label: "在线用户",
                description: "查看当前在线的用户",
                type: "action",
                icon: "i-heroicons-signal",
                click: () => {
                    handleClose();
                    navigateTo("/users/search?status=online");
                },
            },
            {
                id: "users-nearby",
                label: "附近用户",
                description: "基于位置搜索用户",
                type: "action",
                icon: "i-heroicons-map-pin",
                click: () => {
                    handleClose();
                    navigateTo("/users/nearby");
                },
            }
        );
    }

    // 群组相关命令
    if (command.includes('group') || command.includes('群组') || command === '/') {
        advancedCommands.push(
            {
                id: "groups-public",
                label: "公开群组",
                description: "浏览所有公开群组",
                type: "action",
                icon: "i-heroicons-globe-alt",
                click: () => {
                    handleClose();
                    navigateTo("/groups?filter=public");
                },
            },
            {
                id: "groups-owned",
                label: "我的群组",
                description: "管理我创建的群组",
                type: "action",
                icon: "i-heroicons-crown",
                click: () => {
                    handleClose();
                    navigateTo("/groups/owned");
                },
            }
        );
    }

    // 动态相关命令
    if (command.includes('moment') || command.includes('动态') || command === '/') {
        advancedCommands.push(
            {
                id: "moments-trending",
                label: "热门动态",
                description: "查看最受欢迎的动态",
                type: "action",
                icon: "i-heroicons-fire",
                click: () => {
                    handleClose();
                    navigateTo("/moments?sort=trending");
                },
            },
            {
                id: "moments-nearby",
                label: "附近动态",
                description: "查看附近的动态",
                type: "action",
                icon: "i-heroicons-map",
                click: () => {
                    handleClose();
                    navigateTo("/moments?filter=nearby");
                },
            }
        );
    }

    // 系统功能命令
    if (command.includes('setting') || command.includes('设置') || command === '/') {
        advancedCommands.push(
            {
                id: "settings-profile",
                label: "个人资料",
                description: "编辑个人信息和设置",
                type: "action",
                icon: "i-heroicons-user-circle",
                click: () => {
                    handleClose();
                    navigateTo("/settings/profile");
                },
            },
            {
                id: "notifications",
                label: "通知中心",
                description: "查看所有通知消息",
                type: "action",
                icon: "i-heroicons-bell",
                click: () => {
                    handleClose();
                    navigateTo("/notifications");
                },
            }
        );
    }

    // 数据分析命令
    if (command.includes('dashboard') || command.includes('仪表板') || command.includes('统计') || command === '/') {
        advancedCommands.push(
            {
                id: "dashboard",
                label: "数据仪表板",
                description: "查看统计数据和分析",
                type: "action",
                icon: "i-heroicons-chart-bar",
                click: () => {
                    handleClose();
                    navigateTo("/dashboard");
                },
            }
        );
    }

    // 更新搜索结果为高级命令
    if (searchGroups.value[0]) {
        searchGroups.value[0].label = command === '/' ? '所有命令' : '匹配的命令';
        searchGroups.value[0].commands = advancedCommands;
    }
    
    // 清空其他分组
    if (searchGroups.value[1]) searchGroups.value[1].commands = [];
    if (searchGroups.value[2]) searchGroups.value[2].commands = [];
    if (searchGroups.value[3]) searchGroups.value[3].commands = [];
};

// 定义 emit
const emit = defineEmits<{
    close: [];
    'create-moment': [];
}>();

// 关闭弹窗 - useOverlay 会自动管理关闭状态
const handleClose = () => {
    // 重置搜索状态
    searchQuery.value = "";
    
    // 重置所有分组
    if (searchGroups.value[0]) {
        searchGroups.value[0].label = "快速操作";
        searchGroups.value[0].commands = [
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
            {
                id: "search-users",
                label: "搜索用户",
                description: "按用户类型、在线状态筛选",
                type: "action",
                icon: "i-heroicons-user-circle",
                click: () => {
                    handleClose();
                    navigateTo("/users/search");
                },
            },
            {
                id: "search-groups",
                label: "搜索群组",
                description: "按成员数、类型筛选群组",
                type: "action",
                icon: "i-heroicons-magnifying-glass",
                click: () => {
                    handleClose();
                    navigateTo("/groups");
                },
            },
            {
                id: "map-explore",
                label: "地图探索",
                description: "在地图上发现附近内容",
                type: "action",
                icon: "i-heroicons-map",
                click: () => {
                    handleClose();
                    navigateTo("/map");
                },
            },
        ];
    }
    
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
