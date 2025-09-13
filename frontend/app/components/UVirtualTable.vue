<!--
  UVirtualTable - 基于 TanStack Virtual 的虚拟化表格
  完美集成 Nuxt UI v3 的设计系统和 UTable 的 API
-->
<template>
  <div class="relative">
    <!-- 表格头部（固定，使用原生 UTable 渲染） -->
    <div class="sticky top-0 z-10 bg-white dark:bg-neutral-900 border-b border-neutral-200 dark:border-neutral-800">
      <UTable
        :rows="[]"
        :columns="columns"
        :ui="{
          root: 'relative overflow-visible',
          tbody: 'hidden'
        }"
      >
        <!-- 传递头部插槽 -->
        <template v-for="(_, slotName) in headerSlots" :key="slotName" #[slotName]="slotProps">
          <slot :name="slotName" v-bind="slotProps" />
        </template>
      </UTable>
    </div>

    <!-- 虚拟滚动容器 -->
    <div
      ref="containerRef"
      :style="{ height: `${containerHeight}px` }"
      class="overflow-auto"
      :class="ui.wrapper"
    >
      <!-- 虚拟滚动内容 -->
      <div :style="{ height: `${totalHeight}px`, position: 'relative' }">
        <div
          v-for="{ index, item, start, key } in visibleItems"
          :key="key"
          :ref="(el) => measureElement(index, el as HTMLElement)"
          :style="{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '100%',
            transform: `translateY(${start}px)`
          }"
          :class="[
            'hover:bg-neutral-100 dark:hover:bg-neutral-800/50 transition-colors cursor-pointer',
            ui.body.base,
            {
              [ui.body.background.even]: striped && index % 2 === 0,
              [ui.body.background.odd]: striped && index % 2 === 1
            }
          ]"
        >
          <!-- 使用 UTable 的行渲染逻辑 -->
          <div class="flex items-center" :style="{ minHeight: `${estimatedRowHeight}px` }">
            <div
              v-for="column in columns"
              :key="column.key"
              :class="[
                'flex-shrink-0',
                ui.body.cell.base,
                column.class
              ]"
              :style="getColumnStyle(column)"
            >
              <!-- 单元格内容插槽 -->
              <slot
                :name="`${column.key}-data`"
                :row="item"
                :column="column"
                :index="index"
                :get-row-data="(row: any, key: string) => getRowData(row, key)"
              >
                <!-- 默认单元格渲染 -->
                <span :class="ui.body.cell.text">
                  {{ getRowData(item, column.key) }}
                </span>
              </slot>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" :class="ui.loading.wrapper">
      <UIcon :name="ui.loading.icon" :class="ui.loading.icon" />
      <span :class="ui.loading.text">{{ loadingText }}</span>
    </div>

    <!-- 空状态 -->
    <div v-if="rows.length === 0 && !loading" :class="ui.empty.wrapper">
      <slot name="empty">
        <UIcon :name="ui.empty.icon" :class="ui.empty.icon" />
        <p :class="ui.empty.text">{{ emptyText }}</p>
      </slot>
    </div>

    <!-- 滚动控制按钮 -->
    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 scale-90"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition-all duration-300"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-90"
    >
      <UButton
        v-if="showScrollToTop && (virtualizer.scrollOffset || 0) > containerHeight"
        :class="ui.scrollButton.top"
        size="sm"
        variant="soft"
        icon="i-heroicons-arrow-up"
        @click="scrollToTop"
      />
    </Transition>

    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 scale-90"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition-all duration-300"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-90"
    >
      <UButton
        v-if="showScrollToBottom && (virtualizer.scrollOffset || 0) < totalHeight - containerHeight - 100"
        :class="ui.scrollButton.bottom"
        size="sm"
        variant="soft"
        icon="i-heroicons-arrow-down"
        @click="scrollToBottom"
      />
    </Transition>
  </div>
</template>

<script setup lang="ts">
// 自定义 TableColumn 接口
interface CustomTableColumn {
  id: string;
  key: string;
  label: string;
  class?: string;
  width?: string | number;
}

// Props 接口
interface Props {
  // 数据
  rows: Record<string, unknown>[]
  // 列定义
  columns: CustomTableColumn[]
  // 容器高度
  containerHeight?: number
  // 估计行高
  estimatedRowHeight?: number
  // 是否启用条纹
  striped?: boolean
  // 是否显示加载状态
  loading?: boolean
  // 加载文本
  loadingText?: string
  // 空状态文本
  emptyText?: string
  // 是否显示滚动到顶部按钮
  showScrollToTop?: boolean
  // 是否显示滚动到底部按钮
  showScrollToBottom?: boolean
  // 缓冲区大小
  overscan?: number
  // 是否启用动态高度
  enableDynamicSize?: boolean
  // UI 配置
  ui?: Record<string, unknown>
}

const props = withDefaults(defineProps<Props>(), {
  containerHeight: 500,
  estimatedRowHeight: 60,
  striped: false,
  loading: false,
  loadingText: '加载中...',
  emptyText: '暂无数据',
  showScrollToTop: true,
  showScrollToBottom: true,
  overscan: 5,
  enableDynamicSize: false,
  ui: () => ({})
})

// 事件
const emit = defineEmits<{
  'row-click': [row: Record<string, unknown>, index: number]
  'scroll': [offset: number]
}>()

// 默认 UI 配置（继承 UTable 样式并简化）
const defaultUI = {
  root: 'relative overflow-hidden',
  body: {
    base: 'divide-y divide-neutral-200 dark:divide-neutral-800',
    background: {
      even: '',
      odd: 'bg-neutral-50 dark:bg-neutral-800/50'
    },
    cell: {
      base: 'px-4 py-3',
      text: 'text-sm text-neutral-900 dark:text-neutral-100'
    }
  },
  loading: {
    wrapper: 'flex justify-center items-center py-4',
    icon: 'animate-spin w-5 h-5 text-neutral-500', // 与 app.config.ts 保持一致
    text: 'ml-2 text-sm text-neutral-500 dark:text-neutral-400'
  },
  empty: {
    wrapper: 'flex flex-col items-center justify-center py-12 text-center',
    icon: 'w-12 h-12 text-neutral-400 mb-4',
    text: 'text-neutral-500 dark:text-neutral-400'
  },
  scrollButton: {
    top: 'fixed right-4 bottom-20 z-10',
    bottom: 'fixed right-4 bottom-4 z-10'
  }
}

// 合并 UI 配置
const ui = computed(() => ({
  ...defaultUI,
  ...props.ui
}))

// 虚拟滚动
const rowsRef = toRef(props, 'rows')
const {
  containerRef,
  visibleItems,
  totalHeight,
  scrollToTop,
  scrollToBottom,
  measureElement,
  virtualizer
} = useTanStackVirtualScroll(rowsRef, {
  containerHeight: props.containerHeight,
  estimatedItemHeight: props.estimatedRowHeight,
  overscan: props.overscan,
  enableDynamicSize: props.enableDynamicSize
})

// 计算头部插槽
const slots = useSlots()
const headerSlots = computed(() => {
  const headerSlotMap: Record<string, any> = {}
  for (const column of props.columns) {
    const slotName = `${column.key}-header`
    if (slotName in slots) {
      headerSlotMap[slotName] = slots[slotName]
    }
  }
  return headerSlotMap
})

// 工具函数
const getRowData = (row: any, key: string) => {
  return key.split('.').reduce((obj, k) => obj?.[k], row)
}

const getColumnStyle = (column: CustomTableColumn) => {
  const styles: any = {}
  if (column.width) {
    styles.width = typeof column.width === 'number' ? `${column.width}px` : column.width
    styles.flexShrink = 0
  }
  return styles
}

// 监听滚动事件
watch(() => virtualizer.value.scrollOffset, (offset) => {
  emit('scroll', offset || 0)
})
</script>

