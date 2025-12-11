<template>
  <UModal :title="`举报${getTargetTypeLabel(targetType)}`">
    <template #body="{ close }">
      <UFormField
        label="请选择举报原因"
        required
        class="mb-6"
      >
        <URadioGroup
          v-model="selectedReason"
          :items="reportReasons"
          class="space-y-2"
        />
      </UFormField>

      <UFormField
        label="详细描述（可选）"
        :hint="`${description.length}/500`"
        class="mb-6"
      >
        <UTextarea
          v-model="description"
          placeholder="请详细描述您举报的原因，这将帮助我们更好地处理..."
          :rows="4"
          :maxlength="500"
        />
      </UFormField>

      <UAlert
        color="warning"
        variant="subtle"
        icon="i-heroicons-information-circle"
        title="举报须知"
        class="mb-6"
      >
        <template #description>
          <ul class="space-y-1 text-xs mt-1">
            <li>• 我们会认真处理每一条举报，通常在24小时内给出处理结果</li>
            <li>• 恶意举报或虚假举报可能导致您的账户受到限制</li>
            <li>• 举报记录将被保存，您可以在设置中查看处理状态</li>
          </ul>
        </template>
      </UAlert>
    </template>

    <template #footer="{ close }">
      <UButton
        variant="outline"
        :disabled="submitting"
        @click="close"
      >
        取消
      </UButton>
      <UButton
        color="error"
        :loading="submitting"
        :disabled="!selectedReason"
        @click="() => handleSubmit(close)"
      >
        提交举报
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useReportsService } from '~/services/reports.service'
import { getErrorMessage } from '~/utils/error'

interface Props {
  targetType: 'user' | 'moment' | 'comment' | 'message' | 'group'
  targetId: string
  targetTitle?: string
  onSuccess?: () => void
}

const props = withDefaults(defineProps<Props>(), {
  targetTitle: ''
})

const submitting = ref(false)
const selectedReason = ref('')
const description = ref('')
const toast = useToast()

const reportReasons = computed(() => {
  const baseReasons = [
    { value: 'spam', label: '垃圾信息', description: '发布广告、刷屏、重复内容等' },
    { value: 'harassment', label: '骚扰行为', description: '恶意骚扰、人身攻击、威胁等' },
    { value: 'inappropriate_content', label: '不当内容', description: '色情、暴力、仇恨言论等不当内容' },
    { value: 'violence', label: '暴力内容', description: '包含暴力、血腥、恐怖等内容' },
    { value: 'other', label: '其他', description: '其他违反社区规范的行为' }
  ]

  if (props.targetType === 'user') {
    baseReasons.splice(3, 0, {
      value: 'fake_account',
      label: '虚假账户',
      description: '冒充他人、机器人账户、虚假信息等'
    })
  }

  return baseReasons
})

const getTargetTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    user: '用户',
    moment: '生活瞬间',
    comment: '评论',
    message: '消息',
    group: '群组'
  }
  return labels[type] || type
}

const handleSubmit = async (close: () => void) => {
  if (!selectedReason.value) {
    toast.add({ color: 'error', title: '请选择举报原因' })
    return
  }

  submitting.value = true

  try {
    const reportsService = useReportsService()
    await reportsService.create({
      target_type: props.targetType,
      target_id: props.targetId,
      reason: selectedReason.value as any,
      description: description.value.trim() || undefined
    })

    toast.add({ color: 'success', title: '举报提交成功', description: '我们会尽快处理您的举报' })
    props.onSuccess?.()
    close()
  } catch (err: unknown) {
    console.error('提交举报失败:', err)
    toast.add({ color: 'error', title: '提交失败', description: getErrorMessage(err, '网络错误，请稍后重试') })
  } finally {
    submitting.value = false
  }
}
</script>
