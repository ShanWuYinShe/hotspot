<template>
    <UModal :title="`举报${getTargetTypeLabel(targetType)}`" size="lg">
        <template #body="{ close }">
            <UFormField label="请选择举报原因" :required="true" class="mb-6">
                <URadioGroup
                    v-model="selectedReason"
                    :items="reportReasons"
                    class="space-y-2"
                />
            </UFormField>

            <UFormField
                label="详细描述（可选）"
                :hint="`${description.length}/500`"
                class="mb-6"
            >
                <UTextarea
                    v-model="description"
                    placeholder="请详细描述您举报的原因，这将帮助我们更好地处理..."
                    :rows="4"
                    :maxlength="500"
                    autoresize
                />
            </UFormField>

            <UAlert
                color="warning"
                variant="subtle"
                icon="i-lucide-info"
                title="举报须知"
                class="mb-6"
            >
                <template #description>
                    <ul class="text-xs mt-1 text-neutral-600 dark:text-neutral-300 space-y-1">
                        <li>
                            •
                            我们会认真处理每一条举报，通常在24小时内给出处理结果
                        </li>
                        <li>• 恶意举报或虚假举报可能导致您的账户受到限制</li>
                        <li>• 举报记录将被保存，您可以在设置中查看处理状态</li>
                    </ul>
                </template>
            </UAlert>
        </template>

        <template #footer="{ close }">
            <div class="flex justify-end gap-3 w-full">
                <UButton
                    variant="ghost"
                    color="neutral"
                    :disabled="submitting"
                    @click="close"
                >
                    取消
                </UButton>
                <UButton
                    color="error"
                    :loading="submitting"
                    :disabled="!selectedReason"
                    @click="() => handleSubmit(close)"
                >
                    提交举报
                </UButton>
            </div>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import { useReportsService } from "~/services/reports.service";

interface Props {
    targetType: "user" | "moment" | "comment" | "message" | "group";
    targetId: string;
    targetTitle?: string;
    onSuccess?: () => void;
}

const props = withDefaults(defineProps<Props>(), {
    targetTitle: "",
});

const submitting = ref(false);
const selectedReason = ref("");
const description = ref("");
const feedback = useFeedbackStore();
const { requireAuth } = useRequireAuth();

const reportReasons = computed(() => {
    const baseReasons = [
        {
            value: "spam",
            label: "垃圾信息",
            description: "发布广告、刷屏、重复内容等",
        },
        {
            value: "harassment",
            label: "骚扰行为",
            description: "恶意骚扰、人身攻击、威胁等",
        },
        {
            value: "inappropriate_content",
            label: "不当内容",
            description: "色情、暴力、仇恨言论等不当内容",
        },
        {
            value: "violence",
            label: "暴力内容",
            description: "包含暴力、血腥、恐怖等内容",
        },
        {
            value: "other",
            label: "其他",
            description: "其他违反社区规范的行为",
        },
    ];

    if (props.targetType === "user") {
        baseReasons.splice(3, 0, {
            value: "fake_account",
            label: "虚假账户",
            description: "冒充他人、机器人账户、虚假信息等",
        });
    }

    return baseReasons;
});

const getTargetTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
        user: "用户",
        moment: "生活瞬间",
        comment: "评论",
        message: "消息",
        group: "群组",
    };
    return labels[type] || type;
};

const handleSubmit = async (close: () => void) => {
    if (!requireAuth("提交举报")) return;
    if (!selectedReason.value) {
        feedback.error("请选择举报原因");
        return;
    }

    submitting.value = true;

    try {
        const reportsService = useReportsService();
        await reportsService.create({
            target_type: props.targetType,
            target_id: props.targetId,
            reason: selectedReason.value,
            description: description.value.trim() || undefined,
        });

        feedback.success("举报提交成功", "我们会尽快处理您的举报");
        props.onSuccess?.();
        close();
    } catch (err: unknown) {
        console.error("提交举报失败:", err);
        feedback.error(
            "提交失败",
            getErrorMessage(err, "网络错误，请稍后重试"),
        );
    } finally {
        submitting.value = false;
    }
};
</script>
