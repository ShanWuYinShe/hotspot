<template>
  <UModal
    :title="title"
    :dismissible="false"
  >
    <template #body>
      <p>
        {{ message }}
      </p>
    </template>

    <template #footer>
      <UButton
        v-if="cancelText"
        variant="outline"
        color="neutral"
        :label="cancelText"
        @click="emit('close', false)"
      />
      <UButton
        :color="confirmColor"
        :label="confirmText"
        @click="emit('close', true)"
      />
    </template>
  </UModal>
</template>

<script setup lang="ts">
interface Props {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  confirmColor?: 'primary' | 'error' | 'warning' | 'success'
}

withDefaults(defineProps<Props>(), {
  confirmText: '确认',
  cancelText: '取消',
  confirmColor: 'primary'
})

const emit = defineEmits<{
  close: [value: boolean]
}>()
</script>
