<template>
  <div class="w-full flex items-center justify-center min-h-40">
    <UBadge color="neutral" variant="soft" size="sm">正在跳转...</UBadge>
  </div>
</template>

<script setup lang="ts">
// 自动重定向到附近动态
definePageMeta({
  title: '动态',
  description: '正在跳转...',
  layout: 'dashboard',
})

await navigateTo('/moments/nearby', { replace: true })
</script>