<template>
  <div class="container mx-auto px-4 py-8">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900 mb-2">地图</h1>
      <p class="text-gray-600">发现附近的用户和群组</p>
    </div>

    <!-- 地图容器 -->
    <div class="bg-white rounded-lg shadow-sm border h-96 mb-6">
      <div class="h-full flex items-center justify-center text-gray-500">
        <div class="text-center">
          <UIcon name="i-heroicons-map" class="w-16 h-16 mx-auto mb-4" />
          <p>地图功能开发中...</p>
          <p class="text-sm mt-2">将集成高德地图显示附近用户和群组</p>
        </div>
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="flex gap-4 mb-6">
      <UButton @click="refreshLocation" :loading="isLoading">
        <UIcon name="i-heroicons-arrow-path" class="w-4 h-4 mr-2" />
        刷新位置
      </UButton>
      <UButton variant="outline" @click="toggleLocationSharing">
        <UIcon name="i-heroicons-map-pin" class="w-4 h-4 mr-2" />
        {{ isLocationSharing ? '关闭共享' : '开启共享' }}
      </UButton>
    </div>

    <!-- 附近用户列表 -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h3 class="font-semibold">附近用户</h3>
            <UButton size="sm" variant="ghost">
              <UIcon name="i-heroicons-users" class="w-4 h-4" />
            </UButton>
          </div>
        </template>
        
        <div class="space-y-3">
          <div class="text-center py-8 text-gray-500">
            <UIcon name="i-heroicons-users" class="w-12 h-12 mx-auto mb-2" />
            <p>暂无附近用户</p>
          </div>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h3 class="font-semibold">附近群组</h3>
            <UButton size="sm" variant="ghost">
              <UIcon name="i-heroicons-user-group" class="w-4 h-4" />
            </UButton>
          </div>
        </template>
        
        <div class="space-y-3">
          <div class="text-center py-8 text-gray-500">
            <UIcon name="i-heroicons-user-group" class="w-12 h-12 mx-auto mb-2" />
            <p>暂无附近群组</p>
          </div>
        </div>
      </UCard>
    </div>
  </div>
</template>

<script setup>
// 页面元数据
definePageMeta({
  title: '地图',
  requiresAuth: true
})

// 响应式状态
const isLoading = ref(false)
const isLocationSharing = ref(false)

// 刷新位置
const refreshLocation = async () => {
  isLoading.value = true
  try {
    // TODO: 实现位置获取逻辑
    await new Promise(resolve => setTimeout(resolve, 1000))
    const toast = useToast()
    toast.add({
      title: '位置已刷新',
      color: 'success'
    })
  } catch (error) {
    const toast = useToast()
    toast.add({
      title: '位置获取失败',
      description: '请检查位置权限设置',
      color: 'error'
    })
  } finally {
    isLoading.value = false
  }
}

// 切换位置共享
const toggleLocationSharing = () => {
  isLocationSharing.value = !isLocationSharing.value
  const toast = useToast()
  toast.add({
    title: isLocationSharing.value ? '位置共享已开启' : '位置共享已关闭',
    color: 'info'
  })
}
</script>
