<template>
    <UModal :title="modalTitle" :description="modalDescription" size="xl">
        <template #body>
            <UForm :state="form" class="space-y-6" @submit="submitMoment">
                <!-- 瞬间描述 -->
                <UFormField
                    label="瞬间描述"
                    name="description"
                    :hint="`${form.description.length}/${MOMENT_CONSTANTS.MAX_DESCRIPTION_LENGTH}`"
                    :required="true"
                >
                    <UTextarea
                        v-model="form.description"
                        placeholder="分享您此刻的心情和想法..."
                        :rows="4"
                        :maxlength="MOMENT_CONSTANTS.MAX_DESCRIPTION_LENGTH"
                        autoresize
                    />
                </UFormField>

                <!-- 图片上传 -->
                <UFormField
                    label="图片"
                    :hint="
                        props.mode === 'edit'
                            ? '可选：上传后将替换原图片（最多9张）'
                            : '最多9张'
                    "
                >
                    <!-- 编辑模式显示原图 -->
                    <UCard
                        v-if="
                            props.mode === 'edit' &&
                            props.moment?.images?.length
                        "
                        class="mb-3"
                        variant="subtle"
                    >
                        <div class="grid grid-cols-3 sm:grid-cols-4 gap-2">
                            <div
                                v-for="(image, index) in props.moment.images"
                                :key="`${image}-${index}`"
                                class="lsyb-image-container-aspect"
                            >
                                <NuxtImg
                                    :src="image.url"
                                    :alt="`原图片 ${index + 1}`"
                                    class="w-full h-full object-cover"
                                />
                            </div>
                        </div>
                    </UCard>

                    <div class="grid grid-cols-3 sm:grid-cols-4 gap-2">
                        <div
                            v-for="(image, index) in form.images"
                            :key="image.file_id"
                            class="lsyb-image-container-aspect group"
                        >
                            <NuxtImg
                                :src="image.local_preview"
                                :alt="`图片 ${index + 1}`"
                                class="w-full h-full object-cover"
                            />
                            <div
                                class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center"
                            >
                                <UButton
                                    color="error"
                                    variant="solid"
                                    size="xs"
                                    icon="i-lucide-trash-2"
                                    square
                                    @click="removeImage(index)"
                                />
                            </div>
                        </div>

                        <!-- 图片上传按钮 -->
                        <UFileUpload
                            v-if="
                                form.images.length < MOMENT_CONSTANTS.MAX_IMAGES
                            "
                            v-slot="{ open }"
                            v-model="selectedImageFiles"
                            accept="image/*"
                            :multiple="true"
                            class="aspect-square"
                            @update:model-value="handleImagesSelected"
                        >
                        <UButton
                            variant="outline"
                            color="neutral"
                            class="w-full h-full flex flex-col gap-1"
                            :disabled="uploading"
                            @click="() => open()"
                        >
                                <UIcon
                                    v-if="uploading"
                                    name="i-lucide-loader-2"
                                    class="animate-spin w-6 h-6"
                                />
                                <template v-else>
                                    <UIcon
                                        name="i-lucide-camera"
                                        class="w-6 h-6"
                                    />
                                    <span class="text-xs font-medium"
                                        >添加图片</span
                                    >
                                </template>
                            </UButton>
                        </UFileUpload>
                    </div>
                </UFormField>

                <!-- 位置信息 -->
                <UFormField label="位置" :required="true">
                    <div class="flex gap-2">
                        <UCard
                            variant="soft"
                            size="sm"
                            class="flex-1 min-w-0 flex items-center gap-2"
                        >
                            <UIcon
                                name="i-lucide-map-pin"
                                class="w-4 h-4 text-neutral-400 shrink-0"
                            />
                            <span class="truncate text-sm">{{
                                form.location_name || "请选择位置"
                            }}</span>
                        </UCard>
                        <UButton
                            type="button"
                            variant="outline"
                            color="neutral"
                            icon="i-lucide-map"
                            @click="pickLocation"
                        >
                            地图选点
                        </UButton>
                    </div>
                    <div v-if="form.latitude" class="mt-2">
                        <UBadge
                            color="neutral"
                            variant="subtle"
                            size="xs"
                            class="font-mono"
                        >
                            {{ form.latitude.toFixed(6) }},
                            {{ form.longitude?.toFixed(6) }}
                        </UBadge>
                    </div>
                </UFormField>

                <!-- 标签 -->
                <UFormField
                    label="标签"
                    :hint="`最多 ${MOMENT_CONSTANTS.MAX_TAGS} 个`"
                >
                    <UCard variant="soft" class="space-y-3">
                        <div
                            v-if="form.tags.length > 0"
                            class="flex flex-wrap gap-2"
                        >
                            <UBadge
                                v-for="(tag, index) in form.tags"
                                :key="index"
                                color="primary"
                                variant="subtle"
                                class="cursor-pointer pr-1 items-center gap-1"
                                @click="removeTag(index)"
                            >
                                {{ tag }}
                                <UIcon
                                    name="i-lucide-x"
                                    class="w-3 h-3 hover:text-primary-600"
                                />
                            </UBadge>
                        </div>

                        <div class="flex gap-2">
                            <UInput
                                v-model="newTag"
                                placeholder="输入标签"
                                class="flex-1"
                                variant="soft"
                                @keyup.enter="addTag"
                            />
                            <UButton
                                type="button"
                                variant="outline"
                                color="neutral"
                                :disabled="
                                    !newTag.trim() ||
                                    form.tags.length >=
                                        MOMENT_CONSTANTS.MAX_TAGS
                                "
                                @click="addTag"
                            >
                                添加
                            </UButton>
                        </div>
                    </UCard>
                </UFormField>
            </UForm>
        </template>

        <template #footer>
            <div class="flex items-center justify-between w-full">
                <UButton
                    type="button"
                    variant="ghost"
                    color="neutral"
                    :disabled="creating"
                    @click="saveDraft"
                >
                    保存草稿
                </UButton>
                <div class="flex gap-3">
                    <UButton
                        type="button"
                        variant="ghost"
                        color="neutral"
                        :disabled="creating"
                        @click="emit('close')"
                    >
                        取消
                    </UButton>
                    <UButton
                        type="button"
                        :loading="creating"
                        :disabled="!isFormValid"
                        @click="submitMoment"
                    >
                        {{ props.mode === "edit" ? "保存修改" : "发布瞬间" }}
                    </UButton>
                </div>
            </div>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import { MOMENT_CONSTANTS } from "~/config/defaults";
import { useMomentsService } from "~/services/moments.service";
import { useUploadService } from "~/services/upload.service";
import type { MomentItem } from "~/types/api";

interface Props {
    onSuccess?: (momentId: string) => void;
    mode?: "create" | "edit";
    moment?: Pick<
        MomentItem,
        "moment_id" | "description" | "tags" | "images" | "location"
    > | null;
}

const props = withDefaults(defineProps<Props>(), {
    mode: "create",
    moment: null,
});
const emit = defineEmits<{
    close: [value?: string];
}>();

const momentsStore = useMomentsStore();
const locationStore = useLocationStore();

const creating = ref(false);
const uploading = ref(false);
const newTag = ref("");
// 图片上传 (UFileUpload)
const selectedImageFiles = ref<File[] | null>(null);

// 图片数据结构（本地预览 + 服务端 file_id）
interface ImageItem {
    file_id: string;
    local_preview: string; // 本地预览 URL（URL.createObjectURL）
}

// 初始化为空表单（水合安全）
const form = ref({
    description: "",
    images: [] as ImageItem[],
    latitude: null as number | null,
    longitude: null as number | null,
    location_name: "",
    tags: [] as string[],
});

// 客户端水合完成后从 store 恢复草稿
onMounted(() => {
    if (props.mode === "edit" && props.moment) {
        form.value.description = props.moment.description || "";
        form.value.tags = [...(props.moment.tags || [])];
        form.value.latitude = props.moment.location?.latitude ?? null;
        form.value.longitude = props.moment.location?.longitude ?? null;
        form.value.location_name = props.moment.location?.location_name ?? "";
        return;
    }

    if (momentsStore.hasDraft) {
        form.value.description = momentsStore.draft.description;
        // 草稿中的 images 是 file_id 数组，本地预览 URL 无法恢复，需重新上传
        form.value.latitude = momentsStore.draft.latitude;
        form.value.longitude = momentsStore.draft.longitude;
        form.value.location_name = momentsStore.draft.location_name;
        form.value.tags = [...momentsStore.draft.tags];
        return;
    }

    // 无草稿时默认使用当前定位（如果已获取）
    const loc = locationStore.location;
    if (loc) {
        form.value.latitude = loc.latitude;
        form.value.longitude = loc.longitude;
        form.value.location_name = loc.locationName;
    }
});

const isFormValid = computed(() => {
    return (
        form.value.description.trim() &&
        form.value.latitude !== null &&
        form.value.longitude !== null &&
        form.value.location_name.trim()
    );
});

const feedback = useFeedbackStore();
const modal = useModal();

const modalTitle = computed(() =>
    props.mode === "edit" ? "编辑生活瞬间" : "发布生活瞬间",
);
const modalDescription = computed(() =>
    props.mode === "edit" ? "修改您的动态内容" : "分享您的生活瞬间给朋友",
);

const pickLocation = async () => {
    await modal.showLocationPicker((loc) => {
        form.value.latitude = loc.latitude;
        form.value.longitude = loc.longitude;
        form.value.location_name = loc.locationName;
    });
};

// UFileUpload 回调
const handleImagesSelected = async (files: File[] | null | undefined) => {
    if (!files) return;

    const maxFiles = 9 - form.value.images.length;
    const filesToUpload = files.slice(0, maxFiles);

    uploading.value = true;
    try {
        const uploadService = useUploadService();

        for (const file of filesToUpload) {
            try {
                // 使用本地 URL 预览（客户端自己知道上传的内容）
                const localPreview = URL.createObjectURL(file);
                const result = await uploadService.uploadFile(
                    file,
                    "moment_image",
                );
                form.value.images.push({
                    file_id: result.file_id,
                    local_preview: localPreview,
                });
            } catch (err) {
                console.error("图片上传失败:", err);
                feedback.error("图片上传失败", `图片 ${file.name} 上传失败`);
            }
        }
    } finally {
        uploading.value = false;
    }

    // 重置选择
    selectedImageFiles.value = null;
};

const removeImage = (index: number) => {
    const removed = form.value.images.splice(index, 1);
    // 清理本地预览 URL
    if (removed[0]?.local_preview) {
        URL.revokeObjectURL(removed[0].local_preview);
    }
};

const addTag = () => {
    const tag = newTag.value.trim();
    if (
        !tag ||
        form.value.tags.includes(tag) ||
        form.value.tags.length >= MOMENT_CONSTANTS.MAX_TAGS
    )
        return;

    form.value.tags.push(tag);
    newTag.value = "";
};

const removeTag = (index: number) => form.value.tags.splice(index, 1);

const saveDraft = () => {
    momentsStore.saveDraft({
        description: form.value.description,
        images: form.value.images.map((img) => img.file_id), // 草稿只保存 file_id
        tags: form.value.tags,
        latitude: form.value.latitude,
        longitude: form.value.longitude,
        location_name: form.value.location_name,
    });
    feedback.success("草稿已保存");
};

const submitMoment = async () => {
    if (!isFormValid.value) return;

    creating.value = true;
    try {
        const momentService = useMomentsService();
        if (props.mode === "edit" && props.moment?.moment_id) {
            await momentService.update({
                moment_id: props.moment.moment_id,
                description: form.value.description,
                tags: form.value.tags,
                location: {
                    latitude: form.value.latitude!,
                    longitude: form.value.longitude!,
                    location_name: form.value.location_name,
                },
                image_file_ids:
                    form.value.images.length > 0
                        ? form.value.images.map((img) => img.file_id)
                        : undefined,
            });
            feedback.success("保存成功", "动态已更新");
            props.onSuccess?.(props.moment.moment_id);
            emit("close", props.moment.moment_id);
            return;
        }

        const response = await momentService.create({
            description: form.value.description,
            location: {
                latitude: form.value.latitude!,
                longitude: form.value.longitude!,
                location_name: form.value.location_name,
            },
            tags: form.value.tags,
            image_file_ids: form.value.images.map((img) => img.file_id),
        });

        feedback.success("瞬间发布成功");
        momentsStore.clearDraft();
        props.onSuccess?.(response.moment_id);
        emit("close", response.moment_id);
    } catch (err: unknown) {
        console.error("发布瞬间失败:", err);
        feedback.error(
            props.mode === "edit" ? "保存失败" : "发布失败",
            getErrorMessage(err, "请检查网络连接后重试"),
        );
    } finally {
        creating.value = false;
    }
};
</script>
