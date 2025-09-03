<template>
  <div class="flex flex-col items-center space-y-4">
    <!-- 当前头像预览 -->
    <div class="relative">
      <UAvatar
        :alt="user?.nickname"
        :src="avatarPreview || user?.avatar_url"
        size="3xl"
        class="cursor-pointer border-4 border-neutral-200 dark:border-neutral-700 hover:border-primary-300 dark:hover:border-primary-600 transition-colors"
        @click="triggerFileInput"
      >
        {{ user?.nickname?.charAt(0) || "U" }}
      </UAvatar>
      
      <!-- 上传按钮覆盖层 -->
      <div 
        class="absolute inset-0 bg-black bg-opacity-50 rounded-full flex items-center justify-center opacity-0 hover:opacity-100 transition-opacity cursor-pointer"
        @click="triggerFileInput"
      >
        <UIcon name="i-heroicons-camera" class="w-8 h-8 text-white" />
      </div>
    </div>

    <!-- 文件输入 -->
    <input
      ref="fileInput"
      type="file"
      accept="image/*"
      class="hidden"
      @change="handleFileSelect"
    />

    <!-- 上传按钮 -->
    <div class="flex gap-2">
      <UButton 
        variant="outline" 
        size="sm" 
        @click="triggerFileInput"
        :disabled="uploading"
      >
        <UIcon name="i-heroicons-photo" class="w-4 h-4 mr-2" />
        选择图片
      </UButton>
      
      <UButton 
        v-if="selectedFile" 
        color="primary" 
        size="sm" 
        @click="uploadAvatar"
        :loading="uploading"
      >
        <UIcon name="i-heroicons-cloud-arrow-up" class="w-4 h-4 mr-2" />
        上传头像
      </UButton>
    </div>

    <!-- 上传提示 -->
    <p class="text-sm text-neutral-600 dark:text-neutral-400 text-center">
      支持 JPG、PNG 格式，文件大小不超过 2MB
    </p>
  </div>
</template>

<script setup lang="ts">
interface Props {
  user?: {
    nickname?: string;
    avatar_url?: string;
  };
}

const props = defineProps<Props>();
const emit = defineEmits<{
  uploaded: [url: string];
}>();

const toast = useSmartToast();
const fileInput = ref<HTMLInputElement>();
const selectedFile = ref<File | null>(null);
const avatarPreview = ref<string>('');
const uploading = ref(false);

// 触发文件选择
const triggerFileInput = () => {
  fileInput.value?.click();
};

// 处理文件选择
const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  
  if (!file) return;
  
  // 验证文件类型
  if (!file.type.startsWith('image/')) {
    toast.add({
      title: '文件格式错误',
      description: '请选择图片文件',
      color: 'error'
    });
    return;
  }
  
  // 验证文件大小 (2MB)
  if (file.size > 2 * 1024 * 1024) {
    toast.add({
      title: '文件太大',
      description: '图片大小不能超过 2MB',
      color: 'error'
    });
    return;
  }
  
  selectedFile.value = file;
  
  // 创建预览
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
    // TODO: 实现实际的头像上传逻辑
    // 这里应该调用后端API上传文件
    
    // 模拟上传延迟
    await new Promise(resolve => setTimeout(resolve, 1500));
    
    // 模拟上传成功，返回URL
    const mockAvatarUrl = avatarPreview.value;
    
    toast.add({
      title: '头像上传成功',
      description: '您的头像已更新',
      color: 'success'
    });
    
    // 发送上传成功事件
    emit('uploaded', mockAvatarUrl);
    
    // 清理状态
    selectedFile.value = null;
    
  } catch (error: any) {
    toast.add({
      title: '上传失败',
      description: error.message || '请稍后重试',
      color: 'error'
    });
  } finally {
    uploading.value = false;
  }
};

// 清理预览
watch(() => props.user?.avatar_url, () => {
  if (!selectedFile.value) {
    avatarPreview.value = '';
  }
});
</script>
