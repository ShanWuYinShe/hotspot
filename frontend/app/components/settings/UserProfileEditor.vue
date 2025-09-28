<template>
  <div class="space-y-8">
    <!-- 头像编辑 -->
    <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
      <template #header>
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white">头像设置</h3>
      </template>
      
      <div class="flex flex-col md:flex-row items-center gap-6">
        <div class="relative">
          <UAvatar
            :src="profileForm.avatar_url || currentAvatar"
            :alt="profileForm.nickname"
            size="3xl"
            class="ring-4 ring-primary-100 dark:ring-primary-900"
          />
          
          <!-- 上传按钮 -->
          <UFileUpload
            v-model="avatarFile"
            accept="image/*"
            variant="button"
            size="sm"
            class="absolute bottom-0 right-0 bg-primary-600 hover:bg-primary-700 text-white rounded-full p-2"
            @change="handleAvatarUpload"
          >
            <UIcon name="i-heroicons-camera" class="w-4 h-4" />
          </UFileUpload>
        </div>
        
        <div class="flex-1 text-center md:text-left">
          <h4 class="text-lg font-medium text-slate-900 dark:text-white mb-2">
            更换头像
          </h4>
          <p class="text-sm text-slate-600 dark:text-slate-400 mb-4">
            支持 JPG、PNG 格式，文件大小不超过 5MB
          </p>
          <div class="flex flex-wrap gap-2 justify-center md:justify-start">
            <UButton
              size="sm"
              variant="outline"
              @click="selectDefaultAvatar"
            >
              选择默认头像
            </UButton>
            <UButton
              size="sm"
              variant="outline"
              @click="removeAvatar"
              :disabled="!profileForm.avatar_url"
            >
              移除头像
            </UButton>
          </div>
        </div>
      </div>
    </UCard>

    <!-- 基本信息 -->
    <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
      <template #header>
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white">基本信息</h3>
      </template>
      
      <DynamicForm
        :fields="basicInfoFields"
        :schema="basicInfoSchema"
        :loading="saving"
        :initial-values="profileForm"
        @submit="saveBasicInfo"
        @change="updateProfileForm"
      />
    </UCard>

    <!-- 联系方式 -->
    <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
      <template #header>
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white">联系方式</h3>
      </template>
      
      <DynamicForm
        :fields="contactFields"
        :schema="contactSchema"
        :loading="saving"
        :initial-values="profileForm"
        @submit="saveContactInfo"
        @change="updateProfileForm"
      />
    </UCard>

    <!-- 个人设置 -->
    <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
      <template #header>
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white">个人设置</h3>
      </template>
      
      <div class="space-y-6">
        <!-- 隐私设置 -->
        <div>
          <h4 class="text-base font-medium text-slate-900 dark:text-white mb-4">隐私设置</h4>
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-slate-700 dark:text-slate-300">
                  允许陌生人查看我的资料
                </label>
                <p class="text-xs text-slate-500 dark:text-slate-400">
                  关闭后，只有好友可以查看您的详细资料
                </p>
              </div>
              <UToggle v-model="profileForm.allow_stranger_view" />
            </div>
            
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-slate-700 dark:text-slate-300">
                  显示在线状态
                </label>
                <p class="text-xs text-slate-500 dark:text-slate-400">
                  其他用户可以看到您是否在线
                </p>
              </div>
              <UToggle v-model="profileForm.show_online_status" />
            </div>
            
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-slate-700 dark:text-slate-300">
                  允许通过手机号搜索
                </label>
                <p class="text-xs text-slate-500 dark:text-slate-400">
                  其他用户可以通过手机号找到您
                </p>
              </div>
              <UToggle v-model="profileForm.searchable_by_phone" />
            </div>
          </div>
        </div>

        <!-- 位置设置 -->
        <div>
          <h4 class="text-base font-medium text-slate-900 dark:text-white mb-4">位置设置</h4>
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-slate-700 dark:text-slate-300">
                  允许位置共享
                </label>
                <p class="text-xs text-slate-500 dark:text-slate-400">
                  在地图上显示您的位置
                </p>
              </div>
              <UToggle v-model="profileForm.location_sharing_enabled" />
            </div>
            
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-slate-700 dark:text-slate-300">
                  精确位置
                </label>
                <p class="text-xs text-slate-500 dark:text-slate-400">
                  显示精确位置，否则只显示大概区域
                </p>
              </div>
              <UToggle 
                v-model="profileForm.precise_location" 
                :disabled="!profileForm.location_sharing_enabled"
              />
            </div>
          </div>
        </div>
      </div>
    </UCard>

    <!-- 账户安全 -->
    <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
      <template #header>
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white">账户安全</h3>
      </template>
      
      <div class="space-y-4">
        <div class="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-700/50 rounded-lg">
          <div>
            <h4 class="text-sm font-medium text-slate-900 dark:text-white">修改密码</h4>
            <p class="text-xs text-slate-500 dark:text-slate-400">定期更换密码以保护账户安全</p>
          </div>
          <UButton
            variant="outline"
            size="sm"
            @click="openChangePassword"
          >
            修改密码
          </UButton>
        </div>
        
        <div class="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-700/50 rounded-lg">
          <div>
            <h4 class="text-sm font-medium text-slate-900 dark:text-white">两步验证</h4>
            <p class="text-xs text-slate-500 dark:text-slate-400">为您的账户添加额外的安全保护</p>
          </div>
          <UButton
            variant="outline"
            size="sm"
            @click="setup2FA"
          >
            {{ profileForm.two_factor_enabled ? '管理' : '启用' }}
          </UButton>
        </div>
        
        <div class="flex items-center justify-between p-4 bg-red-50 dark:bg-red-900/20 rounded-lg">
          <div>
            <h4 class="text-sm font-medium text-red-900 dark:text-red-300">注销账户</h4>
            <p class="text-xs text-red-600 dark:text-red-400">永久删除您的账户和所有数据</p>
          </div>
          <UButton
            color="error"
            variant="outline"
            size="sm"
            @click="openDeleteAccount"
          >
            注销账户
          </UButton>
        </div>
      </div>
    </UCard>

    <!-- 保存按钮 -->
    <div class="flex items-center justify-end gap-4 pt-6 border-t border-slate-200 dark:border-slate-700">
      <UButton
        variant="outline"
        @click="resetForm"
        :disabled="saving"
      >
        重置
      </UButton>
      <UButton
        @click="saveAllChanges"
        :loading="saving"
        :disabled="!hasChanges"
      >
        保存所有更改
      </UButton>
    </div>

  </div>
</template>

<script setup lang="ts">
import { z } from 'zod'

interface UserProfile {
  user_id: string
  nickname: string
  display_name: string
  bio: string
  email: string
  phone: string
  avatar_url: string
  birthday: string
  gender: string
  location: string
  website: string
  allow_stranger_view: boolean
  show_online_status: boolean
  searchable_by_phone: boolean
  location_sharing_enabled: boolean
  precise_location: boolean
  two_factor_enabled: boolean
}

interface Props {
  profile: UserProfile
}

const props = defineProps<Props>()

const emit = defineEmits<{
  save: [profile: UserProfile]
  'avatar-change': [url: string]
}>()

// 状态
const saving = ref(false)
const currentAvatar = ref(props.profile.avatar_url)
const avatarFile = ref<File | null>(null)

// Modal 管理
const modal = useModal()

// 表单数据
const profileForm = reactive<UserProfile>({ ...props.profile })
const originalForm = { ...props.profile }

// 计算属性
const hasChanges = computed(() => {
  return JSON.stringify(profileForm) !== JSON.stringify(originalForm)
})

// 表单字段配置
const basicInfoFields = [
  {
    name: 'nickname',
    key: 'nickname',
    label: '昵称',
    type: 'text' as const,
    placeholder: '输入您的昵称',
    required: true
  },
  {
    name: 'display_name',
    key: 'display_name',
    label: '显示名称',
    type: 'text' as const,
    placeholder: '输入显示名称'
  },
  {
    name: 'bio',
    key: 'bio',
    label: '个人简介',
    type: 'textarea' as const,
    placeholder: '介绍一下自己...',
    rows: 3
  },
  {
    name: 'birthday',
    key: 'birthday',
    label: '生日',
    type: 'date' as const
  },
  {
    name: 'gender',
    key: 'gender',
    label: '性别',
    type: 'select' as const,
    options: [
      { label: '保密', value: '' },
      { label: '男', value: 'male' },
      { label: '女', value: 'female' }
    ]
  },
  {
    name: 'location',
    key: 'location',
    label: '所在地',
    type: 'text' as const,
    placeholder: '输入您的所在地'
  },
  {
    name: 'website',
    key: 'website',
    label: '个人网站',
    type: 'text' as const,
    placeholder: 'https://example.com'
  }
]

const contactFields = [
  {
    name: 'email',
    key: 'email',
    label: '邮箱地址',
    type: 'email' as const,
    placeholder: '输入邮箱地址',
    required: true
  },
  {
    name: 'phone',
    key: 'phone',
    label: '手机号码',
    type: 'text' as const,
    placeholder: '输入手机号码'
  }
]

const passwordFields = [
  {
    name: 'currentPassword',
    key: 'currentPassword',
    label: '当前密码',
    type: 'password' as const,
    placeholder: '输入当前密码',
    required: true
  },
  {
    name: 'newPassword',
    key: 'newPassword',
    label: '新密码',
    type: 'password' as const,
    placeholder: '输入新密码',
    required: true
  },
  {
    name: 'confirmPassword',
    key: 'confirmPassword',
    label: '确认新密码',
    type: 'password' as const,
    placeholder: '再次输入新密码',
    required: true
  }
]

// 验证模式
const basicInfoSchema = z.object({
  nickname: z.string().min(1, '昵称不能为空').max(20, '昵称不能超过20个字符'),
  display_name: z.string().max(30, '显示名称不能超过30个字符').optional(),
  bio: z.string().max(200, '个人简介不能超过200个字符').optional(),
  website: z.string().url('请输入有效的网址').optional().or(z.literal(''))
})

const contactSchema = z.object({
  email: z.string().email('请输入有效的邮箱地址'),
  phone: z.string().regex(/^1[3-9]\d{9}$/, '请输入有效的手机号码').optional().or(z.literal(''))
})

const passwordSchema = z.object({
  currentPassword: z.string().min(1, '请输入当前密码'),
  newPassword: z.string().min(6, '新密码至少6位字符'),
  confirmPassword: z.string().min(1, '请确认新密码')
}).refine(data => data.newPassword === data.confirmPassword, {
  message: '两次输入的密码不一致',
  path: ['confirmPassword']
})

// 方法
const handleAvatarUpload = async (event: Event) => {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return

  if (file.size > 5 * 1024 * 1024) {
    useNotifications().error('文件大小不能超过5MB')
    return
  }

  try {
    // 这里应该调用上传API
    const formData = new FormData()
    formData.append('avatar', file)
    
    // 模拟上传
    const reader = new FileReader()
    reader.onload = (e) => {
      const url = e.target?.result as string
      profileForm.avatar_url = url
      currentAvatar.value = url
      emit('avatar-change', url)
    }
    reader.readAsDataURL(file)
    
    useNotifications().success('头像上传成功')
  } catch (error) {
    useNotifications().error('头像上传失败')
  }
}

const selectDefaultAvatar = () => {
  const defaultAvatars = [
    'https://api.dicebear.com/7.x/avataaars/svg?seed=default1',
    'https://api.dicebear.com/7.x/avataaars/svg?seed=default2',
    'https://api.dicebear.com/7.x/avataaars/svg?seed=default3'
  ]
  
  const randomAvatar = defaultAvatars[Math.floor(Math.random() * defaultAvatars.length)]
  profileForm.avatar_url = randomAvatar
  currentAvatar.value = randomAvatar
  emit('avatar-change', randomAvatar)
}

const removeAvatar = () => {
  profileForm.avatar_url = ''
  currentAvatar.value = ''
  emit('avatar-change', '')
}

const updateProfileForm = (data: any) => {
  Object.assign(profileForm, data)
}

const saveBasicInfo = async (data: any) => {
  saving.value = true
  try {
    Object.assign(profileForm, data)
    useNotifications().success('基本信息保存成功')
  } catch (error) {
    useNotifications().error('保存失败')
  } finally {
    saving.value = false
  }
}

const saveContactInfo = async (data: any) => {
  saving.value = true
  try {
    Object.assign(profileForm, data)
    useNotifications().success('联系方式保存成功')
  } catch (error) {
    useNotifications().error('保存失败')
  } finally {
    saving.value = false
  }
}

const saveAllChanges = async () => {
  saving.value = true
  try {
    emit('save', { ...profileForm })
    Object.assign(originalForm, profileForm)
    useNotifications().success('所有更改已保存')
  } catch (error) {
    useNotifications().error('保存失败')
  } finally {
    saving.value = false
  }
}

const resetForm = () => {
  Object.assign(profileForm, originalForm)
  currentAvatar.value = originalForm.avatar_url
}

// 打开修改密码 Modal
const openChangePassword = async () => {
  // TODO: 实现密码修改功能
  const { info } = useNotifications()
  info('密码修改功能', '此功能正在开发中')
}

// 打开注销账户 Modal
const openDeleteAccount = async () => {
  // TODO: 实现账户删除功能
  const { info } = useNotifications()
  info('账户删除功能', '此功能正在开发中')
}

const setup2FA = () => {
  useNotifications().info('两步验证功能即将上线')
}

// 监听props变化
watch(() => props.profile, (newProfile) => {
  Object.assign(profileForm, newProfile)
  Object.assign(originalForm, newProfile)
  currentAvatar.value = newProfile.avatar_url
}, { deep: true })
</script>
