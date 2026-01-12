<template>
    <UModal title="编辑资料" description="更新头像、昵称和个人简介">
        <template #body>
            <div class="space-y-6">
                <section
                    class="flex flex-col items-center space-y-3 pb-4"
                >
                    <AvatarUpload
                        :user="userForAvatar"
                        @uploaded="handleAvatarUploaded"
                    />
                    <p class="text-sm text-neutral-500 dark:text-neutral-400">
                        点击头像更换图片
                    </p>
                    <div class="w-full border-b border-neutral-100 dark:border-neutral-800"></div>
                </section>

                <UForm
                    :schema="schema"
                    :state="form"
                    class="space-y-5"
                    @submit="save"
                >
                    <UFormField label="昵称" name="nickname" :required="true">
                        <UInput
                            v-model="form.nickname"
                            placeholder="输入昵称"
                            :maxlength="20"
                            icon="i-lucide-user"
                        />
                    </UFormField>

                    <UFormField
                        label="个人简介"
                        name="bio"
                        :hint="`${(form.bio || '').length}/120`"
                    >
                        <UTextarea
                            v-model="form.bio"
                            placeholder="一句话介绍你自己..."
                            :rows="3"
                            :maxlength="120"
                            autoresize
                        />
                    </UFormField>

                    <UFormField
                        label="标签（可选）"
                        name="tags"
                        hint="用英文逗号分隔，如：摄影, 徒步"
                    >
                        <UInput
                            v-model="tagsText"
                            placeholder="输入标签"
                            icon="i-lucide-tags"
                        />
                    </UFormField>
                </UForm>
            </div>
        </template>

        <template #footer>
            <div class="flex justify-end gap-3 w-full">
                <UButton
                    type="button"
                    variant="ghost"
                    color="neutral"
                    :disabled="saving"
                    @click="emit('close')"
                >
                    取消
                </UButton>
                <UButton
                    type="button"
                    :loading="saving"
                    :disabled="!canSubmit"
                    @click="save"
                >
                    保存
                </UButton>
            </div>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import { z } from "zod";
import AvatarUpload from "~/components/forms/AvatarUpload.vue";
import { useUsersService } from "~/services/users.service";

const emit = defineEmits<{ close: [] }>();

const authStore = useAuthStore();
const feedback = useFeedbackStore();
const userService = useUsersService();

const currentUser = computed(() => authStore.user);

const userForAvatar = computed(() => ({
    nickname: currentUser.value?.nickname,
    avatar_url: currentUser.value?.avatar_url,
}));

const schema = z.object({
    nickname: z
        .string()
        .trim()
        .min(1, "请输入昵称")
        .max(20, "昵称最多 20 个字符"),
    bio: z
        .string()
        .max(120, "个人简介最多 120 个字符")
        .optional()
        .or(z.literal("")),
});

const form = reactive({
    nickname: currentUser.value?.nickname ?? "",
    bio: currentUser.value?.bio ?? "",
});

const tagsText = ref((currentUser.value?.tags ?? []).join(", "));
const saving = ref(false);

const canSubmit = computed(() => !!form.nickname.trim() && !saving.value);

const normalizeTags = (text: string): string[] | undefined => {
    const tags = text
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean)
        .slice(0, 10);
    return tags.length ? tags : undefined;
};

const refreshMe = async () => {
    const me = await userService.getMe();
    authStore.updateCurrentUser({
        nickname: me.nickname,
        avatar_url: me.avatar_url,
        bio: me.bio,
        tags: me.tags,
    });
};

const save = async () => {
    if (saving.value) return;

    saving.value = true;
    try {
        await userService.updateProfile({
            nickname: form.nickname.trim(),
            bio: form.bio?.trim() || undefined,
            tags: normalizeTags(tagsText.value),
        });
        await refreshMe();
        feedback.success("保存成功");
        emit("close");
    } catch (e: unknown) {
        feedback.error("保存失败", getErrorMessage(e));
    } finally {
        saving.value = false;
    }
};

const handleAvatarUploaded = async (fileId: string) => {
    if (saving.value) return;

    saving.value = true;
    try {
        await userService.updateProfile({ avatar_file_id: fileId });
        await refreshMe();
        feedback.success("头像已更新");
    } catch (e: unknown) {
        feedback.error("更新头像失败", getErrorMessage(e));
    } finally {
        saving.value = false;
    }
};
</script>
