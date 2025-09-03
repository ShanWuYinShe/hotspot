<template>
  <div class="space-y-4">
    <UForm :state="groupForm" :validate="validateGroup" @submit="createGroup">
    <!-- 群组名称 -->
    <UFormField label="群组名称" name="name" required>
      <UInput
        v-model="groupForm.name"
        placeholder="输入群组名称"
        :maxlength="50"
      />
    </UFormField>

    <!-- 群组描述 -->
    <UFormField label="群组描述" name="description">
      <UTextarea
        v-model="groupForm.description"
        placeholder="简单介绍一下这个群组..."
        :rows="3"
        :maxlength="200"
      />
    </UFormField>

    <!-- 群组密码 -->
    <UFormField label="群组密码" name="password" description="可选，设置后加入群组需要密码">
      <UInput
        v-model="groupForm.password"
        type="password"
        placeholder="设置群组密码（可选）"
        :maxlength="20"
      />
    </UFormField>

    <!-- 位置信息 -->
    <UFormField label="位置信息" name="location" description="获取当前位置信息用于创建群组">
      <div v-if="coordinates" class="text-sm text-neutral-600 dark:text-neutral-400">
        <UIcon name="i-heroicons-map-pin" class="h-4 w-4 inline mr-1" />
        当前位置：{{ coordinates.latitude.toFixed(4) }}, {{ coordinates.longitude.toFixed(4) }}
      </div>
      
      <UButton
        v-else
        @click="getLocation"
        :loading="isLoadingLocation"
        variant="outline"
        size="sm"
      >
        <UIcon name="i-heroicons-map-pin" class="h-4 w-4 mr-2" />
        获取位置
      </UButton>
    </UFormField>

    <!-- 提交按钮 -->
    <div class="flex justify-end space-x-3 pt-4">
      <UButton variant="ghost" @click="$emit('cancel')">
        取消
      </UButton>
      <UButton
        type="submit"
        :loading="isCreating"
        :disabled="!groupForm.name.trim()"
      >
        创建群组
      </UButton>
    </div>
    </UForm>
  </div>
</template>

<script setup lang="ts">
// 类型定义
interface GroupForm {
  name: string
  description: string  
  password: string
}

// Emits
const emit = defineEmits<{
  success: [group: any];
  cancel: [];
}>()

const toast = useSmartToast()
const groupsStore = useGroupsStore()
const { coordinates, getCurrentPosition, isLoading: isLoadingLocation } = useGeolocation()
const { requestGeolocation } = usePermissions()

// 表单数据
const groupForm = ref<GroupForm>({
  name: '',
  description: '',
  password: ''
})

const isCreating = ref(false)

// 表单校验
type ValidateResult = { path: string; message: string }[]

function validateGroup(): ValidateResult | void {
  const errs: ValidateResult = []
  const name = groupForm.value.name.trim()
  const desc = groupForm.value.description.trim()
  const pwd = groupForm.value.password.trim()
  if (!name) errs.push({ path: 'name', message: '群组名称为必填项' })
  if (name.length > 50) errs.push({ path: 'name', message: '群组名称不能超过50个字符' })
  if (desc.length > 200) errs.push({ path: 'description', message: '群组描述不能超过200个字符' })
  if (pwd.length > 20) errs.push({ path: 'password', message: '群组密码不能超过20个字符' })
  return errs.length ? errs : undefined
}

// 获取位置
async function getLocation() {
  try {
    // 先请求权限
    const granted = await requestGeolocation()
    if (granted) {
      await getCurrentPosition()
      toast.success('位置获取成功')
    } else {
      toast.warning({
        title: '位置权限被拒绝',
        description: '将使用默认位置创建群组'
      })
    }
  } catch (error) {
    console.error('获取位置失败:', error)
    toast.warning({
      title: '位置获取失败',
      description: '将使用默认位置创建群组'
    })
  }
}

// 创建群组
async function createGroup() {
  if (!groupForm.value.name.trim()) {
    toast.error('请输入群组名称')
    return
  }

  isCreating.value = true

  try {
    // 检查并获取地理位置
    let currentCoords = coordinates.value
    
    if (!currentCoords || !currentCoords.latitude || !currentCoords.longitude) {
      // 尝试获取位置权限
      toast.info({
        title: '正在获取位置...',
        description: '创建群组需要位置信息'
      })
      
      try {
        currentCoords = await getCurrentPosition()
      } catch (locationError) {
        console.warn('位置获取失败:', locationError)
        
        // 使用默认位置作为降级方案（北京市中心）
        currentCoords = {
          latitude: 39.9042,
          longitude: 116.4074,
          accuracy: 0
        }
        
        toast.warning({
          title: '使用默认位置',
          description: '无法获取当前位置，将使用默认位置创建群组'
        })
      }
    }

    // 显示创建中状态
    toast.info({
      title: '正在创建群组...',
      description: '请稍等'
    })
    
    // 调用群组服务创建群组（默认位置名与后端保持一致）
    const createdGroup = await groupsStore.createGroup({
      name: groupForm.value.name.trim(),
      description: groupForm.value.description.trim() || undefined,
      password: groupForm.value.password.trim() || undefined,
      latitude: currentCoords.latitude,
      longitude: currentCoords.longitude,
      location_name: "默认位置" // 后端会根据坐标自动获取位置名称
    })

    // 重置表单
    groupForm.value = {
      name: '',
      description: '',
      password: ''
    }
    
    toast.success({
      title: '群组创建成功',
      description: `"${createdGroup?.name || '新群组'}" 群组已创建`
    })
    
    // 触发成功事件
    emit('success', createdGroup)
    
  } catch (error: any) {
    const msg = error?.message || '创建群组失败'
    const detail = error?.details?.message || ''
    toast.error({
      title: '创建失败',
      description: detail || msg
    })
  } finally {
    isCreating.value = false
  }
}
</script>

