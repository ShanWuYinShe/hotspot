<template>
  <div class="space-y-4">
    <!-- 内容输入 -->
    <UTextarea
      v-model="content"
      placeholder="分享你的精彩时刻..."
      :rows="4"
      class="resize-none"
    />
    
    <!-- 图片上传区域 -->
    <UFormField label="添加图片" class="space-y-3">
      <template #hint>
        <span class="text-xs text-neutral-500">{{ uploadedFiles.length }}/9</span>
      </template>
      
      <UFileUpload
        v-model="uploadedFiles"
        multiple
        accept="image/*"
        :max-files="9"
        icon="i-heroicons-photo"
        label="点击上传或拖拽图片到这里"
        description="支持 JPG, PNG, GIF 格式，最多9张图片"
        class="min-h-32"
        @change="handleFileChange"
      />
    </UFormField>
    
    <!-- 标签选择 -->
    <UFormField label="添加标签">
      <div class="flex flex-wrap gap-2">
        <UBadge
          v-for="tag in recommendTags"
          :key="tag"
          :variant="selectedTags.includes(tag) ? 'solid' : 'outline'"
          class="cursor-pointer"
          @click="toggleTag(tag)"
          size="xs"
        >
          {{ tag }}
        </UBadge>
      </div>
    </UFormField>
    
    <!-- 隐私设置 -->
    <UFormField label="隐私设置">
      <USelect
        v-model="privacy"
        :options="privacyOptions"
        class="w-full"
      />
    </UFormField>
    
    <!-- 上传进度条 -->
    <div v-if="isUploading" class="space-y-2">
      <div class="flex items-center justify-between text-sm text-neutral-600">
        <span>正在上传图片...</span>
        <span>{{ uploadProgress }}%</span>
      </div>
      <UProgress :value="uploadProgress" :max="100" />
    </div>
    
    <!-- 发布按钮 -->
    <div class="flex justify-end space-x-3 pt-4">
      <UButton variant="ghost" @click="$emit('cancel')" :disabled="isUploading">
        取消
      </UButton>
      <UButton
        @click="publishMoment"
        :loading="publishing || isUploading"
        :disabled="!canPublish || isUploading"
      >
        发布动态
      </UButton>
    </div>
  </div>
</template>

<script setup lang="ts">

// Emits
const emit = defineEmits<{
  success: [];
  cancel: [];
}>()

const toast = useSmartToast()
const authStore = useAuthStore()
const momentsStore = useMomentsStore()
const { coordinates, getCurrentPosition } = useGeolocation()

// 表单数据
const content = ref('')
const uploadedFiles = ref<File[]>([])
const selectedTags = ref<string[]>([])
const privacy = ref('public')
const publishing = ref(false)
const uploadProgress = ref(0)
const isUploading = ref(false)

// 推荐标签
const recommendTags = ref(['美食', '风景', '生活', '心情', '旅行', '摄影'])

// 隐私选项
const privacyOptions = [
  { label: '公开', value: 'public' },
  { label: '好友可见', value: 'friends' },
  { label: '仅自己可见', value: 'private' }
]

// 计算属性
const canPublish = computed(() => {
  return content.value.trim().length > 0 || uploadedFiles.value.length > 0
})

// 文件处理
function handleFileChange(files: File[]) {
  // UFileUpload 组件已经处理了文件验证和限制
  uploadedFiles.value = files
}

// 标签处理
function toggleTag(tag: string) {
  const index = selectedTags.value.indexOf(tag)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tag)
  }
}

// 模拟图片上传进度
async function simulateImageUpload(): Promise<string[]> {
  if (uploadedFiles.value.length === 0) return []
  
  isUploading.value = true
  uploadProgress.value = 0
  
  return new Promise((resolve) => {
    const interval = setInterval(() => {
      uploadProgress.value += Math.random() * 20
      
      if (uploadProgress.value >= 100) {
        uploadProgress.value = 100
        clearInterval(interval)
        
        setTimeout(() => {
          isUploading.value = false
          uploadProgress.value = 0
          // 注意：图片上传功能需要后端 API 支持
          // const formData = new FormData()
          // uploadedFiles.value.forEach(file => formData.append('files', file))
          // const response = await $fetch('/api/upload/images', { method: 'POST', body: formData })
          // resolve(response.data.urls)
          
          // 暂时返回空数组，等待后端API实现
          resolve([])
        }, 500)
      }
    }, 200)
  })
}

// 发布动态
async function publishMoment() {
  if (!canPublish.value || publishing.value || isUploading.value) return
  
  try {
    // 如果有图片，先上传图片
    let imageUrls: string[] = []
    if (uploadedFiles.value.length > 0) {
      imageUrls = await simulateImageUpload()
    }
    
    publishing.value = true
    
    // 获取位置信息（可选）
    let currentCoords = coordinates.value
    if (!currentCoords) {
      try {
        currentCoords = await getCurrentPosition()
      } catch (error) {
        console.warn('无法获取位置信息，使用默认位置')
      }
    }
    
    // 准备动态数据
    const momentData = {
      content: content.value.trim(),
      description: selectedTags.value.join(' '), // 将标签作为描述
      tags: selectedTags.value,
      privacy: privacy.value,
      images: uploadedFiles.value, // 使用 UFileUpload 的文件
      latitude: currentCoords?.latitude || 39.9042,
      longitude: currentCoords?.longitude || 116.4074,
      location_name: "默认位置"
    }
    
    // 调用服务发布动态（这里简化处理，实际需要调用后端API）
    await momentsStore.createMoment(momentData)
    
    toast.success({
      title: '动态发布成功',
      description: '图片已上传，动态已发布'
    })
    
    // 清空表单
    content.value = ''
    uploadedFiles.value = []
    selectedTags.value = []
    privacy.value = 'public'
    
    // 触发成功事件
    emit('success')
    
  } catch (error: any) {
    console.error('发布动态失败:', error)
    toast.error({
      title: '发布失败',
      description: error.message || '请重试'
    })
  } finally {
    publishing.value = false
  }
}
</script>