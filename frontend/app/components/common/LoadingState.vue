<template>
  <UEmpty
    icon="i-lucide-loader-2"
    :title="title || '加载中...'"
    :description="description"
    variant="naked"
    class="py-12"
  >
    <template #body>
      <UButton loading variant="ghost" color="neutral" />

      <div
        v-if="showProgress && progress !== undefined"
        class="w-full max-w-xs"
      >
        <UProgress
          :model-value="progress"
          :max="100"
          color="primary"
          class="h-2"
        />
        <div class="mt-2 flex justify-center">
          <UBadge color="neutral" variant="soft" size="xs">
            {{ Math.round(progress) }}%
          </UBadge>
        </div>
      </div>
    </template>
  </UEmpty>
</template>

<script setup lang="ts">
interface Props {
  title?: string;
  description?: string;
  showProgress?: boolean;
  progress?: number;
}

defineProps<Props>();
</script>