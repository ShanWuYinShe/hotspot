<template>
  <UModal title="注销账户" :close="{ onClick: () => $emit('close') }">
    <template #body>
      <div class="space-y-4">
        <UAlert
          color="error"
          variant="soft"
          title="警告"
          description="此操作不可撤销，您的所有数据将被永久删除"
        />
        
        <div class="space-y-2">
          <p class="text-sm text-neutral-600 dark:text-neutral-400">
            注销账户将会：
          </p>
          <ul class="text-sm text-neutral-600 dark:text-neutral-400 list-disc list-inside space-y-1">
            <li>删除您的个人资料和设置</li>
            <li>删除您发布的所有动态</li>
            <li>删除您的好友关系</li>
            <li>删除您的群组成员身份</li>
            <li>删除您的聊天记录</li>
          </ul>
        </div>
        
        <UFormField label="确认删除" required>
          <UInput
            v-model="confirmText"
            placeholder="请输入 DELETE 确认删除"
            :error="hasError"
          />
          <p v-if="hasError" class="text-sm text-red-600 mt-1">
            请输入 DELETE 确认删除
          </p>
        </UFormField>
      </div>
    </template>

    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton variant="outline" @click="handleCancel">
          取消
        </UButton>
        <UButton 
          color="error" 
          :disabled="!isConfirmed"
          :loading="deleting"
          @click="handleDelete"
        >
          确认注销
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useUserService } from '~/services/users.service'

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

const confirmText = ref('')
const hasError = ref(false)
const deleting = ref(false)

const isConfirmed = computed(() => confirmText.value === 'DELETE')

// 监听输入变化，清除错误状态
watch(confirmText, () => {
  if (hasError.value) {
    hasError.value = false
  }
})

const handleCancel = () => {
  emit('cancel')
  emit('close')
}

const handleDelete = async () => {
  if (!isConfirmed.value) {
    hasError.value = true
    return
  }
  
  deleting.value = true
  
  try {
    const { deleteAccount: deleteUserAccount } = useUserService()
    await deleteUserAccount()
    
    const toast = useToast()
    toast.add({ color: 'success',
      title: '账户已注销',
      description: '您的账户已成功注销'
    })
    
    // 清除认证状态并跳转到首页
    const authStore = useAuthStore()
    authStore.clearSession()
    
    emit('success')
    emit('close')
    
    // 跳转到首页
    await navigateTo('/')
    } catch (err: any) {
      const toast = useToast()
      toast.add({ color: 'error',
      title: '注销失败',
      description: err?.message || '请稍后重试'
    })
  } finally {
    deleting.value = false
  }
}
</script>
