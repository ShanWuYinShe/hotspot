<template>
  <UDropdownMenu>
    <template #trigger>
      <slot name="trigger">
        <UButton
          icon="i-heroicons-ellipsis-horizontal"
          variant="ghost"
          color="gray"
          size="sm"
        />
      </slot>
    </template>

    <template #content>
      <div class="py-1">
        <UDropdownMenuItem
          icon="i-heroicons-chat-bubble-left"
          label="发送消息"
          @click="$emit('chatRequested', user)"
        />
        <UDropdownMenuItem
          icon="i-heroicons-user-plus"
          label="添加好友"
          @click="$emit('friendRequested', user)"
        />
        <UDropdownMenuItem
          icon="i-heroicons-map-pin"
          label="查看位置"
          @click="$emit('locationRequested', user)"
        />
        <UDropdownMenuItem
          icon="i-heroicons-information-circle"
          label="查看资料"
          @click="$emit('profileRequested', user)"
        />
        <div class="border-t border-gray-200 dark:border-gray-700 my-1"></div>
        <UDropdownMenuItem
          icon="i-heroicons-flag"
          label="举报用户"
          color="red"
          @click="$emit('reportRequested', user)"
        />
      </div>
    </template>
  </UDropdownMenu>
</template>

<script setup lang="ts">
import type { User } from '~/types'

interface Props {
  user: User
}

interface Emits {
  chatRequested: [user: User]
  friendRequested: [user: User]
  locationRequested: [user: User]
  profileRequested: [user: User]
  reportRequested: [user: User]
}

defineProps<Props>()
defineEmits<Emits>()
</script>
