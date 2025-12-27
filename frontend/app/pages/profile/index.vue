<template>
  <div class="w-full">
    <UContainer class="max-w-5xl space-y-8">
      <!-- 顶部个人信息卡片 -->
      <section v-if="loading">
        <UCard variant="outline" class="w-full h-48 mb-6">
          <div class="flex items-center gap-6">
            <UAvatar size="3xl" icon="i-lucide-user" color="neutral" />
            <div class="flex-1 space-y-4 w-full md:w-auto text-center md:text-left">
              <USkeleton class="h-10 w-48 mx-auto md:mx-0" />
              <USkeleton class="h-6 w-32 mx-auto md:mx-0" />
              <div class="flex gap-8 justify-center md:justify-start mt-6">
                <USkeleton class="h-16 w-24" />
                <USkeleton class="h-16 w-24" />
                <USkeleton class="h-16 w-24" />
              </div>
            </div>
          </div>
        </UCard>
      </section>

      <section v-else class="relative animate-fade-in-up">
        <UCard
          variant="outline"
          class="p-6 md:p-8"
        >
          <div class="flex flex-col md:flex-row items-center gap-8 md:gap-12">
            <!-- 头像区域 -->
            <div class="relative group shrink-0">
              <div class="relative">
                <UAvatar
                  :src="userProfile.avatar_url"
                  :alt="userProfile.nickname"
                  size="3xl"
                  imgClass="object-cover"
                >
                  <span class="text-4xl font-light">{{ userProfile.nickname?.charAt(0) || 'U' }}</span>
                </UAvatar>
                <!-- 升级/状态标识 -->
                <div class="absolute -bottom-1 -right-1">
                   <UBadge 
                    :label="currentUser?.user_type === 'normal' ? '正式' : '临时'"
                    :color="currentUser?.user_type === 'normal' ? 'success' : 'warning'"
                    variant="solid"
                  />
                </div>
              </div>
              
              <!-- 头像上传覆盖层 -->
              <UFileUpload
                v-slot="{ open }"
                v-model="selectedAvatarFile"
                accept="image/*"
                class="absolute inset-0"
                @update:model-value="handleAvatarSelected"
              >
                <div
                  class="hs-avatar-upload-overlay"
                  @click="open()"
                >
                  <UIcon name="i-lucide-camera" :size="48" />
                </div>
              </UFileUpload>
            </div>
            
            <!-- 信息区域 -->
            <div class="flex-1 text-center md:text-left min-w-0">
              <div class="mb-6 space-y-3">
                <UUser
                  :name="userProfile.nickname || '未设置昵称'"
                  :description="currentUser?.user_type === 'normal' ? '正式用户' : '临时用户'"
                />
                <div class="flex items-center justify-center md:justify-start gap-3">
                  <UButton
                    variant="soft"
                    color="neutral"
                    size="sm"
                    icon="i-lucide-copy"
                    @click="copyId"
                  >
                    ID: {{ userProfile.user_id }}
                  </UButton>
                </div>
              </div>

              <!-- 数据统计 -->
              <div class="grid grid-cols-3 gap-3 max-w-md mx-auto md:mx-0">
                <UCard variant="soft" data-hs-card="interactive" @click="navigateTo('/moments')">
                  <div class="flex flex-col items-center md:items-start gap-2">
                    <div class="text-xl font-bold">{{ stats.momentsCount }}</div>
                    <UBadge>动态</UBadge>
                  </div>
                </UCard>
                <UCard variant="soft" data-hs-card="interactive" @click="navigateTo('/friends')">
                  <div class="flex flex-col items-center md:items-start gap-2">
                    <div class="text-xl font-bold">{{ stats.friendsCount }}</div>
                    <UBadge>好友</UBadge>
                  </div>
                </UCard>
                <UCard variant="soft" data-hs-card="interactive" @click="navigateTo('/groups')">
                  <div class="flex flex-col items-center md:items-start gap-2">
                    <div class="text-xl font-bold">{{ stats.groupsCount }}</div>
                    <UBadge>群组</UBadge>
                  </div>
                </UCard>
              </div>
            </div>

          </div>
        </UCard>
      </section>

      <!-- 主要内容网格 -->
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 animate-fade-in-up delay-75">
        <!-- 左侧栏：详细信息 (4 cols) -->
        <div class="lg:col-span-4 space-y-6">
          <UCard class="h-full">
            <h3 class="text-lg font-semibold mb-6">账户信息</h3>
            
            <div class="space-y-6">
              <div class="flex items-start gap-4 group">
                <UBadge color="primary" variant="soft" size="lg" square icon="i-lucide-calendar" />
                <div>
                  <UBadge>注册时间</UBadge>
                  <div class="mt-1 text-sm font-semibold">
                    {{ userProfile.created_at ? formatDate(userProfile.created_at, 'long') : '未知' }}
                  </div>
                </div>
              </div>

              <div class="flex items-start gap-4 group">
                <UBadge color="success" variant="soft" size="lg" square icon="i-lucide-map-pin" />
                <div>
                  <UBadge>当前位置</UBadge>
                  <div class="mt-1 text-sm font-semibold">
                    {{ userProfile.location_name || '未设置位置' }}
                  </div>
                </div>
              </div>
              
              <div class="flex items-start gap-4 group">
                <UBadge color="primary" variant="soft" size="lg" square icon="i-lucide-sparkles" />
                <div>
                  <UBadge>个人标签</UBadge>
                  <div class="flex flex-wrap gap-2">
                    <UBadge 
                      v-for="tag in userProfile.tags" 
                      :key="tag"
                    >
                      {{ tag }}
                    </UBadge>
                    <UBadge v-if="!userProfile.tags?.length">暂无标签</UBadge>
                  </div>
                </div>
              </div>
            </div>
          </UCard>

          <!-- 临时用户升级提示 -->
          <UCard
            v-if="currentUser?.user_type === 'temp'"
            variant="subtle"
          >
            <div class="flex items-center gap-3">
              <UBadge color="info" variant="soft" size="lg" square icon="i-lucide-rocket" />
              <h3 class="text-lg font-semibold">升级账户</h3>
            </div>
            <p class="mt-2 text-sm leading-relaxed">
              升级为正式用户，解锁创建群组、发布无限动态等更多高级功能。
            </p>
            <div class="mt-4">
              <UButton :block="true" color="primary" @click="navigateTo('/settings/upgrade')">
                立即升级
              </UButton>
            </div>
          </UCard>
        </div>

        <!-- 右侧栏：编辑表单 (8 cols) -->
        <div class="lg:col-span-8">
          <UCard>
            <div class="flex items-center justify-between mb-8">
               <div class="flex items-center gap-3">
                  <UBadge color="neutral" variant="soft" size="lg" square icon="i-lucide-square-pen" />
                  <div>
                    <h3 class="text-lg font-semibold">编辑资料</h3>
                    <UBadge class="mt-1">更新您的公开信息</UBadge>
                  </div>
               </div>
            </div>

            <UForm :state="formState" class="space-y-8" @submit="handleSave">
              <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                <UFormField label="昵称" name="nickname" required>
                  <UInput 
                    v-model="formState.nickname" 
                    placeholder="您的昵称"
                    icon="i-lucide-user"
                  />
                </UFormField>

                <UFormField label="邮箱" name="email">
                  <UInput 
                    v-model="formState.email" 
                    type="email"
                    placeholder="未绑定邮箱"
                    icon="i-lucide-mail"
                    :disabled="true"
                  />
                </UFormField>
              </div>

              <UFormField label="个人简介" name="bio">
                <UTextarea 
                  v-model="formState.bio" 
                  placeholder="写一句话介绍自己，让大家更了解你..."
                  :rows="5"
                />
                <template #hint>
                  <UBadge>{{ formState.bio?.length || 0 }}/200</UBadge>
                </template>
              </UFormField>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                <UFormField label="手机号" name="phone">
                  <UInput 
                    v-model="formState.phone" 
                    placeholder="未绑定手机"
                    icon="i-lucide-phone"
                    :disabled="true"
                  />
                </UFormField>

                <UFormField label="位置" name="location">
                  <UCard variant="subtle">
                    <div class="flex items-center gap-2 min-w-0">
                      <UIcon name="i-lucide-map-pin" :size="16" />
                      <span class="truncate">{{ formState.location_name || '未设置位置' }}</span>
                    </div>
                  </UCard>
                </UFormField>
              </div>

              <USeparator />

              <div class="flex items-center justify-end gap-4 pt-2">
                <UButton 
                  type="button"
                  variant="ghost" 
                  color="neutral"
                  @click="resetForm"
                >
                  重置
                </UButton>
                <UButton 
                  type="submit"
                  color="primary"
                  :loading="saving"
                  :disabled="!hasChanges"
                >
                  保存更改
                </UButton>
              </div>
            </UForm>
          </UCard>
        </div>
      </div>
    </UContainer>
  </div>
</template>

<script setup lang="ts">
import { useUsersService } from '~/services/users.service'
import type { UserInfo } from '~/types/api'
import { useDashboardHeader } from '~/composables/useDashboardHeader'

definePageMeta({
  title: '个人资料',
  description: '管理您的个人信息和账户设置',
  requiresAuth: true,
  layout: 'dashboard',
  dashboardHeader: true
})

const authStore = useAuthStore()
const userService = useUsersService()
const feedback = useFeedbackStore()
const { withErrorHandling } = useErrorHandler()
const { formatDate } = useFormatters()
const dashboardHeader = useDashboardHeader()

watchEffect(() => {
  dashboardHeader.setActions([
    {
      key: 'preview',
      label: '预览主页',
      icon: 'i-lucide-eye',
      variant: 'outline',
      action: () => previewProfile()
    }
  ])
})

// 水合安全：用户信息
const currentUser = useHydratedStore(authStore, 'user', null) as Ref<UserInfo | null>

// 原始用户资料
const userProfile = ref({
  user_id: 0 as number,
  nickname: '',
  bio: '',
  avatar_url: '',
  email: '',
  phone: '',
  location_name: '',
  is_online: false,
  last_online: '',
  created_at: '',
  tags: [] as string[]
})

// 用户统计
const stats = ref({
  momentsCount: 0,
  friendsCount: 0,
  groupsCount: 0
})

// 表单状态
const formState = ref({
  nickname: '',
  bio: '',
  email: '',
  phone: '',
  location_name: '',
  avatar_file_id: ''
})

const saving = ref(false)
const loading = ref(true)
const selectedAvatarFile = ref<File | null>(null)

// 检测是否有更改
const hasChanges = computed(() => {
  return (
    formState.value.nickname !== userProfile.value.nickname ||
    formState.value.bio !== (userProfile.value.bio || '') ||
    formState.value.avatar_file_id !== ''
  )
})

const { copy } = useClipboard()

const copyId = async () => {
  if (userProfile.value.user_id) {
    try {
      await copy(userProfile.value.user_id.toString())
      feedback.success('ID 已复制', undefined, 2000)
    } catch {
      feedback.error('复制失败')
    }
  }
}

// 头像更改处理
const handleAvatarSelected = async (file: File | null | undefined) => {
  if (!file) return

  const reader = new FileReader()
  reader.onload = (e) => {
    userProfile.value.avatar_url = e.target?.result as string
  }
  reader.readAsDataURL(file)

  feedback.info('头像已更新（预览）', '请点击保存以应用更改')
  selectedAvatarFile.value = null
}

// 保存表单
const handleSave = async () => {
  if (!hasChanges.value) return
  
  await withErrorHandling(async () => {
    await userService.updateProfile({
      nickname: formState.value.nickname || undefined,
      avatar_file_id: formState.value.avatar_file_id || undefined,
      bio: formState.value.bio || undefined
    })
    
    userProfile.value.nickname = formState.value.nickname
    userProfile.value.bio = formState.value.bio
    formState.value.avatar_file_id = ''
    
    if (currentUser.value) {
      authStore.updateCurrentUser({ nickname: formState.value.nickname })
    }
    
    feedback.success('个人资料保存成功')
  }, {
    title: '保存失败',
    loading: saving
  })
}

// 重置表单
const resetForm = () => {
  formState.value = {
    nickname: userProfile.value.nickname,
    bio: userProfile.value.bio || '',
    email: userProfile.value.email || '',
    phone: userProfile.value.phone || '',
    location_name: userProfile.value.location_name || '',
    avatar_file_id: ''
  }
}

// 预览资料
const previewProfile = () => {
  if (userProfile.value.user_id) {
    navigateTo(`/users/${userProfile.value.user_id}`, { open: { target: '_blank' } })
  }
}

// 加载用户统计
const loadStats = async () => {
  try {
    if (userProfile.value.user_id) {
      const data = await userService.getStats({ id: userProfile.value.user_id })
      stats.value = {
        momentsCount: data.moments_count,
        friendsCount: data.friends_count,
        groupsCount: data.groups_count
      }
    }
  } catch {
    // 忽略统计加载失败
  }
}

// 页面加载时获取用户资料
onMounted(async () => {
  await withErrorHandling(async () => {
    const profile = await userService.getMe()
    userProfile.value = {
      user_id: profile.user_id,
      nickname: profile.nickname,
      bio: profile.bio || '',
      avatar_url: profile.avatar_url || '',
      email: profile.email || '',
      phone: profile.phone || '',
      location_name: profile.location_name || '',
      is_online: profile.is_online,
      last_online: profile.last_online || '',
      created_at: profile.created_at,
      tags: profile.tags || []
    }
    
    resetForm()
    loadStats()
  }, {
    title: '加载用户资料失败',
    loading: loading
  })
})
</script>
