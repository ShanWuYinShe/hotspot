<template>
  <div class="w-full">
    <div v-if="canRenderAdmin">
      <UCard v-if="!canManageRbac">
        <div class="py-6">
          <UEmpty
            icon="i-lucide-lock-keyhole"
            title="无权限"
            description="你没有管理员权限管理（admin.rbac.manage）权限。"
          />
        </div>
      </UCard>

      <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- 管理员列表 -->
        <UCard>
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-users-round" :size="20" />
              <h2 class="font-semibold">管理员</h2>
            </div>
            <UButton
              icon="i-lucide-plus"
              size="sm"
              color="primary"
              @click="openGrantModal()"
            >
              授权
            </UButton>
          </div>
        </template>

        <UTable
          :data="admins"
          :columns="adminColumns"
          :loading="loadingAdmins"
          :loading-state="{ icon: 'i-lucide-refresh-cw', label: '加载中...' }"
          :empty-state="{ icon: 'i-lucide-users-round', label: '暂无管理员' }"
        >
          <template #user-data="{ row }">
            <div class="flex items-center gap-2 min-w-0">
              <UAvatar :src="row.original.user.avatar_url" :alt="row.original.user.nickname" size="sm" />
              <div class="min-w-0">
                <div class="font-medium truncate">{{ row.original.user.nickname }}</div>
                <UBadge color="neutral" variant="soft" size="xs">ID: {{ row.original.user.user_id }}</UBadge>
              </div>
            </div>
          </template>
          <template #super-data="{ row }">
            <UBadge :label="row.original.is_super ? '是' : '否'" :color="row.original.is_super ? 'green' : 'neutral'" variant="soft" />
          </template>
          <template #roles-data="{ row }">
            <div class="flex flex-wrap gap-1">
              <UBadge
                v-for="r in row.original.roles"
                :key="r.id"
                :label="r.name"
                color="neutral"
                variant="subtle"
                size="xs"
              />
              <UBadge v-if="row.original.roles.length === 0" color="neutral" variant="soft" size="xs">无角色</UBadge>
            </div>
          </template>
          <template #actions-data="{ row }">
            <div class="flex items-center gap-1">
              <UButton
                icon="i-lucide-pencil"
                variant="ghost"
                color="neutral"
                size="sm"
                tooltip="编辑"
                @click="openGrantModal(row.original)"
              />
              <UButton
                icon="i-lucide-trash-2"
                variant="ghost"
                color="error"
                size="sm"
                tooltip="撤销"
                :disabled="row.original.is_super && !adminStore.isSuper"
                @click="revokeAdmin(row.original.user.user_id)"
              />
            </div>
          </template>
        </UTable>
      </UCard>

      <!-- 角色列表 -->
      <UCard>
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-shield-check" :size="20" />
              <h2 class="font-semibold">角色</h2>
            </div>
            <UButton
              icon="i-lucide-plus"
              size="sm"
              color="primary"
              @click="openRoleModal()"
            >
              新建角色
            </UButton>
          </div>
        </template>

        <UTable
          :data="roles"
          :columns="roleColumns"
          :loading="loadingRoles"
          :loading-state="{ icon: 'i-lucide-refresh-cw', label: '加载中...' }"
          :empty-state="{ icon: 'i-lucide-shield-check', label: '暂无角色' }"
        >
          <template #name-data="{ row }">
            <div class="min-w-0">
              <div class="font-medium truncate">{{ row.original.name }}</div>
              <UBadge v-if="row.original.description" color="neutral" variant="soft" size="xs" class="mt-1 max-w-full truncate">
                {{ row.original.description }}
              </UBadge>
            </div>
          </template>
          <template #permissions-data="{ row }">
            <div class="flex flex-wrap gap-1">
              <UBadge
                v-for="p in row.original.permissions"
                :key="p"
                :label="p"
                color="neutral"
                variant="subtle"
                size="xs"
              />
              <UBadge v-if="row.original.permissions.length === 0" color="neutral" variant="soft" size="xs">无权限</UBadge>
            </div>
          </template>
          <template #actions-data="{ row }">
            <div class="flex items-center gap-1">
              <UButton
                icon="i-lucide-pencil"
                variant="ghost"
                color="neutral"
                size="sm"
                tooltip="编辑"
                @click="openRoleModal(row.original)"
              />
              <UButton
                icon="i-lucide-trash-2"
                variant="ghost"
                color="error"
                size="sm"
                tooltip="删除"
                @click="deleteRole(row.original.id)"
              />
            </div>
          </template>
        </UTable>
      </UCard>
      </div>
    </div>

    <div v-else class="w-full space-y-3">
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-10 w-full" />
      <USkeleton class="h-64 w-full" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { useAdminRbacService } from '~/services/admin_rbac.service'
import { useAdminStore } from '~/stores/admin.store'
import type { AdminAdminItem, AdminRoleItem, AdminPermissionItem } from '~/types/api'
import { useDashboardHeader } from '~/composables/useDashboardHeader'
import { useHydratedStore } from '~/composables/useHydratedStore'
import type { GrantAdminPayload } from '~/components/modals/admin/GrantAdminModal.vue'
import type { RolePayload } from '~/components/modals/admin/RoleModal.vue'

definePageMeta({
  title: '权限管理',
  description: '管理管理员账号、角色与权限',
  requiresAuth: true,
  layout: 'dashboard',
  middleware: 'admin',
  dashboardHeader: true
})

const rbac = useAdminRbacService()
const adminStore = useAdminStore()
const authStore = useAuthStore()
const isAuthenticated = useHydratedStore(authStore, 'isAuthenticated', false)
const canRenderAdmin = computed(() => isAuthenticated.value && adminStore.hasAccess)
const modal = useModal()
const feedback = useFeedbackStore()
const { withErrorHandling } = useErrorHandler()
const dashboardHeader = useDashboardHeader()
const overlay = useOverlay()

const canManageRbac = computed(() => adminStore.hasPermission('admin.rbac.manage'))

const loadingAdmins = ref(false)
const loadingRoles = ref(false)
const loadingPerms = ref(false)

const admins = ref<AdminAdminItem[]>([])
const roles = ref<AdminRoleItem[]>([])
const permissions = ref<AdminPermissionItem[]>([])

const adminColumns: TableColumn<AdminAdminItem>[] = [
  { id: 'user', accessorKey: 'user.user_id', header: '用户' },
  { id: 'super', accessorKey: 'is_super', header: 'Super' },
  { id: 'roles', accessorKey: 'roles', header: '角色' },
  { id: 'actions', accessorKey: 'user.user_id', header: '操作' }
]

const roleColumns: TableColumn<AdminRoleItem>[] = [
  { id: 'name', accessorKey: 'name', header: '角色' },
  { id: 'permissions', accessorKey: 'permissions', header: '权限' },
  { id: 'actions', accessorKey: 'id', header: '操作' }
]

const roleOptions = computed(() => roles.value.map(r => ({ label: r.name, value: r.id })))
const permissionOptions = computed(() => permissions.value.map(p => ({ label: `${p.key} · ${p.description}`, value: p.key })))

const loadAll = async () => {
  if (!canManageRbac.value) return
  loadingAdmins.value = true
  loadingRoles.value = true
  loadingPerms.value = true

  await Promise.all([
    withErrorHandling(async () => {
      const res = await rbac.listAdmins()
      admins.value = res.items
    }, { title: '加载管理员失败', loading: loadingAdmins }),
    withErrorHandling(async () => {
      const res = await rbac.listRoles()
      roles.value = res.items
    }, { title: '加载角色失败', loading: loadingRoles }),
    withErrorHandling(async () => {
      const res = await rbac.listPermissions()
      permissions.value = res.items
    }, { title: '加载权限字典失败', loading: loadingPerms })
  ])
}

// ===== 授权管理员 =====
const savingGrant = ref(false)
const openGrantModal = async (row?: AdminAdminItem) => {
  const { default: GrantAdminModal } = await import('~/components/modals/admin/GrantAdminModal.vue')
  const m = overlay.create(GrantAdminModal)
  const instance = m.open({
    title: row ? '编辑管理员' : '授权管理员',
    isEditing: !!row,
    canSetSuper: adminStore.isSuper,
    roleOptions: roleOptions.value,
    initial: row
      ? {
          user_id: row.user.user_id,
          is_super: row.is_super,
          role_ids: row.roles.map(r => r.id)
        }
      : undefined
  })

  const payload = await instance.result as GrantAdminPayload | null
  if (!payload) return

  savingGrant.value = true
  await withErrorHandling(async () => {
    await rbac.grantAdmin({
      user_id: payload.user_id,
      is_super: adminStore.isSuper ? payload.is_super : undefined,
      role_ids: payload.role_ids
    })
    feedback.success('操作成功', row ? '管理员已更新' : '管理员已授权')
    await loadAll()
  }, { title: '保存失败', loading: savingGrant })
}

const revokeAdmin = async (userId: number) => {
  const confirmed = await modal.showConfirm({
    title: '撤销管理员',
    message: `确定要撤销用户 ${userId} 的管理员权限吗？`,
    confirmText: '撤销',
    confirmColor: 'error'
  })
  if (!confirmed) return

  await withErrorHandling(async () => {
    await rbac.revokeAdmin({ user_id: userId })
    feedback.success('操作成功', '已撤销管理员')
    await loadAll()
  }, { title: '撤销失败' })
}

// ===== 角色管理 =====
const savingRole = ref(false)
const openRoleModal = async (row?: AdminRoleItem) => {
  const { default: RoleModal } = await import('~/components/modals/admin/RoleModal.vue')
  const m = overlay.create(RoleModal)
  const instance = m.open({
    title: row ? '编辑角色' : '新建角色',
    isEditing: !!row,
    permissionOptions: permissionOptions.value,
    initial: row
      ? { id: row.id, name: row.name, description: row.description, permission_keys: [...row.permissions] }
      : undefined
  })

  const payload = await instance.result as RolePayload | null
  if (!payload) return

  savingRole.value = true
  await withErrorHandling(async () => {
    if (!row) {
      const created = await rbac.createRole({ name: payload.name, description: payload.description })
      await rbac.setRolePermissions({ id: created.id, permission_keys: payload.permission_keys })
      feedback.success('角色已创建')
    } else {
      await rbac.updateRole({
        id: row.id,
        name: payload.name,
        description: payload.description,
        clear_description: payload.description === undefined ? true : undefined
      })
      await rbac.setRolePermissions({ id: row.id, permission_keys: payload.permission_keys })
      feedback.success('角色已更新')
    }
    await loadAll()
  }, { title: '保存角色失败', loading: savingRole })
}

const deleteRole = async (id: string) => {
  const confirmed = await modal.showConfirm({
    title: '删除角色',
    message: '确定要删除该角色吗？（该角色将从所有管理员中移除）',
    confirmText: '删除',
    confirmColor: 'error'
  })
  if (!confirmed) return

  await withErrorHandling(async () => {
    await rbac.deleteRole({ id })
    feedback.success('角色已删除')
    await loadAll()
  }, { title: '删除失败' })
}

watchEffect(() => {
  dashboardHeader.setActions([
    {
      key: 'refresh',
      label: '刷新',
      icon: 'i-lucide-refresh-cw',
      variant: 'outline',
      loading: loadingAdmins.value || loadingRoles.value || loadingPerms.value,
      action: () => loadAll()
    }
  ])
})

onMounted(() => {
  loadAll()
})
</script>

