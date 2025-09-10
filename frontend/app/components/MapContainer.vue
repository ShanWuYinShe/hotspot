<template>
  <UCard class="h-96 mb-6 overflow-hidden" aria-label="地图容器">
    <div
      v-if="!amapEnabled"
      class="h-full flex items-center justify-center"
    >
      <EmptyState
        icon="i-heroicons-map"
        title="地图未启用"
        description="未检测到高德地图 Key，请在环境变量中配置 NUXT_PUBLIC_AMAP_KEY 后刷新。"
      >
        <template #illustration>
          <EmptyNearby size="lg" />
        </template>
      </EmptyState>
    </div>
    <div v-else ref="mapEl" class="h-full w-full"></div>
  </UCard>
</template>

<script setup lang="ts">
interface Props {
  amapEnabled: boolean
}

defineProps<Props>()

const mapEl = ref<HTMLDivElement | null>(null)

defineExpose({
  mapEl
})
</script>
