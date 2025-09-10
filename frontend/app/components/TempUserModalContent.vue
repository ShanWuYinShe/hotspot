<template>
  <UModal>
    <template #header>
      <div class="flex items-center gap-2">
        <UIcon name="i-heroicons-rocket-launch" class="w-5 h-5 text-primary-500" />
        <h3 class="text-lg font-semibold">{{ t('userMenu.tempModal.title') }}</h3>
      </div>
    </template>

    <template #body>
      <div class="space-y-4">
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
              @click="handleClose" 
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
  </UModal>
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

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

// 响应式状态
const isCreating = ref(false)
const tempState = ref({ nickname: '' })

// 使用通用表单验证
const { validateLength, createFormValidator } = useFormValidation()

const validateTempState = createFormValidator({
  nickname: [(value) => value && value.length > 20 ? t('userMenu.tempModal.nicknameTooLong') : undefined]
})

// 处理关闭
const handleClose = () => {
  emit('close')
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
    handleClose()
  } catch (error) {
    // 错误已在父组件处理，这里不需要额外处理
    console.error('创建临时用户失败:', error)
  } finally {
    isCreating.value = false
  }
}
</script>
