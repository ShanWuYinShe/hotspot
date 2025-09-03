<template>
  <UModal 
    v-model:open="isOpen"
    :overlay="false"
    :transition="false"
    class="search-modal"
  >
    <template #default></template>
    
    <template #content>
      <UCommandPalette
        v-model="searchQuery"
        :groups="searchGroups"
        :loading="isSearching"
        placeholder="搜索用户、群组、动态..."
        :empty-state="emptyState"
        @update:model-value="handleSearch"
        @close="closeModal"
      >
        <template #default="{ item }">
          <div class="flex items-center space-x-3">
            <UAvatar 
              v-if="item.type === 'user'" 
              :alt="item.label" 
              size="sm"
            >
              {{ item.label?.charAt(0) || 'U' }}
            </UAvatar>
            <div 
              v-else-if="item.type === 'group'"
              class="w-8 h-8 bg-blue-100 dark:bg-blue-900 rounded-lg flex items-center justify-center"
            >
              <UIcon name="i-heroicons-user-group" class="w-4 h-4 text-blue-600 dark:text-blue-400" />
            </div>
            <div 
              v-else-if="item.type === 'moment'"
              class="w-8 h-8 bg-purple-100 dark:bg-purple-900 rounded-lg flex items-center justify-center"
            >
              <UIcon name="i-heroicons-photo" class="w-4 h-4 text-purple-600 dark:text-purple-400" />
            </div>
            <div 
              v-else
              class="w-8 h-8 bg-neutral-100 dark:bg-neutral-800 rounded-lg flex items-center justify-center"
            >
              <UIcon :name="item.icon || 'i-heroicons-magnifying-glass'" class="w-4 h-4" />
            </div>
            
            <div class="flex-1">
              <p class="font-medium text-sm">{{ item.label }}</p>
              <p v-if="item.description" class="text-xs text-neutral-500">{{ item.description }}</p>
            </div>
            
            <UBadge v-if="item.badge" size="xs" :color="item.badgeColor || 'neutral'">
              {{ item.badge }}
            </UBadge>
          </div>
        </template>
      </UCommandPalette>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'

interface SearchItem {
  id: string;
  label: string;
  description?: string;
  type: 'user' | 'group' | 'moment' | 'action';
  icon?: string;
  badge?: string;
  badgeColor?: string;
  to?: string;
  click?: () => void;
}

interface SearchGroup {
  key: string;
  label: string;
  commands: SearchItem[];
}

// Props
interface Props {
  open?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  open: false
});

// Emits
const emit = defineEmits<{
  'update:open': [value: boolean];
  close: [];
}>();

// 响应式状态
const isOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
});

const searchQuery = ref('');
const isSearching = ref(false);

// 空状态配置
const emptyState = computed(() => ({
  icon: 'i-heroicons-magnifying-glass',
  label: '没有找到结果',
  queryLabel: searchQuery.value ? `没有找到与 "${searchQuery.value}" 相关的结果` : '开始搜索'
}));

// 搜索结果分组
const searchGroups = ref<SearchGroup[]>([
  {
    key: 'actions',
    label: '快速操作',
    commands: [
      {
        id: 'create-group',
        label: '创建群组',
        description: '创建一个新的群组',
        type: 'action',
        icon: 'i-heroicons-plus',
        click: () => {
          closeModal();
          navigateTo('/groups/create');
        }
      },
      {
        id: 'create-moment',
        label: '发布动态',
        description: '分享你的精彩瞬间',
        type: 'action',
        icon: 'i-heroicons-camera',
        click: () => {
          closeModal();
          navigateTo('/moments/create');
        }
      },
      {
        id: 'nearby-users',
        label: '附近用户',
        description: '发现周围的朋友',
        type: 'action',
        icon: 'i-heroicons-users',
        click: () => {
          closeModal();
          navigateTo('/nearby-users');
        }
      }
    ]
  },
  {
    key: 'users',
    label: '用户',
    commands: []
  },
  {
    key: 'groups',
    label: '群组',
    commands: []
  },
  {
    key: 'moments',
    label: '动态',
    commands: []
  }
]);

// 搜索处理
const handleSearch = useDebounceFn(async (query: string) => {
  if (!query.trim()) {
    // 重置为默认快速操作
    searchGroups.value[1].commands = [];
    searchGroups.value[2].commands = [];
    searchGroups.value[3].commands = [];
    return;
  }

  isSearching.value = true;
  
  try {
    // TODO: 实现真实的搜索API调用
    // const results = await $fetch('/api/search', { query: { q: query } });
    
    // 模拟搜索结果
    await new Promise(resolve => setTimeout(resolve, 300));
    
    // 模拟用户搜索结果
    const mockUsers: SearchItem[] = query.includes('用户') ? [
      {
        id: 'user-1',
        label: '张三',
        description: '附近 500m',
        type: 'user',
        badge: '在线',
        badgeColor: 'green',
        click: () => {
          closeModal();
          navigateTo('/users/1');
        }
      }
    ] : [];

    // 模拟群组搜索结果
    const mockGroups: SearchItem[] = query.includes('群') || query.includes('组') ? [
      {
        id: 'group-1',
        label: '技术交流群',
        description: '120 名成员',
        type: 'group',
        badge: '活跃',
        badgeColor: 'blue',
        click: () => {
          closeModal();
          navigateTo('/groups/1');
        }
      }
    ] : [];

    // 模拟动态搜索结果
    const mockMoments: SearchItem[] = query.includes('动态') || query.includes('照片') ? [
      {
        id: 'moment-1',
        label: '美丽的日落',
        description: '2小时前 · 李四',
        type: 'moment',
        badge: '热门',
        badgeColor: 'orange',
        click: () => {
          closeModal();
          navigateTo('/moments/1');
        }
      }
    ] : [];

    // 更新搜索结果
    searchGroups.value[1].commands = mockUsers;
    searchGroups.value[2].commands = mockGroups;
    searchGroups.value[3].commands = mockMoments;
    
  } catch (error) {
    console.error('搜索失败:', error);
  } finally {
    isSearching.value = false;
  }
}, 300);

// 关闭弹窗
const closeModal = () => {
  isOpen.value = false;
  emit('close');
  // 重置搜索状态
  searchQuery.value = '';
  searchGroups.value[1].commands = [];
  searchGroups.value[2].commands = [];
  searchGroups.value[3].commands = [];
};

// 监听搜索查询变化
watch(searchQuery, handleSearch);
</script>

<style scoped>
.search-modal :deep(.command-palette) {
  border: none;
  box-shadow: none;
}
</style>

