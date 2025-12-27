<template>
  <UModal
    width="md"
    title="转让群主"
    description="选择新的群主"
  >
    <template #body="{ close }">
      <UAlert
        color="warning"
        variant="soft"
        title="重要提示"
        description="转让群主后，您将失去群主权限，新群主将拥有最高管理权限。此操作不可撤销。"
        class="mb-6"
      />

      <UInput
        v-model="searchQuery"
        placeholder="搜索成员..."
        icon="i-lucide-search"
        :disabled="isLoading"
        class="mb-4"
      />

      <!-- 加载状态 -->
      <template v-if="isLoading">
        <USkeleton
          v-for="i in 3"
          :key="i"
          class="h-16 rounded-2xl mb-2"
        />
      </template>

      <!-- 成员列表 -->
      <section
        v-else-if="filteredMembers.length > 0"
        class="max-h-96 overflow-y-auto space-y-2"
      >
        <UCard
          v-for="member in filteredMembers"
          :key="member.user.user_id"
        >
          <section class="flex items-center justify-between gap-3">
            <header class="flex items-center gap-3 min-w-0">
              <UAvatar
                :src="member.user.avatar_url"
                :alt="member.user.nickname"
                size="md"
              />
              <article class="min-w-0">
                <p class="font-medium truncate">{{ member.user.nickname }}</p>
                <UBadge
                  variant="subtle"
                  color="neutral"
                  size="xs"
                >
                  {{ getRoleLabel(member.role) }}
                </UBadge>
              </article>
            </header>

            <UButton
              size="sm"
              color="warning"
              :disabled="isTransferring"
              @click="() => selectMember(member, close)"
            >
              选择
            </UButton>
          </section>
        </UCard>
      </section>

      <UAlert
        v-else
        icon="i-lucide-users"
        color="neutral"
        variant="soft"
        title="暂无成员"
        description="没有可以转让的成员"
      />
    </template>

    <template #footer="{ close }">
      <UButton
        variant="ghost"
        class="flex-1"
        :disabled="isTransferring"
        @click="close"
      >
        取消
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import type { GroupMemberItem } from '~/types/api'
import { useGroupsService } from '~/services/groups.service'

interface Props {
  groupId: string
  members: GroupMemberItem[]
  onSuccess?: (newOwnerId: number) => void
}

const props = defineProps<Props>()
const feedback = useFeedbackStore()
const modal = useModal()
const groupsService = useGroupsService()

const searchQuery = ref('')
const isLoading = ref(false)
const isTransferring = ref(false)

const availableMembers = computed(() => props.members.filter(m => m.role !== 'owner'))

const filteredMembers = computed(() => {
  if (!searchQuery.value.trim()) return availableMembers.value
  const query = searchQuery.value.toLowerCase()
  return availableMembers.value.filter(member =>
    member.user.nickname.toLowerCase().includes(query)
  )
})

const getRoleLabel = (role: string) => {
  const labels: Record<string, string> = { admin: '管理员', member: '普通成员' }
  return labels[role] || role
}

const selectMember = async (member: GroupMemberItem, close: () => void) => {
  const confirmed = await modal.showConfirm({
    title: '确认转让',
    message: `确定将群主转让给 "${member.user.nickname}" 吗？此操作不可撤销。`,
    confirmText: '确认转让',
    cancelText: '取消',
    confirmColor: 'warning'
  })

  if (confirmed) {
    await transferOwnership(member, close)
  }
}

const transferOwnership = async (member: GroupMemberItem, close: () => void) => {
  isTransferring.value = true

  try {
    await groupsService.transferOwnership({
      id: props.groupId,
      new_owner_id: member.user.user_id
    })

    feedback.success('转让成功', `已将群主转让给 ${member.user.nickname}`)

    props.onSuccess?.(member.user.user_id)
    close()
  } catch (error) {
    console.error('转让失败:', error)
    feedback.error('转让失败', (error as Error).message)
  } finally {
    isTransferring.value = false
  }
}
</script>
