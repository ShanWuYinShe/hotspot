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
          
          <!-- 头像上传组件 -->
          <AvatarUpload @uploaded="handleAvatarUploaded" />
        </div>
        
        <div class="flex-1 text-center md:text-left">
          <h4 class="text-lg font-medium text-slate-900 dark:text-white mb-2">
            更换头像
          </h4>
          <p class="text-sm text-slate-600 dark:text-slate-400 mb-4">
            支持 JPG、PNG 格式，文件大小不超过 5MB
          </p>
        </div>
      </div>
    </UCard>

    <!-- 基本信息 - 仅包含 API 支持的字段 (nickname, bio, avatar_url) -->
    <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
      <template #header>
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white">基本信息</h3>
      </template>
      
      <form @submit.prevent="saveBasicInfo" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            昵称 *
          </label>
          <UInput
            v-model="profileForm.nickname"
            placeholder="请输入昵称"
            maxlength="50"
            required
          />
        </div>
        
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            个人简介
          </label>
          <UTextarea
            v-model="profileForm.bio"
            placeholder="介绍一下自己吧..."
            :rows="4"
            maxlength="200"
          />
        </div>
      </form>
    </UCard>

    <!-- 联系方式（只读 - 显示API返回的email和phone） -->
    <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
      <template #header>
        <h3 class="text-lg font-semibold text-slate-900 dark:text-white">联系方式</h3>
      </template>
      
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            邮箱
          </label>
          <UInput
            :model-value="profileForm.email || '未设置'"
            disabled
            placeholder="未设置"
          />
          <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
            邮箱用于账户注册和找回密码
          </p>
        </div>
        
        <div>
          <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            手机号
          </label>
          <UInput
            :model-value="profileForm.phone || '未设置'"
            disabled
            placeholder="未设置"
          />
          <p class="text-xs text-slate-500 dark:text-slate-400 mt-1">
            手机号用于账户注册和找回密码
          </p>
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
// 根据 API 文档 3.2.2 MyProfileData 定义用户资料类型
interface UserProfile {
  user_id: string
  nickname: string
  bio: string
  avatar_url: string
  email: string
  phone: string
  location_name: string
  is_online: boolean
  last_seen: string
  created_at: string
}

interface Props {
  profile: UserProfile
}

const props = defineProps<Props>()

const emit = defineEmits<{
  save: [profile: UserProfile]
  'avatar-change': [url: string]
}>()

const toast = useToast()
const modal = useModal()
const saving = ref(false)
const currentAvatar = ref(props.profile.avatar_url)

// 表单数据（仅包含 API 支持的字段）
const profileForm = reactive({
  user_id: props.profile.user_id,
  nickname: props.profile.nickname,
  bio: props.profile.bio,
  avatar_url: props.profile.avatar_url,
  email: props.profile.email,
  phone: props.profile.phone,
  location_name: props.profile.location_name,
  is_online: props.profile.is_online,
  last_seen: props.profile.last_seen,
  created_at: props.profile.created_at
})
const originalForm = reactive({ ...profileForm })

// 计算是否有修改
const hasChanges = computed(() => {
  return JSON.stringify(profileForm) !== JSON.stringify(originalForm)
})

// 方法
const handleAvatarUploaded = (url: string) => {
  profileForm.avatar_url = url
  currentAvatar.value = url
  emit('avatar-change', url)
}

const saveBasicInfo = async () => {
  saving.value = true
  try {
    emit('save', { ...profileForm })
    Object.assign(originalForm, profileForm)
    toast.add({ color: 'success', title: '基本信息保存成功' })
  } catch (error) {
    toast.add({ color: 'error', title: '保存失败' })
  } finally {
    saving.value = false
  }
}

const saveAllChanges = async () => {
  saving.value = true
  try {
    emit('save', { ...profileForm })
    Object.assign(originalForm, profileForm)
    toast.add({ color: 'success', title: '所有更改已保存' })
  } catch (error) {
    toast.add({ color: 'error', title: '保存失败' })
  } finally {
    saving.value = false
  }
}

const resetForm = () => {
  Object.assign(profileForm, originalForm)
  currentAvatar.value = originalForm.avatar_url
}

// 打开修改密码 Modal
const openChangePassword = () => {
  modal.showChangePasswordModal()
}

// 打开注销账户 Modal
const openDeleteAccount = () => {
  modal.showDeleteAccountModal()
}

// 监听props变化
watch(() => props.profile, (newProfile) => {
  Object.assign(profileForm, newProfile)
  Object.assign(originalForm, newProfile)
  currentAvatar.value = newProfile.avatar_url
}, { deep: true })
</script>
