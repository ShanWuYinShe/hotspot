<template>
  <UModal size="md">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-lock-closed" class="w-6 h-6 text-neutral-600 dark:text-neutral-300" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-white">
              修改密码
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <form @submit.prevent="handleSubmit" class="space-y-5">
          <!-- 提示信息 -->
          <UAlert
            color="info"
            variant="soft"
            title="您已通过 Token 认证"
            description="无需验证旧密码，直接设置新密码即可"
            icon="i-heroicons-information-circle"
          />

          <!-- 新密码 -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              新密码 *
            </label>
            <UInput
              v-model="form.newPassword"
              type="password"
              placeholder="请输入新密码（至少8个字符）"
              :disabled="isSubmitting"
              required
              minlength="8"
            />
            <p class="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
              密码长度至少为8个字符，建议包含大小写字母、数字和特殊字符
            </p>
          </div>

          <!-- 确认新密码 -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              确认新密码 *
            </label>
            <UInput
              v-model="form.confirmPassword"
              type="password"
              placeholder="请再次输入新密码"
              :disabled="isSubmitting"
              required
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
              color="primary"
              class="flex-1"
              :loading="isSubmitting"
              :disabled="!isFormValid"
            >
              确认修改
            </UButton>
          </div>
        </form>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { hashPassword, checkPasswordStrength } from '~/utils/password'
import { useAuthService } from '~/services/auth.service'
import { useAuthStore } from '~/stores/auth.store'

interface Props {
  onSuccess?: () => void
  onClose?: () => void
}

const props = defineProps<Props>()
const toast = useToast()

const form = reactive({
  newPassword: '',
  confirmPassword: ''
})

const isSubmitting = ref(false)
const errorMessage = ref('')

const isFormValid = computed(() => {
  return form.newPassword.length >= 8 &&
         form.confirmPassword.length >= 8 &&
         form.newPassword === form.confirmPassword
})

const handleClose = () => {
  if (!isSubmitting.value) {
    props.onClose?.()
  }
}

const handleSubmit = async () => {
  errorMessage.value = ''
  
  // 验证新密码和确认密码是否一致
  if (form.newPassword !== form.confirmPassword) {
    errorMessage.value = '两次输入的新密码不一致'
    return
  }
  
  // 验证密码强度
  const strength = checkPasswordStrength(form.newPassword)
  if (strength.level === 'weak') {
    errorMessage.value = '密码强度太弱，请使用更复杂的密码'
    return
  }
  
  isSubmitting.value = true
  
  try {
    const authService = useAuthService()
    const authStore = useAuthStore()
    
    // 计算新密码哈希
    const newPasswordHash = await hashPassword(form.newPassword)
    
    // 直接调用 set_password API（已登录用户无需验证旧密码）
    await authService.setPassword({
      password_hash: newPasswordHash
    })
    
    toast.add({
      title: '密码修改成功',
      description: '请使用新密码重新登录',
      color: 'success'
    })
    
    props.onSuccess?.()
    handleClose()
    
    // 登出并跳转到登录页
    await authStore.logout()
    await navigateTo('/auth/login')
  } catch (error) {
    console.error('修改密码失败:', error)
    errorMessage.value = (error as Error).message || '修改密码失败'
  } finally {
    isSubmitting.value = false
  }
}
</script>
