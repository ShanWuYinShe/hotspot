<template>
    <UModal title="发布动态" :ui="{ width: 'sm:max-w-2xl' }" prevent-close>
        <UCard>
            <template #header>
                <div class="flex items-center justify-between">
                    <h2 class="text-xl font-semibold">发布动态</h2>
                    <UButton
                        variant="ghost"
                        icon="i-heroicons-x-mark"
                        @click="handleClose"
                    />
                </div>
            </template>

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
                        <div class="text-sm text-neutral-600 dark:text-neutral-400">
                            {{ formatCurrentTime() }}
                        </div>
                    </div>
                </div>

                <!-- 文本内容 -->
                <div>
                    <UTextarea
                        v-model="content"
                        placeholder="分享你的精彩时刻..."
                        :rows="4"
                        :maxlength="500"
                        class="text-base"
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
                        <span class="text-sm text-neutral-500">（最多9张）</span>
                    </div>

                    <UFileUpload
                        v-model="uploadedFiles"
                        multiple
                        accept="image/*"
                        :max-files="9"
                        icon="i-heroicons-photo"
                        label="点击上传或拖拽图片到这里"
                        description="支持 JPG, PNG, GIF 格式"
                        layout="grid"
                        class="min-h-24"
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
                    <div v-if="selectedTags.length > 0" class="flex flex-wrap gap-2">
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
                        <span class="text-sm text-neutral-600 dark:text-neutral-400">
                            推荐标签:
                        </span>
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
            </div>

            <template #footer>
                <div class="flex justify-end gap-3">
                    <UButton variant="ghost" @click="handleClose">
                        取消
                    </UButton>
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
    </UModal>
</template>

<script setup lang="ts">
// useOverlay 会自动管理 Modal 状态，无需手动绑定 isOpen

interface Emits {
    close: []
    success: []
}

const emit = defineEmits<Emits>()

const authStore = useAuthStore()
const momentsStore = useMomentsStore()
const toast = useSmartToast()
const { coordinates, getCurrentPosition } = useGeolocation()

// 表单状态
const content = ref("")
const locationName = ref("")
const selectedTags = ref<string[]>([])
const newTag = ref("")
const publishing = ref(false)
const gettingLocation = ref(false)
const uploadedFiles = ref<File[]>([])

// 推荐标签
const recommendTags = ref([
    "美食", "风景", "生活", "心情", "旅行", "摄影", "健身", "音乐"
])

// 计算属性
const canPublish = computed(() => {
    return content.value.trim().length > 0 || uploadedFiles.value.length > 0
})

// 位置处理
async function getCurrentLocation() {
    gettingLocation.value = true
    try {
        await getCurrentPosition()
        toast.success("定位成功")
    } catch (error) {
        toast.error("定位失败，请检查位置权限设置")
    } finally {
        gettingLocation.value = false
    }
}

// 标签处理
function addTag() {
    const tag = newTag.value.trim()
    if (!tag) return

    if (selectedTags.value.includes(tag)) {
        toast.warning("标签已存在")
        return
    }

    if (selectedTags.value.length >= 10) {
        toast.warning("最多只能添加10个标签")
        return
    }

    selectedTags.value.push(tag)
    newTag.value = ""
}

function addRecommendTag(tag: string) {
    if (!selectedTags.value.includes(tag)) {
        selectedTags.value.push(tag)
    }
}

function removeTag(tag: string) {
    const index = selectedTags.value.indexOf(tag)
    if (index > -1) {
        selectedTags.value.splice(index, 1)
    }
}

// 发布处理
async function handlePublish() {
    if (!canPublish.value || publishing.value) return

    publishing.value = true
    try {
        const payload = {
            description: content.value.trim(),
            latitude: coordinates.value?.latitude || 0,
            longitude: coordinates.value?.longitude || 0,
            location_name: locationName.value.trim(),
            tags: selectedTags.value,
        }

        await momentsStore.createMoment(payload)
        
        toast.success("发布成功")
        resetForm()
        emit('success')
        emit('close')
    } catch (e: any) {
        toast.error("发布失败", e?.message)
    } finally {
        publishing.value = false
    }
}


// 重置表单
function resetForm() {
    content.value = ""
    locationName.value = ""
    selectedTags.value = []
    newTag.value = ""
    uploadedFiles.value = []
}

// 处理关闭
const handleClose = () => {
    if (content.value.trim() || uploadedFiles.value.length > 0) {
        // 有内容时确认关闭
        if (confirm("确定要关闭吗？未保存的内容将丢失。")) {
            resetForm()
            emit('close') // 通知 useOverlay 关闭
        }
    } else {
        resetForm()
        emit('close') // 通知 useOverlay 关闭
    }
}

function formatCurrentTime() {
    return new Date().toLocaleString("zh-CN", {
        year: "numeric",
        month: "long",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    })
}

</script>
