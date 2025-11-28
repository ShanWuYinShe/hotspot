<template>
  <UModal size="2xl">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-camera" class="w-6 h-6 text-primary-500" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
              发布生活瞬间
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <form @submit.prevent="createMoment" class="space-y-6">
          <!-- 瞬间描述 -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              瞬间描述 *
            </label>
            <UTextarea
              v-model="form.description"
              placeholder="分享您此刻的心情和想法..."
              :rows="4"
              :maxlength="MOMENT_CONSTANTS.MAX_DESCRIPTION_LENGTH"
              required
            />
            <div class="flex justify-between items-center mt-1">
              <span class="text-xs text-neutral-500">
                {{ form.description.length }}/{{ MOMENT_CONSTANTS.MAX_DESCRIPTION_LENGTH }}
              </span>
            </div>
          </div>

          <!-- 图片上传 -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              图片（最多9张）
            </label>
            <div class="grid grid-cols-3 gap-3">
              <!-- 已上传的图片 -->
              <div
                v-for="(image, index) in form.images"
                :key="index"
                class="relative aspect-square bg-neutral-100 dark:bg-neutral-800 rounded-lg overflow-hidden"
              >
                <img
                  :src="image"
                  :alt="`图片 ${index + 1}`"
                  class="w-full h-full object-cover"
                />
                <UButton
                  variant="solid"
                  color="red"
                  size="xs"
                  icon="i-heroicons-x-mark"
                  class="absolute top-1 right-1"
                  @click="removeImage(index)"
                />
              </div>
              
              <!-- 上传按钮 -->
              <div
                v-if="form.images.length < MOMENT_CONSTANTS.MAX_IMAGES"
                class="aspect-square border-2 border-dashed border-neutral-300 dark:border-neutral-600 rounded-lg flex items-center justify-center cursor-pointer hover:border-primary-500 transition-colors"
                @click="triggerImageUpload"
              >
                <div class="text-center">
                  <UIcon name="i-heroicons-camera" class="w-6 h-6 text-neutral-400 mx-auto mb-1" />
                  <span class="text-xs text-neutral-500">添加图片</span>
                </div>
              </div>
            </div>
            <input
              ref="imageInput"
              type="file"
              accept="image/*"
              multiple
              class="hidden"
              @change="handleImageUpload"
            />
          </div>

          <!-- 位置信息 -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              位置信息 *
            </label>
            <div class="space-y-3">
              <UInput
                v-model="form.location_name"
                placeholder="输入位置名称"
                icon="i-heroicons-map-pin"
                required
              />
              <div class="flex gap-3">
                <UInput
                  v-model.number="form.latitude"
                  placeholder="纬度"
                  type="number"
                  step="any"
                  required
                  class="flex-1"
                />
                <UInput
                  v-model.number="form.longitude"
                  placeholder="经度"
                  type="number"
                  step="any"
                  required
                  class="flex-1"
                />
                <UButton
                  variant="outline"
                  icon="i-heroicons-map-pin"
                  @click="getCurrentLocation"
                  :loading="gettingLocation"
                  size="sm"
                >
                  获取位置
                </UButton>
              </div>
            </div>
          </div>

          <!-- 标签 -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              标签（最多10个）
            </label>
            <div class="flex flex-wrap gap-2 mb-3">
              <UBadge
                v-for="(tag, index) in form.tags"
                :key="index"
                color="primary"
                variant="soft"
                class="cursor-pointer"
                @click="removeTag(index)"
              >
                {{ tag }}
                <UIcon name="i-heroicons-x-mark" class="w-3 h-3 ml-1" />
              </UBadge>
            </div>
            <div class="flex gap-2">
              <UInput
                v-model="newTag"
                placeholder="输入标签"
                @keyup.enter="addTag"
                class="flex-1"
                size="sm"
              />
              <UButton
                variant="outline"
                @click="addTag"
                :disabled="!newTag.trim() || form.tags.length >= MOMENT_CONSTANTS.MAX_TAGS"
                size="sm"
              >
                添加
              </UButton>
            </div>
          </div>

          <!-- 隐私设置 -->
          <div>
            <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
              隐私设置
            </label>
            <USelectMenu
              v-model="form.visibility"
              :options="visibilityOptions"
              option-attribute="label"
              value-attribute="value"
            />
          </div>

          <!-- 操作按钮 -->
          <div class="flex justify-end gap-3 pt-4 border-t border-neutral-200 dark:border-neutral-700">
            <UButton
              variant="outline"
              @click="saveDraft"
              :disabled="creating"
              size="sm"
            >
              保存草稿
            </UButton>
            <UButton
              variant="outline"
              @click="handleClose"
              :disabled="creating"
            >
              取消
            </UButton>
            <UButton
              type="submit"
              :loading="creating"
              :disabled="!isFormValid"
            >
              发布瞬间
            </UButton>
          </div>
        </form>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { MOMENT_CONSTANTS, DEFAULTS } from '~/config/defaults'
import { useMomentsService } from '~/services/moments.service'
import { useUploadService } from '~/services/upload.service'
import { useGeocodingService } from '~/services/geocoding.service'

interface Props {
  onSuccess?: (momentId: string) => void
  onCancel?: () => void
  onClose?: () => void
}

const props = defineProps<Props>()

// 状态管理
const creating = ref(false)
const gettingLocation = ref(false)
const isSubmitting = ref(false)
const newTag = ref('')
const imageInput = ref<HTMLInputElement>()

// 表单数据
const form = ref({
  description: '',
  images: [] as string[],
  latitude: null as number | null,
  longitude: null as number | null,
  location_name: '',
  tags: [] as string[],
  visibility: 'public'
})

// 隐私选项
const visibilityOptions = [
  { value: 'public', label: '公开' },
  { value: 'friends', label: '仅好友可见' },
  { value: 'private', label: '仅自己可见' }
]

// 计算属性
const isFormValid = computed(() => {
  return form.value.description.trim() &&
         form.value.latitude !== null &&
         form.value.longitude !== null &&
         form.value.location_name.trim()
})

// 工具函数
const toast = useToast()
const locationStore = useLocationStore()

// Toast 辅助函数
const success = (title: string, description?: string) => {
  toast.add({ title, description, color: 'success' })
}
const error = (title: string, description?: string) => {
  toast.add({ title, description, color: 'error' })
}

// 获取当前位置或默认位置
const getCurrentLocationOrDefault = async (): Promise<{ latitude: number; longitude: number }> => {
  // 先尝试从 store 获取
  if (locationStore.coordinates) {
    return locationStore.coordinates
  }
  
  // 尝试获取浏览器位置
  try {
    await locationStore.getCurrentPosition()
    if (locationStore.coordinates) {
      return locationStore.coordinates
    }
  } catch (e) {
    console.warn('获取位置失败，使用默认位置')
  }
  
  // 返回系统默认位置（南天门）
  return {
    latitude: DEFAULTS.latitude,
    longitude: DEFAULTS.longitude
  }
}

// 方法
const getCurrentLocation = async () => {
  gettingLocation.value = true
  try {
    const location = await getCurrentLocationOrDefault()
    form.value.latitude = location.latitude
    form.value.longitude = location.longitude
    
    // 使用地理编码服务获取位置名称
    const geocodingService = useGeocodingService()
    form.value.location_name = await geocodingService.getLocationName(
      location.latitude,
      location.longitude
    )
    
    success('位置获取成功')
  } catch (err) {
    console.error('获取位置失败:', err)
    error('位置获取失败', '请手动输入位置信息')
  } finally {
    gettingLocation.value = false
  }
}

const triggerImageUpload = () => {
  imageInput.value?.click()
}

const handleImageUpload = async (event: Event) => {
  const files = (event.target as HTMLInputElement).files
  if (!files) return
  
  const maxFiles = 9 - form.value.images.length
  const filesToUpload = Array.from(files).slice(0, maxFiles)
  
  isSubmitting.value = true
  try {
    const uploadService = useUploadService()
    
    // 批量上传图片
    for (const file of filesToUpload) {
      try {
        const result = await uploadService.uploadFile(file, 'moment_image')
        form.value.images.push(result.url)
      } catch (err) {
        console.error('图片上传失败:', err)
        error(`图片 ${file.name} 上传失败`)
      }
    }
  } finally {
    isSubmitting.value = false
  }
  
  // 清空input
  if (imageInput.value) {
    imageInput.value.value = ''
  }
}

const removeImage = (index: number) => {
  form.value.images.splice(index, 1)
}

const addTag = () => {
  const tag = newTag.value.trim()
  if (!tag || form.value.tags.includes(tag) || form.value.tags.length >= MOMENT_CONSTANTS.MAX_TAGS) {
    return
  }
  
  form.value.tags.push(tag)
  newTag.value = ''
}

const removeTag = (index: number) => {
  form.value.tags.splice(index, 1)
}

const saveDraft = () => {
  localStorage.setItem(MOMENT_CONSTANTS.DRAFT_STORAGE_KEY, JSON.stringify(form.value))
  success('草稿已保存')
}

const createMoment = async () => {
  if (!isFormValid.value) return
  
  creating.value = true
  try {
    const momentService = useMomentsService()
    const response = await momentService.createMoment({
      description: form.value.description,
      location: {
        latitude: form.value.latitude!,
        longitude: form.value.longitude!,
        name: form.value.location_name
      },
      tags: form.value.tags,
      images: form.value.images
    })
    
    success('瞬间发布成功')
    
    // 清除草稿
    localStorage.removeItem(MOMENT_CONSTANTS.DRAFT_STORAGE_KEY)
    
    // 调用成功回调
    props.onSuccess?.(response.id)
    
    // 关闭Modal
    handleClose()
  } catch (err: any) {
    console.error('发布瞬间失败:', err)
    error('发布失败', err?.message || '请检查网络连接后重试')
  } finally {
    creating.value = false
  }
}

const handleClose = () => {
  props.onClose?.()
}

// 页面加载时恢复草稿
onMounted(() => {
  const draft = localStorage.getItem(MOMENT_CONSTANTS.DRAFT_STORAGE_KEY)
  if (draft) {
    try {
      const draftData = JSON.parse(draft)
      Object.assign(form.value, draftData)
    } catch (err) {
      console.error('恢复草稿失败:', err)
    }
  }
})
</script>