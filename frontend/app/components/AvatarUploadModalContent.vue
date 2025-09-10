<template>
  <UModal :title="$t('settings.avatar.title')">
    <template #body>
      <div class="p-6">
        <AvatarUpload 
          :user="user" 
          @uploaded="handleUploaded"
        />
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
interface Props {
  user?: any
}

const props = defineProps<Props>()
const emit = defineEmits<{
  uploaded: [data: any]
  close: []
}>()

// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

const handleUploaded = (data: any) => {
  emit('uploaded', data)
  emit('close')
}
</script>
