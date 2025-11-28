<template>
  <UModal size="2xl">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-cog-6-tooth" class="w-6 h-6 text-purple-500" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
              群组管理
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <div class="space-y-6">
          <!-- 邀请新成员 -->
          <UCard>
            <template #header>
              <div class="flex items-center justify-between">
                <h4 class="font-medium">邀请新成员</h4>
                <UButton
                  size="sm"
                  @click="showInviteModal"
                  :disabled="!canInvite"
                >
                  <UIcon name="i-heroicons-user-plus" class="mr-1" />
                  邀请成员
                </UButton>
              </div>
            </template>

            <!-- 邀请码分享 -->
            <div class="bg-neutral-50 dark:bg-neutral-900 rounded-lg p-4">
              <div class="flex items-center justify-between mb-3">
                <div>
                  <h5 class="font-medium text-sm">群组邀请码</h5>
                  <p class="text-xs text-neutral-500 mt-1">
                    分享此邀请码让朋友快速加入群组
                  </p>
                </div>
                <UButton
                  size="sm"
                  variant="outline"
                  @click="generateInviteCode"
                  :loading="generatingCode"
                >
                  <UIcon name="i-heroicons-link" class="mr-1" />
                  生成邀请码
                </UButton>
              </div>
              
              <div v-if="inviteCode" class="flex items-center gap-2">
                <UInput
                  :model-value="inviteCode"
                  readonly
                  class="flex-1"
                />
                <UButton
                  size="sm"
                  variant="outline"
                  @click="copyInviteCode"
                >
                  <UIcon name="i-heroicons-clipboard" />
                </UButton>
                <UButton
                  size="sm"
                  variant="outline"
                  @click="shareInviteCode"
                >
                  <UIcon name="i-heroicons-share" />
                </UButton>
              </div>
            </div>
          </UCard>

          <!-- 成员管理 -->
          <UCard>
            <template #header>
              <h4 class="font-medium">成员管理</h4>
            </template>

            <div class="space-y-3 max-h-64 overflow-y-auto">
              <div
                v-for="member in members"
                :key="member.user_id"
                class="flex items-center justify-between p-3 bg-neutral-50 dark:bg-neutral-800 rounded-lg"
              >
                <div class="flex items-center gap-3">
                  <UAvatar
                    :alt="member.nickname"
                    :src="member.avatar_url"
                    size="sm"
                  >
                    {{ member.nickname?.charAt(0) || 'U' }}
                  </UAvatar>
                  
                  <div>
                    <div class="flex items-center gap-2">
                      <span class="font-medium text-sm">{{ member.nickname }}</span>
                      <UBadge
                        :color="getRoleColor(member.role)"
                        :label="getRoleLabel(member.role)"
                        size="xs"
                      />
                    </div>
                    <p class="text-xs text-neutral-500">
                      加入时间：{{ formatTime(member.joined_at) }}
                    </p>
                  </div>
                </div>
                
                <UDropdown
                  v-if="canManageMember(member)"
                  :items="getMemberActions(member)"
                  :popper="{ placement: 'bottom-end' }"
                >
                  <UButton
                    variant="ghost"
                    icon="i-heroicons-ellipsis-horizontal"
                    size="xs"
                  />
                </UDropdown>
              </div>
            </div>
          </UCard>

          <!-- 群组设置 -->
          <UCard>
            <template #header>
              <h4 class="font-medium">群组设置</h4>
            </template>

            <div class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                  群组名称
                </label>
                <UInput
                  v-model="groupSettings.name"
                  :disabled="!canEditGroup"
                />
              </div>
              
              <div>
                <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                  群组描述
                </label>
                <UTextarea
                  v-model="groupSettings.description"
                  :rows="3"
                  :disabled="!canEditGroup"
                />
              </div>
              
              <div>
                <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                  最大成员数
                </label>
                <UInput
                  v-model.number="groupSettings.max_members"
                  type="number"
                  :min="members.length"
                  :max="1000"
                  :disabled="!canEditGroup"
                />
              </div>
              
              <div class="flex items-center justify-between">
                <div>
                  <label class="text-sm font-medium text-neutral-700 dark:text-neutral-300">
                    需要密码加入
                  </label>
                  <p class="text-xs text-neutral-500">
                    开启后新成员需要输入密码才能加入
                  </p>
                </div>
                <UToggle
                  v-model="groupSettings.password_required"
                  :disabled="!canEditGroup"
                />
              </div>
              
              <div v-if="groupSettings.password_required">
                <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                  群组密码
                </label>
                <UInput
                  v-model="groupSettings.password"
                  type="password"
                  placeholder="设置新密码"
                  :disabled="!canEditGroup"
                />
              </div>
            </div>
          </UCard>

          <!-- 危险操作 -->
          <UCard v-if="canDeleteGroup">
            <template #header>
              <h4 class="font-medium text-red-600 dark:text-red-400">危险操作</h4>
            </template>

            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <div>
                  <h5 class="text-sm font-medium text-neutral-900 dark:text-white">
                    转移群主
                  </h5>
                  <p class="text-xs text-neutral-500">
                    将群主权限转移给其他管理员
                  </p>
                </div>
                <UButton
                  color="warning"
                  variant="outline"
                  size="sm"
                  @click="transferOwnership"
                >
                  转移群主
                </UButton>
              </div>
              
              <div class="flex items-center justify-between">
                <div>
                  <h5 class="text-sm font-medium text-red-600 dark:text-red-400">
                    解散群组
                  </h5>
                  <p class="text-xs text-neutral-500">
                    永久删除群组，此操作不可撤销
                  </p>
                </div>
                <UButton
                  color="error"
                  variant="outline"
                  size="sm"
                  @click="deleteGroup"
                >
                  解散群组
                </UButton>
              </div>
            </div>
          </UCard>
        </div>

        <!-- 底部操作 -->
        <div class="flex justify-end gap-3 mt-6 pt-4 border-t border-neutral-200 dark:border-neutral-700">
          <UButton
            variant="outline"
            @click="handleClose"
            :disabled="saving"
          >
            取消
          </UButton>
          <UButton
            @click="saveSettings"
            :loading="saving"
            :disabled="!canEditGroup"
          >
            保存设置
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { TIME_CONSTANTS, GROUP_CONSTANTS } from '~/config/defaults'
import type { GroupResponse, GroupMemberResponseData } from '~/types/api'
import { useGroupsService } from '~/services/groups.service'

interface Props {
  groupId: string
  onSuccess?: () => void
  onClose?: () => void
}

const props = defineProps<Props>()

// 状态管理
const loading = ref(false)
const saving = ref(false)
const generatingCode = ref(false)
const inviteCode = ref('')
const members = ref<GroupMemberResponseData[]>([])
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
const { showConfirm } = useModal()

// Toast 辅助函数
const success = (title: string, description?: string) => {
  toast.add({ title, description, color: 'success' })
}
const error = (title: string, description?: string) => {
  toast.add({ title, description, color: 'error' })
}

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

const canManageMember = (member: any) => {
  if (currentUserRole.value === 'owner') return member.role !== 'owner'
  if (currentUserRole.value === 'admin') return member.role === 'member'
  return false
}

const getMemberActions = (member: any) => {
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
    
    // 获取群组详情和成员列表
    const [groupResponse, membersResponse] = await Promise.all([
      groupService.openTheDoor({ id: props.groupId }),
      groupService.getMembers({ group_chat_id: props.groupId })
    ])
    
    // 设置群组基本信息
    groupSettings.value = {
      name: groupResponse.name || '',
      description: groupResponse.description || '',
      max_members: GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT,
      password_required: false,
      password: ''
    }
    
    // 设置成员列表
    members.value = membersResponse.items || []
    
    // 获取当前用户在群组中的角色
    const authStore = useAuthStore()
    const currentUserId = authStore.user?.user_id
    const currentMember = members.value.find(m => m.user_id === currentUserId)
    currentUserRole.value = currentMember?.role || 'member'
    
    return { groupSettings: groupSettings.value, members: members.value }
  } catch (err: any) {
    loadError.value = err.message || '加载失败'
    const toast = useToast()
    error('加载失败')
    throw err
  } finally {
    dataLoading.value = false
  }
}

const generateInviteCode = async () => {
  generatingCode.value = true
  try {
    const groupService = useGroupsService()
    // TODO: 实现邀请码生成
    const response = { invite_code: 'TEMP_CODE_' + Date.now() }
    inviteCode.value = response.invite_code
    success('邀请码生成成功')
  } catch (err: any) {
    console.error('生成邀请码失败:', err)
    error('生成失败')
  } finally {
    generatingCode.value = false
  }
}

const copyInviteCode = async () => {
  try {
    await navigator.clipboard.writeText(inviteCode.value)
    success('邀请码已复制到剪贴板')
  } catch (err) {
    error('复制失败')
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
      success('邀请链接已复制到剪贴板')
    }
  } catch (err) {
    console.error('分享失败:', err)
  }
}

const showInviteModal = () => {
  const { showInviteMembersModal } = useModal()
  showInviteMembersModal(props.groupId)
}

const promoteMember = async (member: any) => {
  const { showConfirm } = useModal()
  const confirmed = await showConfirm({
    title: '设为管理员',
    message: `确定要将 ${member.nickname} 设为管理员吗？`,
    confirmText: '确认',
    cancelText: '取消'
  })
  
  if (!confirmed) return
  
  try {
    const groupService = useGroupsService()
    await groupService.setMemberRole({
      group_chat_id: props.groupId,
      user_id: member.user_id,
      role: 'admin'
    })
    member.role = 'admin'
    success('设置成功')
  } catch (err: any) {
    error('设置失败')
  }
}

const demoteMember = async (member: any) => {
  const { showConfirm } = useModal()
  const confirmed = await showConfirm({
    title: '取消管理员',
    message: `确定要取消 ${member.nickname} 的管理员权限吗？`,
    confirmText: '确认',
    cancelText: '取消'
  })
  
  if (!confirmed) return
  
  try {
    const groupService = useGroupsService()
    await groupService.setMemberRole({
      group_chat_id: props.groupId,
      user_id: member.user_id,
      role: 'member'
    })
    member.role = 'member'
    success('取消成功')
  } catch (err: any) {
    error('取消失败')
  }
}

const kickMember = async (member: any) => {
  const { showConfirm } = useModal()
  const confirmed = await showConfirm({
    title: '踢出群组',
    message: `确定要将 ${member.nickname} 踢出群组吗？`,
    confirmText: '确认踢出',
    cancelText: '取消',
    confirmColor: 'error'
  })
  
  if (!confirmed) return
  
  try {
    const groupService = useGroupsService()
    await groupService.removeMember({
      group_chat_id: props.groupId,
      user_id: member.user_id
    })
    const index = members.value.findIndex(m => m.user_id === member.user_id)
    if (index > -1) {
      members.value.splice(index, 1)
    }
    success('踢出成功')
  } catch (err: any) {
    error('踢出失败')
  }
}

const transferOwnership = async () => {
  const { showTransferOwnershipModal } = useModal()
  showTransferOwnershipModal(
    props.groupId,
    members.value.filter(m => m.role !== 'owner'),
    () => {
      loadGroupData()
      handleClose()
    }
  )
}

const deleteGroup = async () => {
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
    await groupService.deleteGroup({ group_chat_id: props.groupId })
    success('群组已解散')
    handleClose()
  } catch (err: any) {
    error('解散失败')
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    const groupService = useGroupsService()
    await groupService.updateGroup({
      group_chat_id: props.groupId,
      name: groupSettings.value.name,
      description: groupSettings.value.description || undefined,
      settings: {
        max_members: groupSettings.value.max_members
      }
    })
    success('设置已保存')
    props.onSuccess?.()
  } catch (err: any) {
    console.error('保存设置失败:', err)
    error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleClose = () => {
  props.onClose?.()
}

// 页面加载
onMounted(() => {
  loadGroupData()
})
</script>
