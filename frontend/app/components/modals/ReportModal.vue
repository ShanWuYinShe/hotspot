<template>
  <UModal>
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center gap-3 mb-6">
          <UIcon name="i-heroicons-shield-exclamation" class="w-6 h-6 text-red-500" />
          <h3 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
            举报{{ getTargetTypeLabel(targetType) }}
          </h3>
        </div>

        <!-- 举报原因选择 -->
        <div class="mb-6">
          <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-3">
            请选择举报原因 *
          </label>
          <div class="space-y-2">
            <div
              v-for="reason in reportReasons"
              :key="reason.value"
              class="flex items-center"
            >
              <URadio
                v-model="selectedReason"
                :value="reason.value"
                :label="reason.label"
                :description="reason.description"
              />
            </div>
          </div>
        </div>

        <!-- 详细描述 -->
        <div class="mb-6">
          <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
            详细描述（可选）
          </label>
          <UTextarea
            v-model="description"
            placeholder="请详细描述您举报的原因，这将帮助我们更好地处理..."
            :rows="4"
            :maxlength="500"
          />
          <div class="flex justify-end mt-1">
            <span class="text-xs text-neutral-500">
              {{ description.length }}/500
            </span>
          </div>
        </div>

        <!-- 举报须知 -->
        <div class="mb-6 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
          <div class="flex items-start gap-2">
            <UIcon name="i-heroicons-information-circle" class="w-5 h-5 text-amber-600 mt-0.5" />
            <div class="text-sm text-amber-800 dark:text-amber-200">
              <p class="font-medium mb-1">举报须知：</p>
              <ul class="space-y-1 text-xs">
                <li>• 我们会认真处理每一条举报，通常在24小时内给出处理结果</li>
                <li>• 恶意举报或虚假举报可能导致您的账户受到限制</li>
                <li>• 举报记录将被保存，您可以在设置中查看处理状态</li>
              </ul>
            </div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex justify-end gap-3">
          <UButton
            variant="outline"
            @click="handleCancel"
            :disabled="submitting"
          >
            取消
          </UButton>
          <UButton
            color="red"
            @click="handleSubmit"
            :loading="submitting"
            :disabled="!selectedReason"
          >
            提交举报
          </UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
interface Props {
  targetType: 'user' | 'moment' | 'comment' | 'message' | 'group'
  targetId: string
  targetTitle?: string // 被举报对象的标题或名称
  onSuccess?: () => void
  onCancel?: () => void
  onClose?: () => void
}

const props = withDefaults(defineProps<Props>(), {
  targetTitle: ''
})

// 状态管理
const submitting = ref(false)
const selectedReason = ref('')
const description = ref('')

// 举报原因配置
const reportReasons = computed(() => {
  const baseReasons = [
    {
      value: 'spam',
      label: '垃圾信息',
      description: '发布广告、刷屏、重复内容等'
    },
    {
      value: 'harassment',
      label: '骚扰行为',
      description: '恶意骚扰、人身攻击、威胁等'
    },
    {
      value: 'inappropriate_content',
      label: '不当内容',
      description: '色情、暴力、仇恨言论等不当内容'
    },
    {
      value: 'violence',
      label: '暴力内容',
      description: '包含暴力、血腥、恐怖等内容'
    },
    {
      value: 'other',
      label: '其他',
      description: '其他违反社区规范的行为'
    }
  ]

  // 针对用户举报添加特定原因
  if (props.targetType === 'user') {
    baseReasons.splice(3, 0, {
      value: 'fake_account',
      label: '虚假账户',
      description: '冒充他人、机器人账户、虚假信息等'
    })
  }

  return baseReasons
})

// 工具函数
const toast = useToast()

const getTargetTypeLabel = (type: string) => {
  const labels = {
    user: '用户',
    moment: '生活瞬间',
    comment: '评论',
    message: '消息',
    group: '群组'
  }
  return labels[type] || type
}

// 方法
const handleSubmit = async () => {
  if (!selectedReason.value) {
    error('请选择举报原因')
    return
  }

  submitting.value = true

  try {
    const reportsService = useReportsService()
    await reportsService.createReport({
      target_type: props.targetType,
      target_id: props.targetId,
      reason: selectedReason.value,
      description: description.value.trim() || undefined
    })

    success('举报提交成功', '我们会尽快处理您的举报')
    
    // 调用成功回调
    props.onSuccess?.()
    
    // 关闭Modal
    handleClose()
  } catch (err: any) {
    console.error('提交举报失败:', err)
    error('提交失败', err?.message || '网络错误，请稍后重试')
  } finally {
    submitting.value = false
  }
}

const handleCancel = () => {
  props.onCancel?.()
  handleClose()
}

const handleClose = () => {
  // 重置表单
  selectedReason.value = ''
  description.value = ''
  
  props.onClose?.()
}

// 监听ESC键关闭
onMounted(() => {
  const handleKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      handleCancel()
    }
  }

  document.addEventListener('keydown', handleKeydown)

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })
})
</script>
