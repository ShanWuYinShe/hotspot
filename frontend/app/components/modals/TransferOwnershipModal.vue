<template>
  <UModal size="md">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-arrow-right-circle" class="w-6 h-6 text-warning-500" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
              转让群主
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <!-- 警告提示 -->
        <UAlert
          color="warning"
          variant="soft"
          title="重要提示"
          description="转让群主后，您将失去群主权限，新群主将拥有最高管理权限。此操作不可撤销。"
          class="mb-6"
        />

        <!-- 搜索成员 -->
        <div class="mb-4">
          <UInput
            v-model="searchQuery"
            placeholder="搜索成员..."
            icon="i-heroicons-magnifying-glass"
            :disabled="isLoading"
          />
        </div>

        <!-- 成员列表 -->
        <div v-if="isLoading" class="space-y-2">
          <USkeleton class="h-16" v-for="i in 3" :key="i" />
        </div>

        <div v-else-if="filteredMembers.length > 0" class="max-h-96 overflow-y-auto space-y-2">
          <div
            v-for="member in filteredMembers"
            :key="member.user_id"
            class="flex items-center justify-between p-3 rounded-lg border border-neutral-200 dark:border-neutral-700 hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
          >
            <div class="flex items-center gap-3">
              <UAvatar
                :src="member.avatar_url"
                :alt="member.nickname"
                size="md"
              />
              <div>
                <div class="font-medium text-neutral-900 dark:text-white">
                  {{ member.nickname }}
                </div>
                <div class="text-xs text-neutral-500">
                  {{ getRoleLabel(member.role) }}
                </div>
              </div>
            </div>

            <UButton
              size="sm"
              color="warning"
              @click="selectMember(member)"
              :disabled="isTransferring"
            >
              选择
            </UButton>
          </div>
        </div>

        <EmptyState
          v-else
          icon="i-heroicons-users"
          title="暂无成员"
          description="没有可以转让的成员"
        />

        <!-- 操作按钮 -->
        <div class="flex gap-3 pt-4 mt-6 border-t border-neutral-200 dark:border-neutral-700">
          <UButton
            variant="ghost"
            class="flex-1"
            @click="handleClose"
            :disabled="isTransferring"
          >
            取消
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useGroupsService } from '~/services/groups.service'

interface Member {
  user_id: string
  nickname: string
  avatar_url?: string
  role: 'owner' | 'admin' | 'member'
}

interface Props {
  groupId: string
  members: Member[]
  onSuccess?: (newOwnerId: string) => void
  onClose?: () => void
}

const props = defineProps<Props>()
const toast = useToast()
const modal = useModal()

const groupsService = useGroupsService()

const searchQuery = ref('')
const isLoading = ref(false)
const isTransferring = ref(false)

// 可以转让的成员（排除群主自己）
const availableMembers = computed(() => {
  return props.members.filter(m => m.role !== 'owner')
})

// 过滤成员列表
const filteredMembers = computed(() => {
  if (!searchQuery.value.trim()) {
    return availableMembers.value
  }
  
  const query = searchQuery.value.toLowerCase()
  return availableMembers.value.filter(member => 
    member.nickname.toLowerCase().includes(query)
  )
})

// 获取角色标签
const getRoleLabel = (role: string) => {
  const labels: Record<string, string> = {
    admin: '管理员',
    member: '普通成员'
  }
  return labels[role] || role
}

// 选择成员
const selectMember = (member: Member) => {
  modal.showConfirmModal({
    title: '确认转让',
    message: `确定将群主转让给 "${member.nickname}" 吗？此操作不可撤销。`,
    confirmText: '确认转让',
    cancelText: '取消',
    confirmColor: 'warning',
    onConfirm: async () => {
      await transferOwnership(member)
    }
  })
}

// 转让群主
const transferOwnership = async (member: Member) => {
  isTransferring.value = true
  
  try {
    await groupsService.transferOwnership({
      group_chat_id: props.groupId,
      new_owner_id: member.user_id
    })
    
    toast.add({
      title: '转让成功',
      description: `已将群主转让给 ${member.nickname}`,
      color: 'success'
    })
    
    props.onSuccess?.(member.user_id)
    handleClose()
  } catch (error) {
    console.error('转让失败:', error)
    toast.add({
      title: '转让失败',
      description: (error as Error).message,
      color: 'error'
    })
  } finally {
    isTransferring.value = false
  }
}

const handleClose = () => {
  if (!isTransferring.value) {
    props.onClose?.()
  }
}
</script>



