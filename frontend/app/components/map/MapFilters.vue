<template>
  <div class="mb-6">
    <UFieldGroup>
      <UButton 
        variant="ghost" 
        size="sm"
        :class="{ 'bg-primary/10 text-primary': activeFilters.includes('users') }"
        @click="$emit('toggle-filter', 'users')"
      >
        <UIcon name="i-heroicons-users" class="w-4 h-4 mr-1" />
        用户 ({{ userCount }})
      </UButton>
      <UButton 
        variant="ghost" 
        size="sm"
        :class="{ 'bg-primary/10 text-primary': activeFilters.includes('groups') }"
        @click="$emit('toggle-filter', 'groups')"
      >
        <UIcon name="i-heroicons-user-group" class="w-4 h-4 mr-1" />
        群组 ({{ groupCount }})
      </UButton>
      <UButton 
        variant="ghost" 
        size="sm"
        :class="{ 'bg-primary/10 text-primary': activeFilters.includes('moments') }"
        @click="$emit('toggle-filter', 'moments')"
      >
        <UIcon name="i-heroicons-photo" class="w-4 h-4 mr-1" />
        动态 ({{ momentCount }})
      </UButton>
    </UFieldGroup>
  </div>
</template>

<script setup lang="ts">
interface Props {
  activeFilters: ('users' | 'groups' | 'moments')[]
  userCount: number
  groupCount: number
  momentCount: number
}

defineProps<Props>()

defineEmits<{
  'toggle-filter': [filter: 'users' | 'groups' | 'moments']
}>()
</script>
