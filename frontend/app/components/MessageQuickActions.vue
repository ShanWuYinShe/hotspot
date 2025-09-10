<template>
  <div class="flex flex-wrap gap-2 mb-6">
    <!-- 新建对话 -->
    <UButton 
      @click="$emit('start-new-chat')"
      size="sm"
      variant="outline"
    >
      <UIcon name="i-heroicons-plus" />
      新对话
    </UButton>

    <!-- 加入群组 -->
    <UTooltip text="前往群组页加入或创建群组">
      <UButton 
        @click="joinGroup"
        size="sm" 
        variant="outline"
        color="success"
      >
        <UIcon name="i-heroicons-user-group" />
        加入群组
      </UButton>
    </UTooltip>

    <!-- 扫码加好友 -->
    <UTooltip text="扫码加好友（开发中）">
      <UButton 
        @click="scanQRCode"
        size="sm" 
        variant="outline"
        color="info"
      >
        <UIcon name="i-heroicons-qr-code" />
        扫码
      </UButton>
    </UTooltip>

    <!-- 附近的人 -->
    <UTooltip text="查看附近的人">
      <UButton 
        @click="findNearbyPeople"
        size="sm" 
        variant="outline"
        color="secondary"
      >
        <UIcon name="i-heroicons-map-pin" />
        附近的人
      </UButton>
    </UTooltip>

  <!-- 消息请求（占位，无伪造计数） -->
    <UTooltip text="消息请求（开发中）">
      <UButton 
        @click="showMessageRequests"
        size="sm" 
        variant="outline"
        color="warning"
      >
        <UIcon name="i-heroicons-envelope" />
        消息请求
      </UButton>
    </UTooltip>
  </div>
</template>

<script setup lang="ts">
interface Emits {
  (e: 'start-new-chat'): void
}

defineEmits<Emits>()

const toast = useSmartToast()


// 功能实现
const joinGroup = () => {
  navigateTo('/groups')
}

const scanQRCode = () => {
  toast.add({
    title: '功能开发中',
    description: '扫码加好友功能即将上线',
    color: 'info'
  })
}

const findNearbyPeople = () => {
  navigateTo('/users/nearby')
}

const showMessageRequests = () => {
  toast.add({
    title: '功能开发中',
    description: '消息请求功能即将上线',
    color: 'info'
  })
}

// 无伪造数据：真实未读请求数可在未来接入后端接口后再显示徽章
</script>