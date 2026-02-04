<template>
    <UModal size="md" title="删除账户" description="此操作无法撤销，请谨慎操作">
        <template #body="{ close }">
            <UAlert
                color="error"
                variant="soft"
                title="此操作无法撤销"
                description="删除账户后，您的所有数据将被永久删除，包括个人资料、发布的内容、好友关系等。"
                class="mb-6"
            />

            <UForm
                :state="formState"
                class="space-y-5"
                @submit="() => handleSubmit(close)"
            >
                <UFormField
                    name="confirmText"
                    :required="true"
                    label="请输入 DELETE_MY_ACCOUNT 来确认删除"
                >
                    <UInput
                        v-model="confirmText"
                        placeholder="DELETE_MY_ACCOUNT"
                        :disabled="isSubmitting"
                    />
                </UFormField>

                <UFormField name="reason" label="删除原因（可选）">
                    <UTextarea
                        v-model="reason"
                        placeholder="告诉我们您删除账户的原因，帮助我们改进产品"
                        :rows="3"
                        :disabled="isSubmitting"
                    />
                </UFormField>

                <UAlert
                    v-if="errorMessage"
                    color="error"
                    variant="soft"
                    :title="errorMessage"
                    icon="i-lucide-triangle-alert"
                />
            </UForm>
        </template>

        <template #footer="{ close }">
            <UButton
                variant="ghost"
                color="neutral"
                class="flex-1"
                :disabled="isSubmitting"
                @click="close"
            >
                取消
            </UButton>
            <UButton
                color="error"
                class="flex-1"
                :loading="isSubmitting"
                :disabled="!isFormValid"
                @click="() => handleSubmit(close)"
            >
                确认删除
            </UButton>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import { useUsersService } from "~/services/users.service";
import { useAuthStore } from "~/stores/auth.store";

interface Props {
    onSuccess?: () => void;
}

const emit = defineEmits<{ close: [value?: unknown] }>();

const props = defineProps<Props>();
const feedback = useFeedbackStore();
const router = useRouter();
const authStore = useAuthStore();

const confirmText = ref("");
const reason = ref("");
const isSubmitting = ref(false);
const errorMessage = ref("");

const formState = computed(() => ({
    confirmText: confirmText.value,
    reason: reason.value,
}));

const isFormValid = computed(() => confirmText.value === "DELETE_MY_ACCOUNT");

const handleSubmit = async (close: () => void) => {
    errorMessage.value = "";

    if (!isFormValid.value) {
        errorMessage.value = "请正确输入确认文本";
        return;
    }

    isSubmitting.value = true;

    try {
        const userService = useUsersService();

        await userService.deleteAccount({
            confirmation: "DELETE_MY_ACCOUNT",
        });

        feedback.success("账户已删除", "感谢您使用我们的产品，再见");

        authStore.logoutCurrent();
        await router.push("/");

        props.onSuccess?.();
        close();
    } catch (error) {
        console.error("删除账户失败:", error);
        errorMessage.value =
            (error as Error).message || "删除账户失败，请稍后重试";
    } finally {
        isSubmitting.value = false;
    }
};
</script>
