<template>
    <div class="mx-auto max-w-2xl px-4 sm:px-6 lg:px-8 py-14 sm:py-16">
        <!-- 标题栏 -->
        <UCard class="mb-6">
            <template #header>
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-3">
                        <UButton
                            variant="ghost"
                            icon="i-heroicons-arrow-left"
                            @click="handleBack"
                        />
                        <h1 class="text-2xl font-bold">发布动态</h1>
                    </div>
                    <UButton
                        :disabled="!canPublish"
                        :loading="publishing"
                        @click="handlePublish"
                    >
                        发布
                    </UButton>
                </div>
            </template>
        </UCard>

        <UCard>
            <div class="space-y-6">
                <!-- 用户信息 -->
                <div class="flex items-center gap-3">
                    <UAvatar :alt="authStore.user?.nickname" size="md">
                        {{ authStore.user?.nickname?.charAt(0) || "U" }}
                    </UAvatar>
                    <div>
                        <div class="font-semibold">
                            {{ authStore.user?.nickname }}
                        </div>
                        <div
                            class="text-sm text-neutral-600 dark:text-neutral-400"
                        >
                            {{ formatCurrentTime() }}
                        </div>
                    </div>
                </div>

                <!-- 文本内容 -->
                <div>
                    <UTextarea
                        v-model="content"
                        placeholder="分享你的精彩时刻..."
                        :rows="6"
                        :maxlength="500"
                        class="text-lg"
                    />
                    <div class="flex justify-end mt-2">
                        <span class="text-sm text-neutral-500">
                            {{ content.length }}/500
                        </span>
                    </div>
                </div>

                <!-- 图片上传区域 -->
                <div class="space-y-3">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-photo" class="w-5 h-5" />
                        <span class="font-medium">图片</span>
                        <span class="text-sm text-neutral-500"
                            >（最多9张）</span
                        >
                    </div>

                    <UFileUpload
                        v-model="uploadedFiles"
                        multiple
                        accept="image/*"
                        :max-files="9"
                        icon="i-heroicons-photo"
                        label="点击上传或拖拽图片到这里"
                        description="支持 JPG, PNG, GIF 格式，最多9张图片"
                        layout="grid"
                        class="min-h-32"
                        @change="handleFileChange"
                    />
                </div>

                <!-- 位置信息 -->
                <div class="space-y-3">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-map-pin" class="w-5 h-5" />
                        <span class="font-medium">位置</span>
                    </div>

                    <div class="flex gap-3">
                        <UInput
                            v-model="locationName"
                            placeholder="添加位置信息..."
                            class="flex-1"
                        />
                        <UButton
                            variant="outline"
                            icon="i-heroicons-map-pin"
                            :loading="gettingLocation"
                            @click="getCurrentLocation"
                        >
                            {{ coordinates ? "重新定位" : "获取位置" }}
                        </UButton>
                    </div>

                    <div
                        v-if="coordinates"
                        class="text-sm text-neutral-600 dark:text-neutral-400"
                    >
                        <UIcon
                            name="i-heroicons-check-circle"
                            class="w-4 h-4 inline mr-1 text-green-500"
                        />
                        位置已获取: {{ coordinates.latitude.toFixed(6) }},
                        {{ coordinates.longitude.toFixed(6) }}
                    </div>
                </div>

                <!-- 标签 -->
                <div class="space-y-3">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-hashtag" class="w-5 h-5" />
                        <span class="font-medium">标签</span>
                    </div>

                    <!-- 已选标签 -->
                    <div
                        v-if="selectedTags.length > 0"
                        class="flex flex-wrap gap-2"
                    >
                        <UBadge
                            v-for="tag in selectedTags"
                            :key="tag"
                            variant="soft"
                            size="xs"
                            class="cursor-pointer"
                            @click="removeTag(tag)"
                        >
                            #{{ tag }}
                        </UBadge>
                    </div>

                    <!-- 添加标签 -->
                    <div class="flex gap-2">
                        <UInput
                            v-model="newTag"
                            placeholder="输入标签..."
                            class="flex-1"
                            @keyup.enter="addTag"
                            @keyup.space="addTag"
                        />
                        <UButton
                            variant="outline"
                            :disabled="!newTag.trim()"
                            @click="addTag"
                        >
                            添加
                        </UButton>
                    </div>

                    <!-- 推荐标签 -->
                    <div class="space-y-2">
                        <span
                            class="text-sm text-neutral-600 dark:text-neutral-400"
                            >推荐标签:</span
                        >
                        <div class="flex flex-wrap gap-2">
                            <UButton
                                v-for="tag in recommendTags"
                                :key="tag"
                                variant="ghost"
                                size="xs"
                                :disabled="selectedTags.includes(tag)"
                                @click="addRecommendTag(tag)"
                            >
                                #{{ tag }}
                            </UButton>
                        </div>
                    </div>
                </div>

                <!-- 隐私设置 -->
                <div class="space-y-3">
                    <div class="flex items-center gap-2">
                        <UIcon name="i-heroicons-eye" class="w-5 h-5" />
                        <span class="font-medium">隐私设置</span>
                    </div>

                    <!-- 使用 URadioGroup 替代原生 radio 输入 -->
                    <URadioGroup v-model="privacy" :options="privacyOptions" />
                </div>
            </div>
        </UCard>
        <!-- 离开确认弹窗已改为 useModals.openConfirmModal 程序化管理 -->
    </div>
</template>

<script setup lang="ts">
definePageMeta({
    title: "发布动态",
    requiresAuth: true,
});

const authStore = useAuthStore();
const router = useRouter();
const momentsStore = useMomentsStore();
const toast = useSmartToast();
const { openConfirmModal } = useModals();
const { coordinates, getCurrentPosition } = useGeolocation();

// 表单状态
const content = ref("");
const locationName = ref("");
const selectedTags = ref<string[]>([]);
const newTag = ref("");
const privacy = ref("public");
const publishing = ref(false);
const gettingLocation = ref(false);
// leave confirm handled via useModals.openConfirmModal
// pending navigation state no longer required

// 隐私选项配置
const privacyOptions = [
    {
        value: "public",
        label: "公开",
        description: "所有人都可以看到",
    },
    {
        value: "friends",
        label: "好友可见",
        description: "仅好友可以看到",
    },
];

// 图片相关
const uploadedFiles = ref<File[]>([]);

// 推荐标签
const recommendTags = ref([
    "美食",
    "风景",
    "生活",
    "心情",
    "旅行",
    "摄影",
    "健身",
    "音乐",
]);

// 计算属性
const canPublish = computed(() => {
    return content.value.trim().length > 0 || uploadedFiles.value.length > 0;
});

// 文件处理
function handleFileChange(files: File[]) {
    // UFileUpload 组件已经处理了文件验证和限制
    uploadedFiles.value = files;
}

// 位置处理
async function getCurrentLocation() {
    gettingLocation.value = true;
    try {
        await getCurrentPosition();
        toast.add({
            title: "定位成功",
            color: "success",
        });
    } catch (error) {
        toast.add({
            title: "定位失败",
            description: "请检查位置权限设置",
            color: "error",
        });
    } finally {
        gettingLocation.value = false;
    }
}

// 标签处理
function addTag() {
    const tag = newTag.value.trim();
    if (!tag) return;

    if (selectedTags.value.includes(tag)) {
        toast.add({
            title: "标签已存在",
            color: "warning",
        });
        return;
    }

    if (selectedTags.value.length >= 10) {
        toast.add({
            title: "标签过多",
            description: "最多只能添加10个标签",
            color: "warning",
        });
        return;
    }

    selectedTags.value.push(tag);
    newTag.value = "";
}

function addRecommendTag(tag: string) {
    if (!selectedTags.value.includes(tag)) {
        selectedTags.value.push(tag);
    }
}

function removeTag(tag: string) {
    const index = selectedTags.value.indexOf(tag);
    if (index > -1) {
        selectedTags.value.splice(index, 1);
    }
}

// 发布处理
async function handlePublish() {
    if (!canPublish.value || publishing.value) return;

    publishing.value = true;
    try {
        // 注意：图片上传功能需要后端 API 支持，暂时使用占位逻辑
        // 这里应该先上传图片，获取图片URL

        const payload = {
            description: content.value.trim(),
            latitude: coordinates.value?.latitude || 0,
            longitude: coordinates.value?.longitude || 0,
            location_name: locationName.value.trim(),
            tags: selectedTags.value,
        };

        await momentsStore.createMoment(payload);

        toast.add({
            title: "发布成功",
            color: "success",
        });

        // 返回动态广场
        navigateTo("/moments");
    } catch (e: any) {
        toast.add({
            title: "发布失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        publishing.value = false;
    }
}

// 返回处理
function handleBack() {
    if (content.value.trim() || uploadedFiles.value.length > 0) {
        openConfirmModal({
            title: "确认离开",
            message: "确定要离开吗？未保存的内容将丢失。",
            confirmText: "确认离开",
            cancelText: "取消",
            confirmColor: "warning",
            onConfirm: async () => { await navigateTo("/moments"); },
        });
    } else {
        navigateTo("/moments");
    }
}

function formatCurrentTime() {
    return new Date().toLocaleString("zh-CN", {
        year: "numeric",
        month: "long",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    });
}

// 页面离开确认
onBeforeRouteLeave((to) => {
    if (content.value.trim() || uploadedFiles.value.length > 0) {
        openConfirmModal({
            title: "确认离开",
            message: "确定要离开吗？未保存的内容将丢失。",
            confirmText: "确认离开",
            cancelText: "取消",
            confirmColor: "warning",
            onConfirm: () => {
                if (to?.fullPath) router.push(to.fullPath);
                else if (typeof to === "string") router.push(to);
                else navigateTo("/moments");
            },
        });
        return false;
    }
    return true;
});

// handled by useModals.openConfirmModal

// handled by useModals.openConfirmModal

// 清理文件引用
onUnmounted(() => {
    // UFileUpload 组件会自动处理文件清理
    uploadedFiles.value = [];
});
</script>
