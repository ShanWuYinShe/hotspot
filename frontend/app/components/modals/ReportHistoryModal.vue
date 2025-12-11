<template>
  <UModal
    size="2xl"
    title="我的举报记录"
    :description="`共 ${reports.length} 条举报记录`"
  >
    <template #body>
      <UTabs
        v-model="activeTab"
        :items="tabItems"
        size="sm"
        class="mb-4"
      />

      <section class="max-h-96 overflow-y-auto space-y-3">
        <!-- 举报列表 -->
        <UCard
          v-for="report in filteredReports"
          :key="report.id"
          :ui="{ body: 'p-4' }"
        >
          <header class="flex items-start justify-between mb-2">
            <section class="flex items-center gap-2">
              <UBadge
                :color="getStatusColor(report.status)"
                :label="getStatusLabel(report.status)"
                size="xs"
              />
              <span class="text-sm text-secondary-content">{{ getTargetTypeLabel(report.target_type) }}</span>
            </section>
            <time class="text-xs text-secondary-content">{{ formatTime(report.created_at) }}</time>
          </header>

          <h4 class="font-medium mb-1">{{ getReasonLabel(report.reason) }}</h4>
          <p
            v-if="report.description"
            class="text-sm text-secondary-content mb-2"
          >
            {{ report.description }}
          </p>

          <p
            v-if="report.handled_at"
            class="text-xs text-secondary-content mt-2"
          >
            处理时间：{{ formatTime(report.handled_at) }}
          </p>
        </UCard>

        <!-- 空状态 -->
        <UAlert
          v-if="!loading && filteredReports.length === 0"
          icon="i-heroicons-document-text"
          color="neutral"
          variant="subtle"
          :title="activeTab === 'all' ? '暂无举报记录' : `暂无${getTabLabel(activeTab)}举报`"
          :description="activeTab === 'all' ? '您还没有提交过任何举报' : '该状态下没有举报记录'"
        />

        <!-- 加载状态 -->
        <template v-if="loading">
          <USkeleton
            v-for="i in 3"
            :key="i"
            class="h-20 rounded-lg"
          />
        </template>
      </section>
    </template>

    <template #footer="{ close }">
      <UButton
        variant="outline"
        icon="i-heroicons-arrow-path"
        :loading="loading"
        @click="loadReports"
      >
        刷新
      </UButton>
      <UButton
        variant="outline"
        @click="close"
      >
        关闭
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useReportsService } from '~/services/reports.service'

const loading = ref(false)
const reports = ref<any[]>([])
const activeTab = ref('all')

const tabItems = [
  { value: 'all', label: '全部' },
  { value: 'pending', label: '待处理' },
  { value: 'reviewing', label: '处理中' },
  { value: 'resolved', label: '已解决' },
  { value: 'dismissed', label: '已驳回' }
]

const filteredReports = computed(() => {
  if (activeTab.value === 'all') return reports.value
  return reports.value.filter(report => report.status === activeTab.value)
})

const { formatTime } = useFormatters()

type BadgeColor = 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'

const getStatusColor = (status: string): BadgeColor => {
  const colors: Record<string, BadgeColor> = {
    pending: 'warning',
    reviewing: 'info',
    resolved: 'success',
    dismissed: 'error'
  }
  return colors[status] || 'neutral'
}

const getStatusLabel = (status: string) => {
  const labels: Record<string, string> = {
    pending: '待处理',
    reviewing: '处理中',
    resolved: '已解决',
    dismissed: '已驳回'
  }
  return labels[status] || status
}

const getTargetTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    user: '用户',
    moment: '生活瞬间',
    comment: '评论',
    message: '消息',
    group: '群组'
  }
  return labels[type] || type
}

const getReasonLabel = (reason: string) => {
  const labels: Record<string, string> = {
    spam: '垃圾信息',
    harassment: '骚扰行为',
    inappropriate_content: '不当内容',
    fake_account: '虚假账户',
    violence: '暴力内容',
    other: '其他'
  }
  return labels[reason] || reason
}

const getTabLabel = (tab: string) => tabItems.find(t => t.value === tab)?.label || tab

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

onMounted(() => loadReports())
</script>
