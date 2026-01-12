<template>
  <UModal
    title="创建群组"
    description="填写群组信息，并在地图上选一个点作为群组位置"
    size="lg"
  >
    <template #body>
      <UForm
        :schema="schema"
        :state="form"
        class="space-y-6"
        @submit="createGroup"
      >
        <UFormField label="群组名称" name="name" :required="true">
          <UInput
            v-model="form.name"
            placeholder="输入群组名称"
            :maxlength="50"
            icon="i-lucide-users-round"
          />
        </UFormField>

        <UFormField
          label="群组描述"
          name="description"
          :hint="`${(form.description || '').length}/200`"
        >
          <UTextarea
            v-model="form.description"
            placeholder="描述群组的主题和目的..."
            :rows="3"
            :maxlength="200"
            autoresize
          />
        </UFormField>

        <UFormField
          label="群组位置"
          name="location_name"
          :required="true"
          :hint="locationHint"
        >
          <!-- 用“选择控件”表达只读位置（而不是输入框），点击进入地图选点 -->
          <UButton
            type="button"
            variant="soft"
            color="neutral"
            icon="i-lucide-map-pin"
            trailing-icon="i-lucide-chevron-right"
            :block="true"
            class="justify-between"
            @click="pickLocation"
          >
            <span class="truncate">
              {{ form.location_name || '请选择位置' }}
            </span>
          </UButton>
        </UFormField>

        <UAccordion
          :items="advancedItems"
          type="single"
          collapsible
          variant="soft"
        >
          <template #advanced>
            <div class="space-y-5 px-1 py-3">
              <UFormField
                label="加入方式"
                name="join_method"
                hint="开放加入：任何人可直接加入；需要审批：管理员审核；仅邀请：只能通过邀请加入"
              >
                <USelectMenu
                  v-model="form.join_method"
                  :items="joinMethodItems"
                  value-key="value"
                  label-key="label"
                />
              </UFormField>

              <UFormField
                name="is_searchable"
                class="flex items-start justify-between gap-4"
              >
                <template #label>
                  <div class="font-semibold">允许被搜索</div>
                </template>
                <template #description>
                  关闭后，群组不会出现在搜索/附近发现中
                </template>
                <USwitch v-model="form.is_searchable" />
              </UFormField>

              <UFormField label="最大成员数" name="max_members">
                <UInput
                  v-model.number="form.max_members"
                  type="number"
                  :min="GROUP_CONSTANTS.MIN_MEMBERS"
                  :max="GROUP_CONSTANTS.MAX_MEMBERS_LIMIT"
                  :placeholder="GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT.toString()"
                />
              </UFormField>

              <UFormField label="群组密码（可选）" name="password" hint="设置后，直接加入需要输入密码">
                <UInput
                  v-model="form.password"
                  type="password"
                  placeholder="不设置则为公开加入"
                  icon="i-lucide-lock"
                />
              </UFormField>
            </div>
          </template>
        </UAccordion>
      </UForm>
    </template>

    <template #footer>
      <div class="flex justify-end gap-3 w-full">
        <UButton
          type="button"
          variant="ghost"
          color="neutral"
          :disabled="creating"
          @click="emit('close')"
        >
          取消
        </UButton>
        <UButton
          type="button"
          :loading="creating"
          :disabled="!canSubmit"
          @click="createGroup"
        >
          创建群组
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { z } from 'zod'
import { GROUP_CONSTANTS } from '~/config/defaults'
import { useGroupsService } from '~/services/groups.service'

interface Props {
  onSuccess?: (groupId: string) => void
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: [value?: string]
}>()

type JoinMethod = 'open' | 'approval' | 'invite_only'

const joinMethodItems: Array<{ label: string; value: JoinMethod }> = [
  { label: '开放加入', value: 'open' },
  { label: '需要审批', value: 'approval' },
  { label: '仅邀请', value: 'invite_only' }
]

const advancedItems = [
  { label: '高级设置', icon: 'i-lucide-settings-2', slot: 'advanced' }
]

const schema = z.object({
  name: z.string().trim().min(1, '请输入群组名称').max(50, '群组名称过长'),
  description: z.string().max(200, '群组描述过长').optional().or(z.literal('')),
  // state 中在未选点前为 undefined，因此 schema 也应允许 undefined，最终提交仍由 canSubmit 把关
  latitude: z.number().optional(),
  longitude: z.number().optional(),
  location_name: z.string().trim().min(1, '请选择位置'),
  join_method: z.enum(['open', 'approval', 'invite_only']).default('open'),
  is_searchable: z.boolean().default(true),
  max_members: z.number().int().min(GROUP_CONSTANTS.MIN_MEMBERS).max(GROUP_CONSTANTS.MAX_MEMBERS_LIMIT),
  password: z.string().optional()
})

const form = reactive({
  name: '',
  description: '',
  latitude: undefined as number | undefined,
  longitude: undefined as number | undefined,
  location_name: '',
  join_method: 'open' as JoinMethod,
  is_searchable: true,
  max_members: GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT,
  password: ''
})

const creating = ref(false)
const feedback = useFeedbackStore()
const modal = useModal()
const locationStore = useLocationStore()

onMounted(() => {
  const current = locationStore.location
  if (!current) return

  if (form.latitude === undefined) form.latitude = current.latitude
  if (form.longitude === undefined) form.longitude = current.longitude
  if (!form.location_name) form.location_name = current.locationName
})

const canSubmit = computed(() => {
  return Boolean(form.name.trim() && form.location_name.trim() && form.latitude !== undefined && form.longitude !== undefined)
})

const locationHint = computed(() => {
  if (form.latitude === undefined || form.longitude === undefined) return '请选择一个位置点'
  return `${form.latitude.toFixed(6)}, ${form.longitude.toFixed(6)}`
})

async function pickLocation() {
  await modal.showLocationPicker((loc) => {
    form.latitude = loc.latitude
    form.longitude = loc.longitude
    form.location_name = loc.locationName
  })
}

async function createGroup() {
  if (!canSubmit.value || creating.value) return
  if (form.latitude === undefined || form.longitude === undefined) return

  creating.value = true
  try {
    const groupService = useGroupsService()
    const response = await groupService.create({
      name: form.name.trim(),
      description: form.description.trim() || undefined,
      avatar_file_id: undefined,
      tags: undefined,
      settings: {
        join_method: form.join_method,
        is_searchable: form.is_searchable,
        max_members: form.max_members,
        latitude: form.latitude,
        longitude: form.longitude,
        location_name: form.location_name,
        password: form.password.trim() || undefined
      }
    })

    feedback.success('群组创建成功', '您已成为群主，可以开始邀请朋友加入了！')
    props.onSuccess?.(response.group_chat_id)
    emit('close', response.group_chat_id)
  } catch (err: unknown) {
    console.error('创建群组失败:', err)
    feedback.error('创建失败', getErrorMessage(err, '请检查网络连接后重试'))
  } finally {
    creating.value = false
  }
}
</script>
