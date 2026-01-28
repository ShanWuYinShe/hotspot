<template>
  <div class="flex flex-col gap-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">系统公告</h1>
        <p class="text-neutral-500 dark:text-neutral-400 mt-1">管理系统公告通知</p>
      </div>
      <UButton icon="i-lucide-plus" @click="openCreateModal">
        发布公告
      </UButton>
    </div>

    <!-- Announcements List -->
    <div class="space-y-4">
      <UCard
        v-for="announcement in announcements"
        :key="announcement.announcement_id"
        variant="outline"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="flex-1 space-y-2">
            <div class="flex items-center gap-2">
              <UBadge :color="getSeverityColor(announcement.severity)" variant="soft">
                {{ getSeverityLabel(announcement.severity) }}
              </UBadge>
              <h3 class="font-semibold text-lg">{{ announcement.title }}</h3>
            </div>
            <p class="text-neutral-600 dark:text-neutral-400">{{ announcement.content }}</p>
            <div class="flex items-center gap-4 text-sm text-neutral-500">
              <span class="flex items-center gap-1">
                <UIcon name="i-lucide-user" class="size-4" />
                {{ announcement.created_by.nickname }}
              </span>
              <span class="flex items-center gap-1">
                <UIcon name="i-lucide-clock" class="size-4" />
                {{ formatTime(announcement.created_at) }}
              </span>
              <a
                v-if="announcement.action_url"
                :href="announcement.action_url"
                target="_blank"
                class="flex items-center gap-1 text-primary-500 hover:underline"
              >
                <UIcon name="i-lucide-external-link" class="size-4" />
                查看链接
              </a>
            </div>
          </div>
          <UButton
            variant="ghost"
            color="error"
            icon="i-lucide-trash-2"
            size="sm"
            @click="confirmDelete(announcement)"
          />
        </div>
      </UCard>

      <!-- Empty State -->
      <UCard v-if="!loading && announcements.length === 0" variant="outline">
        <div class="text-center py-8 text-neutral-500">
          <UIcon name="i-lucide-megaphone" class="size-12 mx-auto mb-4 opacity-50" />
          <p>暂无系统公告</p>
        </div>
      </UCard>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-8">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-neutral-400" />
      </div>

      <!-- Load More -->
      <div v-if="hasMore" class="flex justify-center">
        <UButton variant="ghost" :loading="loadingMore" @click="loadMore">
          加载更多
        </UButton>
      </div>
    </div>

    <!-- Create Modal -->
    <UModal v-model:open="showCreateModal">
      <template #header>
        <div class="flex items-center gap-2">
          <UIcon name="i-lucide-megaphone" />
          <span>发布公告</span>
        </div>
      </template>
      <template #body>
        <div class="space-y-4">
          <UFormField label="公告标题" required>
            <UInput v-model="form.title" placeholder="请输入公告标题" />
          </UFormField>

          <UFormField label="公告内容" required>
            <UTextarea v-model="form.content" placeholder="请输入公告内容" :rows="4" />
          </UFormField>

          <UFormField label="严重程度">
            <USelect
              v-model="form.severity"
              :items="severityOptions"
            />
          </UFormField>

          <UFormField label="操作链接">
            <UInput v-model="form.action_url" placeholder="https://..." />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="showCreateModal = false">取消</UButton>
          <UButton :loading="creating" @click="doCreate">发布</UButton>
        </div>
      </template>
    </UModal>

    <!-- Delete Confirm Modal -->
    <UModal v-model:open="showDeleteModal">
      <template #header>
        <div class="flex items-center gap-2 text-red-600">
          <UIcon name="i-lucide-trash-2" />
          <span>删除公告</span>
        </div>
      </template>
      <template #body>
        <p>确定要删除公告 <strong>{{ announcementToDelete?.title }}</strong> 吗？</p>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="ghost" color="neutral" @click="showDeleteModal = false">取消</UButton>
          <UButton color="error" :loading="deleting" @click="doDelete">确认删除</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { useAdminAnnouncementsService } from '~/services/admin_announcements.service'
import type { SystemAnnouncementItem } from '~/types/api'

definePageMeta({
  layout: 'dashboard',
  requiresAuth: true,
  middleware: ['admin']
})

const adminAnnouncementsService = useAdminAnnouncementsService()
const { formatTime } = useFormatters()
const toast = useToast()

// State
const announcements = ref<SystemAnnouncementItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const cursor = ref<string | undefined>()

const showCreateModal = ref(false)
const creating = ref(false)
const form = reactive({
  title: '',
  content: '',
  severity: 'info' as 'info' | 'warning' | 'error',
  action_url: ''
})

const showDeleteModal = ref(false)
const deleting = ref(false)
const announcementToDelete = ref<SystemAnnouncementItem | null>(null)

const severityOptions = [
  { label: '提示', value: 'info' },
  { label: '警告', value: 'warning' },
  { label: '紧急', value: 'error' }
]

// Methods
function getSeverityColor(severity: string) {
  switch (severity) {
    case 'info': return 'primary'
    case 'warning': return 'warning'
    case 'error': return 'error'
    default: return 'neutral'
  }
}

function getSeverityLabel(severity: string) {
  switch (severity) {
    case 'info': return '提示'
    case 'warning': return '警告'
    case 'error': return '紧急'
    default: return severity
  }
}

async function fetchAnnouncements(reset = false) {
  if (reset) {
    cursor.value = undefined
    announcements.value = []
  }

  loading.value = reset
  loadingMore.value = !reset

  try {
    const res = await adminAnnouncementsService.list({
      cursor: cursor.value,
      limit: 20
    })
    announcements.value = reset ? res.items : [...announcements.value, ...res.items]
    hasMore.value = res.pagination.has_more
    cursor.value = res.pagination.next_cursor
  } catch (e) {
    toast.add({ title: '加载失败', color: 'error' })
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function loadMore() {
  fetchAnnouncements(false)
}

function openCreateModal() {
  form.title = ''
  form.content = ''
  form.severity = 'info'
  form.action_url = ''
  showCreateModal.value = true
}

async function doCreate() {
  if (!form.title.trim() || !form.content.trim()) {
    toast.add({ title: '请填写标题和内容', color: 'warning' })
    return
  }

  creating.value = true
  try {
    await adminAnnouncementsService.create({
      title: form.title,
      content: form.content,
      severity: form.severity,
      action_url: form.action_url || undefined
    })
    toast.add({ title: '发布成功', color: 'success' })
    showCreateModal.value = false
    fetchAnnouncements(true)
  } catch (e) {
    toast.add({ title: '发布失败', color: 'error' })
  } finally {
    creating.value = false
  }
}

function confirmDelete(announcement: SystemAnnouncementItem) {
  announcementToDelete.value = announcement
  showDeleteModal.value = true
}

async function doDelete() {
  if (!announcementToDelete.value) return
  deleting.value = true
  try {
    await adminAnnouncementsService.delete({
      announcement_id: announcementToDelete.value.announcement_id
    })
    toast.add({ title: '删除成功', color: 'success' })
    showDeleteModal.value = false
    fetchAnnouncements(true)
  } catch (e) {
    toast.add({ title: '删除失败', color: 'error' })
  } finally {
    deleting.value = false
  }
}

onMounted(() => {
  fetchAnnouncements(true)
})
</script>
