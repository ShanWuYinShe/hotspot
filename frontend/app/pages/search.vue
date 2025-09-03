<template>
  <div class="max-w-5xl mx-auto px-4 py-6">
    <div class="mb-6">
      <h1 class="text-2xl font-semibold mb-2">{{ $t('nav.search') }}</h1>
      <p class="text-neutral-600 dark:text-neutral-400">搜索用户、群组和内容</p>
    </div>
    
    <!-- 搜索输入框 -->
    <div class="mb-6">
      <UInput
        v-model="searchQuery"
        placeholder="搜索用户、群组..."
        icon="i-heroicons-magnifying-glass"
        size="lg"
        class="w-full"
        @keyup.enter="performSearch"
      />
    </div>

    <UTabs v-model="tab" :items="tabs" class="mb-6" />

    <div v-if="tab === 'groups'" class="space-y-4">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-user-group" class="w-5 h-5" />
            <span>群组搜索</span>
          </div>
        </template>
        <div class="flex items-center justify-between">
          <div class="text-neutral-600 dark:text-neutral-300">发现附近的群组和感兴趣的社区</div>
          <UButton to="/groups/discover" color="primary" icon="i-heroicons-arrow-right">
            群组发现
          </UButton>
        </div>
      </UCard>
    </div>

    <div v-else class="space-y-4">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-users" class="w-5 h-5" />
            <span>用户搜索</span>
          </div>
        </template>
        <div class="flex items-center justify-between">
          <div class="text-neutral-600 dark:text-neutral-300">查找朋友和附近的用户</div>
          <div class="flex gap-2">
            <UButton to="/users/search" color="primary" icon="i-heroicons-magnifying-glass">
              搜索用户
            </UButton>
            <UButton to="/nearby-users" variant="outline" icon="i-heroicons-map-pin">
              附近用户
            </UButton>
          </div>
        </div>
      </UCard>
    </div>
  </div>
</template>

<script setup lang="ts">
const tab = ref<'groups' | 'users'>('groups')
const tabs = [
  { label: '群组', value: 'groups' },
  { label: '用户', value: 'users' }
]
</script>


