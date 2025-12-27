<template>
  <UModal
    width="md"
    title="修改密码"
    description="定期修改密码可以提高账户安全性"
  >
    <template #body="{ close }">
      <UForm
        :state="form"
        class="space-y-5"
        @submit="() => handleSubmit(close)"
      >
        <UFormField
          label="当前密码"
          name="oldPassword"
          :required="true"
        >
          <UInput
            v-model="form.oldPassword"
            type="password"
            placeholder="请输入当前密码"
            :disabled="isSubmitting"
          />
        </UFormField>

        <UFormField
          label="新密码"
          name="newPassword"
          hint="密码长度至少为8个字符，建议包含大小写字母、数字和特殊字符"
          :required="true"
        >
          <UInput
            v-model="form.newPassword"
            type="password"
            placeholder="请输入新密码（至少8个字符）"
            :disabled="isSubmitting"
            minlength="8"
          />
        </UFormField>

        <UFormField
          label="确认新密码"
          name="confirmPassword"
          :required="true"
        >
          <UInput
            v-model="form.confirmPassword"
            type="password"
            placeholder="请再次输入新密码"
            :disabled="isSubmitting"
          />
        </UFormField>

        <UAlert
          v-if="errorMessage"
          color="error"
          variant="soft"
          :title="errorMessage"
          icon="i-lucide-triangle-alert"
        />
      </UForm>
    </template>

    <template #footer="{ close }">
      <UButton
        variant="ghost"
        class="flex-1"
        :disabled="isSubmitting"
        @click="close"
      >
        取消
      </UButton>
      <UButton
        color="primary"
        class="flex-1"
        :loading="isSubmitting"
        :disabled="!isFormValid"
        @click="() => handleSubmit(close)"
      >
        确认修改
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { checkPasswordStrength } from '~/utils/password'
import { useAuthService } from '~/services/auth.service'
import { useAuthStore } from '~/stores/auth.store'

interface Props {
  onSuccess?: () => void
}

const props = defineProps<Props>()
const feedback = useFeedbackStore()

const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const isSubmitting = ref(false)
const errorMessage = ref('')

const isFormValid = computed(() => {
  return form.oldPassword.length >= 1 &&
         form.newPassword.length >= 8 &&
         form.confirmPassword.length >= 8 &&
         form.newPassword === form.confirmPassword
})

const handleSubmit = async (close: () => void) => {
  errorMessage.value = ''

  if (form.newPassword !== form.confirmPassword) {
    errorMessage.value = '两次输入的新密码不一致'
    return
  }

  const strength = checkPasswordStrength(form.newPassword)
  if (strength.level === 'weak') {
    errorMessage.value = '密码强度太弱，请使用更复杂的密码'
    return
  }

  isSubmitting.value = true

  try {
    const authService = useAuthService()
    const authStore = useAuthStore()

    await authService.changePassword({
      old_password: form.oldPassword,
      new_password: form.newPassword
    })

    feedback.success('密码修改成功', '请使用新密码重新登录')

    props.onSuccess?.()
    close()

    await authService.logout()
    authStore.logoutCurrent()
    await navigateTo('/auth?mode=login')
  } catch (error) {
    console.error('修改密码失败:', error)
    errorMessage.value = (error as Error).message || '修改密码失败，请检查当前密码是否正确'
  } finally {
    isSubmitting.value = false
  }
}
</script>
