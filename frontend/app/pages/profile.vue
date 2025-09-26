<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-8 sm:py-10">
        <div class="max-w-5xl mx-auto">
            <!-- 个人信息卡片 -->
            <UCard class="mb-4">
                <div class="text-center">
                    <!-- 头像 -->
                    <div class="mb-3">
                        <UAvatar
                            :alt="user?.nickname"
                            size="3xl"
                            class="mx-auto"
                        >
                            {{ user?.nickname?.charAt(0) || "U" }}
                        </UAvatar>
                    </div>

                    <!-- 基本信息 -->
                    <h1 class="text-2xl font-bold text-neutral-900 mb-2">
                        {{ user?.nickname || $t("common.user") }}
                    </h1>
                    <p class="text-neutral-600 mb-4">ID: {{ user?.user_id }}</p>

                    <!-- 未绑定/临时用户引导 EmptyState -->
                    <div v-if="false" class="mb-4">
                        <UCard>
                            <EmptyState
                                icon="i-heroicons-user"
                                :title="$t('profile.temporary.title')"
                                :description="
                                    $t('profile.temporary.description')
                                "
                                :action-text="
                                    $t('profile.temporary.actionText')
                                "
                                :action-handler="goUpgrade"
                                :secondary-action-text="
                                    $t('profile.temporary.secondaryActionText')
                                "
                                :secondary-action-handler="showBenefits"
                            >
                                <template #illustration>
                                    <EmptyAccount size="md" />
                                </template>
                            </EmptyState>
                        </UCard>
                    </div>

                    <!-- 操作按钮 -->
                    <div class="flex justify-center gap-3 sm:gap-4">
                        <UButton variant="soft" @click="editProfile">
                            <UIcon
                                name="i-heroicons-pencil-square"
                                class="w-4 h-4 mr-2"
                            />
                            {{ $t("profile.actions.edit") }}
                        </UButton>
                        <UButton variant="outline" @click="viewSettings">
                            <UIcon
                                name="i-heroicons-cog-6-tooth"
                                class="w-4 h-4 mr-2"
                            />
                            {{ $t("profile.actions.settings") }}
                        </UButton>
                    </div>
                </div>
            </UCard>

            <!-- 统计信息 -->
            <div class="grid grid-cols-3 gap-3 mb-6">
                <UCard class="text-center">
                    <div>
                        <div class="text-2xl font-bold text-neutral-900 mb-1">
                            0
                        </div>
                        <div class="text-sm text-neutral-600">
                            {{ $t("profile.stats.friends") }}
                        </div>
                    </div>
                </UCard>
                <UCard class="text-center">
                    <div>
                        <div class="text-2xl font-bold text-neutral-900 mb-1">
                            0
                        </div>
                        <div class="text-sm text-neutral-600">
                            {{ $t("profile.stats.groups") }}
                        </div>
                    </div>
                </UCard>
                <UCard class="text-center">
                    <div>
                        <div class="text-2xl font-bold text-neutral-900 mb-1">
                            0
                        </div>
                        <div class="text-sm text-neutral-600">
                            {{ $t("profile.stats.moments") }}
                        </div>
                    </div>
                </UCard>
            </div>

            <!-- 功能菜单 -->
            <UCard>
                <template #header>
                    <h3 class="font-semibold">
                        {{ $t("profile.menu.title") }}
                    </h3>
                </template>

                <div class="space-y-1">
                    <UButton
                        variant="ghost"
                        block
                        justify="start"
                        @click="navigateTo('/groups')"
                    >
                        <UIcon
                            name="i-heroicons-user-group"
                            class="w-5 h-5 mr-3"
                        />
                        {{ $t("nav.myGroups") }}
                    </UButton>

                    <UButton
                        variant="ghost"
                        block
                        justify="start"
                        @click="navigateTo('/moments')"
                    >
                        <UIcon name="i-heroicons-photo" class="w-5 h-5 mr-3" />
                        {{ $t("nav.moments") }}
                    </UButton>

                    <UButton
                        variant="ghost"
                        block
                        justify="start"
                        @click="navigateTo('/settings')"
                    >
                        <UIcon
                            name="i-heroicons-cog-6-tooth"
                            class="w-5 h-5 mr-3"
                        />
                        {{ $t("nav.settings") }}
                    </UButton>

                    <UButton
                        variant="ghost"
                        block
                        justify="start"
                        color="error"
                        @click="openLogoutConfirm"
                    >
                        <UIcon
                            name="i-heroicons-arrow-right-on-rectangle"
                            class="w-5 h-5 mr-3"
                        />
                        {{ $t("profile.actions.logout") }}
                    </UButton>
                </div>
            </UCard>
        </div>
    </div>

    <!-- 退出登录确认由 useModals.openConfirmModal 管理 -->
</template>

<script setup lang="ts">
// 页面元数据
definePageMeta({
    title: "profile.title",
    requiresAuth: true,
    layout: 'dashboard'
});
import { defineAsyncComponent } from "vue";
const EmptyAccount = defineAsyncComponent(() => import("~/components/illustrations/EmptyAccount.vue"));

const authStore = useAuthStore();
const toast = useSmartToast();
const { t } = useI18n();

// 当前用户信息
const user = computed(() => authStore.user);

// 编辑资料
const editProfile = () => {
    toast.add({
        title: t("profile.toasts.featureUnavailableTitle"),
        description: t("profile.toasts.featureUnavailableDesc"),
        color: "info",
    });
};

// 查看设置
const viewSettings = () => {
    navigateTo("/settings");
};

// 升级/权益引导
const goUpgrade = () => {
    navigateTo("/settings");
};
const showBenefits = () => {
    toast.add({
        title: t("profile.benefits.title"),
        description: t("profile.benefits.description"),
        color: "info",
    });
};

// 退出登录
const { openConfirmModal } = useModals();
const openLogoutConfirm = () => {
    openConfirmModal({
        title: t("profile.logout.confirmTitle"),
        message: t("profile.logout.confirmBody"),
        confirmText: t("profile.logout.confirmButton"),
        cancelText: t("common.cancel"),
        confirmColor: "warning",
        onConfirm: () => confirmLogout(),
    });
};
const loggingOut = ref(false);
const confirmLogout = async () => {
    if (loggingOut.value) return;
    loggingOut.value = true;
    try {
        await authStore.logout();
        toast.add({
            title: t("profile.toasts.logoutSuccess"),
            color: "success",
        });

        await navigateTo("/auth/login");
    } catch (error) {
        toast.add({
            title: t("profile.toasts.logoutFailedTitle"),
            description: t("profile.toasts.logoutFailedDesc"),
            color: "error",
        });
    } finally {
        loggingOut.value = false;
    }
};
</script>
