<template>
  <div class="space-y-4">
    <!-- 头部工具栏 -->
    <div v-if="showToolbar" class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <!-- 搜索框 -->
        <UInput
          v-if="searchable"
          v-model="searchQuery"
          placeholder="搜索..."
          icon="i-heroicons-magnifying-glass"
          class="w-64"
          @input="handleSearch"
        />
        
        <!-- 筛选器 -->
        <USelectMenu
          v-if="filters.length > 0"
          v-model="selectedFilter"
          :options="filters"
          placeholder="筛选"
          @change="handleFilterChange"
        />
      </div>
      
      <!-- 操作按钮 -->
      <div class="flex items-center gap-2">
        <UButton
          v-for="action in toolbarActions"
          :key="action.key"
          :color="action.color || 'primary'"
          :variant="action.variant || 'outline'"
          :size="action.size || 'sm'"
          @click="action.action"
        >
          <UIcon v-if="action.icon" :name="action.icon" class="w-4 h-4 mr-1" />
          {{ action.label }}
        </UButton>
      </div>
    </div>
    
    <!-- 数据展示区域 -->
    <div class="relative">
      <!-- 加载状态 -->
      <div v-if="loading" class="space-y-4">
        <USkeleton
          v-for="i in skeletonCount"
          :key="i"
          class="h-20 w-full"
        />
      </div>
      
      <!-- 空状态 -->
      <div v-else-if="!loading && filteredData.length === 0" class="text-center py-12">
        <UIcon :name="emptyIcon" class="w-12 h-12 text-neutral-400 dark:text-neutral-500 mx-auto mb-4" />
        <h3 class="text-lg font-medium text-neutral-900 dark:text-white mb-2">{{ emptyTitle }}</h3>
        <p class="text-neutral-500 dark:text-neutral-400 mb-4">{{ emptyDescription }}</p>
        <UButton
          v-if="emptyAction"
          :color="emptyAction.color || 'primary'"
          @click="emptyAction.action"
        >
          {{ emptyAction.label }}
        </UButton>
      </div>
      
      <!-- 数据列表 -->
      <div v-else class="space-y-3">
        <!-- 使用 UTable 如果数据是表格格式 -->
        <UTable
          v-if="isTableData"
          :rows="paginatedData"
          :columns="tableColumns"
          :loading="loading"
          :empty-state="{ icon: emptyIcon, label: emptyTitle, description: emptyDescription }"
          @select="handleTableSelect"
        />
        <!-- 使用组件渲染 -->
        <component
          v-else
          v-for="(item, index) in paginatedData"
          :key="getItemKey(item, index)"
          :is="itemComponent"
          v-bind="getItemProps(item)"
          @click="handleItemClick(item)"
          v-on="getItemEvents(item)"
        />
      </div>
      
      <!-- 分页 -->
      <div v-if="showPagination && totalPages > 1" class="flex justify-center mt-6">
        <UPagination
          v-model="currentPage"
          :total="totalItems"
          :page-size="pageSize"
          show-first
          show-last
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

export interface FilterOption {
  label: string
  value: string
}

export interface ToolbarAction {
  key: string
  label: string
  icon?: string
  color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
  variant?: 'solid' | 'outline' | 'soft' | 'ghost' | 'link'
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  action: () => void
}

export interface EmptyAction {
  label: string
  color?: 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
  action: () => void
}

interface Props {
  // 数据相关
  data: any[]
  loading?: boolean
  itemComponent?: Component | string
  getItemKey?: (item: any, index: number) => string | number
  getItemProps?: (item: any) => Record<string, any>
  getItemEvents?: (item: any) => Record<string, Function>
  
  // 表格相关
  isTableData?: boolean
  tableColumns?: any[]
  
  // 搜索和筛选
  searchable?: boolean
  searchFields?: string[]
  filters?: FilterOption[]
  
  // 分页
  showPagination?: boolean
  pageSize?: number
  
  // 工具栏
  showToolbar?: boolean
  toolbarActions?: ToolbarAction[]
  
  // 空状态
  emptyIcon?: string
  emptyTitle?: string
  emptyDescription?: string
  emptyAction?: EmptyAction
  
  // 其他
  skeletonCount?: number
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  searchable: false,
  searchFields: () => ['name', 'title', 'description'],
  filters: () => [],
  showPagination: true,
  pageSize: 10,
  showToolbar: true,
  toolbarActions: () => [],
  emptyIcon: 'i-heroicons-inbox',
  emptyTitle: '暂无数据',
  emptyDescription: '当前没有可显示的内容',
  skeletonCount: 3,
  isTableData: false,
  tableColumns: () => [],
  getItemKey: (item: any, index: number) => item.id || index,
  getItemProps: () => ({}),
  getItemEvents: () => ({})
})

const emit = defineEmits<{
  itemClick: [item: any]
  search: [query: string]
  filterChange: [filter: string]
  tableSelect: [rows: any[]]
}>()

// 搜索和筛选状态
const searchQuery = ref('')
const selectedFilter = ref('')
const currentPage = ref(1)

// 计算属性
const filteredData = computed(() => {
  let result = [...props.data]
  
  // 搜索过滤
  if (searchQuery.value && props.searchable) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(item => 
      props.searchFields.some(field => 
        String(item[field] || '').toLowerCase().includes(query)
      )
    )
  }
  
  // 筛选器过滤
  if (selectedFilter.value) {
    result = result.filter(item => {
      // 这里可以根据具体需求实现筛选逻辑
      return true
    })
  }
  
  return result
})

const totalItems = computed(() => filteredData.value.length)
const totalPages = computed(() => Math.ceil(totalItems.value / props.pageSize))

const paginatedData = computed(() => {
  if (!props.showPagination) return filteredData.value
  
  const start = (currentPage.value - 1) * props.pageSize
  const end = start + props.pageSize
  return filteredData.value.slice(start, end)
})

// 事件处理
const handleSearch = () => {
  currentPage.value = 1
  emit('search', searchQuery.value)
}

const handleFilterChange = () => {
  currentPage.value = 1
  emit('filterChange', selectedFilter.value)
}

const handleItemClick = (item: any) => {
  emit('itemClick', item)
}

const handleTableSelect = (e: Event, row: any) => {
  emit('tableSelect', [row])
}

// 监听数据变化，重置分页
watch(() => props.data, () => {
  currentPage.value = 1
})
</script>
