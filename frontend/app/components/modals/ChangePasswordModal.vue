<template>
  <UModal title="修改密码" :close="{ onClick: () => $emit('close') }">
    <template #body>
      <DynamicForm
        :fields="passwordFields"
        :schema="passwordSchema"
        :loading="changingPassword"
        @submit="changePassword"
      />
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { z } from 'zod'
import { useUsersService } from '~/services/users.service'

interface Props {
  onSuccess?: () => void
  onCancel?: () => void
}

const props = withDefaults(defineProps<Props>(), {})

const emit = defineEmits<{
  success: []
  cancel: []
  close: []
}>()

const changingPassword = ref(false)

// 密码表单字段
const passwordFields = [
  {
    name: 'currentPassword',
    label: '当前密码',
    type: 'password' as const,
    placeholder: '请输入当前密码',
    required: true
  },
  {
    name: 'newPassword',
    label: '新密码',
    type: 'password' as const,
    placeholder: '请输入新密码',
    required: true
  },
  {
    name: 'confirmPassword',
    label: '确认新密码',
    type: 'password' as const,
    placeholder: '请再次输入新密码',
    required: true
  }
]

// 密码验证 Schema
const passwordSchema = z.object({
  currentPassword: z.string().min(1, '请输入当前密码'),
  newPassword: z.string().min(6, '新密码至少6位').max(50, '密码不能超过50位'),
  confirmPassword: z.string().min(1, '请确认新密码')
}).refine((data) => data.newPassword === data.confirmPassword, {
  message: '两次输入的密码不一致',
  path: ['confirmPassword']
})

// 修改密码
const changePassword = async (data: any) => {
  changingPassword.value = true
  
  const toast = useToast()
  
  try {
    const { setPassword } = useUserService()
    await setPassword(data.newPassword)
    
    toast.add({ color: 'success',
      title: '密码修改成功',
      description: '您的密码已成功修改'
    })
    
    emit('success')
    emit('close')
    } catch (err: any) {
      toast.add({ color: 'error',
      title: '密码修改失败',
      description: err?.message || '请稍后重试'
    })
  } finally {
    changingPassword.value = false
  }
}
</script>
