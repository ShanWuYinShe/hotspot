<template>
  <div v-if="show" class="post-dialog-overlay" @click="close">
    <div class="post-dialog-content" @click.stop>
      <!-- 头部 -->
      <div class="post-dialog-header">
        <h3 class="post-dialog-title">发布动态</h3>
        <button @click="close" class="post-dialog-close">
          <Icon name="mdi:close" size="24" />
        </button>
      </div>

      <!-- 内容区域 -->
      <div class="post-dialog-body">
        <!-- 用户信息 -->
        <div class="post-user-info">
          <div class="post-user-avatar">
            U
          </div>
          <div class="post-user-details">
            <div class="post-user-name">当前用户</div>
            <div class="post-user-visibility">公开</div>
          </div>
        </div>

        <!-- 内容输入 -->
        <textarea v-model="content" placeholder="分享你的想法..." class="post-content-input"></textarea>

        <!-- 图片预览 -->
        <div v-if="selectedImages.length > 0" class="post-images-preview">
          <div class="images-grid">
            <div v-for="(image, index) in selectedImages" :key="index" class="image-item">
              <img :src="image.preview" />
              <button @click="removeImage(index)" class="remove-btn">
                <Icon name="mdi:close" size="14" />
              </button>
            </div>
          </div>
        </div>

        <!-- 位置信息 -->
        <div v-if="selectedLocation" class="post-location-info">
          <div class="location-content">
            <Icon name="mdi:map-marker-outline" size="16" />
            <span class="location-text">{{ selectedLocation }}</span>
            <button @click="selectedLocation = ''" class="remove-location">
              <Icon name="mdi:close" size="16" />
            </button>
          </div>
        </div>

        <!-- 心情选择 -->
        <div v-if="selectedMood" class="post-mood-info">
          <div class="mood-tag">
            <Icon :name="getMoodIcon(selectedMood)" size="14" class="mood-icon" />
            {{ selectedMood }}
            <button @click="selectedMood = ''" class="remove-mood">
              <Icon name="mdi:close" size="12" />
            </button>
          </div>
        </div>

        <!-- 功能按钮 -->
        <div class="post-actions">
          <div class="action-buttons">
            <input ref="imageInput" type="file" multiple accept="image/*" class="hidden-input"
              @change="handleImageSelect">
            <button @click="imageInput?.click()" class="action-btn photo-btn">
              <Icon name="mdi:image-outline" size="20" />
              <span>照片</span>
            </button>

            <button @click="showLocationPicker = true" class="action-btn location-btn">
              <Icon name="mdi:map-marker-outline" size="20" />
              <span>位置</span>
            </button>

            <button @click="showMoodPicker = true" class="action-btn mood-btn">
              <Icon name="mdi:emoticon-outline" size="20" />
              <span>心情</span>
            </button>
          </div>

          <div class="publish-actions">
            <span class="char-count" :class="{ warning: content.length > 200, error: content.length > 280 }">
              {{ content.length }}/280
            </span>
            <button @click="submitPost" :disabled="!content.trim() && selectedImages.length === 0"
              :class="['publish-btn', { loading: isSubmitting }]">
              <span>发布</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 位置选择器 -->
    <div v-if="showLocationPicker" class="post-dialog-overlay" @click="showLocationPicker = false">
      <div class="location-picker" @click.stop>
        <div class="post-dialog-header">
          <h4 class="post-dialog-title">选择位置</h4>
        </div>
        <div class="post-dialog-body">
          <input v-model="locationSearch" placeholder="搜索位置..." class="location-input" />
          <div class="location-list">
            <div v-for="location in nearbyLocations" :key="location" @click="selectLocation(location)"
              class="location-option">
              {{ location }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 心情选择器 -->
    <div v-if="showMoodPicker" class="post-dialog-overlay" @click="showMoodPicker = false">
      <div class="mood-picker" @click.stop>
        <div class="post-dialog-header">
          <h4 class="post-dialog-title">选择心情</h4>
        </div>
        <div class="post-dialog-body">
          <div class="mood-grid">
            <div v-for="mood in moods" :key="mood" @click="selectMood(mood)"
              :class="['mood-option', { selected: selectedMood === mood }]">
              <div class="mood-emoji">{{ getMoodEmoji(mood) }}</div>
              <span class="mood-label">{{ mood }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'submit', post: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// Template refs
const imageInput = ref<HTMLInputElement>()

// Form data
const content = ref('')
const selectedImages = ref<any[]>([])
const selectedLocation = ref('')
const selectedMood = ref('')
const showLocationPicker = ref(false)
const showMoodPicker = ref(false)
const locationSearch = ref('')
const isSubmitting = ref(false)

const nearbyLocations = [
  '当前位置',
  '北京市朝阳区',
  '三里屯',
  '国贸CBD',
  '望京',
  '其他位置...'
]

const moods = ['开心', '兴奋', '平静', '难过', '愤怒', '惊喜']

const close = () => {
  emit('close')
  // 重置表单
  content.value = ''
  selectedImages.value = []
  selectedLocation.value = ''
  selectedMood.value = ''
  showLocationPicker.value = false
  showMoodPicker.value = false
  locationSearch.value = ''
  isSubmitting.value = false
}

const handleImageSelect = (event: Event) => {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (!files) return

  Array.from(files).forEach(file => {
    const reader = new FileReader()
    reader.onload = (e) => {
      selectedImages.value.push({
        file,
        preview: e.target?.result as string
      })
    }
    reader.readAsDataURL(file)
  })
}

const removeImage = (index: number) => {
  selectedImages.value.splice(index, 1)
}

const selectLocation = (location: string) => {
  selectedLocation.value = location
  showLocationPicker.value = false
}

const selectMood = (mood: string) => {
  selectedMood.value = mood
  showMoodPicker.value = false
}

const submitPost = async () => {
  if ((!content.value.trim() && selectedImages.value.length === 0) || isSubmitting.value) {
    return
  }

  isSubmitting.value = true

  try {
    const postData = {
      content: content.value,
      images: selectedImages.value,
      location: selectedLocation.value,
      mood: selectedMood.value,
      created_at: new Date().toISOString()
    }

    emit('submit', postData)
    close()
  } catch (error) {
    console.error('发布失败:', error)
    isSubmitting.value = false
  }
}

const getMoodIcon = (mood: string) => {
  const moodIcons: Record<string, string> = {
    '开心': 'mdi:emoticon-happy-outline',
    '难过': 'mdi:emoticon-sad-outline',
    '兴奋': 'mdi:emoticon-excited-outline',
    '平静': 'mdi:emoticon-neutral-outline',
    '愤怒': 'mdi:emoticon-angry-outline',
    '惊喜': 'mdi:emoticon-surprised-outline'
  }
  return moodIcons[mood] || 'mdi:emoticon-outline'
}

const getMoodEmoji = (mood: string) => {
  const moodEmojis: Record<string, string> = {
    '开心': '😊',
    '难过': '😢',
    '兴奋': '🤩',
    '平静': '😌',
    '愤怒': '😠',
    '惊喜': '😮'
  }
  return moodEmojis[mood] || '😊'
}
</script>

<style scoped>
</style>
