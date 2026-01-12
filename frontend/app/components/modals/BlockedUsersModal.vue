<template>
    <UModal
        title="黑名单管理"
        :description="`共 ${blockedUsers.length} 个被拉黑用户`"
    >
        <template #body="{ close }">
            <UInput
                v-model="searchQuery"
                placeholder="搜索被拉黑的用户..."
                icon="i-lucide-search"
                class="mb-4 w-full"
                variant="soft"
            />

            <section
                ref="listEl"
                class="max-h-96 overflow-y-auto space-y-1 pr-1"
                @scroll="onListScroll"
            >
                <!-- 用户列表 -->
                <UCard
                    v-for="user in filteredUsers"
                    :key="user.user_id"
                    variant="ghost"
                    class="group"
                >
                    <div class="flex items-center justify-between gap-3">
                        <div class="flex items-center gap-3 min-w-0 flex-1">
                            <UAvatar
                                :src="user.avatar_url"
                                :alt="user.nickname"
                                size="md"
                            />
                            <div class="min-w-0">
                                <div
                                    class="font-semibold text-neutral-900 dark:text-white truncate"
                                >
                                    {{ user.nickname }}
                                </div>
                                <UBadge
                                    variant="subtle"
                                    color="neutral"
                                    size="xs"
                                    class="font-mono mt-0.5"
                                >
                                    ID: {{ user.user_id }}
                                </UBadge>
                            </div>
                        </div>

                        <div
                            class="flex items-center gap-2 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity"
                        >
                            <UTooltip text="查看资料">
                                <UButton
                                    variant="ghost"
                                    color="neutral"
                                    icon="i-lucide-user"
                                    size="xs"
                                    @click="viewUserProfile(user, close)"
                                />
                            </UTooltip>
                            <UTooltip text="解除拉黑">
                                <UButton
                                    color="error"
                                    variant="soft"
                                    icon="i-lucide-user-check"
                                    size="xs"
                                    :loading="unblockingUsers.has(user.user_id)"
                                    @click="unblockUser(user)"
                                />
                            </UTooltip>
                        </div>
                    </div>
                </UCard>

                <!-- 空状态 -->
                <UEmpty
                    v-if="!loading && filteredUsers.length === 0"
                    icon="i-lucide-user-minus"
                    title="黑名单为空"
                    description="您还没有拉黑任何用户"
                    variant="naked"
                    class="py-8"
                />

                <!-- 加载状态 -->
                <template v-if="loading">
                    <div
                        v-for="i in 3"
                        :key="i"
                        class="flex items-center gap-3 p-3"
                    >
                        <USkeleton shape="circle" class="h-10 w-10" />
                        <div class="space-y-2 flex-1">
                            <USkeleton class="h-4 w-24" />
                            <USkeleton class="h-3 w-16" />
                        </div>
                    </div>
                </template>
            </section>

            <!-- 回到顶部（列表很长时） -->
            <div v-if="showScrollTop" class="absolute bottom-20 right-8">
                <UButton
                    variant="soft"
                    color="neutral"
                    size="sm"
                    icon="i-lucide-arrow-up"
                    @click="scrollToTop"
                />
            </div>
        </template>

        <template #footer="{ close }">
            <UButton
                variant="ghost"
                color="neutral"
                icon="i-lucide-refresh-cw"
                :loading="loading"
                @click="loadBlockedUsers"
            >
                刷新
            </UButton>
            <UButton color="neutral" @click="close"> 关闭 </UButton>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import { useUsersService } from "~/services/users.service";
import type { UserInfo } from "~/types/api";

const authStore = useAuthStore();
const loading = ref(false);
const blockedUsers = ref<UserInfo[]>([]);
const searchQuery = ref("");
const unblockingUsers = ref(new Set<number>());
const feedback = useFeedbackStore();

const filteredUsers = computed(() => {
    if (!searchQuery.value.trim()) return blockedUsers.value;
    const query = searchQuery.value.toLowerCase();
    return blockedUsers.value.filter((user) =>
        user.nickname.toLowerCase().includes(query),
    );
});

const listEl = ref<HTMLElement | null>(null);
const showScrollTop = ref(false);

const onListScroll = () => {
    const el = listEl.value;
    if (!el) return;
    showScrollTop.value = el.scrollTop > 160;
};

const scrollToTop = () => {
    listEl.value?.scrollTo({ top: 0, behavior: "smooth" });
};

const loadBlockedUsers = async () => {
    loading.value = true;
    try {
        const userService = useUsersService();
        const response = await userService.getBlockedUsers();
        blockedUsers.value = (response.items || []).map((i) => i.user);
    } catch (err) {
        console.error("加载拉黑用户列表失败:", err);
        feedback.error("加载失败", "无法获取拉黑用户列表");
    } finally {
        loading.value = false;
    }
};

const viewUserProfile = (user: UserInfo, close: () => void) => {
    close();
    if (user.user_id === authStore.user?.user_id) {
        navigateTo("/profile");
    } else {
        navigateTo(`/users/${user.user_id}`);
    }
};

const unblockUser = async (user: UserInfo) => {
    const { showConfirm } = useModal();
    const confirmed = await showConfirm({
        title: "解除拉黑",
        message: `确定要解除对 ${user.nickname} 的拉黑吗？`,
        confirmText: "确认解除",
        cancelText: "取消",
        confirmColor: "primary",
    });

    if (!confirmed) return;

    unblockingUsers.value.add(user.user_id);

    try {
        const userService = useUsersService();
        await userService.unblock({ user_id: user.user_id });

        const index = blockedUsers.value.findIndex(
            (u) => u.user_id === user.user_id,
        );
        if (index > -1) blockedUsers.value.splice(index, 1);

        feedback.success("解除成功", `已解除对 ${user.nickname} 的拉黑`);
    } catch (err: unknown) {
        console.error("解除拉黑失败:", err);
        feedback.error(
            "解除失败",
            getErrorMessage(err, "无法解除拉黑，请重试"),
        );
    } finally {
        unblockingUsers.value.delete(user.user_id);
    }
};

onMounted(() => loadBlockedUsers());
</script>
