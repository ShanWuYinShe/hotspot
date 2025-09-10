<template>
  <div class="flex flex-wrap items-center gap-3 mb-6">
    <UButton size="lg" :loading="isLoading" @click="$emit('refresh-location')">
      <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 mr-2" />
      刷新位置
    </UButton>
    <UButton size="lg" variant="outline" @click="$emit('toggle-location-sharing')">
      <UIcon name="i-heroicons-map-pin" class="w-4 h-4 mr-2" />
      {{ isLocationSharing ? "关闭共享" : "开启共享" }}
    </UButton>
    <UAlert v-if="permissionDenied" color="warning" variant="soft" class="text-sm">
      已拒绝位置权限，请在浏览器地址栏位置图标中重新授权
    </UAlert>
  </div>
</template>

<script setup lang="ts">
interface Props {
  isLoading: boolean
  isLocationSharing: boolean
  permissionDenied: boolean
}

defineProps<Props>()

defineEmits<{
  'refresh-location': []
  'toggle-location-sharing': []
}>()
</script>
