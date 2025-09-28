<template>
  <UModal size="xl">
    <template #body>
      <div class="p-6">
        <!-- 标题 -->
        <div class="flex items-center justify-between mb-6">
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-user-group" class="w-6 h-6 text-primary-500" />
            <h3 class="text-lg font-semibold text-neutral-900 dark:text-neutral-100">
              创建群组
            </h3>
          </div>
          <UButton
            variant="ghost"
            icon="i-heroicons-x-mark"
            size="sm"
            @click="handleClose"
          />
        </div>

        <!-- 创建方式选择 -->
        <div v-if="!showForm" class="space-y-4">
          <p class="text-neutral-600 dark:text-neutral-400 mb-6">
            选择创建方式，快速开始您的群组之旅
          </p>
          
          <!-- 快速创建 -->
          <UCard 
            class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-primary-200 dark:hover:border-primary-800"
            @click="quickCreate"
          >
            <div class="flex items-center gap-4 p-4">
              <div class="flex items-center justify-center w-12 h-12 bg-primary-100 dark:bg-primary-900/50 rounded-xl">
                <UIcon name="i-heroicons-bolt" class="w-6 h-6 text-primary-600 dark:text-primary-300" />
              </div>
              <div class="flex-1">
                <h4 class="font-semibold text-neutral-900 dark:text-white mb-1">快速创建</h4>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">
                  使用当前位置和默认设置快速创建群组
                </p>
              </div>
              <UBadge color="primary" variant="soft" size="sm">推荐</UBadge>
            </div>
          </UCard>
          
          <!-- 自定义创建 -->
          <UCard 
            class="cursor-pointer transition-all duration-200 hover:shadow-md hover:border-purple-200 dark:hover:border-purple-800"
            @click="customCreate"
          >
            <div class="flex items-center gap-4 p-4">
              <div class="flex items-center justify-center w-12 h-12 bg-purple-100 dark:bg-purple-900/50 rounded-xl">
                <UIcon name="i-heroicons-cog-6-tooth" class="w-6 h-6 text-purple-600 dark:text-purple-300" />
              </div>
              <div class="flex-1">
                <h4 class="font-semibold text-neutral-900 dark:text-white mb-1">自定义创建</h4>
                <p class="text-sm text-neutral-600 dark:text-neutral-400">
                  详细配置群组信息、位置和权限设置
                </p>
              </div>
            </div>
          </UCard>
        </div>

        <!-- 创建表单 -->
        <form v-if="showForm" @submit.prevent="createGroup" class="space-y-6">
          <!-- 基本信息 -->
          <div class="space-y-4">
            <h4 class="font-medium text-neutral-900 dark:text-white">基本信息</h4>
            
            <div>
              <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                群组名称 *
              </label>
              <UInput
                v-model="form.name"
                placeholder="输入群组名称"
                :maxlength="50"
                required
              />
            </div>
            
            <div>
              <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                群组描述
              </label>
              <UTextarea
                v-model="form.description"
                placeholder="描述群组的主题和目的..."
                :rows="3"
                :maxlength="200"
              />
              <div class="flex justify-end mt-1">
                <span class="text-xs text-neutral-500">
                  {{ (form.description || '').length }}/200
                </span>
              </div>
            </div>
          </div>

          <!-- 位置信息 -->
          <div class="space-y-4">
            <h4 class="font-medium text-neutral-900 dark:text-white">位置信息</h4>
            
            <div>
              <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                群组位置 *
              </label>
              <UInput
                v-model="form.location_name"
                placeholder="输入位置名称"
                icon="i-heroicons-map-pin"
                required
              />
            </div>
            
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

          <!-- 群组设置 -->
          <div v-if="!isQuickCreate" class="space-y-4">
            <h4 class="font-medium text-neutral-900 dark:text-white">群组设置</h4>
            
            <div>
              <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                最大成员数
              </label>
              <UInput
                v-model.number="form.max_members"
                type="number"
                :min="GROUP_CONSTANTS.MIN_MEMBERS"
                :max="GROUP_CONSTANTS.MAX_MEMBERS_LIMIT"
                :placeholder="GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT.toString()"
              />
            </div>
            
            <div>
              <label class="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
                群组密码（可选）
              </label>
              <UInput
                v-model="form.password"
                type="password"
                placeholder="设置群组密码以限制加入"
              />
              <p class="text-xs text-neutral-500 mt-1">
                设置密码后，用户需要输入密码才能加入群组
              </p>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex justify-end gap-3 pt-4 border-t border-neutral-200 dark:border-neutral-700">
            <UButton
              variant="outline"
              @click="showForm = false"
              :disabled="creating"
            >
              返回
            </UButton>
            <UButton
              type="submit"
              :loading="creating"
              :disabled="!isFormValid"
            >
              {{ isQuickCreate ? '快速创建' : '创建群组' }}
            </UButton>
          </div>
        </form>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { GROUP_CONSTANTS } from '~/config/defaults'

interface Props {
  onSuccess?: (groupId: string) => void
  onCancel?: () => void
  onClose?: () => void
}

const props = defineProps<Props>()

// 状态管理
const creating = ref(false)
const gettingLocation = ref(false)
const showForm = ref(false)
const isQuickCreate = ref(false)

// 表单数据
const form = ref({
  name: '',
  description: '',
  latitude: null as number | null,
  longitude: null as number | null,
  location_name: '',
  max_members: GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT,
  password: ''
})

// 计算属性
const isFormValid = computed(() => {
  return form.value.name.trim() &&
         form.value.latitude !== null &&
         form.value.longitude !== null &&
         form.value.location_name.trim()
})

// 工具函数
const { success, error } = useNotifications()
const { getCurrentLocationOrDefault } = useLocation()

// 方法
const quickCreate = async () => {
  isQuickCreate.value = true
  
  // 获取当前位置
  try {
    const location = await getCurrentLocationOrDefault()
    form.value.latitude = location.latitude
    form.value.longitude = location.longitude
    form.value.location_name = '当前位置'
    form.value.name = `${new Date().toLocaleDateString()} 的群组`
    form.value.description = '欢迎加入我们的群组！'
  } catch (err) {
    error('位置获取失败', '请手动输入位置信息')
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
    const location = await getCurrentLocationOrDefault()
    form.value.latitude = location.latitude
    form.value.longitude = location.longitude
    
    // TODO: 调用地理编码API获取位置名称
    form.value.location_name = '当前位置'
    
    success('位置获取成功')
  } catch (err) {
    console.error('获取位置失败:', err)
    error('位置获取失败', '请手动输入位置信息')
  } finally {
    gettingLocation.value = false
  }
}

const createGroup = async () => {
  if (!isFormValid.value) return
  
  creating.value = true
  try {
    const groupService = useGroupService()
    const response = await groupService.createGroup({
      name: form.value.name,
      description: form.value.description || undefined,
      latitude: form.value.latitude!,
      longitude: form.value.longitude!,
      location_name: form.value.location_name,
      max_members: form.value.max_members,
      password: form.value.password || undefined
    })
    
    success('群组创建成功', '您已成为群主，可以开始邀请朋友加入了！')
    
    // 调用成功回调
    props.onSuccess?.(response.group_id)
    
    // 关闭Modal
    handleClose()
  } catch (err: any) {
    console.error('创建群组失败:', err)
    error('创建失败', err?.message || '请检查网络连接后重试')
  } finally {
    creating.value = false
  }
}

const handleClose = () => {
  // 重置表单
  showForm.value = false
  isQuickCreate.value = false
  form.value = {
    name: '',
    description: '',
    latitude: null,
    longitude: null,
    location_name: '',
    max_members: GROUP_CONSTANTS.MAX_MEMBERS_DEFAULT,
    password: ''
  }
  
  props.onClose?.()
}
</script>
