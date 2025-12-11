<template>
  <UAlert
    :title="title"
    :description="description"
    :icon="icon"
    class="border-t"
  >
    <template #actions>
      <div class="flex gap-2">
        <UButton
          v-for="action in actions"
          :key="action.key"
          :icon="action.icon"
          :color="action.color || 'primary'"
          :variant="action.variant || 'solid'"
          :size="action.size || 'sm'"
          :loading="action.loading"
          :disabled="action.disabled"
          @click="action.action"
        >
          {{ action.label }}
        </UButton>
      </div>
    </template>
  </UAlert>
</template>

<script setup lang="ts">
export interface ActionItem {
  key: string
  label: string
  icon?: string
  color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
  variant?: 'solid' | 'outline' | 'soft' | 'ghost' | 'link'
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  loading?: boolean
  disabled?: boolean
  action: () => void | Promise<void>
}

interface Props {
  title: string
  description?: string
  icon: string
  actions: ActionItem[]
}

defineProps<Props>()
</script>
