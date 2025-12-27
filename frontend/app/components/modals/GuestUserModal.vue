<template>
  <UModal title="创建临时用户">
    <template #body>
      <UAlert
        icon="i-lucide-info"
        color="info"
        variant="soft"
        :title="t('userMenu.tempModal.infoTitle')"
        :description="t('userMenu.tempModal.infoDescription')"
        class="mb-4"
      />

      <UForm
        :state="tempState"
        class="space-y-4"
        @submit="handleSubmit"
      >
        <UFormField
          :label="t('userMenu.tempModal.nicknameLabel')"
          name="nickname"
          :hint="t('userMenu.tempModal.nicknameHelp')"
        >
          <UInput
            v-model="tempState.nickname"
            :placeholder="t('userMenu.tempModal.nicknamePlaceholder')"
            icon="i-lucide-user"
          />
        </UFormField>
      </UForm>
    </template>

    <template #footer>
      <UButton
        variant="ghost"
        color="neutral"
        @click="handleClose"
      >
        {{ t('userMenu.tempModal.cancel') }}
      </UButton>
      <UButton
        :loading="isCreating"
        icon="i-lucide-rocket"
        @click="handleSubmit"
      >
        {{ t('userMenu.tempModal.startExperience') }}
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
interface Props {
  onCreateTemp?: (nickname?: string) => Promise<void>
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()

const isCreating = ref(false)
const tempState = ref({ nickname: '' })

const handleClose = () => emit('close')

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
    console.error('创建临时用户失败:', error)
  } finally {
    isCreating.value = false
  }
}
</script>
