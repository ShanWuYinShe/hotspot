<template>
  <UModal
    :title="title"
    @update:open="(open) => { if (!open) emit('close') }"
  >
    <template #body>
      <UForm
        :schema="activeSchema"
        :state="form"
        class="space-y-6"
        @submit="handleSubmit"
      >
        <UFormField
          v-if="step === 'request'"
          label="登录ID"
          name="login_id"
          :required="true"
          hint="邮箱或手机号"
        >
          <UInput v-model="form.login_id" placeholder="请输入登录ID" />
        </UFormField>

        <UFormField
          v-else-if="step === 'verify'"
          label="验证码"
          name="code"
          :required="true"
          :hint="`已发送到：${form.login_id}`"
        >
          <UInput
            v-model="form.code"
            placeholder="请输入 6 位验证码"
            inputmode="numeric"
            autocomplete="one-time-code"
          />
        </UFormField>

        <template v-else>
          <UFormField label="新密码" name="password" :required="true" hint="至少 6 位">
            <UInput v-model="form.password" type="password" placeholder="请输入新密码" />
          </UFormField>
          <UFormField label="确认新密码" name="confirm_password" :required="true">
            <UInput v-model="form.confirm_password" type="password" placeholder="请再次输入新密码" />
          </UFormField>
        </template>
      </UForm>
    </template>

    <template #footer>
      <UButton
        variant="outline"
        :disabled="loading"
        @click="emit('close')"
      >
        取消
      </UButton>

      <UButton
        v-if="step !== 'request'"
        variant="outline"
        :disabled="loading"
        @click="goBack"
      >
        上一步
      </UButton>

      <UButton
        :loading="loading"
        :disabled="!canContinue"
        @click="handleSubmit"
      >
        {{ primaryActionText }}
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import * as z from 'zod'
import { useAuthService } from '~/services/auth.service'

type Step = 'request' | 'verify' | 'reset'

const emit = defineEmits<{
  close: []
}>()

const authService = useAuthService()
const feedback = useFeedbackStore()

const step = ref<Step>('request')
const loading = ref(false)

const form = reactive({
  login_id: '',
  reset_id: '',
  code: '',
  reset_token: '',
  password: '',
  confirm_password: ''
})

const requestSchema = z.object({
  login_id: z.string().trim().min(1, '请输入登录ID')
})

const verifySchema = z.object({
  code: z.string().trim().regex(/^\d{6}$/, '请输入 6 位验证码')
})

const resetSchema = z.object({
  password: z.string().min(6, '密码至少 6 位').max(128, '密码最多 128 位'),
  confirm_password: z.string()
}).refine((v) => v.password === v.confirm_password, {
  message: '两次输入的密码不一致',
  path: ['confirm_password']
})

const activeSchema = computed(() => {
  if (step.value === 'request') return requestSchema
  if (step.value === 'verify') return verifySchema
  return resetSchema
})

const title = computed(() => {
  if (step.value === 'request') return '重置密码'
  if (step.value === 'verify') return '输入验证码'
  return '设置新密码'
})

const primaryActionText = computed(() => {
  if (step.value === 'request') return '发送验证码'
  if (step.value === 'verify') return '验证'
  return '重置密码'
})

const canContinue = computed(() => {
  if (step.value === 'request') return Boolean(form.login_id.trim())
  if (step.value === 'verify') return /^\d{6}$/.test(form.code.trim())
  return Boolean(form.password) && Boolean(form.confirm_password)
})

const goBack = () => {
  if (loading.value) return
  if (step.value === 'verify') step.value = 'request'
  else if (step.value === 'reset') step.value = 'verify'
}

const handleSubmit = async () => {
  if (loading.value) return
  if (!canContinue.value) return

  loading.value = true
  try {
    if (step.value === 'request') {
      const data = await authService.requestPasswordReset({ login_id: form.login_id.trim() })
      form.reset_id = data.reset_id
      form.code = ''
      step.value = 'verify'
      feedback.success('验证码已发送', '请查收并输入 6 位验证码')
      return
    }

    if (step.value === 'verify') {
      const data = await authService.verifyResetCode({ reset_id: form.reset_id, code: form.code.trim() })
      form.reset_token = data.reset_token
      form.password = ''
      form.confirm_password = ''
      step.value = 'reset'
      return
    }

    await authService.resetPassword({ reset_token: form.reset_token, password: form.password })
    feedback.success('密码已重置', '请使用新密码登录')
    emit('close')
  } catch (err: unknown) {
    console.error('[ResetPasswordModal] failed:', err)
    feedback.error('操作失败', getErrorMessage(err, '请稍后重试'))
  } finally {
    loading.value = false
  }
}
</script>

