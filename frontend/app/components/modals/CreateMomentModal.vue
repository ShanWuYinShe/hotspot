<template>
  <UModal
    size="2xl"
    title="发布生活瞬间"
    description="分享您的生活瞬间给朋友"
  >
    <template #body="{ close }">
      <UForm
        :state="form"
        class="space-y-6"
        @submit="() => createMoment(close)"
      >
        <!-- 瞬间描述 -->
        <UFormField
          label="瞬间描述"
          name="description"
          :hint="`${form.description.length}/${MOMENT_CONSTANTS.MAX_DESCRIPTION_LENGTH}`"
          required
        >
          <UTextarea
            v-model="form.description"
            placeholder="分享您此刻的心情和想法..."
            :rows="4"
            :maxlength="MOMENT_CONSTANTS.MAX_DESCRIPTION_LENGTH"
            class="w-full"
          />
        </UFormField>

        <!-- 图片上传 -->
        <UFormField
          label="图片"
          hint="最多9张"
        >
          <UCard
            :ui="{ body: 'p-4' }"
            variant="subtle"
          >
            <section class="grid grid-cols-3 gap-3">
              <article
                v-for="(image, index) in form.images"
                :key="image.file_id"
                class="relative aspect-square rounded-lg overflow-hidden"
              >
                <NuxtImg
                  :src="image.local_preview"
                  :alt="`图片 ${index + 1}`"
                  class="w-full h-full object-cover"
                />
                <UButton
                  variant="solid"
                  color="error"
                  size="xs"
                  icon="i-heroicons-x-mark"
                  class="absolute top-1 right-1"
                  @click="removeImage(index)"
                />
              </article>

              <UButton
                v-if="form.images.length < MOMENT_CONSTANTS.MAX_IMAGES"
                variant="outline"
                class="aspect-square flex flex-col items-center justify-center gap-1"
                :loading="uploading"
                @click="triggerImageUpload"
              >
                <UIcon
                  name="i-heroicons-camera"
                  class="icon-lg"
                />
                <span class="text-xs">添加图片</span>
              </UButton>
            </section>
          </UCard>
          <UFileUpload
            v-slot="{ open }"
            v-model="selectedImageFiles"
            accept="image/*"
            multiple
            class="hidden"
            @update:model-value="handleImagesSelected"
          >
            <button ref="imageUploadTrigger" class="hidden" @click="open()" />
          </UFileUpload>
        </UFormField>

        <!-- 位置信息 -->
        <UFormField
          label="位置"
          required
        >
          <div class="flex gap-2">
            <UInput
              v-model="form.location_name"
              placeholder="点击获取当前位置"
              icon="i-heroicons-map-pin"
              class="flex-1"
              readonly
              required
            />
            <UButton
              variant="outline"
              icon="i-heroicons-map-pin"
              :loading="gettingLocation"
              @click="getCurrentLocation"
            >
              {{ form.latitude ? '重新获取' : '获取位置' }}
            </UButton>
          </div>
          <p v-if="form.latitude" class="text-xs text-muted mt-1">
            位置已获取
          </p>
        </UFormField>

        <!-- 标签 -->
        <UFormField
          label="标签"
          hint="最多10个"
        >
          <UCard
            :ui="{ body: 'p-4 space-y-3' }"
            variant="subtle"
          >
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
                  name="i-heroicons-x-mark"
                  class="icon-xs ml-1"
                />
              </UBadge>
            </section>

            <section class="flex gap-2">
              <UInput
                v-model="newTag"
                placeholder="输入标签"
                size="sm"
                class="flex-1"
                @keyup.enter="addTag"
              />
              <UButton
                variant="outline"
                size="sm"
                :disabled="!newTag.trim() || form.tags.length >= MOMENT_CONSTANTS.MAX_TAGS"
                @click="addTag"
              >
                添加
              </UButton>
            </section>
          </UCard>
        </UFormField>

        <!-- 隐私设置 -->
        <UFormField label="隐私设置">
          <USelect
            v-model="form.visibility"
            :items="visibilityOptions"
          />
        </UFormField>
      </UForm>
    </template>

    <template #footer="{ close }">
      <UButton
        variant="outline"
        size="sm"
        :disabled="creating"
        @click="saveDraft"
      >
        保存草稿
      </UButton>
      <UButton
        variant="outline"
        :disabled="creating"
        @click="close"
      >
        取消
      </UButton>
      <UButton
        :loading="creating"
        :disabled="!isFormValid"
        @click="() => createMoment(close)"
      >
        发布瞬间
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { MOMENT_CONSTANTS } from '~/config/defaults'
import { useMomentsService } from '~/services/moments.service'
import { useUploadService } from '~/services/upload.service'
import { geocoding } from '~/utils/geocoding'
import { getErrorMessage } from '~/utils/error'

interface Props {
  onSuccess?: (momentId: string) => void
}

const props = defineProps<Props>()

const momentsStore = useMomentsStore()

const creating = ref(false)
const gettingLocation = ref(false)
const uploading = ref(false)
const newTag = ref('')
// 图片上传 (UFileUpload)
const selectedImageFiles = ref<File[] | null>(null)
const imageUploadTrigger = ref<HTMLButtonElement | null>(null)

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
  tags: [] as string[],
  visibility: 'public'
})

// 客户端水合完成后从 store 恢复草稿
onMounted(() => {
  if (momentsStore.hasDraft) {
    form.value.description = momentsStore.draft.description
    // 草稿中的 images 是 file_id 数组，本地预览 URL 无法恢复，需重新上传
    form.value.latitude = momentsStore.draft.latitude
    form.value.longitude = momentsStore.draft.longitude
    form.value.location_name = momentsStore.draft.location_name
    form.value.tags = [...momentsStore.draft.tags]
  }
})

const visibilityOptions = [
  { value: 'public', label: '公开' },
  { value: 'friends', label: '仅好友可见' },
  { value: 'private', label: '仅自己可见' }
]

const isFormValid = computed(() => {
  return form.value.description.trim() &&
         form.value.latitude !== null &&
         form.value.longitude !== null &&
         form.value.location_name.trim()
})

const toast = useToast()
const geolocation = useGeolocation()

const getCurrentLocation = async () => {
  gettingLocation.value = true
  try {
    const position = await geolocation.getPosition()
    if (!position) {
      gettingLocation.value = false
      return
    }

    form.value.latitude = position.coords.latitude
    form.value.longitude = position.coords.longitude

    form.value.location_name = await geocoding.getLocationName(
      position.coords.latitude,
      position.coords.longitude
    )

    toast.add({ color: 'success', title: '位置获取成功' })
  } catch (err) {
    console.error('获取位置失败:', err)
    toast.add({ color: 'error', title: '位置获取失败', description: '请手动输入位置信息' })
  } finally {
    gettingLocation.value = false
  }
}

const triggerImageUpload = () => imageUploadTrigger.value?.click()

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
        toast.add({ color: 'error', title: `图片 ${file.name} 上传失败` })
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
  toast.add({ color: 'success', title: '草稿已保存' })
}

const createMoment = async (close: () => void) => {
  if (!isFormValid.value) return

  creating.value = true
  try {
    const momentService = useMomentsService()
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

    toast.add({ color: 'success', title: '瞬间发布成功' })
    momentsStore.clearDraft()
    props.onSuccess?.(response.id)
    close()
  } catch (err: unknown) {
    console.error('发布瞬间失败:', err)
    toast.add({ color: 'error', title: '发布失败', description: getErrorMessage(err, '请检查网络连接后重试') })
  }   finally {
    creating.value = false
  }
}
</script>
