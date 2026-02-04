<template>
    <UModal title="群组管理" description="管理群组设置、成员和权限">
    <template #body>
        <div class="p-6">
            <section class="space-y-6">
                <!-- 邀请新成员 -->
                <UCard variant="subtle">
                    <template #header>
                        <header class="flex items-center justify-between">
                            <h4
                                class="text-lg font-semibold text-neutral-900 dark:text-white"
                            >
                                邀请新成员
                            </h4>
                            <UButton
                                :disabled="!canInvite"
                                icon="i-lucide-user-plus"
                                size="sm"
                                @click="showInviteModal = true"
                            >
                                邀请成员
                            </UButton>
                        </header>
                    </template>

                    <UCard variant="soft" class="mt-0">
                        <header class="flex items-center justify-between mb-3">
                            <article>
                                <h5
                                    class="font-semibold text-neutral-900 dark:text-white text-sm"
                                >
                                    群组分享链接
                                </h5>
                                <p
                                    class="text-xs text-neutral-500 dark:text-neutral-400 mt-1"
                                >
                                    分享此链接让朋友快速找到并加入群组
                                </p>
                            </article>
                            <UButton
                                variant="outline"
                                size="xs"
                                :loading="generatingCode"
                                icon="i-lucide-link"
                                @click="generateInviteCode"
                            >
                                生成链接
                            </UButton>
                        </header>

                        <section
                            v-if="inviteCode"
                            class="flex items-center gap-2"
                        >
                            <UInput
                                :model-value="inviteCode"
                                :readonly="true"
                                class="flex-1"
                                size="xs"
                            />
                            <UButton
                                variant="outline"
                                icon="i-lucide-clipboard"
                                size="xs"
                                @click="copyInviteCode"
                            />
                            <UButton
                                variant="outline"
                                icon="i-lucide-share-2"
                                size="xs"
                                @click="shareInviteCode"
                            />
                        </section>
                    </UCard>
                </UCard>

                <!-- 成员管理 -->
                <UCard variant="subtle">
                    <template #header>
                        <h4
                            class="text-lg font-semibold text-neutral-900 dark:text-white"
                        >
                            成员管理
                        </h4>
                    </template>

                    <ul
                        ref="membersListEl"
                        class="space-y-3 max-h-64 overflow-y-auto pr-1"
                        @scroll="onMembersListScroll"
                    >
                        <li
                            v-for="member in members"
                            :key="member.user.user_id"
                            class="min-w-0"
                        >
                            <UCard variant="outline">
                                <div
                                    class="flex items-center justify-between gap-3"
                                >
                                    <header
                                        class="flex items-center gap-3 min-w-0"
                                    >
                                        <UAvatar
                                            :alt="member.user.nickname"
                                            :src="member.user.avatar_url"
                                            size="sm"
                                        >
                                            {{
                                                member.user.nickname?.charAt(
                                                    0,
                                                ) || "U"
                                            }}
                                        </UAvatar>

                                        <article class="min-w-0">
                                            <p
                                                class="flex items-center gap-2 min-w-0"
                                            >
                                                <span
                                                    class="font-semibold text-neutral-900 dark:text-white text-sm truncate"
                                                    >{{
                                                        member.user.nickname
                                                    }}</span
                                                >
                                                <UBadge
                                                    :color="
                                                        getRoleColor(
                                                            member.role,
                                                        )
                                                    "
                                                    :label="
                                                        getRoleLabel(
                                                            member.role,
                                                        )
                                                    "
                                                    size="xs"
                                                    variant="soft"
                                                />
                                            </p>
                                            <span
                                                class="text-neutral-400 dark:text-neutral-500 text-xs"
                                            >
                                                加入于
                                                {{
                                                    formatTime(member.joined_at)
                                                }}
                                            </span>
                                        </article>
                                    </header>

                                    <UDropdownMenu
                                        v-if="canManageMember(member)"
                                        :items="getMemberActions(member)"
                                    >
                                        <UButton
                                            variant="ghost"
                                            color="neutral"
                                            size="xs"
                                            icon="i-lucide-ellipsis"
                                        />
                                    </UDropdownMenu>
                                </div>
                            </UCard>
                        </li>

                        <!-- 回到顶部 -->
                        <li
                            v-if="showMembersScrollTop"
                            class="sticky bottom-2 flex justify-end pointer-events-none"
                        >
                            <UButton
                                variant="soft"
                                color="neutral"
                                size="sm"
                                icon="i-lucide-arrow-up"
                                class="pointer-events-auto"
                                @click="scrollMembersToTop"
                            />
                        </li>
                    </ul>
                </UCard>

                <!-- 群组设置 -->
                <UCard variant="subtle">
                    <template #header>
                        <h4
                            class="text-lg font-semibold text-neutral-900 dark:text-white"
                        >
                            群组设置
                        </h4>
                    </template>

                    <UForm :state="groupSettings" class="space-y-4">
                        <UFormField label="群组名称">
                            <UInput
                                v-model="groupSettings.name"
                                :disabled="!canEditGroup"
                            />
                        </UFormField>

                        <UFormField label="群组描述">
                            <UTextarea
                                v-model="groupSettings.description"
                                :rows="3"
                                :disabled="!canEditGroup"
                                autoresize
                            />
                        </UFormField>

                        <UFormField label="最大成员数">
                            <UInput
                                v-model.number="groupSettings.max_members"
                                type="number"
                                :min="members.length"
                                :max="1000"
                                :disabled="!canEditGroup"
                            />
                        </UFormField>

                        <UCard variant="soft" class="mt-2">
                            <div
                                class="flex items-center justify-between gap-4 p-2"
                            >
                                <div class="flex-1">
                                    <div
                                        class="font-semibold text-neutral-900 dark:text-white text-sm"
                                    >
                                        需要密码加入
                                    </div>
                                    <div
                                        class="text-xs text-neutral-500 dark:text-neutral-400"
                                    >
                                        开启后新成员需要输入密码才能加入
                                    </div>
                                </div>
                                <USwitch
                                    v-model="groupSettings.password_required"
                                    :disabled="!canEditGroup"
                                />
                            </div>
                        </UCard>

                        <UFormField
                            v-if="groupSettings.password_required"
                            label="群组密码"
                        >
                            <UInput
                                v-model="groupSettings.password"
                                type="password"
                                placeholder="设置新密码"
                                :disabled="!canEditGroup"
                            />
                        </UFormField>
                    </UForm>
                </UCard>

                <!-- 危险操作 -->
                <UCard v-if="canDeleteGroup" variant="outline">
                    <template #header>
                        <div
                            class="flex items-center gap-2 text-red-500 dark:text-red-400"
                        >
                            <UIcon
                                name="i-lucide-triangle-alert"
                                class="size-4"
                            />
                            <span class="font-bold text-sm">危险操作</span>
                        </div>
                    </template>

                    <ul class="space-y-4">
                        <li class="flex items-center justify-between">
                            <article>
                                <h5
                                    class="font-semibold text-neutral-900 dark:text-white text-sm"
                                >
                                    转移群主
                                </h5>
                                <p
                                    class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5"
                                >
                                    将群主权限转移给其他管理员
                                </p>
                            </article>
                            <UButton
                                color="warning"
                                variant="outline"
                                size="sm"
                                @click="showTransferModal = true"
                            >
                                转移群主
                            </UButton>
                        </li>

                        <li class="flex items-center justify-between">
                            <article>
                                <h5
                                    class="font-semibold text-neutral-900 dark:text-white text-sm"
                                >
                                    解散群组
                                </h5>
                                <p
                                    class="text-xs text-neutral-500 dark:text-neutral-400 mt-0.5"
                                >
                                    永久删除群组，此操作不可撤销
                                </p>
                            </article>
                            <UButton
                                color="error"
                                variant="outline"
                                size="sm"
                                @click="confirmDelete"
                            >
                                解散群组
                            </UButton>
                        </li>
                    </ul>
                </UCard>
            </section>
        </div>
    </template>

        <template #footer="{ close }">
            <div class="flex justify-end gap-3 w-full">
                <UButton
                    variant="ghost"
                    color="neutral"
                    :disabled="saving"
                    @click="close"
                >
                    取消
                </UButton>
                <UButton
                    :loading="saving"
                    :disabled="!canEditGroup"
                    @click="saveSettings"
                >
                    保存设置
                </UButton>
            </div>
        </template>
    </UModal>

    <!-- 子模态框 -->
    <InviteMembersModal
        v-model:open="showInviteModal"
        :group-id="groupId"
        @success="loadGroupData"
    />

    <TransferOwnershipModal
        v-model:open="showTransferModal"
        :group-id="groupId"
        :members="members"
        @success="handleTransferSuccess"
    />

    <ConfirmModal
        v-model:open="showConfirmModal"
        :title="confirmConfig.title"
        :message="confirmConfig.message"
        :confirm-text="confirmConfig.confirmText"
        :confirm-color="confirmConfig.confirmColor"
        @close="handleConfirm"
    />
</template>

<script setup lang="ts">
import { GROUP_CONSTANTS } from "~/config/defaults";
import { GroupRole, type GroupRoleValue } from "~/config/constants";
import type { GroupMemberItem } from "~/types/api";
import { useGroupsService } from "~/services/groups.service";
import InviteMembersModal from "./InviteMembersModal.vue";
import TransferOwnershipModal from "./TransferOwnershipModal.vue";
import ConfirmModal from "./ConfirmModal.vue";

interface Props {
    groupId: string;
    onSuccess?: () => void;
}

const props = defineProps<Props>();
const emit = defineEmits<{ close: [] }>();

const authStore = useAuthStore();
const groupsService = useGroupsService();
const feedback = useFeedbackStore();
const { formatTime } = useFormatters();
const { copy } = useClipboard();
const { share, isSupported: isShareSupported } = useShare();

// 状态
const saving = ref(false);
const generatingCode = ref(false);
const inviteCode = ref("");
const members = ref<GroupMemberItem[]>([]);
const currentUserRole = ref<GroupRoleValue>(GroupRole.Member);
const showInviteModal = ref(false);
const showTransferModal = ref(false);
const showConfirmModal = ref(false);

// 确认弹窗配置
const confirmConfig = reactive({
    title: "",
    message: "",
    confirmText: "确认",
    confirmColor: "primary" as "primary" | "error" | "warning" | "neutral",
    action: null as (() => Promise<void>) | null,
});

// 成员列表滚动
const membersListEl = ref<HTMLElement | null>(null);
const showMembersScrollTop = ref(false);

const onMembersListScroll = () => {
    const el = membersListEl.value;
    if (!el) return;
    showMembersScrollTop.value = el.scrollTop > 160;
};

const scrollMembersToTop = () => {
    membersListEl.value?.scrollTo({ top: 0, behavior: "smooth" });
};

// 群组设置
const groupSettings = ref({
    name: "",
    description: "",
    max_members: 500,
    password_required: false,
    password: "",
});

// 计算属性
const canInvite = computed(() =>
    [GroupRole.Owner, GroupRole.Admin].includes(currentUserRole.value),
);
const canEditGroup = computed(() =>
    [GroupRole.Owner, GroupRole.Admin].includes(currentUserRole.value),
);
const canDeleteGroup = computed(() => currentUserRole.value === GroupRole.Owner);

type BadgeColor =
    | "primary"
    | "secondary"
    | "success"
    | "info"
    | "warning"
    | "error"
    | "neutral";

const getRoleColor = (role: string): BadgeColor => {
    const colors: Record<string, BadgeColor> = {
        [GroupRole.Owner]: "error",
        [GroupRole.Admin]: "warning",
        [GroupRole.Member]: "neutral",
    };
    return colors[role] || "neutral";
};

const getRoleLabel = (role: string) => {
    const labels: Record<string, string> = {
        [GroupRole.Owner]: "群主",
        [GroupRole.Admin]: "管理员",
        [GroupRole.Member]: "成员",
    };
    return labels[role] || role;
};

const canManageMember = (member: GroupMemberItem): boolean => {
    if (currentUserRole.value === GroupRole.Owner) return member.role !== GroupRole.Owner;
    if (currentUserRole.value === GroupRole.Admin) return member.role === GroupRole.Member;
    return false;
};

const getMemberActions = (member: GroupMemberItem) => {
    const actions = [];

    if (currentUserRole.value === GroupRole.Owner) {
        if (member.role === GroupRole.Member) {
            actions.push([
                {
                    label: "设为管理员",
                    icon: "i-lucide-shield-check",
                    click: () => confirmPromote(member),
                },
            ]);
        } else if (member.role === GroupRole.Admin) {
            actions.push([
                {
                    label: "取消管理员",
                    icon: "i-lucide-shield-alert",
                    click: () => confirmDemote(member),
                },
            ]);
        }
    }

    if (canManageMember(member)) {
        actions.push([
            {
                label: "踢出群组",
                icon: "i-lucide-user-minus",
                color: "error" as const,
                click: () => confirmKick(member),
            },
        ]);
    }

    return actions;
};

const loadGroupData = async () => {
    try {
        const coords = await useGeolocation().getCoordinates().catch(() => ({ latitude: 0, longitude: 0 }));

        const [settingsResponse, membersResponse, groupDetail] = await Promise.all([
            groupsService.getSettings({ group_chat_id: props.groupId }),
            groupsService.getMembers({ group_chat_id: props.groupId }),
            groupsService.openTheDoor({
                group_chat_id: props.groupId,
                reference_point: {
                    latitude: coords.latitude,
                    longitude: coords.longitude
                }
            })
        ]);

        groupSettings.value = {
            name: groupDetail.name,

            description: groupDetail.description || "",
            max_members:
                settingsResponse.settings?.max_members ||
                GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT,
            password_required: settingsResponse.has_password || false,
            password: "",
        };


        // members
        members.value = membersResponse.items || [];

        const currentUserId = authStore.user?.user_id;
        if (currentUserId) {
            const currentMember = members.value.find(
                (m) => m.user?.user_id === currentUserId,
            );
            currentUserRole.value = (currentMember?.role as GroupRoleValue) || GroupRole.Member;
        }
    } catch (err) {
        feedback.error("加载失败");
    }
};

// Helper to fetch full details
const fetchFullDetails = async () => {
    // ...
};

// Actions
const generateInviteCode = async () => {
    generatingCode.value = true;
    try {
        const baseUrl = window.location.origin;
        inviteCode.value = `${baseUrl}/groups/${props.groupId}`;
        feedback.success("分享链接已生成");
    } catch (err) {
        feedback.error("生成失败");
    } finally {
        generatingCode.value = false;
    }
};

const copyInviteCode = async () => {
    try {
        await copy(inviteCode.value);
        feedback.success("复制成功");
    } catch (err) {
        feedback.error("复制失败");
    }
};

const shareInviteCode = async () => {
    const shareData = {
        title: `加入群组`,
        text: `邀请您加入群组！链接：${inviteCode.value}`,
        url: inviteCode.value,
    };
    if (isShareSupported.value) await share(shareData);
    else await copyInviteCode();
};

// Confirmation Logic
const confirmPromote = (member: GroupMemberItem) => {
    confirmConfig.title = "设为管理员";
    confirmConfig.message = `确定要将 ${member.user?.nickname} 设为管理员吗？`;
    confirmConfig.confirmText = "确认";
    confirmConfig.confirmColor = "primary";
    confirmConfig.action = async () => {
        await groupsService.setMemberRole({
            group_chat_id: props.groupId,
            user_id: member.user.user_id,
            role: GroupRole.Admin,
        });
        member.role = GroupRole.Admin;
        feedback.success("设置成功");
    };
    showConfirmModal.value = true;
};

const confirmDemote = (member: GroupMemberItem) => {
    confirmConfig.title = "取消管理员";
    confirmConfig.message = `确定要取消 ${member.user?.nickname} 的管理员权限吗？`;
    confirmConfig.confirmText = "确认";
    confirmConfig.confirmColor = "warning";
    confirmConfig.action = async () => {
        await groupsService.setMemberRole({
            group_chat_id: props.groupId,
            user_id: member.user.user_id,
            role: GroupRole.Member,
        });
        member.role = GroupRole.Member;
        feedback.success("取消成功");
    };
    showConfirmModal.value = true;
};

const confirmKick = (member: GroupMemberItem) => {
    confirmConfig.title = "踢出群组";
    confirmConfig.message = `确定要将 ${member.user?.nickname} 踢出群组吗？`;
    confirmConfig.confirmText = "确认踢出";
    confirmConfig.confirmColor = "error";
    confirmConfig.action = async () => {
        await groupsService.removeMember({
            group_chat_id: props.groupId,
            user_id: member.user.user_id,
        });
        members.value = members.value.filter(
            (m) => m.user.user_id !== member.user.user_id,
        );
        feedback.success("踢出成功");
    };
    showConfirmModal.value = true;
};

const confirmDelete = () => {
    confirmConfig.title = "解散群组";
    confirmConfig.message = `确定要解散群组吗？此操作不可撤销。`;
    confirmConfig.confirmText = "确认解散";
    confirmConfig.confirmColor = "error";
    confirmConfig.action = async () => {
        await groupsService.deleteGroup({ group_chat_id: props.groupId });
        feedback.success("群组已解散");
        emit("close");
        // redirect?
    };
    showConfirmModal.value = true;
};

const handleConfirm = async (confirmed: boolean) => {
    if (confirmed && confirmConfig.action) {
        try {
            await confirmConfig.action();
        } catch (e) {
            feedback.error("操作失败");
        }
    }
    showConfirmModal.value = false;
};

const handleTransferSuccess = (newOwnerId: number) => {
    showTransferModal.value = false;
    loadGroupData();
};

const saveSettings = async () => {
    saving.value = true;
    try {
        await groupsService.update({
            group_chat_id: props.groupId,
            name: groupSettings.value.name,
            description: groupSettings.value.description,
        });
        feedback.success("设置已保存");
        props.onSuccess?.();
        emit("close");
    } catch (err) {
        feedback.error("保存失败");
    } finally {
        saving.value = false;
    }
};

onMounted(() => {
    loadGroupData();
});
</script>
