<template>
  <!-- 固定顶部的轻量反馈区：移动端比 toast 更友好 -->
  <div class="fixed inset-x-0 bottom-0 md:top-0 md:bottom-auto z-50 pointer-events-none">
    <UContainer as="div" class="pb-[env(safe-area-inset-bottom)] pb-3 md:pt-3 md:pb-0 space-y-2">
      <UAlert
        v-for="m in messages"
        :key="m.id"
        :title="m.title"
        :description="m.description"
        :color="m.color"
        variant="soft"
        :close="{ color: 'neutral', variant: 'link', size: 'sm' }"
        class="pointer-events-auto"
        @update:open="(open) => { if (!open) dismiss(m.id) }"
      />
    </UContainer>
  </div>
</template>

<script setup lang="ts">
import { useFeedbackStore } from '~/stores/feedback.store'

const feedback = useFeedbackStore()
const messages = computed(() => feedback.messages)
const dismiss = (id: string) => feedback.dismiss(id)
</script>

