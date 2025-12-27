<template>
  <UModal
    :title="modalTitle"
    :description="modalDescription"
  >
    <template #body>
      <UForm
        :state="form"
        class="space-y-6"
        @submit="submitMoment"
      >
        <!-- 瞬间描述 -->
        <UFormField
          label="瞬间描述"
          name="description"
          :hint="`${form.description.length}/${MOMENT_CONSTANTS.MAX_DESCRIPTION_LENGTH}`"
          :required="true"
        >
          <UTextarea
            v-model="form.description"
            placeholder="分享您此刻的心情和想法..."
            :rows="4"
            :maxlength="MOMENT_CONSTANTS.MAX_DESCRIPTION_LENGTH"
          />
        </UFormField>

        <!-- 图片上传 -->
        <UFormField
          label="图片"
          :hint="props.mode === 'edit' ? '可选：上传后将替换原图片（最多9张）' : '最多9张'"
        >
          <UCard v-if="props.mode === 'edit' && props.moment?.images?.length" class="mb-3">
            <section class="grid grid-cols-3 gap-3">
              <article
                v-for="(image, index) in props.moment.images"
                :key="`${image}-${index}`"
                class="relative aspect-square rounded-xl overflow-hidden ring-1 ring-gray-200 dark:ring-gray-800"
              >
                <NuxtImg
                  :src="image"
                  :alt="`原图片 ${index + 1}`"
                  class="w-full h-full object-cover"
                />
              </article>
            </section>
          </UCard>
          <UCard>
            <section class="grid grid-cols-3 gap-3">
              <article
                v-for="(image, index) in form.images"
                :key="image.file_id"
                class="relative aspect-square rounded-xl overflow-hidden ring-1 ring-gray-200 dark:ring-gray-800"
              >
                <NuxtImg
                  :src="image.local_preview"
                  :alt="`图片 ${index + 1}`"
                  class="w-full h-full object-cover"
                />
                <UButton
                  type="button"
                  variant="solid"
                  color="error"
                  size="sm"
                  icon="i-lucide-x"
                  class="absolute top-1 right-1"
                  @click="removeImage(index)"
                />
              </article>

              <!-- 图片上传 (使用作用域插槽) -->
              <UFileUpload
                v-if="form.images.length < MOMENT_CONSTANTS.MAX_IMAGES"
                v-slot="{ open }"
                v-model="selectedImageFiles"
                accept="image/*"
                :multiple="true"
                @update:model-value="handleImagesSelected"
              >
                <UButton
                  type="button"
                  variant="outline"
                  class="aspect-square flex flex-col items-center justify-center gap-1"
                  :loading="uploading"
                  @click="() => open()"
                >
                  <UIcon
                    name="i-lucide-camera"
                    :size="24"
                  />
                  <span class="text-xs">添加图片</span>
                </UButton>
              </UFileUpload>
            </section>
          </UCard>
        </UFormField>

        <!-- 位置信息 -->
        <UFormField
          label="位置"
          :required="true"
        >
          <div class="flex gap-2">
            <UCard variant="subtle" class="flex-1 min-w-0">
              <div class="flex items-center gap-2 min-w-0">
                <UIcon name="i-lucide-map-pin" :size="16" />
                <span class="truncate">{{ form.location_name || '请选择位置' }}</span>
              </div>
            </UCard>
            <UButton
              type="button"
              variant="outline"
              icon="i-lucide-map"
              @click="pickLocation"
            >
              地图选点
            </UButton>
          </div>
          <div v-if="form.latitude" class="mt-1">
            <UBadge color="neutral" variant="soft" size="xs">
              {{ form.latitude.toFixed(6) }}, {{ form.longitude?.toFixed(6) }}
            </UBadge>
          </div>
        </UFormField>

        <!-- 标签 -->
        <UFormField
          label="标签"
          :hint="`最多 ${MOMENT_CONSTANTS.MAX_TAGS} 个`"
        >
          <UCard>
            <div class="space-y-3">
              <section
                v-if="form.tags.length > 0"
                class="flex flex-wrap gap-2"
              >
                <UBadge
                  v-for="(tag, index) in form.tags"
                  :key="index"
                  color="primary"
                  variant="soft"
                  class="cursor-pointer"
                  @click="removeTag(index)"
                >
                  {{ tag }}
                  <UIcon
                    name="i-lucide-x"
                    :size="14"
                    class="ml-1"
                  />
                </UBadge>
              </section>

              <section class="flex gap-2">
                <UInput
                  v-model="newTag"
                  placeholder="输入标签"
                  class="flex-1"
                  @keyup.enter="addTag"
                />
                <UButton
                  type="button"
                  variant="outline"
                  :disabled="!newTag.trim() || form.tags.length >= MOMENT_CONSTANTS.MAX_TAGS"
                  @click="addTag"
                >
                  添加
                </UButton>
              </section>
            </div>
          </UCard>
        </UFormField>
      </UForm>
    </template>

    <template #footer>
      <UButton
        type="button"
        variant="outline"
        :disabled="creating"
        @click="saveDraft"
      >
        保存草稿
      </UButton>
      <UButton
        type="button"
        variant="outline"
        :disabled="creating"
        @click="emit('close')"
      >
        取消
      </UButton>
      <UButton
        type="button"
        :loading="creating"
        :disabled="!isFormValid"
        @click="submitMoment"
      >
        {{ props.mode === 'edit' ? '保存修改' : '发布瞬间' }}
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { MOMENT_CONSTANTS } from '~/config/defaults'
import { useMomentsService } from '~/services/moments.service'
import { useUploadService } from '~/services/upload.service'
import type { MomentSearchItem } from '~/types/api'

interface Props {
  onSuccess?: (momentId: string) => void
  mode?: 'create' | 'edit'
  moment?: Pick<MomentSearchItem, 'id' | 'description' | 'tags' | 'images' | 'geo'> | null
}

const props = withDefaults(defineProps<Props>(), {
  mode: 'create',
  moment: null
})
const emit = defineEmits<{
  close: [value?: string]
}>()

const momentsStore = useMomentsStore()
const locationStore = useLocationStore()

const creating = ref(false)
const uploading = ref(false)
const newTag = ref('')
// 图片上传 (UFileUpload)
const selectedImageFiles = ref<File[] | null>(null)

// 图片数据结构（本地预览 + 服务端 file_id）
interface ImageItem {
  file_id: string;
  local_preview: string;  // 本地预览 URL（URL.createObjectURL）
}

// 初始化为空表单（水合安全）
const form = ref({
  description: '',
  images: [] as ImageItem[],
  latitude: null as number | null,
  longitude: null as number | null,
  location_name: '',
  tags: [] as string[]
})

// 客户端水合完成后从 store 恢复草稿
onMounted(() => {
  if (props.mode === 'edit' && props.moment) {
    form.value.description = props.moment.description || ''
    form.value.tags = [...(props.moment.tags || [])]
    form.value.latitude = props.moment.geo?.latitude ?? null
    form.value.longitude = props.moment.geo?.longitude ?? null
    form.value.location_name = props.moment.geo?.location_name ?? ''
    return
  }

  if (momentsStore.hasDraft) {
    form.value.description = momentsStore.draft.description
    // 草稿中的 images 是 file_id 数组，本地预览 URL 无法恢复，需重新上传
    form.value.latitude = momentsStore.draft.latitude
    form.value.longitude = momentsStore.draft.longitude
    form.value.location_name = momentsStore.draft.location_name
    form.value.tags = [...momentsStore.draft.tags]
    return
  }

  // 无草稿时默认使用当前定位（如果已获取）
  const loc = locationStore.location
  if (loc) {
    form.value.latitude = loc.latitude
    form.value.longitude = loc.longitude
    form.value.location_name = loc.locationName
  }
})

const isFormValid = computed(() => {
  return form.value.description.trim() &&
         form.value.latitude !== null &&
         form.value.longitude !== null &&
         form.value.location_name.trim()
})

const feedback = useFeedbackStore()
const modal = useModal()

const modalTitle = computed(() => (props.mode === 'edit' ? '编辑生活瞬间' : '发布生活瞬间'))
const modalDescription = computed(() => (props.mode === 'edit' ? '修改您的动态内容' : '分享您的生活瞬间给朋友'))

const pickLocation = async () => {
  await modal.showLocationPicker((loc) => {
    form.value.latitude = loc.latitude
    form.value.longitude = loc.longitude
    form.value.location_name = loc.locationName
  })
}

// UFileUpload 回调
const handleImagesSelected = async (files: File[] | null | undefined) => {
  if (!files) return

  const maxFiles = 9 - form.value.images.length
  const filesToUpload = files.slice(0, maxFiles)

  uploading.value = true
  try {
    const uploadService = useUploadService()

    for (const file of filesToUpload) {
      try {
        // 使用本地 URL 预览（客户端自己知道上传的内容）
        const localPreview = URL.createObjectURL(file)
        const result = await uploadService.uploadFile(file, 'moment_image')
        form.value.images.push({
          file_id: result.file_id,
          local_preview: localPreview
        })
      } catch (err) {
        console.error('图片上传失败:', err)
        feedback.error('图片上传失败', `图片 ${file.name} 上传失败`)
      }
    }
  } finally {
    uploading.value = false
  }

  // 重置选择
  selectedImageFiles.value = null
}

const removeImage = (index: number) => {
  const removed = form.value.images.splice(index, 1)
  // 清理本地预览 URL
  if (removed[0]?.local_preview) {
    URL.revokeObjectURL(removed[0].local_preview)
  }
}

const addTag = () => {
  const tag = newTag.value.trim()
  if (!tag || form.value.tags.includes(tag) || form.value.tags.length >= MOMENT_CONSTANTS.MAX_TAGS) return

  form.value.tags.push(tag)
  newTag.value = ''
}

const removeTag = (index: number) => form.value.tags.splice(index, 1)

const saveDraft = () => {
  momentsStore.saveDraft({
    description: form.value.description,
    images: form.value.images.map(img => img.file_id),  // 草稿只保存 file_id
    tags: form.value.tags,
    latitude: form.value.latitude,
    longitude: form.value.longitude,
    location_name: form.value.location_name
  })
  feedback.success('草稿已保存')
}

const submitMoment = async () => {
  if (!isFormValid.value) return

  creating.value = true
  try {
    const momentService = useMomentsService()
    if (props.mode === 'edit' && props.moment?.id) {
      await momentService.update({
        id: props.moment.id,
        description: form.value.description,
        tags: form.value.tags,
        location: {
          latitude: form.value.latitude!,
          longitude: form.value.longitude!,
          name: form.value.location_name
        },
        image_file_ids: form.value.images.length > 0 ? form.value.images.map(img => img.file_id) : undefined
      })
      feedback.success('保存成功', '动态已更新')
      props.onSuccess?.(props.moment.id)
      emit('close', props.moment.id)
      return
    }

    const response = await momentService.create({
      description: form.value.description,
      location: {
        latitude: form.value.latitude!,
        longitude: form.value.longitude!,
        name: form.value.location_name
      },
      tags: form.value.tags,
      image_file_ids: form.value.images.map(img => img.file_id)
    })

    feedback.success('瞬间发布成功')
    momentsStore.clearDraft()
    props.onSuccess?.(response.id)
    emit('close', response.id)
  } catch (err: unknown) {
    console.error('发布瞬间失败:', err)
    feedback.error(props.mode === 'edit' ? '保存失败' : '发布失败', getErrorMessage(err, '请检查网络连接后重试'))
  }   finally {
    creating.value = false
  }
}
</script>
