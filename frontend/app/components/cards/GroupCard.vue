<template>
    <UCard
        variant="outline"
        class="h-full flex flex-col"
        @click="$emit('click', group)"
    >
        <!-- 群组头部 -->
        <template #header>
            <div class="flex items-center justify-between gap-3">
                <div class="flex items-center gap-3 min-w-0">
                    <UAvatar
                        :alt="group.name"
                        :src="group.avatar_url"
                        size="md"
                        :chip="
                            group.has_password
                                ? {
                                      color: 'warning',
                                      position: 'bottom-right',
                                      text: '🔒',
                                  }
                                : undefined
                        "
                    >
                        {{ group.name?.charAt(0) }}
                    </UAvatar>

                    <div class="flex-1 min-w-0">
                        <div class="flex items-center gap-2">
                            <h3
                                class="text-lg font-semibold text-neutral-900 dark:text-white truncate"
                            >
                                {{ group.name }}
                            </h3>
                            <UBadge
                                v-if="isJoined"
                                variant="soft"
                                size="xs"
                                color="primary"
                            >
                                {{ t('groups.card.joined') }}
                            </UBadge>
                        </div>

                        <div class="flex items-center gap-2 mt-1 flex-wrap">
                            <UBadge
                                v-if="isOwner"
                                color="success"
                                variant="subtle"
                                size="xs"
                                >{{ t('enums.groupRole.owner') }}</UBadge
                            >
                            <UBadge
                                v-else-if="isAdmin"
                                color="warning"
                                variant="subtle"
                                size="xs"
                                >{{ t('enums.groupRole.admin') }}</UBadge
                            >

                            <UBadge
                                :label="t('groups.card.members', { count: group.members_count })"
                                icon="i-lucide-users"
                                color="neutral"
                                variant="subtle"
                                size="xs"
                            />
                        </div>
                    </div>
                </div>

                <!-- 操作按钮 -->
                <div class="flex items-center gap-1 shrink-0" @click.stop>
                    <!-- 未加入：显示加入按钮 -->
                    <UButton
                        v-if="!isJoined"
                        variant="soft"
                        size="sm"
                        color="primary"
                        icon="i-lucide-plus"
                        @click.stop="$emit('join', group)"
                    >
                        {{ t('groups.card.join') }}
                    </UButton>

                    <!-- 已加入：显示操作按钮 -->
                    <template v-else>
                        <UTooltip :text="t('groups.card.enter')" :portal="false">
                            <UButton
                                variant="ghost"
                                color="neutral"
                                size="sm"
                                icon="i-lucide-messages-square"
                                @click.stop="$emit('enter', group)"
                            />
                        </UTooltip>

                        <UDropdownMenu :items="memberActions">
                            <UButton
                                variant="ghost"
                                color="neutral"
                                size="sm"
                                icon="i-lucide-more-vertical"
                            />
                        </UDropdownMenu>
                    </template>
                </div>
            </div>
        </template>

        <!-- 群组内容 -->
        <div class="flex-1 space-y-3">
            <!-- 群组描述 -->
            <p
                v-if="group.description"
                class="text-sm text-neutral-500 dark:text-neutral-400 line-clamp-2"
            >
                {{ group.description }}
            </p>

            <!-- 位置和距离信息 -->
            <div
                v-if="group.location"
                class="flex items-center gap-2 mt-auto pt-2"
            >
                <UBadge
                    v-if="group.location.location_name"
                    color="neutral"
                    variant="soft"
                    size="xs"
                    class="flex items-center gap-1 max-w-[120px]"
                >
                    <UIcon name="i-lucide-map-pin" class="size-3" />
                    <span class="truncate">{{ group.location.location_name }}</span>
                </UBadge>

                <UBadge
                    v-if="group.location.distance !== undefined"
                    color="primary"
                    variant="soft"
                    size="xs"
                    class="flex items-center gap-1"
                >
                    <UIcon name="i-lucide-navigation" class="size-3" />
                    <span>{{ formatDistance(group.location.distance) }}</span>
                </UBadge>
            </div>
        </div>

        <!-- 群组底部信息 -->
        <template #footer>
            <div
                class="flex items-center justify-between gap-2 text-sm text-neutral-500 dark:text-neutral-400"
            >
                <span v-if="group.created_at" class="flex items-center gap-1">
                    <UIcon name="i-lucide-calendar" class="size-3" />
                    {{ formatTime(group.created_at) }}
                </span>
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">
import type { RelativeLocation, GroupItem, GroupMyState } from "~/types/api";
import type { DropdownMenuItem } from "@nuxt/ui";
import { useGroupsService } from "~/services/groups.service";
import { GroupRole, type GroupRoleValue } from "~/config/constants";

interface GroupCardData extends GroupItem {
    created_at?: string;
}

interface Props {
    group: GroupCardData;
}

interface Emits {
    click: [group: Props["group"]];
    join: [group: Props["group"]];
    enter: [group: Props["group"]];
    more: [group: Props["group"]];
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const { formatTime, formatDistance } = useFormatters();
const modal = useModal();
const feedback = useFeedbackStore();
const groupService = useGroupsService();
const { t } = useI18n();

// 计算属性：判断用户是否已加入群组
const isJoined = computed(() => {
    return props.group.my_state?.is_joined ?? false;
});

// 计算属性：获取用户角色
const userRole = computed<GroupRoleValue | undefined>(() => {
    return props.group.my_state?.role as GroupRoleValue | undefined;
});

// 计算属性：判断是否是创建者
const isOwner = computed(() => userRole.value === GroupRole.Owner);

// 计算属性：判断是否是管理员
const isAdmin = computed(() => userRole.value === GroupRole.Admin);

const handleGroupSettings = () =>
    navigateTo(`/groups/${props.group.group_chat_id}/settings`);
const handleManageMembers = () =>
    navigateTo(`/groups/${props.group.group_chat_id}/members`);

const handleLeaveGroup = async () => {
    const confirmed = await modal.showConfirm({
        title: t('groups.card.leaveConfirmTitle'),
        message: t('groups.card.leaveConfirmBody', { name: props.group.name }),
        confirmText: t('groups.card.confirmLeave'),
        confirmColor: "error",
    });

    if (!confirmed) return;

    try {
        await groupService.leaveGroup({ group_chat_id: props.group.group_chat_id });
        feedback.success(t('groups.card.leaveSuccess'));
        await refreshNuxtData("groups");
    } catch (error: unknown) {
        feedback.error(t('groups.card.leaveFailed'), getErrorMessage(error));
    }
};

const memberActions = computed((): DropdownMenuItem[][] => {
    const actions: DropdownMenuItem[] = [
        {
            label: t('groups.card.settings'),
            icon: "i-lucide-settings",
            onSelect: handleGroupSettings,
        },
    ];

    if (isOwner.value || isAdmin.value) {
        actions.push({
            label: t('groups.card.manageMembers'),
            icon: "i-lucide-users",
            onSelect: handleManageMembers,
        });
    }

    actions.push({
        label: t('groups.card.leave'),
        icon: "i-lucide-log-out",
        color: "error",
        onSelect: handleLeaveGroup,
    });

    return [actions];
});
</script>
