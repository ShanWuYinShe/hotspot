<template>
    <UModal
        :title="title"
        :description="description"
        @update:open="(val) => !val && handleClose()"
    >
        <template #body>
            <div class="space-y-4">
                <UAlert
                    v-if="report"
                    icon="i-lucide-info"
                    color="neutral"
                    variant="soft"
                    title="举报信息"
                    :description="`${report.reason}: ${report.description || '无描述'}`"
                />

                <UForm :state="form" @submit="handleSubmit">
                    <UFormField label="处理备注" help="可选，用于记录处理理由">
                        <UTextarea v-model="form.note" placeholder="例如：内容违规，已删除" />
                    </UFormField>
                </UForm>
            </div>
        </template>

        <template #footer>
            <div class="flex justify-end gap-3 w-full">
                <UButton variant="ghost" color="neutral" @click="handleClose">取消</UButton>
                <UButton :color="actionColor" :loading="loading" @click="handleSubmit">
                    {{ actionText }}
                </UButton>
            </div>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import { useReportsService } from '~/services/reports.service'
import type { ReportItem } from '~/types/api'

const props = defineProps<{
    open: boolean
    report?: ReportItem
    action: 'resolve' | 'dismiss'
}>()

const emit = defineEmits<{
    'update:open': [value: boolean]
    'success': []
}>()

const reportsService = useReportsService()
const feedback = useFeedbackStore()

const loading = ref(false)
const form = reactive({
    note: ''
})

const title = computed(() => props.action === 'resolve' ? '确认处理举报' : '确认忽略举报')
const description = computed(() => props.action === 'resolve' ? '该操作将标记举报为已解决。' : '该操作将标记举报为已忽略。')
const actionText = computed(() => props.action === 'resolve' ? '确认处理' : '确认忽略')
const actionColor = computed(() => props.action === 'resolve' ? 'primary' : 'neutral')

function handleClose() {
    emit('update:open', false)
    form.note = ''
}

async function handleSubmit() {
    if (!props.report) return
    loading.value = true
    try {
        await reportsService.adminHandle({
            report_id: props.report.report_id,
            action: props.action,
            note: form.note
        })
        feedback.success('操作成功')
        emit('success')
        handleClose()
    } catch (e) {
        feedback.error('操作失败')
    } finally {
        loading.value = false
    }
}
</script>
