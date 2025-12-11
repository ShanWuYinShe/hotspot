<template>
  <UModal
    size="xl"
    title="创建群组"
    description="选择创建方式，快速开始您的群组之旅"
  >
    <template #body="{ close }">
      <!-- 创建方式选择 -->
      <nav
        v-if="!showForm"
        class="space-y-4"
      >
        <UCard
          class="interactive-card cursor-pointer"
          @click="quickCreate"
        >
          <section class="flex items-center gap-4 p-2">
            <UAvatar
              icon="i-heroicons-bolt"
              size="lg"
              :ui="{ icon: 'text-primary' }"
            />
            <article class="flex-1">
              <header class="flex items-center gap-2">
                <h4 class="font-semibold">快速创建</h4>
                <UBadge
                  color="primary"
                  variant="soft"
                  size="xs"
                >
                  推荐
                </UBadge>
              </header>
              <p class="text-sm text-secondary-content">使用当前位置和默认设置快速创建群组</p>
            </article>
            <UIcon
              name="i-heroicons-chevron-right"
              class="text-muted"
            />
          </section>
        </UCard>

        <UCard
          class="interactive-card cursor-pointer"
          @click="customCreate"
        >
          <section class="flex items-center gap-4 p-2">
            <UAvatar
              icon="i-heroicons-cog-6-tooth"
              size="lg"
              color="purple"
            />
            <article class="flex-1">
              <h4 class="font-semibold">自定义创建</h4>
              <p class="text-sm text-secondary-content">详细配置群组信息、位置和权限设置</p>
            </article>
            <UIcon
              name="i-heroicons-chevron-right"
              class="text-muted"
            />
          </section>
        </UCard>
      </nav>

      <!-- 创建表单 -->
      <UForm
        v-if="showForm"
        :state="form"
        class="space-y-6"
        @submit.prevent
      >
        <UFormField
          label="群组名称"
          name="name"
          required
        >
          <UInput
            v-model="form.name"
            placeholder="输入群组名称"
            :maxlength="50"
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
          />
        </UFormField>

        <UFormField label="群组位置">
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

        <template v-if="!isQuickCreate">
          <UFormField label="最大成员数">
            <UInput
              v-model.number="form.max_members"
              type="number"
              :min="GROUP_CONSTANTS.MIN_MEMBERS"
              :max="GROUP_CONSTANTS.MAX_MEMBERS_LIMIT"
              :placeholder="GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT.toString()"
            />
          </UFormField>

          <UFormField
            label="群组密码（可选）"
            hint="设置密码后，用户需要输入密码才能加入群组"
          >
            <UInput
              v-model="form.password"
              type="password"
              placeholder="设置群组密码以限制加入"
            />
          </UFormField>
        </template>
      </UForm>
    </template>

    <template #footer="{ close }">
      <UButton
        v-if="showForm"
        variant="outline"
        :disabled="creating"
        @click="showForm = false"
      >
        返回
      </UButton>
      <UButton
        v-if="showForm"
        :loading="creating"
        :disabled="!isFormValid"
        @click="() => createGroup(close)"
      >
        {{ isQuickCreate ? '快速创建' : '创建群组' }}
      </UButton>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { GROUP_CONSTANTS } from '~/config/defaults'
import { useGroupsService } from '~/services/groups.service'
import { geocoding } from '~/utils/geocoding'
import { getErrorMessage } from '~/utils/error'

interface Props {
  onSuccess?: (groupId: string) => void
}

const props = defineProps<Props>()

const creating = ref(false)
const gettingLocation = ref(false)
const showForm = ref(false)
const isQuickCreate = ref(false)

const form = ref({
  name: '',
  description: '',
  latitude: null as number | null,
  longitude: null as number | null,
  location_name: '',
  max_members: GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT,
  password: ''
})

const isFormValid = computed(() => {
  return form.value.name.trim() &&
    form.value.latitude !== null &&
    form.value.longitude !== null &&
    form.value.location_name.trim()
})

const toast = useToast()
const geolocation = useGeolocation()

const quickCreate = async () => {
  isQuickCreate.value = true

  const position = await geolocation.getPosition({ silent: true })
  if (position) {
    form.value.latitude = position.coords.latitude
    form.value.longitude = position.coords.longitude
    form.value.location_name = '当前位置'
    form.value.name = `${new Date().toLocaleDateString()} 的群组`
    form.value.description = '欢迎加入我们的群组！'
  } else {
    toast.add({ color: 'warning', title: '未能获取位置', description: '请手动输入位置信息' })
  }

  showForm.value = true
}

const customCreate = () => {
  isQuickCreate.value = false
  showForm.value = true
}

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

const createGroup = async (close: () => void) => {
  if (!isFormValid.value) return

  creating.value = true
  try {
    const groupService = useGroupsService()

    let passwordHash: string | undefined
    if (form.value.password) {
      const { hashPassword } = await import('~/utils/password')
      passwordHash = await hashPassword(form.value.password)
    }

    const response = await groupService.create({
      name: form.value.name,
      description: form.value.description || undefined,
      avatar_url: undefined,
      settings: {
        join_method: 'open',
        is_searchable: true,
        max_members: form.value.max_members,
        password_hash: passwordHash
      }
    })

    toast.add({ color: 'success', title: '群组创建成功', description: '您已成为群主，可以开始邀请朋友加入了！' })
    props.onSuccess?.(response.id)
    close()
  } catch (err: unknown) {
    console.error('创建群组失败:', err)
    toast.add({ color: 'error', title: '创建失败', description: getErrorMessage(err, '请检查网络连接后重试') })
  } finally {
    creating.value = false
  }
}
</script>
