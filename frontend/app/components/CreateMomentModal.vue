<template>
  <UModal v-model="isOpen">
    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-semibold">发布动态</h3>
          <UButton
            color="neutral"
            variant="ghost"
            icon="i-heroicons-x-mark"
            @click="close"
          />
        </div>
      </template>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- 内容输入 -->
        <UTextarea
          v-model="form.description"
          placeholder="分享你的精彩瞬间..."
          :rows="4"
          :maxlength="500"
          required
        />
        
        <!-- 字数统计 -->
        <div class="text-right text-sm text-gray-500">
          {{ form.description.length }}/500
        </div>

        <!-- 位置信息 -->
        <div class="space-y-2">
          <label class="text-sm font-medium">位置信息</label>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-map-pin" class="w-4 h-4 text-primary" />
            <span class="text-sm">{{ locationText }}</span>
            <UButton
              size="xs"
              variant="outline"
              @click="refreshLocation"
              :loading="locationLoading"
            >
              刷新
            </UButton>
          </div>
        </div>

        <!-- 标签输入 -->
        <div class="space-y-2">
          <label class="text-sm font-medium">添加标签</label>
          
          <!-- 热门标签快选 -->
          <div class="flex flex-wrap gap-2 mb-2">
            <UBadge
              v-for="tag in popularTags"
              :key="tag"
              :color="form.tags.includes(tag) ? 'primary' : 'neutral'"
              variant="soft"
              class="cursor-pointer text-xs"
              @click="toggleTag(tag)"
            >
              {{ tag }}
            </UBadge>
          </div>
          
          <!-- 标签输入组件 -->
          <UInputTags
            v-model="form.tags"
            placeholder="输入标签并按回车添加..."
            :max="10"
            :max-length="20"
            size="md"
            icon="i-heroicons-hashtag"
            @add-tag="handleAddTag"
            @remove-tag="handleRemoveTag"
          />
          
          <p class="text-xs text-gray-500">最多可添加10个标签，每个标签最长20个字符</p>
        </div>

        <!-- 图片上传 -->
        <div class="space-y-2">
          <label class="text-sm font-medium">添加图片（可选）</label>
          <UFileUpload
            v-model="uploadedFiles"
            accept="image/*"
            multiple
            variant="area"
            layout="grid"
            class="min-h-32"
            label="拖拽图片到此处或点击选择"
            description="支持 JPG, PNG, GIF 格式，最多9张图片"
            @change="handleImageUpload"
          />
          <p class="text-xs text-gray-500">最多可上传9张图片</p>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="close">
            取消
          </UButton>
          <UButton
            type="submit"
            :loading="submitting"
            :disabled="!canSubmit"
            @click="handleSubmit"
          >
            发布
          </UButton>
        </div>
      </template>
    </UCard>
  </UModal>
</template>

<script setup lang="ts">
import { useMomentService } from '~/services/moment.service'
import { useFeatureStatus } from '~/composables/useFeatureStatus'

interface Props {
  open?: boolean
}

interface Emits {
  'update:open': [value: boolean]
  success: [moment: any]
}

const props = withDefaults(defineProps<Props>(), {
  open: false
})

const emit = defineEmits<Emits>()

// 响应式状态
const isOpen = computed({
  get: () => props.open,
  set: (value) => emit('update:open', value)
})

// 服务和工具
const momentService = useMomentService()
const { getCurrentPosition, coordinates } = useGeolocation()
const toast = useToast()
const { executeFeature } = useFeatureStatus()

// 表单数据
const form = reactive({
  description: '',
  tags: [] as string[],
  images: [] as string[]
})

// 状态管理
const submitting = ref(false)
const locationLoading = ref(false)
const customTag = ref('')
const uploadedFiles = ref<File[]>([])

// 位置信息
const location = ref({
  latitude: 0,
  longitude: 0,
  name: '获取位置中...'
})

// 预设标签
const popularTags = ['美食', '旅行', '生活', '摄影', '音乐', '运动', '工作', '学习']

// 计算属性
const locationText = computed(() => {
  if (location.value.name === '获取位置中...') {
    return '获取位置中...'
  }
  return location.value.name || '未知位置'
})

const canSubmit = computed(() => {
  return form.description.trim().length > 0 && 
         location.value.latitude !== 0 && 
         location.value.longitude !== 0
})

// 生命周期
onMounted(() => {
  getCurrentLocation()
})

// 方法
const getCurrentLocation = async () => {
  locationLoading.value = true
  try {
    const position = await getCurrentPosition()
    location.value = {
      latitude: position.coords.latitude,
      longitude: position.coords.longitude,
      name: '当前位置' // 实际项目中可以调用地理编码API获取地址
    }
  } catch (error) {
    console.error('获取位置失败:', error)
    toast.add({
      title: '位置获取失败',
      description: '请手动刷新位置或检查位置权限',
      color: 'error'
    })
  } finally {
    locationLoading.value = false
  }
}

const refreshLocation = () => {
  getCurrentLocation()
}

const toggleTag = (tag: string) => {
  const index = form.tags.indexOf(tag)
  if (index > -1) {
    form.tags.splice(index, 1)
  } else if (form.tags.length < 10) {
    form.tags.push(tag)
  } else {
    toast.add({
      title: '标签数量限制',
      description: '最多只能添加10个标签',
      color: 'warning'
    })
  }
}

const addCustomTag = () => {
  const tag = customTag.value.trim()
  if (tag && !form.tags.includes(tag)) {
    if (form.tags.length < 10) {
      form.tags.push(tag)
      customTag.value = ''
    } else {
      toast.add({
        title: '标签数量限制',
        description: '最多只能添加10个标签',
        color: 'warning'
      })
    }
  }
}

const removeTag = (tag: string) => {
  const index = form.tags.indexOf(tag)
  if (index > -1) {
    form.tags.splice(index, 1)
  }
}

const handleImageUpload = (event: Event) => {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (!files) return

  Array.from(files).forEach(file => {
    if (form.images.length >= 9) {
      toast.add({
        title: '图片数量限制',
        description: '最多只能上传9张图片',
        color: 'warning'
      })
      return
    }

    // 简单的文件读取，实际项目中应该上传到服务器
    const reader = new FileReader()
    reader.onload = (e) => {
      if (e.target?.result) {
        form.images.push(e.target.result as string)
      }
    }
    reader.readAsDataURL(file)
  })

  // 清空input
  target.value = ''
}

const removeImage = (index: number) => {
  form.images.splice(index, 1)
}

// 标签处理函数
const handleAddTag = (tag: string) => {
  console.log('添加标签:', tag)
}

const handleRemoveTag = (tag: string) => {
  console.log('移除标签:', tag)
}

const handleSubmit = async () => {
  await executeFeature('momentCreate', async () => {
    submitting.value = true
    
    try {
      const result = await momentService.createMoment(
        location.value.latitude,
        location.value.longitude,
        form.description,
        form.tags,
        form.images
      )
      
      toast.add({
        title: '发布成功',
        description: '你的动态已成功发布',
        color: 'success'
      })
      
      emit('success', result)
      close()
      resetForm()
    } catch (error) {
      console.error('发布动态失败:', error)
      throw error // 让 executeFeature 处理错误显示
    } finally {
      submitting.value = false
    }
  })
}

const close = () => {
  isOpen.value = false
}

const resetForm = () => {
  form.description = ''
  form.tags = []
  form.images = []
  customTag.value = ''
}

// 监听弹窗关闭，重置表单
watch(isOpen, (newValue) => {
  if (!newValue) {
    resetForm()
  }
})
</script>