<template>
  <div class="max-w-4xl mx-auto space-y-6">
    <!-- 页面标题 -->
    <UPageHeader
      title="通知中心"
      description="管理您的推送通知设置"
    />

      <!-- 说明卡片 -->
      <UCard class="mb-6">
        <div class="flex items-start gap-4">
          <div class="shrink-0">
            <div class="w-12 h-12 rounded-full bg-primary-subtle flex items-center justify-center">
              <UIcon name="i-heroicons-bell" class="w-6 h-6 text-accent" />
            </div>
          </div>
          <div class="flex-1">
            <h3 class="font-semibold text-primary-content mb-2">
              实时推送通知
            </h3>
            <p class="text-sm text-secondary-content mb-4">
              通知会在事件发生时实时推送到您的设备。您可以在设置中配置接收哪些类型的通知。
            </p>
            <div class="flex gap-2">
              <UButton
                to="/settings"
                variant="outline"
                size="sm"
                icon="i-heroicons-cog-6-tooth"
              >
                通知设置
              </UButton>
            </div>
          </div>
        </div>
      </UCard>

      <!-- 通知类型说明 -->
      <UCard>
        <template #header>
          <h3 class="font-semibold text-primary-content">
            支持的通知类型
          </h3>
        </template>

        <div class="space-y-4">
          <div
            v-for="notificationType in notificationTypes"
            :key="notificationType.type"
            class="flex items-center gap-4 p-3 rounded-lg hover-bg-surface transition-colors"
          >
            <div
              :class="[
                'w-10 h-10 rounded-full flex items-center justify-center',
                notificationType.bgColor
              ]"
            >
              <UIcon :name="notificationType.icon" class="icon-md" />
            </div>
            <div class="flex-1">
              <h4 class="font-medium text-primary-content">
                {{ notificationType.label }}
              </h4>
              <p class="text-sm text-secondary-content">
                {{ notificationType.description }}
              </p>
            </div>
          </div>
        </div>
      </UCard>

      <!-- 待处理事项快捷入口 -->
      <UCard class="mt-6">
        <template #header>
          <h3 class="font-semibold text-primary-content">
            快捷入口
          </h3>
        </template>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <UButton
            to="/friends"
            variant="soft"
            block
            class="justify-start"
            icon="i-heroicons-users"
          >
            我的好友
          </UButton>
          <UButton
            to="/groups"
            variant="soft"
            block
            class="justify-start"
            icon="i-heroicons-user-group"
          >
            我的群组
          </UButton>
          <UButton
            to="/chat"
            variant="soft"
            block
            class="justify-start"
            icon="i-heroicons-chat-bubble-left-right"
          >
            消息列表
          </UButton>
          <UButton
            to="/moments"
            variant="soft"
            block
            class="justify-start"
            icon="i-heroicons-photo"
          >
            生活瞬间
          </UButton>
        </div>
      </UCard>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  requiresAuth: true,
  title: '通知中心',
  layout: 'dashboard'
})

// 通知类型列表
const notificationTypes = [
  {
    type: 'new_follower',
    label: '新关注者',
    description: '当有人关注您时收到通知',
    icon: 'i-heroicons-user-plus',
    bgColor: 'bg-blue-subtle text-blue'
  },
  {
    type: 'mutual_follow',
    label: '互相关注',
    description: '当与某人互相关注成为好友时收到通知',
    icon: 'i-heroicons-heart',
    bgColor: 'bg-pink-subtle text-pink'
  },
  {
    type: 'group_invite',
    label: '群组邀请',
    description: '当您被邀请加入群组时收到通知',
    icon: 'i-heroicons-user-group',
    bgColor: 'bg-green-subtle text-green'
  },
  {
    type: 'message',
    label: '新消息',
    description: '当收到新的私聊或群聊消息时收到通知',
    icon: 'i-heroicons-chat-bubble-left-right',
    bgColor: 'bg-purple-subtle text-purple'
  },
  {
    type: 'moment_like',
    label: '瞬间点赞',
    description: '当您的生活瞬间被点赞时收到通知',
    icon: 'i-heroicons-hand-thumb-up',
    bgColor: 'bg-orange-subtle text-orange'
  },
  {
    type: 'comment',
    label: '评论通知',
    description: '当您的内容收到评论或回复时收到通知',
    icon: 'i-heroicons-chat-bubble-bottom-center-text',
    bgColor: 'bg-teal-subtle text-teal'
  },
  {
    type: 'system',
    label: '系统通知',
    description: '重要的系统公告和更新通知',
    icon: 'i-heroicons-megaphone',
    bgColor: 'bg-yellow-subtle text-yellow'
  }
]
</script>
