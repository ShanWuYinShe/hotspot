<template>
  <UModal
    title="我的举报记录"
    :description="`共 ${reports.length} 条举报记录`"
    size="lg"
  >
    <template #body>
      <div class="p-6">
        <UTabs
          v-model="activeTab"
        :items="tabItems"
        class="mb-4 w-full"
      />

      <section
        ref="listEl"
        class="max-h-96 overflow-y-auto space-y-3 px-1"
        @scroll="onListScroll"
      >
        <!-- 举报列表 -->
        <UCard
          v-for="report in filteredReports"
          :key="report.id"
          variant="subtle"
        >
          <header class="flex items-start justify-between mb-2 gap-2">
            <section class="flex flex-wrap items-center gap-2">
              <UBadge
                :color="getStatusColor(report.status)"
                :label="getStatusLabel(report.status)"
                variant="soft"
                size="xs"
              />
              <UBadge color="neutral" variant="outline" size="xs">{{ getTargetTypeLabel(report.target_type) }}</UBadge>
            </section>
            <span class="text-neutral-400 dark:text-neutral-500 text-xs shrink-0">{{ formatTime(report.created_at) }}</span>
          </header>

          <h4 class="font-semibold text-neutral-900 dark:text-white text-sm mb-1">{{ getReasonLabel(report.reason) }}</h4>
          
          <UCard
            v-if="report.description"
            variant="soft"
            size="sm"
            class="text-xs wrap-break-word"
          >
            {{ report.description }}
          </UCard>

          <div v-if="report.handled_at" class="mt-2 flex items-center gap-1 text-neutral-400 dark:text-neutral-500 text-xs">
            <UIcon name="i-lucide-check-circle" class="size-3" />
            处理于 {{ formatTime(report.handled_at) }}
          </div>
        </UCard>

        <!-- 空状态 -->
        <UEmpty
          v-if="!loading && filteredReports.length === 0"
          icon="i-lucide-file-text"
          :title="activeTab === 'all' ? '暂无举报记录' : `暂无${getTabLabel(activeTab)}举报`"
          :description="activeTab === 'all' ? '您还没有提交过任何举报' : '该状态下没有举报记录'"
          class="py-8"
        />

        <!-- 加载状态 -->
        <template v-if="loading">
          <USkeleton
            v-for="i in 3"
            :key="i"
            class="h-24 w-full rounded-2xl"
          />
        </template>

        <!-- 回到顶部 -->
        <div v-if="showScrollTop" class="sticky bottom-2 flex justify-end pointer-events-none">
          <UButton
            variant="soft"
            color="neutral"
            size="sm"
            icon="i-lucide-arrow-up"
            class="pointer-events-auto"
            @click="scrollToTop"
          />
        </div>
      </div>
    </template>

    <template #footer="{ close }">
      <div class="flex justify-between w-full">
        <UButton
            variant="ghost"
            color="neutral"
            icon="i-lucide-refresh-cw"
            :loading="loading"
            @click="loadReports"
        >
            刷新
        </UButton>
        <UButton
            color="neutral"
            variant="outline"
            @click="close"
        >
            关闭
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { useReportsService } from '~/services/reports.service'

const loading = ref(false)
const reports = ref<any[]>([])
const activeTab = ref('all')

const listEl = ref<HTMLElement | null>(null)
const showScrollTop = ref(false)

const onListScroll = () => {
  const el = listEl.value
  if (!el) return
  showScrollTop.value = el.scrollTop > 160
}

const scrollToTop = () => {
  listEl.value?.scrollTo({ top: 0, behavior: 'smooth' })
}

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
