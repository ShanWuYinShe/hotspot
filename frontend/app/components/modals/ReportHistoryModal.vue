<template>
  <UModal size="2xl">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-document-text" class="w-6 h-6 text-blue-500" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
              我的举报记录
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <!-- 状态筛选 -->
        <div class="mb-4">
          <UTabs v-model="activeTab" :items="tabItems" size="sm" />
        </div>

        <!-- 举报列表 -->
        <div class="max-h-96 overflow-y-auto">
          <div v-if="filteredReports.length > 0" class="space-y-3">
            <div
              v-for="report in filteredReports"
              :key="report.report_id || report.target_id"
              class="p-4 border border-neutral-200 dark:border-neutral-700 rounded-lg hover:bg-neutral-50 dark:hover:bg-neutral-800 transition-colors"
            >
              <div class="flex items-start justify-between mb-2">
                <div class="flex items-center gap-2">
                  <UBadge
                    :color="getStatusColor(report.status)"
                    :label="getStatusLabel(report.status)"
                    size="xs"
                  />
                  <span class="text-sm text-neutral-500">
                    {{ getTargetTypeLabel(report.target_type) }}
                  </span>
                </div>
                <span class="text-xs text-neutral-500">
                  {{ formatTime(report.created_at) }}
                </span>
              </div>
              
              <h4 class="font-medium text-neutral-900 dark:text-neutral-100 mb-1">
                {{ getReasonLabel(report.reason) }}
              </h4>
              
              <p v-if="report.description" class="text-sm text-neutral-600 dark:text-neutral-400 mb-2">
                {{ report.description }}
              </p>
              
              <div v-if="report.handler_note" class="text-xs text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 p-2 rounded">
                <strong>处理说明：</strong>{{ report.handler_note }}
              </div>
              
              <div v-if="report.handled_at" class="text-xs text-neutral-500 mt-2">
                处理时间：{{ formatTime(report.handled_at) }}
              </div>
            </div>
          </div>

          <!-- 空状态 -->
          <div v-else-if="!loading" class="text-center py-8">
            <UIcon name="i-heroicons-document-text" class="w-12 h-12 text-neutral-400 mx-auto mb-3" />
            <h4 class="text-sm font-medium text-neutral-900 dark:text-neutral-100 mb-1">
              {{ activeTab === 'all' ? '暂无举报记录' : `暂无${getTabLabel(activeTab)}举报` }}
            </h4>
            <p class="text-xs text-neutral-600 dark:text-neutral-400">
              {{ activeTab === 'all' ? '您还没有提交过任何举报' : '该状态下没有举报记录' }}
            </p>
          </div>

          <!-- 加载状态 -->
          <div v-if="loading" class="space-y-3">
            <USkeleton class="h-20" v-for="i in 3" :key="i" />
          </div>
        </div>

        <!-- 底部操作 -->
        <div class="flex justify-between items-center mt-6 pt-4 border-t border-neutral-200 dark:border-neutral-700">
          <div class="text-sm text-neutral-600 dark:text-neutral-400">
            共 {{ reports.length }} 条举报记录
          </div>
          <div class="flex gap-2">
            <UButton
              variant="outline"
              size="sm"
              icon="i-heroicons-arrow-path"
              @click="loadReports"
              :loading="loading"
            >
              刷新
            </UButton>
            <UButton
              variant="outline"
              size="sm"
              @click="handleClose"
            >
              关闭
            </UButton>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useReportsService } from '~/services/reports.service'
import type { ReportItem } from '~/types/api'

interface Props {
  onClose?: () => void
}

const props = defineProps<Props>()

// 状态管理
const loading = ref(false)
const reports = ref<ReportItem[]>([])
const activeTab = ref('all')

// 标签页配置
const tabItems = [
  { value: 'all', label: '全部' },
  { value: 'pending', label: '待处理' },
  { value: 'reviewing', label: '处理中' },
  { value: 'resolved', label: '已解决' },
  { value: 'dismissed', label: '已驳回' }
]

// 计算属性
const filteredReports = computed(() => {
  if (activeTab.value === 'all') return reports.value
  return reports.value.filter(report => report.status === activeTab.value)
})

// 工具函数
const { formatTime } = useFormatters()

const statusColors: Record<string, string> = {
  pending: 'orange',
  reviewing: 'blue',
  resolved: 'green',
  dismissed: 'red'
}

const statusLabels: Record<string, string> = {
  pending: '待处理',
  reviewing: '处理中',
  resolved: '已解决',
  dismissed: '已驳回'
}

const targetTypeLabels: Record<string, string> = {
  user: '用户',
  moment: '生活瞬间',
  comment: '评论',
  message: '消息',
  group: '群组'
}

const reasonLabels: Record<string, string> = {
  spam: '垃圾信息',
  harassment: '骚扰行为',
  inappropriate_content: '不当内容',
  fake_account: '虚假账户',
  violence: '暴力内容',
  other: '其他'
}

const getStatusColor = (status: string) => statusColors[status] || 'neutral'
const getStatusLabel = (status: string) => statusLabels[status] || status
const getTargetTypeLabel = (type: string) => targetTypeLabels[type] || type
const getReasonLabel = (reason: string) => reasonLabels[reason] || reason

const getTabLabel = (tab: string) => {
  const item = tabItems.find(t => t.value === tab)
  return item?.label || tab
}

// 方法
const loadReports = async () => {
  loading.value = true
  try {
    const reportsService = useReportsService()
    const response = await reportsService.getMyReports()
    reports.value = response.items || []
  } catch (error) {
    console.error('加载举报记录失败:', error)
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  props.onClose?.()
}

// 监听标签页变化
watch(activeTab, () => {
  // 可以在这里重新加载数据或进行筛选
})

// 页面加载
onMounted(() => {
  loadReports()
})
</script>
