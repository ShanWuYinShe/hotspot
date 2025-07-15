<!-- 图片上传组件 -->
<template>
  <div class="image-uploader">
    <NUpload
      ref="uploadRef"
      :action="uploadAction"
      :headers="uploadHeaders"
      :show-file-list="false"
      :before-upload="beforeUpload"
      :on-success="handleSuccess"
      :on-error="handleError"
      :accept="accept"
      :multiple="multiple"
      :disabled="disabled"
      class="upload-component"
    >
      <slot name="trigger">
        <NButton 
          :type="buttonType" 
          :size="buttonSize"
          :icon="buttonIcon"
          :loading="uploading"
          :disabled="disabled"
        >
          {{ uploading ? uploadingText : buttonText }}
        </NButton>
      </slot>
    </NUpload>

    <!-- 图片预览 -->
    <div v-if="previewImages.length > 0" class="preview-container">
      <div class="preview-grid">
        <div
          v-for="(image, index) in previewImages"
          :key="index"
          class="preview-item"
        >
          <NImage
            :src="image.url"
            :preview-src-list="previewImages.map(img => img.url)"
            :initial-index="index"
            fit="cover"
            class="preview-image"
          />
          <div class="preview-actions">
            <NButton
              size="small"
              type="error"
              circle
              @click="removeImage(index)"
              :icon="Delete"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 上传进度 -->
    <NProgress
      v-if="uploadProgress > 0 && uploadProgress < 100"
      :percentage="uploadProgress"
      :show-text="false"
      class="upload-progress"
    />
  </div>
</template>

<script setup lang="ts">
import { NUpload, NButton, NImage, NProgress, useMessage } from 'naive-ui'
import { TrashOutline as Delete, CloudUploadOutline as Upload, Image as Picture } from '@vicons/ionicons5'
import { extractErrorMessage } from '~/utils/error-handler'

// API实例
const message = useMessage()

interface ImageFile {
  url: string
  name: string
  size?: number
  file?: File
}

interface Props {
  // 上传地址
  action?: string
  // 请求头
  headers?: Record<string, string>
  // 接受的文件类型
  accept?: string
  // 是否多选
  multiple?: boolean
  // 最大文件数量
  maxCount?: number
  // 最大文件大小 (MB)
  maxSize?: number
  // 按钮文字
  buttonText?: string
  // 上传中文字
  uploadingText?: string
  // 按钮类型
  buttonType?: 'primary' | 'success' | 'warning' | 'error' | 'info' | 'default'
  // 按钮大小
  buttonSize?: 'large' | 'medium' | 'small'
  // 按钮图标
  buttonIcon?: any
  // 是否禁用
  disabled?: boolean
  // 初始图片列表
  modelValue?: ImageFile[]
}

const props = withDefaults(defineProps<Props>(), {
  action: '/api/v1/upload/image',
  accept: 'image/*',
  multiple: false,
  maxCount: 9,
  maxSize: 10,
  buttonText: '选择图片',
  uploadingText: '上传中...',
  buttonType: 'primary',
  buttonSize: 'medium',
  buttonIcon: Picture
})

const emit = defineEmits<{
  'update:modelValue': [images: ImageFile[]]
  'success': [response: any, file: File]
  'error': [error: any, file: File]
  'progress': [percentage: number]
}>()

// 状态
const uploading = ref(false)
const uploadProgress = ref(0)
const uploadRef = ref()

// 预览图片列表
const previewImages = ref<ImageFile[]>(props.modelValue || [])

// 上传配置
const uploadAction = computed(() => {
  const config = useRuntimeConfig()
  return config.public.apiBase + '/upload/image'
})

const uploadHeaders = computed(() => {
  const authStore = useAuthStore()
  return {
    'Authorization': `Bearer ${authStore.token}`,
    ...props.headers
  }
})

// 监听外部数据变化
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    previewImages.value = [...newVal]
  }
}, { immediate: true })

// 上传前检查
const beforeUpload = (file: File) => {
  // 检查文件类型
  if (!file.type.startsWith('image/')) {
    message.error('只能上传图片文件')
    return false
  }

  // 检查文件大小
  const isLtMaxSize = file.size / 1024 / 1024 < props.maxSize
  if (!isLtMaxSize) {
    message.error(`图片大小不能超过 ${props.maxSize}MB`)
    return false
  }

  // 检查数量限制
  if (previewImages.value.length >= props.maxCount) {
    message.error(`最多只能上传 ${props.maxCount} 张图片`)
    return false
  }

  uploading.value = true
  uploadProgress.value = 0
  return true
}

// 上传成功
const handleSuccess = (response: any, file: any) => {
  uploading.value = false
  uploadProgress.value = 100

  if (response.success && response.data?.url) {
    const newImage: ImageFile = {
      url: response.data.url,
      name: file.name || 'image',
      size: file.size,
      file: file
    }

    previewImages.value.push(newImage)
    emit('update:modelValue', [...previewImages.value])
    emit('success', response, file)
    
    message.success('图片上传成功')
  } else {
    const errorMessage = extractErrorMessage(response, '上传失败')
    message.error(errorMessage)
  }

  // 重置进度
  setTimeout(() => {
    uploadProgress.value = 0
  }, 1000)
}

// 上传失败
const handleError = (error: any) => {
  uploading.value = false
  uploadProgress.value = 0
  
  console.error('图片上传失败:', error)
  message.error('图片上传失败')
  
  emit('error', error, error.file)
}

// 移除图片
const removeImage = (index: number) => {
  previewImages.value.splice(index, 1)
  emit('update:modelValue', [...previewImages.value])
}

// 清空图片
const clearImages = () => {
  previewImages.value = []
  emit('update:modelValue', [])
}

// 手动触发上传
const triggerUpload = () => {
  uploadRef.value?.submit()
}

// 暴露方法
defineExpose({
  clearImages,
  triggerUpload
})
</script>

<style scoped lang="scss">
.image-uploader {
  .upload-component {
    width: 100%;
  }

  .preview-container {
    margin-top: 1rem;
  }

  .preview-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
    gap: 0.5rem;
  }

  .preview-item {
    position: relative;
    aspect-ratio: 1;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid var(--el-border-color-light);

    .preview-image {
      width: 100%;
      height: 100%;
    }

    .preview-actions {
      position: absolute;
      top: 4px;
      right: 4px;
      opacity: 0;
      transition: opacity 0.2s ease;
    }

    &:hover .preview-actions {
      opacity: 1;
    }
  }

  .upload-progress {
    margin-top: 0.5rem;
  }
}

@media (max-width: 768px) {
  .image-uploader {
    .preview-grid {
      grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
    }
  }
}
</style>
