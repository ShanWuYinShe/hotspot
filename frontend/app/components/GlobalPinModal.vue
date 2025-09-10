<template>
  <PinInputModal
    v-model:open="authStore.pinModalState.isOpen"
    :is-setup="authStore.pinModalState.isSetup"
    @confirm="authStore.handlePinConfirm"
    @cancel="authStore.handlePinCancel"
  />
</template>

<script setup lang="ts">
const authStore = useAuthStore()
</script>

