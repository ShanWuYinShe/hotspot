<template>
  <div class="space-y-4">
    <div class="flex items-center gap-2 mb-4">
      <UIcon name="i-heroicons-rocket-launch" class="w-5 h-5 text-primary-500" />
      <h3 class="text-lg font-semibold">{{ t('userMenu.tempModal.title') }}</h3>
    </div>

    <UAlert 
      icon="i-heroicons-information-circle"
      color="info"
      variant="soft"
      :title="t('userMenu.tempModal.infoTitle')"
      :description="t('userMenu.tempModal.infoDescription')"
    />

    <UForm 
      :state="tempState" 
      :validate="validateTempState" 
      @submit="handleSubmit"
      class="space-y-4"
    >
      <UFormField 
        :label="t('userMenu.tempModal.nicknameLabel')" 
        name="nickname"
        :help="t('userMenu.tempModal.nicknameHelp')"
      >
        <UInput 
          v-model="tempState.nickname" 
          :placeholder="t('userMenu.tempModal.nicknamePlaceholder')"
          icon="i-heroicons-user"
        />
      </UFormField>

      <div class="flex justify-end gap-3 pt-4">
        <UButton 
          @click="$emit('close')" 
          variant="ghost"
          color="neutral"
        >
          {{ t('userMenu.tempModal.cancel') }}
        </UButton>
        <UButton 
          type="submit"
          :loading="isCreating"
          icon="i-heroicons-rocket-launch"
        >
          {{ t('userMenu.tempModal.startExperience') }}
        </UButton>
      </div>
    </UForm>
  </div>
</template>

<script setup lang="ts">
interface Props {
  onCreateTemp?: (nickname?: string) => Promise<void>
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

// 响应式状态
const isCreating = ref(false)
const tempState = ref({ nickname: '' })

// 表单验证
type FormErrors = { path: string; message: string }[]
function validateTempState(state: any): FormErrors { 
  const errors: FormErrors = []
  
  if (state.nickname && state.nickname.length > 20) {
    errors.push({
      path: 'nickname',
      message: t('userMenu.tempModal.nicknameTooLong')
    })
  }
  
  return errors
}

// 处理表单提交
const handleSubmit = async () => {
  if (!props.onCreateTemp) {
    console.error('onCreateTemp callback not provided')
    return
  }

  try {
    isCreating.value = true
    await props.onCreateTemp(tempState.value.nickname)
    emit('close')
  } catch (error) {
    // 错误已在父组件处理，这里不需要额外处理
    console.error('创建临时用户失败:', error)
  } finally {
    isCreating.value = false
  }
}
</script>