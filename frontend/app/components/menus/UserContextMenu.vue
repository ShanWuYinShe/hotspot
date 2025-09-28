<template>
  <UContextMenu>
    <template #trigger>
      <slot name="trigger">
        <UButton
          icon="i-heroicons-ellipsis-horizontal"
          variant="ghost"
          color="neutral"
          size="sm"
        />
      </slot>
    </template>

    <template #content>
      <div class="py-1">
        <UContextMenuItem
          icon="i-heroicons-chat-bubble-left"
          label="发送消息"
          @click="$emit('chatRequested', user)"
        />
        <UContextMenuItem
          icon="i-heroicons-user-plus"
          label="添加好友"
          @click="$emit('friendRequested', user)"
        />
        <UContextMenuItem
          icon="i-heroicons-map-pin"
          label="查看位置"
          @click="$emit('locationRequested', user)"
        />
        <UContextMenuItem
          icon="i-heroicons-information-circle"
          label="查看资料"
          @click="$emit('profileRequested', user)"
        />
        <USeparator />
        <UContextMenuItem
          icon="i-heroicons-flag"
          label="举报用户"
          color="red"
          @click="$emit('reportRequested', user)"
        />
      </div>
    </template>
  </UContextMenu>
</template>

<script setup lang="ts">
import type { UserResponse } from '~/types/api'

interface Props {
  user: UserResponse
}

interface Emits {
  chatRequested: [user: UserResponse]
  friendRequested: [user: UserResponse]
  locationRequested: [user: UserResponse]
  profileRequested: [user: UserResponse]
  reportRequested: [user: UserResponse]
}

defineProps<Props>()
defineEmits<Emits>()
</script>
