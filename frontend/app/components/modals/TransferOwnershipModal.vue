<template>
  <UModal
    size="md"
    title="转让群主"
    description="选择新的群主"
    @update:open="(val) => !val && emit('update:open', false)"
  >
    <template #body>
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
          class="h-16 rounded-lg mb-2"
        />
      </template>

      <!-- 成员列表 -->
      <section
        v-else-if="filteredMembers.length > 0"
        class="max-h-96 overflow-y-auto space-y-2 pr-1"
      >
        <UCard
          v-for="member in filteredMembers"
          :key="member.user.user_id"
          variant="outline"
          size="sm"
        >
          <section class="flex items-center justify-between gap-3">
            <header class="flex items-center gap-3 min-w-0">
              <UAvatar
                :src="member.user.avatar_url"
                :alt="member.user.nickname"
                size="sm"
              />
              <article class="min-w-0">
                <p class="font-semibold text-neutral-900 dark:text-white text-sm truncate">{{ member.user.nickname }}</p>
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
              size="xs"
              color="warning"
              variant="soft"
              :disabled="isTransferring"
              @click="selectMember(member)"
            >
              选择
            </UButton>
          </section>
        </UCard>
      </section>

      <UEmpty
        v-else
        icon="i-lucide-users"
        title="暂无成员"
        description="没有可以转让的成员"
        class="py-8"
      />
    </template>

    <template #footer>
      <UButton
        variant="ghost"
        color="neutral"
        class="w-full"
        :disabled="isTransferring"
        @click="emit('update:open', false)"
      >
        取消
      </UButton>
    </template>
  </UModal>

  <ConfirmModal
    v-model:open="showConfirm"
    title="确认转让"
    :message="`确定将群主转让给 '${selectedMember?.user.nickname}' 吗？此操作不可撤销。`"
    confirm-text="确认转让"
    confirm-color="warning"
    @close="handleConfirm"
  />
</template>

<script setup lang="ts">
import type { GroupMemberItem } from '~/types/api'
import { useGroupsService } from '~/services/groups.service'
import ConfirmModal from './ConfirmModal.vue'

interface Props {
  groupId: string
  members: GroupMemberItem[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
    'update:open': [value: boolean]
    'success': [newOwnerId: number]
}>()

const feedback = useFeedbackStore()
const groupsService = useGroupsService()

const searchQuery = ref('')
const isLoading = ref(false)
const isTransferring = ref(false)
const showConfirm = ref(false)
const selectedMember = ref<GroupMemberItem | null>(null)

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

const selectMember = (member: GroupMemberItem) => {
  selectedMember.value = member
  showConfirm.value = true
}

const handleConfirm = async (confirmed: boolean) => {
    if (confirmed && selectedMember.value) {
        await transferOwnership(selectedMember.value)
    }
    showConfirm.value = false
}

const transferOwnership = async (member: GroupMemberItem) => {
  isTransferring.value = true

  try {
    await groupsService.transferOwnership({
      group_chat_id: props.groupId,
      new_owner_id: member.user.user_id
    })

    feedback.success('转让成功', `已将群主转让给 ${member.user.nickname}`)
    emit('success', member.user.user_id)
    emit('update:open', false)
  } catch (error) {
    feedback.error('转让失败')
  } finally {
    isTransferring.value = false
  }
}
</script>
