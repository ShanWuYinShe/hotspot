<template>
  <div
    v-if="images?.length"
    class="grid gap-1 overflow-hidden rounded-lg border border-neutral-200 dark:border-neutral-800"
    :class="gridClass"
  >
    <div
      v-for="(image, index) in visibleImages"
      :key="index"
      class="relative w-full h-full overflow-hidden bg-neutral-100 dark:bg-neutral-800"
    >
      <NuxtImg
        :src="image"
        :alt="`Image ${index + 1}`"
        loading="lazy"
        class="w-full h-full object-cover transition-transform hover:scale-105 duration-500 cursor-pointer"
        @click.stop="$emit('click', { image, index })"
      />

      <!-- Overlay for remaining images -->
      <div
        v-if="index === visibleImages.length - 1 && remainingCount > 0"
        class="absolute inset-0 flex items-center justify-center bg-white/80 dark:bg-neutral-900/80 backdrop-blur-sm cursor-pointer transition-colors hover:bg-black/40"
        @click.stop="$emit('more')"
      >
        <span class="text-neutral-900 dark:text-white font-bold text-lg">+{{ remainingCount }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  images?: string[]
  limit?: number
}

const props = withDefaults(defineProps<Props>(), {
  images: () => [],
  limit: 9
})

const emit = defineEmits<{
  click: [data: { image: string; index: number }]
  more: []
}>()

const visibleImages = computed(() => props.images.slice(0, props.limit))
const remainingCount = computed(() => Math.max(0, props.images.length - props.limit))

const gridClass = computed(() => {
  const len = visibleImages.value.length
  if (len === 0) return ''
  if (len === 1) return 'grid-cols-1 aspect-video'
  if (len === 2) return 'grid-cols-2 aspect-2/1'
  if (len <= 4) return 'grid-cols-2 aspect-square'
  return 'grid-cols-3 aspect-square'
})
</script>
