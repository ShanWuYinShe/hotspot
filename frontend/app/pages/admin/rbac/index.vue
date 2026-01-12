<template>
    <div class="flex flex-col">
        <!-- Sticky Header Group -->
        <div
            class="sticky top-0 z-10 -mx-3 sm:-mx-4 lg:-mx-6 -mt-5 px-3 sm:px-4 lg:px-6 pt-5 pb-4 mb-6 bg-white dark:bg-neutral-900 border-b border-neutral-200 dark:border-neutral-800"
        >
            <div class="flex flex-col gap-6">
                <!-- Header -->
                <div class="flex items-center justify-between">
                    <div>
                        <h1
                            class="text-2xl font-bold text-neutral-900 dark:text-white"
                        >
                            权限管理
                        </h1>
                        <p class="text-neutral-500 dark:text-neutral-400 mt-1">
                            基于角色的访问控制 (RBAC)
                        </p>
                    </div>
                    <UButton
                        variant="ghost"
                        color="neutral"
                        icon="i-lucide-arrow-left"
                        to="/admin"
                    >
                        返回控制台
                    </UButton>
                </div>

                <UTabs
                    v-model="selectedTab"
                    :items="tabs"
                    class="w-full"
                    :content="false"
                />
            </div>
        </div>

        <!-- Content Area -->
        <div class="space-y-8">
            <!-- 管理员列表 -->
            <div v-if="selectedTab === 0" class="space-y-4">
                <div class="flex justify-end">
                    <UButton icon="i-lucide-plus" @click="openGrantModal"
                        >添加管理员</UButton
                    >
                </div>

                <UCard variant="subtle" class="overflow-hidden">
                    <div class="-m-5 sm:-m-7">
                        <UTable
                            :data="admins"
                            :columns="adminColumns"
                            :loading="loadingAdmins"
                            :empty-state="{
                                icon: 'i-lucide-users',
                                label: '暂无管理员',
                            }"
                        >
                        <template #user-cell="{ row }">
                            <div class="flex items-center gap-3">
                                <UAvatar
                                    :src="row.original.user.avatar_url"
                                    :alt="row.original.user.nickname"
                                    size="sm"
                                />
                                <div>
                                    <div
                                        class="font-medium text-neutral-900 dark:text-white"
                                    >
                                        {{ row.original.user.nickname }}
                                    </div>
                                    <div
                                        class="text-xs text-neutral-500 dark:text-neutral-400"
                                    >
                                        ID: {{ row.original.user.user_id }}
                                    </div>
                                </div>
                            </div>
                        </template>

                        <template #type-cell="{ row }">
                            <UBadge
                                :color="
                                    (row.original.permissions || []).includes(
                                        '*',
                                    )
                                        ? 'error'
                                        : 'primary'
                                "
                                variant="soft"
                                size="xs"
                            >
                                {{
                                    (row.original.permissions || []).includes(
                                        "*",
                                    )
                                        ? "超级管理员"
                                        : "普通管理员"
                                }}
                            </UBadge>
                        </template>

                        <template #role-cell="{ row }">
                            <div class="flex flex-wrap gap-1">
                                <UBadge
                                    v-for="role in row.original.roles"
                                    :key="role.role_id"
                                    color="neutral"
                                    variant="outline"
                                    size="xs"
                                >
                                    {{ role.name }}
                                </UBadge>
                            </div>
                        </template>

                        <template #actions-cell="{ row }">
                            <div class="flex justify-end gap-2">
                                <UButton
                                    variant="ghost"
                                    color="neutral"
                                    size="xs"
                                    icon="i-lucide-edit"
                                    @click="editAdmin(row.original)"
                                />
                                <UButton
                                    variant="ghost"
                                    color="error"
                                    size="xs"
                                    icon="i-lucide-trash-2"
                                    @click="revokeAdmin(row.original)"
                                />
                            </div>
                        </template>
                    </UTable>
                    </div>
                </UCard>
            </div>

            <!-- 角色管理 -->
            <div v-else-if="selectedTab === 1" class="space-y-4">
                <div class="flex justify-end">
                    <UButton icon="i-lucide-plus" @click="openCreateRoleModal"
                        >创建角色</UButton
                    >
                </div>

                <div
                    class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
                >
                    <UCard
                        v-for="role in roles"
                        :key="role.role_id"
                        variant="solid"
                    >
                        <template #header>
                            <div class="flex items-center justify-between">
                                <h3
                                    class="font-semibold text-lg text-neutral-900 dark:text-white"
                                >
                                    {{ role.name }}
                                </h3>
                                <UDropdownMenu :items="getRoleActions(role)">
                                    <UButton
                                        variant="ghost"
                                        color="neutral"
                                        size="xs"
                                        icon="i-lucide-more-vertical"
                                    />
                                </UDropdownMenu>
                            </div>
                        </template>

                        <p
                            class="text-sm text-neutral-500 dark:text-neutral-400 mb-4 h-10 line-clamp-2"
                        >
                            {{ role.description || "暂无描述" }}
                        </p>

                        <div class="space-y-2">
                            <div
                                class="text-xs font-medium text-neutral-500 dark:text-neutral-400"
                            >
                                权限 ({{ role.permissions.length }})
                            </div>
                            <div
                                class="flex flex-wrap gap-1 max-h-24 overflow-y-auto"
                            >
                                <UBadge
                                    v-for="perm in role.permissions"
                                    :key="perm"
                                    color="neutral"
                                    variant="soft"
                                    size="2xs"
                                >
                                    {{ perm }}
                                </UBadge>
                            </div>
                        </div>
                    </UCard>
                </div>
            </div>

            <!-- 权限字典 -->
            <div v-else-if="selectedTab === 2" class="space-y-4">
                <UCard variant="subtle" class="overflow-hidden">
                    <div class="-m-5 sm:-m-7">
                        <UTable
                            :data="permissions"
                            :columns="permissionColumns"
                            :empty-state="{
                                icon: 'i-lucide-book-key',
                                label: '暂无权限定义',
                            }"
                        >
                        <template #key-cell="{ row }">
                            <span
                                class="font-mono text-xs text-neutral-900 dark:text-white"
                                >{{ row.original.key }}</span
                            >
                        </template>
                        <template #description-cell="{ row }">
                            <span
                                class="text-neutral-600 dark:text-neutral-300"
                                >{{ row.original.description }}</span
                            >
                        </template>
                    </UTable>
                    </div>
                </UCard>
            </div>
        </div>

        <!-- 模态框组件 -->
        <GrantAdminModal
            v-model:open="showGrantModal"
            :title="modalTitle"
            :is-editing="isEditing"
            :can-set-super="true"
            :role-options="roleOptions"
            :initial="modalInitial"
            @close="handleModalClose"
        />
        <!-- CreateRoleModal 待实现 -->
    </div>
</template>

<script setup lang="ts">
import { useAdminRbacService } from "~/services/admin_rbac.service";
import GrantAdminModal, {
    type GrantAdminPayload,
} from "~/components/modals/admin/GrantAdminModal.vue";
import type {
    AdminAdminItem,
    AdminRoleItem,
    AdminPermissionItem,
} from "~/types/api";

definePageMeta({
    layout: "dashboard",
    requiresAuth: true,
    middleware: ["admin"],
});

const rbacService = useAdminRbacService();
const feedback = useFeedbackStore();

const selectedTab = ref(0);
const tabs = [
    { label: "管理员", slot: "admins", icon: "i-lucide-users" },
    { label: "角色", slot: "roles", icon: "i-lucide-shield" },
    { label: "权限字典", slot: "permissions", icon: "i-lucide-book-key" },
];

const adminColumns = [
    { accessorKey: "user", header: "用户" },
    { accessorKey: "type", header: "类型" },
    { accessorKey: "role", header: "角色" },
    { id: "actions", header: "操作", class: "text-right" },
];

const permissionColumns = [
    { accessorKey: "key", header: "权限Key" },
    { accessorKey: "description", header: "描述" },
];

// State
const loadingAdmins = ref(false);
const admins = ref<AdminAdminItem[]>([]);
const roles = ref<AdminRoleItem[]>([]);
const permissions = ref<AdminPermissionItem[]>([]);

const showGrantModal = ref(false);
const currentAdmin = ref<AdminAdminItem | undefined>(undefined);

const modalTitle = computed(() =>
    currentAdmin.value ? "编辑管理员" : "添加管理员",
);
const isEditing = computed(() => !!currentAdmin.value);
const roleOptions = computed(() =>
    roles.value.map((r) => ({ label: r.name, value: r.role_id })),
);
const modalInitial = computed(() => {
    if (!currentAdmin.value) return undefined;
    return {
        user_id: currentAdmin.value.user.user_id,
        grant_all_permissions: currentAdmin.value.permissions.includes("*"),
        role_ids: currentAdmin.value.roles.map((r) => r.role_id),
    };
});

// Init
onMounted(async () => {
    await Promise.all([refreshAdmins(), refreshRoles(), refreshPermissions()]);
});

async function refreshAdmins() {
    loadingAdmins.value = true;
    try {
        const res = await rbacService.listAdmins();
        admins.value = res.items;
    } catch (e) {
        // error
    } finally {
        loadingAdmins.value = false;
    }
}

async function refreshRoles() {
    try {
        const res = await rbacService.listRoles();
        roles.value = res.items;
    } catch (e) {
        // error
    }
}

async function refreshPermissions() {
    try {
        const res = await rbacService.listPermissions();
        permissions.value = res.items;
    } catch (e) {
        // error
    }
}

// Actions
function openGrantModal() {
    currentAdmin.value = undefined;
    showGrantModal.value = true;
}

function editAdmin(admin: AdminAdminItem) {
    currentAdmin.value = admin;
    showGrantModal.value = true;
}

async function revokeAdmin(admin: AdminAdminItem) {
    if (!confirm(`确定要撤销 ${admin.user.nickname} 的管理员权限吗？`)) return;
    try {
        await rbacService.revokeAdmin({ user_id: admin.user.user_id });
        feedback.success("操作成功");
        refreshAdmins();
    } catch (e) {
        feedback.error("操作失败");
    }
}

async function handleModalClose(payload: GrantAdminPayload | null) {
    if (!payload) {
        showGrantModal.value = false;
        return;
    }
    try {
        if (isEditing.value) {
            // Edit not implemented in API yet or reuse grant
            await rbacService.grantAdmin(payload);
        } else {
            await rbacService.grantAdmin(payload);
        }
        feedback.success("操作成功");
        refreshAdmins();
        showGrantModal.value = false;
    } catch (e) {
        feedback.error("操作失败");
    }
}

function openCreateRoleModal() {
    // TODO: Implement CreateRoleModal
    feedback.info("功能开发中");
}

function getRoleActions(role: AdminRoleItem) {
    return [
        [
            {
                label: "编辑角色",
                icon: "i-lucide-edit",
                click: () => {
                    /* TODO */
                },
            },
            {
                label: "配置权限",
                icon: "i-lucide-key",
                click: () => {
                    /* TODO */
                },
            },
        ],
        [
            {
                label: "删除角色",
                icon: "i-lucide-trash-2",
                color: "error" as const,
                click: () => {
                    /* TODO */
                },
            },
        ],
    ];
}
</script>
