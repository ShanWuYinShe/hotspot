<template>
  <UModal
    size="2xl"
    title="群组管理"
    description="管理群组设置、成员和权限"
  >
    <template #body="{ close }">
      <section class="space-y-6">
        <!-- 邀请新成员 -->
        <UCard>
          <template #header>
            <header class="flex items-center justify-between">
              <h4 class="font-medium">邀请新成员</h4>
              <UButton
                size="sm"
                :disabled="!canInvite"
                @click="showInviteModal"
              >
                <UIcon
                  name="i-heroicons-user-plus"
                  class="mr-1"
                />
                邀请成员
              </UButton>
            </header>
          </template>

          <UCard
            :ui="{ body: 'p-4' }"
            variant="subtle"
          >
            <header class="flex items-center justify-between mb-3">
              <article>
                <h5 class="font-medium text-sm">群组分享链接</h5>
                <p class="text-xs text-secondary-content mt-1">分享此链接让朋友快速找到并加入群组</p>
              </article>
              <UButton
                size="sm"
                variant="outline"
                :loading="generatingCode"
                @click="generateInviteCode"
              >
                <UIcon
                  name="i-heroicons-link"
                  class="mr-1"
                />
                生成链接
              </UButton>
            </header>

            <section
              v-if="inviteCode"
              class="flex items-center gap-2"
            >
              <UInput
                :model-value="inviteCode"
                readonly
                class="flex-1"
              />
              <UButton
                size="sm"
                variant="outline"
                icon="i-heroicons-clipboard"
                @click="copyInviteCode"
              />
              <UButton
                size="sm"
                variant="outline"
                icon="i-heroicons-share"
                @click="shareInviteCode"
              />
            </section>
          </UCard>
        </UCard>

        <!-- 成员管理 -->
        <UCard>
          <template #header>
            <h4 class="font-medium">成员管理</h4>
          </template>

          <ul class="space-y-3 max-h-64 overflow-y-auto">
            <li
              v-for="member in members"
              :key="member.user.user_id"
              class="flex items-center justify-between p-3 bg-surface rounded-lg"
            >
              <header class="flex items-center gap-3">
                <UAvatar
                  :alt="member.user.nickname"
                  :src="member.user.avatar_url"
                  size="sm"
                >
                  {{ member.user.nickname?.charAt(0) || 'U' }}
                </UAvatar>

                <article>
                  <p class="flex items-center gap-2">
                    <span class="font-medium text-sm">{{ member.user.nickname }}</span>
                    <UBadge
                      :color="getRoleColor(member.role)"
                      :label="getRoleLabel(member.role)"
                      size="xs"
                    />
                  </p>
                  <time class="text-xs text-secondary-content">加入时间：{{ formatTime(member.joined_at) }}</time>
                </article>
              </header>

              <UDropdownMenu
                v-if="canManageMember(member)"
                :items="getMemberActions(member)"
              >
                <UButton
                  variant="ghost"
                  icon="i-heroicons-ellipsis-horizontal"
                  size="sm"
                />
              </UDropdownMenu>
            </li>
          </ul>
        </UCard>

        <!-- 群组设置 -->
        <UCard>
          <template #header>
            <h4 class="font-medium">群组设置</h4>
          </template>

          <UForm
            :state="groupSettings"
            class="space-y-4"
          >
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

            <section class="flex items-center justify-between p-3 bg-surface rounded-lg">
              <article>
                <p class="text-sm font-medium text-secondary-strong">需要密码加入</p>
                <p class="text-xs text-secondary-content">开启后新成员需要输入密码才能加入</p>
              </article>
              <USwitch
                v-model="groupSettings.password_required"
                :disabled="!canEditGroup"
              />
            </section>

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
        <UCard v-if="canDeleteGroup">
          <template #header>
            <h4 class="font-medium text-error">危险操作</h4>
          </template>

          <ul class="space-y-3">
            <li class="flex items-center justify-between">
              <article>
                <h5 class="text-sm font-medium">转移群主</h5>
                <p class="text-xs text-secondary-content">将群主权限转移给其他管理员</p>
              </article>
              <UButton
                color="warning"
                variant="outline"
                size="sm"
                @click="() => transferOwnership(close)"
              >
                转移群主
              </UButton>
            </li>

            <li class="flex items-center justify-between">
              <article>
                <h5 class="text-sm font-medium text-error">解散群组</h5>
                <p class="text-xs text-secondary-content">永久删除群组，此操作不可撤销</p>
              </article>
              <UButton
                color="error"
                variant="outline"
                size="sm"
                @click="() => deleteGroup(close)"
              >
                解散群组
              </UButton>
            </li>
          </ul>
        </UCard>
      </section>
    </template>

    <template #footer="{ close }">
      <UButton
        variant="outline"
        :disabled="saving"
        @click="close"
      >
        取消
      </UButton>
      <UButton
        :loading="saving"
        :disabled="!canEditGroup"
        @click="() => saveSettings(close)"
      >
        保存设置
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { GROUP_CONSTANTS } from '~/config/defaults'
import type { GroupMemberItem } from '~/types/api'
import { useGroupsService } from '~/services/groups.service'
import { getErrorMessage } from '~/utils/error'

interface Props {
  groupId: string
  onSuccess?: () => void
}

const props = defineProps<Props>()

// 使用水合安全的 store 访问
const authStore = useAuthStore()

// 状态管理
const saving = ref(false)
const generatingCode = ref(false)
const inviteCode = ref('')
const members = ref<GroupMemberItem[]>([])
const currentUserRole = ref('member')

// 群组设置
const groupSettings = ref({
  name: '',
  description: '',
  max_members: 500,
  password_required: false,
  password: ''
})

// 计算属性
const canInvite = computed(() => ['owner', 'admin'].includes(currentUserRole.value))
const canEditGroup = computed(() => ['owner', 'admin'].includes(currentUserRole.value))
const canDeleteGroup = computed(() => currentUserRole.value === 'owner')

// 工具函数
const { formatTime } = useFormatters()
const toast = useToast()

// 定义 Badge 颜色类型
type BadgeColor = 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'

const getRoleColor = (role: string): BadgeColor => {
  const colors: Record<string, BadgeColor> = {
    owner: 'error',
    admin: 'warning',
    member: 'info'
  }
  return colors[role] || 'neutral'
}

const getRoleLabel = (role: string) => {
  const labels: Record<string, string> = {
    owner: '群主',
    admin: '管理员',
    member: '成员'
  }
  return labels[role] || role
}

const canManageMember = (member: GroupMemberItem): boolean => {
  if (currentUserRole.value === 'owner') return member.role !== 'owner'
  if (currentUserRole.value === 'admin') return member.role === 'member'
  return false
}

const getMemberActions = (member: GroupMemberItem) => {
  const actions = []

  if (currentUserRole.value === 'owner') {
    if (member.role === 'member') {
      actions.push([{
        label: '设为管理员',
        icon: 'i-heroicons-shield-check',
        click: () => promoteMember(member)
      }])
    } else if (member.role === 'admin') {
      actions.push([{
        label: '取消管理员',
        icon: 'i-heroicons-shield-exclamation',
        click: () => demoteMember(member)
      }])
    }
  }

  if (canManageMember(member)) {
    actions.push([{
      label: '踢出群组',
      icon: 'i-heroicons-user-minus',
      click: () => kickMember(member)
    }])
  }

  return actions
}

// 简化的异步数据管理
const dataLoading = ref(false)
const loadError = ref<string | null>(null)

const loadGroupData = async () => {
  if (dataLoading.value) return

  dataLoading.value = true
  loadError.value = null

  try {
    const groupService = useGroupsService()
    const geolocation = useGeolocation()
    const coords = await geolocation.getCoordinates()

    // 获取群组详情和成员列表
    const [settingsResponse, membersResponse] = await Promise.all([
      groupService.getSettings({ 
        id: props.groupId
      }),
      groupService.getMembers({ id: props.groupId })
    ])

    // 设置群组基本信息
    groupSettings.value = {
      name: settingsResponse.settings?.location_name || '',
      description: '',
      max_members: settingsResponse.settings?.max_members || GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT,
      password_required: settingsResponse.has_password || false,
      password: ''
    }

    // 设置成员列表
    members.value = membersResponse.items || []

    // 获取当前用户在群组中的角色
    const currentUserId = authStore.user?.user_id
    const currentMember = members.value.find((m) => m.user?.user_id === currentUserId)
    currentUserRole.value = currentMember?.role || 'member'

    return { groupSettings: groupSettings.value, members: members.value }
  } catch (err: unknown) {
    loadError.value = getErrorMessage(err, '加载失败')
    toast.add({ color: 'error', title: '加载失败' })
    throw err
  } finally {
    dataLoading.value = false
  }
}

const generateInviteCode = async () => {
  generatingCode.value = true
  try {
    // 生成群组分享链接（用户点击后进入群组详情页，可以申请加入）
    const baseUrl = window.location.origin
    const groupLink = `${baseUrl}/groups/${props.groupId}`
    inviteCode.value = groupLink
    toast.add({ color: 'success', title: '分享链接已生成' })
  } catch (err: unknown) {
    console.error('生成分享链接失败:', err)
    toast.add({ color: 'error', title: '生成失败', description: getErrorMessage(err) })
  } finally {
    generatingCode.value = false
  }
}

const copyInviteCode = async () => {
  try {
    await navigator.clipboard.writeText(inviteCode.value)
    toast.add({ color: 'success', title: '邀请码已复制到剪贴板' })
  } catch (err) {
    toast.add({ color: 'error', title: '复制失败' })
  }
}

const shareInviteCode = async () => {
  const shareData = {
    title: `加入群组：${groupSettings.value.name}`,
    text: `邀请您加入我们的群组！邀请码：${inviteCode.value}`,
    url: `${window.location.origin}/groups/join?code=${inviteCode.value}`
  }

  try {
    if (navigator.share) {
      await navigator.share(shareData)
    } else {
      await navigator.clipboard.writeText(shareData.url)
      toast.add({ color: 'success', title: '邀请链接已复制到剪贴板' })
    }
  } catch (err) {
    console.error('分享失败:', err)
  }
}

const showInviteModal = async () => {
  const modal = useModal()
  await modal.showInviteMembers(props.groupId)
}

const promoteMember = async (member: GroupMemberItem) => {
  const { showConfirm } = useModal()
  const confirmed = await showConfirm({
    title: '设为管理员',
    message: `确定要将 ${member.user?.nickname} 设为管理员吗？`,
    confirmText: '确认',
    cancelText: '取消'
  })

  if (!confirmed) return

  try {
    const groupService = useGroupsService()
    await groupService.setMemberRole({
      id: props.groupId,
      user_id: member.user.user_id,
      role: 'admin'
    })
    member.role = 'admin'
    toast.add({ color: 'success', title: '设置成功' })
  } catch (err: unknown) {
    toast.add({ color: 'error', title: '设置失败', description: getErrorMessage(err) })
  }
}

const demoteMember = async (member: GroupMemberItem) => {
  const { showConfirm } = useModal()
  const confirmed = await showConfirm({
    title: '取消管理员',
    message: `确定要取消 ${member.user?.nickname} 的管理员权限吗？`,
    confirmText: '确认',
    cancelText: '取消'
  })

  if (!confirmed) return

  try {
    const groupService = useGroupsService()
    await groupService.setMemberRole({
      id: props.groupId,
      user_id: member.user.user_id,
      role: 'member'
    })
    member.role = 'member'
    toast.add({ color: 'success', title: '取消成功' })
  } catch (err: unknown) {
    toast.add({ color: 'error', title: '取消失败', description: getErrorMessage(err) })
  }
}

const kickMember = async (member: GroupMemberItem) => {
  const { showConfirm } = useModal()
  const confirmed = await showConfirm({
    title: '踢出群组',
    message: `确定要将 ${member.user?.nickname} 踢出群组吗？`,
    confirmText: '确认踢出',
    cancelText: '取消',
    confirmColor: 'error'
  })

  if (!confirmed) return

  try {
    const groupService = useGroupsService()
    await groupService.removeMember({
      id: props.groupId,
      user_id: member.user.user_id
    })
    const userId = member.user.user_id
    const index = members.value.findIndex((m) => m.user.user_id === userId)
    if (index > -1) {
      members.value.splice(index, 1)
    }
    toast.add({ color: 'success', title: '踢出成功' })
  } catch (err: unknown) {
    toast.add({ color: 'error', title: '踢出失败', description: getErrorMessage(err) })
  }
}

const transferOwnership = async (close: () => void) => {
  const modal = useModal()
  const newOwnerId = await modal.showTransferOwnership(
    props.groupId,
    members.value.filter(m => m.role !== 'owner')
  )
  if (newOwnerId) {
    loadGroupData()
    close()
  }
}

const deleteGroup = async (close: () => void) => {
  const { showConfirm } = useModal()
  const confirmed = await showConfirm({
    title: '解散群组',
    message: `确定要解散群组"${groupSettings.value.name}"吗？此操作不可撤销，所有成员将被移除，群组数据将被永久删除。`,
    confirmText: '确认解散',
    cancelText: '取消',
    confirmColor: 'error'
  })

  if (!confirmed) return

  try {
    const groupService = useGroupsService()
    await groupService.deleteGroup({ id: props.groupId })
    toast.add({ color: 'success', title: '群组已解散' })
    close()
  } catch (err: unknown) {
    toast.add({ color: 'error', title: '解散失败', description: getErrorMessage(err) })
  }
}

const saveSettings = async (close: () => void) => {
  saving.value = true
  try {
    const groupService = useGroupsService()
    await groupService.update({
      id: props.groupId,
      name: groupSettings.value.name,
      description: groupSettings.value.description
    })
    toast.add({ color: 'success', title: '设置已保存' })
    props.onSuccess?.()
    close()
  } catch (err: unknown) {
    console.error('保存设置失败:', err)
    toast.add({ color: 'error', title: '保存失败', description: getErrorMessage(err) })
  } finally {
    saving.value = false
  }
}

// 页面加载
onMounted(() => {
  loadGroupData()
})
</script>
