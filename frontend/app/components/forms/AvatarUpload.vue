<template>
    <div class="flex flex-col items-center space-y-4">
        <!-- 当前头像预览 -->
        <div class="relative">
            <UTooltip text="点击更换头像">
                <UFileUpload
                    v-slot="{ open }"
                    v-model="selectedFile"
                    accept="image/*"
                    class="cursor-pointer"
                    @update:model-value="handleFileSelected"
                >
                    <div @click="open()">
                        <UAvatar
                            :alt="user?.nickname"
                            :src="avatarPreview || user?.avatar_url"
                            size="3xl"
                            class="border-4 border-default"
                        >
                            {{ user?.nickname?.charAt(0) || "U" }}
                        </UAvatar>

                        <!-- 上传按钮覆盖层 -->
                        <div
                            class="absolute inset-0 bg-black/50 rounded-full flex items-center justify-center opacity-0 hover:opacity-100 transition-opacity"
                        >
                            <UIcon name="i-heroicons-camera" class="icon-lg text-white" />
                        </div>
                    </div>
                </UFileUpload>
            </UTooltip>
        </div>

        <!-- 拖拽上传区域 (使用 UFileUpload) -->
        <UFileUpload
            v-model="selectedFile"
            accept="image/*"
            icon="i-heroicons-photo"
            label="点击选择或拖拽图片到这里"
            description="仅支持图片，最大 2MB"
            class="w-full max-w-xs min-h-32"
            @update:model-value="handleFileSelected"
        />

        <!-- 上传按钮 -->
        <div class="flex gap-2">
            <UFileUpload
                v-slot="{ open }"
                v-model="selectedFile"
                accept="image/*"
                class="inline-block"
                @update:model-value="handleFileSelected"
            >
                <UTooltip text="从本地选择图片文件">
                    <UButton variant="outline" size="sm" :disabled="uploading" @click="open()">
                        <UIcon name="i-heroicons-photo" class="icon-sm mr-2" />
                        选择图片
                    </UButton>
                </UTooltip>
            </UFileUpload>

            <UTooltip v-if="selectedFile" text="上传并设置为头像">
                <UButton
                    color="primary"
                    size="sm"
                    :loading="uploading"
                    @click="uploadAvatar"
                >
                    <UIcon name="i-heroicons-cloud-arrow-up" class="icon-sm mr-2" />
                    上传头像
                </UButton>
            </UTooltip>
        </div>

        <!-- 上传提示 -->
        <p class="text-sm text-secondary-content text-center">
            支持 JPG、PNG 格式，文件大小不超过 2MB
        </p>
    </div>
</template>

<script setup lang="ts">
import { useUploadService } from '~/services/upload.service'
import { getErrorMessage } from '~/utils/error'

interface Props {
    user?: {
        nickname?: string;
        avatar_url?: string;
    };
}

const props = defineProps<Props>();
const emit = defineEmits<{
    uploaded: [fileId: string];
}>();

const toast = useToast()
const selectedFile = ref<File | null>(null);
const avatarPreview = ref<string>("");
const uploading = ref(false);

// UFileUpload 回调 - 处理文件选择
const handleFileSelected = (file: File | null | undefined) => {
    if (!file) return;
    processFile(file);
};

// 处理文件
const processFile = (file: File) => {
    // 验证文件类型
    if (!file.type.startsWith("image/")) {
        toast.add({
            title: "文件格式错误",
            description: "请选择图片文件",
            color: "error",
        });
        selectedFile.value = null;
        return;
    }
    // 验证文件大小 (2MB)
    if (file.size > 2 * 1024 * 1024) {
        toast.add({
            title: "文件太大",
            description: "图片大小不能超过 2MB",
            color: "error",
        });
        selectedFile.value = null;
        return;
    }
    
    // 生成预览
    const reader = new FileReader();
    reader.onload = (e) => {
        avatarPreview.value = e.target?.result as string;
    };
    reader.readAsDataURL(file);
};

// 上传头像
const uploadAvatar = async () => {
    if (!selectedFile.value) return;

    uploading.value = true;

    try {
        // 使用 upload.service.ts 上传头像
        const uploadService = useUploadService();
        
        try {
            const response = await uploadService.uploadFile(selectedFile.value, 'avatar');
            
            toast.add({
                title: "头像上传成功",
                description: "您的头像已更新",
                color: "success",
            });

            // 发送上传成功事件（本地预览已在选择文件时生成）
            emit("uploaded", response.file_id);
        } catch (error: unknown) {
            toast.add({
                title: "头像上传失败",
                description: getErrorMessage(error, "请稍后重试"),
                color: "error",
            });
            throw error;
        }

        // 清理状态
        selectedFile.value = null;
    } catch (error: unknown) {
        toast.add({
            title: "上传失败",
            description: getErrorMessage(error, "请稍后重试"),
            color: "error",
        });
    } finally {
        uploading.value = false;
    }
};

// 清理预览
watch(
    () => props.user?.avatar_url,
    () => {
        if (!selectedFile.value) {
            avatarPreview.value = "";
        }
    },
);
</script>
