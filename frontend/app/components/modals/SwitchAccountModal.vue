<template>
  <UModal title="切换账户">
    <template #body>
      <p class="text-secondary-content mb-4">
        {{ t('userMenu.switchModal.description', { name: account?.nickname }) }}
      </p>

      <UAlert
        v-if="account?.user_type && isTempUser(account.user_type)"
        icon="i-heroicons-exclamation-triangle"
        color="warning"
        variant="soft"
        :title="t('userMenu.switchModal.tempWarningTitle')"
        :description="t('userMenu.switchModal.tempWarningDescription')"
      />
    </template>

    <template #footer>
      <UButton
        variant="ghost"
        color="neutral"
        @click="handleClose"
      >
        {{ t('userMenu.switchModal.cancel') }}
      </UButton>
      <UButton
        color="warning"
        icon="i-heroicons-arrow-path"
        :loading="isSwitching"
        @click="handleConfirm"
      >
        {{ t('userMenu.switchModal.confirm') }}
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { isTempUser } from '~/utils/userTypes'

interface Props {
  account?: { user_id: number; nickname?: string; avatar_url?: string; user_type?: 'temp' | 'normal' | 'admin' | 'system' }
  onConfirm?: () => Promise<void>
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()

const isSwitching = ref(false)

const handleClose = () => emit('close')

const handleConfirm = async () => {
  if (!props.onConfirm) {
    console.error('onConfirm callback not provided')
    return
  }

  try {
    isSwitching.value = true
    await props.onConfirm()
    handleClose()
  } catch (error) {
    console.error('切换账户失败:', error)
  } finally {
    isSwitching.value = false
  }
}
</script>
