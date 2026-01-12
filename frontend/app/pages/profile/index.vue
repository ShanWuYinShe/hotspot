<template>
    <div class="flex flex-col gap-6 pb-20">
        <!-- Header -->
        <div class="flex items-center justify-between shrink-0">
            <div>
                <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                    个人资料
                </h1>
            </div>
            <UButton
                label="编辑"
                icon="i-lucide-edit"
                variant="outline"
                color="neutral"
                @click="editProfile"
            />
        </div>

        <div class="max-w-4xl mx-auto w-full space-y-8">
            <!-- 头部信息 -->
            <UCard variant="subtle" class="overflow-hidden">
                <div
                    class="flex flex-col md:flex-row gap-8 items-center md:items-start"
                >
                    <div class="relative group shrink-0">
                        <UAvatar
                            :src="userProfile.avatar_url"
                            :alt="userProfile.nickname"
                            size="3xl"
                            class="ring-4 ring-white dark:ring-neutral-900 shadow-lg"
                        />
                        <!-- 简单的头像上传遮罩 -->
                        <div
                            class="absolute inset-0 flex items-center justify-center bg-black/50 rounded-full opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                            @click="triggerFileInput"
                        >
                            <UIcon
                                name="i-lucide-camera"
                                class="text-white size-8"
                            />
                            <input
                                ref="fileInput"
                                type="file"
                                accept="image/*"
                                class="hidden"
                                @change="handleFileChange"
                            />
                        </div>
                    </div>

                    <div class="flex-1 text-center md:text-left space-y-4">
                        <div>
                            <h1
                                class="text-3xl font-bold text-neutral-900 dark:text-white"
                            >
                                {{ userProfile.nickname }}
                            </h1>
                            <p
                                class="text-neutral-500 dark:text-neutral-400 mt-1 max-w-lg"
                            >
                                {{ userProfile.bio || "暂无简介" }}
                            </p>
                        </div>

                        <div
                            class="flex flex-wrap justify-center md:justify-start gap-2"
                        >
                            <UBadge
                                color="neutral"
                                variant="soft"
                                icon="i-lucide-map-pin"
                            >
                                {{ userProfile.location_name || "未知位置" }}
                            </UBadge>
                            <UBadge
                                color="neutral"
                                variant="soft"
                                icon="i-lucide-calendar"
                            >
                                注册于 {{ formatDate(userProfile.created_at) }}
                            </UBadge>
                        </div>

                        <div class="flex gap-4 justify-center md:justify-start">
                            <div
                                class="text-center cursor-pointer hover:opacity-80 transition-opacity"
                                @click="navigateTo('/moments')"
                            >
                                <div
                                    class="text-xl font-semibold text-neutral-900 dark:text-white"
                                >
                                    {{ stats.momentsCount }}
                                </div>
                                <div
                                    class="text-xs text-neutral-500 dark:text-neutral-400"
                                >
                                    动态
                                </div>
                            </div>
                            <div
                                class="text-center cursor-pointer hover:opacity-80 transition-opacity"
                                @click="navigateTo('/friends')"
                            >
                                <div
                                    class="text-xl font-semibold text-neutral-900 dark:text-white"
                                >
                                    {{ stats.friendsCount }}
                                </div>
                                <div
                                    class="text-xs text-neutral-500 dark:text-neutral-400"
                                >
                                    好友
                                </div>
                            </div>
                            <div
                                class="text-center cursor-pointer hover:opacity-80 transition-opacity"
                                @click="navigateTo('/groups')"
                            >
                                <div
                                    class="text-xl font-semibold text-neutral-900 dark:text-white"
                                >
                                    {{ stats.groupsCount }}
                                </div>
                                <div
                                    class="text-xs text-neutral-500 dark:text-neutral-400"
                                >
                                    群组
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </UCard>

            <!-- 内容区域 -->
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                <!-- 左侧：详细信息 -->
                <div class="space-y-6">
                    <UCard variant="soft">
                        <template #header
                            ><h3
                                class="font-semibold text-neutral-900 dark:text-white"
                            >
                                标签
                            </h3></template
                        >
                        <div class="flex flex-wrap gap-2">
                            <UBadge
                                v-for="tag in userProfile.tags"
                                :key="tag"
                                variant="outline"
                                color="neutral"
                            >
                                {{ tag }}
                            </UBadge>
                            <span
                                v-if="!userProfile.tags?.length"
                                class="text-sm text-neutral-500 dark:text-neutral-400"
                                >暂无标签</span
                            >
                        </div>
                    </UCard>

                    <UCard variant="soft">
                        <template #header
                            ><h3
                                class="font-semibold text-neutral-900 dark:text-white"
                            >
                                账号
                            </h3></template
                        >
                        <div class="space-y-2 text-sm">
                            <div class="flex justify-between">
                                <span
                                    class="text-neutral-500 dark:text-neutral-400"
                                    >ID</span
                                >
                                <span
                                    class="font-mono text-neutral-900 dark:text-white"
                                    >{{ userProfile.user_id }}</span
                                >
                            </div>
                            <div class="flex justify-between">
                                <span
                                    class="text-neutral-500 dark:text-neutral-400"
                                    >类型</span
                                >
                                <span
                                    class="text-neutral-900 dark:text-white"
                                    >{{
                                        currentUser?.user_type === UserType.Normal
                                            ? "正式用户"
                                            : "临时用户"
                                    }}</span
                                >
                            </div>
                        </div>
                    </UCard>
                </div>

                <!-- 右侧：最近动态占位 -->
                <div class="md:col-span-2">
                    <UCard
                        variant="subtle"
                        class="h-full min-h-[300px] flex items-center justify-center"
                    >
                        <div class="text-center">
                            <UIcon
                                name="i-lucide-image"
                                class="size-12 text-neutral-300 dark:text-neutral-600 mx-auto mb-2"
                            />
                            <p class="text-neutral-500 dark:text-neutral-400">
                                动态列表功能开发中...
                            </p>
                            <UButton variant="link" to="/moments" class="mt-2"
                                >查看我的动态</UButton
                            >
                        </div>
                    </UCard>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useUsersService } from "~/services/users.service";
import type { UserStatsData } from "~/types/api";
import { UserType } from "~/config/constants";

definePageMeta({
    title: "个人资料",
    layout: "dashboard",
    requiresAuth: true,
});

const authStore = useAuthStore();
const userService = useUsersService();
const modal = useModal();
const { formatDate } = useFormatters();
const feedback = useFeedbackStore();

const currentUser = useHydratedStore(authStore, "user", null);
const userProfile = ref<any>({});
const stats = ref({ momentsCount: 0, friendsCount: 0, groupsCount: 0 });
const fileInput = ref<HTMLInputElement | null>(null);

const editProfile = () => modal.showEditProfile();

const triggerFileInput = () => {
    fileInput.value?.click();
};

const handleFileChange = async (event: Event) => {
    const target = event.target as HTMLInputElement;
    const file = target.files?.[0];
    if (!file) return;

    // Implement avatar upload logic immediately or wait for save?
    // Typically upload immediately and update URL.
    // For now just show toast.
    feedback.info("头像上传功能需配合后端API");
};

onMounted(async () => {
    try {
        const profile = await userService.getMe();
        userProfile.value = profile;

        const userId = currentUser.value?.user_id;
        if (!userId) return;

        const statData: UserStatsData = await userService.getStats({
            user_id: userId,
        });
        stats.value = {
            momentsCount: statData.moments_count || 0,
            friendsCount: statData.friends_count || 0,
            groupsCount: statData.groups_count || 0,
        };
    } catch {}
});
</script>
