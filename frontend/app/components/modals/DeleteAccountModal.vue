<template>
  <UModal size="md">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-exclamation-triangle" class="w-6 h-6 text-error-500" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">
              删除账户
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <!-- 警告信息 -->
        <UAlert
          color="error"
          variant="soft"
          title="此操作无法撤销"
          description="删除账户后，您的所有数据将被永久删除，包括个人资料、发布的内容、好友关系等。"
          class="mb-6"
        />

        <form @submit.prevent="handleSubmit" class="space-y-5">
          <!-- 确认文本 -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              请输入 <code class="px-2 py-1 bg-neutral-100 dark:bg-neutral-800 rounded">DELETE_MY_ACCOUNT</code> 来确认删除 *
            </label>
            <UInput
              v-model="confirmText"
              placeholder="DELETE_MY_ACCOUNT"
              :disabled="isSubmitting"
              required
            />
          </div>

          <!-- 删除原因（可选） -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              删除原因（可选）
            </label>
            <UTextarea
              v-model="reason"
              placeholder="告诉我们您删除账户的原因，帮助我们改进产品"
              :rows="3"
              :disabled="isSubmitting"
            />
          </div>

          <!-- 错误提示 -->
          <UAlert
            v-if="errorMessage"
            color="error"
            variant="soft"
            :title="errorMessage"
            icon="i-heroicons-exclamation-triangle"
          />

          <!-- 操作按钮 -->
          <div class="flex gap-3 pt-4">
            <UButton
              type="button"
              variant="ghost"
              class="flex-1"
              @click="handleClose"
              :disabled="isSubmitting"
            >
              取消
            </UButton>
            <UButton
              type="submit"
              color="error"
              class="flex-1"
              :loading="isSubmitting"
              :disabled="!isFormValid"
            >
              确认删除
            </UButton>
          </div>
        </form>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useUserService } from '~/services/users.service'
import { useAuthStore } from '~/stores/auth.store'

interface Props {
  onSuccess?: () => void
  onClose?: () => void
}

const props = defineProps<Props>()
const toast = useToast()
const router = useRouter()
const authStore = useAuthStore()

const confirmText = ref('')
const reason = ref('')
const isSubmitting = ref(false)
const errorMessage = ref('')

const isFormValid = computed(() => {
  return confirmText.value === 'DELETE_MY_ACCOUNT'
})

const handleClose = () => {
  if (!isSubmitting.value) {
    props.onClose?.()
  }
}

const handleSubmit = async () => {
  errorMessage.value = ''
  
  if (!isFormValid.value) {
    errorMessage.value = '请正确输入确认文本'
    return
  }
  
  isSubmitting.value = true
  
  try {
    const userService = useUserService()
    
    await userService.deleteAccount({
      confirmation: 'DELETE_MY_ACCOUNT'
    })
    
    toast.add({
      title: '账户已删除',
      description: '感谢您使用我们的产品，再见',
      color: 'success'
    })
    
    // 登出并跳转到首页
    await authStore.logout()
    await router.push('/')
    
    props.onSuccess?.()
    handleClose()
  } catch (error) {
    console.error('删除账户失败:', error)
    errorMessage.value = (error as Error).message || '删除账户失败，请稍后重试'
  } finally {
    isSubmitting.value = false
  }
}
</script>
