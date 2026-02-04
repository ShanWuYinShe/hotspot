<template>
  <div
    v-if="visible"
    class="fixed right-4 bottom-[calc(env(safe-area-inset-bottom)+1rem)] z-40"
  >
    <UTooltip text="回到顶部">
      <UButton
        variant="soft"
        color="neutral"
        size="md"
        icon="i-lucide-arrow-up"
        :square="true"
        @click="scrollToTop"
      />
    </UTooltip>
  </div>
</template>

<script setup lang="ts">
const THRESHOLD = 600
const { y } = useWindowScroll()

const visible = computed(() => y.value > THRESHOLD)

const scrollToTop = () => {
  if (import.meta.client) {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}
</script>

