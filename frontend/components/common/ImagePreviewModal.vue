<template>
  <NDialog
    v-model="visible"
    title="图片预览"
    width="90%"
    :show-close="true"
    :close-on-click-modal="true"
    class="image-preview-modal"
  >
    <div class="image-preview-container">
      <div v-if="images.length > 0" class="image-gallery">
        <!-- 主图显示 -->
        <div class="main-image">
          <img 
            :src="images[currentIndex]" 
            :alt="`图片 ${currentIndex + 1}`"
            class="preview-image"
            @load="handleImageLoad"
            @error="handleImageError"
          />
          
          <!-- 导航按钮 -->
          <div v-if="images.length > 1" class="image-nav">
            <NButton
              v-if="currentIndex > 0"
              @click="previousImage"
              class="nav-btn prev-btn"
              circle
              size="large"
            >
              <Icon name="mdi:arrow-left" size="16" />
            </NButton>
            
            <NButton
              v-if="currentIndex < images.length - 1"
              @click="nextImage"
              class="nav-btn next-btn"
              circle
              size="large"
            >
              <Icon name="mdi:arrow-right" size="16" />
            </NButton>
          </div>
          
          <!-- 图片计数 -->
          <div v-if="images.length > 1" class="image-counter">
            {{ currentIndex + 1 }} / {{ images.length }}
          </div>
        </div>
        
        <!-- 缩略图列表 -->
        <div v-if="images.length > 1" class="thumbnails">
          <div
            v-for="(image, index) in images"
            :key="index"
            :class="['thumbnail', { active: index === currentIndex }]"
            @click="currentIndex = index"
          >
            <img :src="image" :alt="`缩略图 ${index + 1}`" />
          </div>
        </div>
      </div>
      
      <!-- 图片操作 -->
      <div class="image-actions">
        <NButton @click="downloadImage" type="primary" plain>
          <Icon name="mdi:download" size="16" />
          下载
        </NButton>
        <NButton @click="shareImage" type="success" plain>
          <Icon name="mdi:share" size="16" />
          分享
        </NButton>
      </div>
    </div>
  </NDialog>
</template>

<script setup lang="ts">

import { ref, computed, watch } from 'vue'
import { NDialog, NButton, useMessage } from 'naive-ui'

// API实例
const message = useMessage()

interface Props {
  modelValue: boolean
  images: string[]
  initialIndex?: number
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'download', image: string): void
  (e: 'share', image: string): void
}

const props = withDefaults(defineProps<Props>(), {
  initialIndex: 0
})

const emit = defineEmits<Emits>()

// 响应式数据
const currentIndex = ref(props.initialIndex)

// 计算属性
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 监听初始索引变化
watch(() => props.initialIndex, (newIndex) => {
  currentIndex.value = newIndex
})

// 方法
const previousImage = () => {
  if (currentIndex.value > 0) {
    currentIndex.value--
  }
}

const nextImage = () => {
  if (currentIndex.value < props.images.length - 1) {
    currentIndex.value++
  }
}

const handleImageLoad = () => {
  // 图片加载成功
}

const handleImageError = () => {
  message.error('图片加载失败')
}

const downloadImage = async () => {
  try {
    const currentImage = props.images[currentIndex.value]
    // 创建一个隐藏的 a 标签来下载图片
    const link = document.createElement('a')
    link.href = currentImage
    link.download = `image-${currentIndex.value + 1}.jpg`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    
    message.success('图片下载开始')
    emit('download', currentImage)
  } catch (error) {
    message.error('下载失败')
    console.error('Download error:', error)
  }
}

const shareImage = async () => {
  try {
    const currentImage = props.images[currentIndex.value]
    
    if (navigator.share) {
      // 使用原生分享 API（如果支持）
      await navigator.share({
        title: '分享图片',
        text: '查看这张图片',
        url: currentImage
      })
    } else {
      // 复制链接到剪贴板
      await navigator.clipboard.writeText(currentImage)
      message.success('图片链接已复制到剪贴板')
    }
    
    emit('share', currentImage)
  } catch (error) {
    message.error('分享失败')
    console.error('Share error:', error)
  }
}

// 键盘事件处理
const handleKeydown = (event: KeyboardEvent) => {
  if (!visible.value) return
  
  switch (event.key) {
    case 'ArrowLeft':
      previousImage()
      break
    case 'ArrowRight':
      nextImage()
      break
    case 'Escape':
      visible.value = false
      break
  }
}

// 组件挂载时添加键盘事件监听
onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

// 组件卸载时移除键盘事件监听
onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped lang="scss">
.image-preview-modal {
  :deep(.el-dialog__body) {
    padding: 0;
  }
}

.image-preview-container {
  background: #000;
  border-radius: 8px;
  overflow: hidden;
}

.image-gallery {
  position: relative;
}

.main-image {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
  max-height: 70vh;
  background: #000;

  .preview-image {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
    border-radius: 4px;
  }
}

.image-nav {
  position: absolute;
  top: 50%;
  left: 0;
  right: 0;
  transform: translateY(-50%);
  display: flex;
  justify-content: space-between;
  padding: 0 1rem;
  pointer-events: none;

  .nav-btn {
    pointer-events: auto;
    background: rgba(0, 0, 0, 0.6) !important;
    border: none !important;
    color: white !important;
    backdrop-filter: blur(4px);
    
    &:hover {
      background: rgba(0, 0, 0, 0.8) !important;
      transform: scale(1.1);
    }
  }

  .prev-btn {
    margin-left: 0;
  }

  .next-btn {
    margin-right: 0;
  }
}

.image-counter {
  position: absolute;
  top: 1rem;
  right: 1rem;
  background: rgba(0, 0, 0, 0.6);
  color: white;
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.875rem;
  backdrop-filter: blur(4px);
}

.thumbnails {
  display: flex;
  gap: 0.5rem;
  padding: 1rem;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  overflow-x: auto;
  
  .thumbnail {
    width: 60px;
    height: 60px;
    border-radius: 4px;
    overflow: hidden;
    cursor: pointer;
    opacity: 0.6;
    transition: all 0.2s ease;
    border: 2px solid transparent;

    &.active {
      opacity: 1;
      border-color: var(--el-color-primary);
    }

    &:hover {
      opacity: 0.8;
      transform: scale(1.05);
    }

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }
}

.image-actions {
  display: flex;
  gap: 1rem;
  padding: 1rem;
  justify-content: center;
  background: rgba(0, 0, 0, 0.3);

  .el-button {
    background: rgba(255, 255, 255, 0.1) !important;
    border-color: rgba(255, 255, 255, 0.2) !important;
    color: white !important;
    backdrop-filter: blur(4px);

    &:hover {
      background: rgba(255, 255, 255, 0.2) !important;
      transform: translateY(-2px);
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .image-preview-modal {
    :deep(.el-dialog) {
      width: 95% !important;
      margin: 2.5vh auto !important;
    }
  }

  .main-image {
    min-height: 300px;
    max-height: 60vh;
  }

  .image-nav {
    padding: 0 0.5rem;
  }

  .thumbnails {
    padding: 0.5rem;
    
    .thumbnail {
      width: 50px;
      height: 50px;
    }
  }

  .image-actions {
    padding: 0.75rem;
    
    .el-button {
      font-size: 0.875rem;
    }
  }
}

// 深色模式适配
.dark {
  .image-preview-container {
    background: #1a1a1a;
  }

  .main-image {
    background: #1a1a1a;
  }
}
</style>
