<template>
  <UCard variant="outline">
    <div class="flex items-start gap-4">
      <div class="shrink-0">
        <UAvatar
          :icon="icon"
          size="sm"
          color="primary"
        />
      </div>

      <div class="flex-1 min-w-0">
        <div class="space-y-1">
          <div class="text-base font-semibold">
            {{ title }}
          </div>
          <div v-if="description" class="text-sm">
            {{ description }}
          </div>
        </div>

        <div class="flex flex-wrap gap-2 mt-4">
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
      </div>
    </div>
  </UCard>
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
