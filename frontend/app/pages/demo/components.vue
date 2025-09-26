<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-50 via-slate-100/50 to-slate-50 dark:from-slate-900 dark:via-slate-800/30 dark:to-slate-900">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- 页面标题 -->
      <div class="mb-12 text-center">
        <h1 class="text-4xl font-bold text-slate-800 dark:text-slate-100 mb-4">
          HotSpot 组件展示
        </h1>
        <p class="text-xl text-slate-600 dark:text-slate-400 max-w-3xl mx-auto">
          展示项目中的通用组件和设计系统，体现"简洁而不简陋"的设计理念
        </p>
      </div>

      <!-- 组件展示区域 -->
      <div class="space-y-16">
        <!-- ActionPanel 展示 -->
        <section>
          <div class="mb-8">
            <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100 mb-2">
              ActionPanel 操作面板
            </h2>
            <p class="text-slate-600 dark:text-slate-400">
              统一的操作面板组件，用于显示重要信息和操作按钮
            </p>
          </div>
          
          <div class="space-y-6">
            <!-- 成功状态 -->
            <ActionPanel
              title="操作成功"
              description="您的更改已成功保存"
              icon="i-heroicons-check-circle"
              :actions="[
                { key: 'continue', label: '继续', color: 'primary', action: () => toast.success('继续操作') },
                { key: 'back', label: '返回', variant: 'outline', action: () => toast.info('返回上一页') }
              ]"
            />
            
            <!-- 警告状态 -->
            <ActionPanel
              title="需要确认"
              description="此操作将永久删除数据，无法恢复"
              icon="i-heroicons-exclamation-triangle"
              :actions="[
                { key: 'cancel', label: '取消', variant: 'outline', action: () => toast.info('已取消') },
                { key: 'confirm', label: '确认删除', color: 'error', action: () => toast.error('已删除') }
              ]"
            />
          </div>
        </section>

        <!-- DataView 展示 -->
        <section>
          <div class="mb-8">
            <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100 mb-2">
              DataView 数据展示
            </h2>
            <p class="text-slate-600 dark:text-slate-400">
              通用的数据展示组件，支持搜索、筛选、分页等功能
            </p>
          </div>
          
          <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
            <DataView
              :data="demoUsers"
              :loading="demoLoading"
              :searchable="true"
              :search-fields="['nickname', 'email']"
              :show-pagination="true"
              :page-size="5"
              :toolbar-actions="demoToolbarActions"
              item-component="UserCard"
              :get-item-key="(item) => item.user_id"
              :get-item-props="(item) => ({ user: item })"
              :get-item-events="(item) => ({ 
                click: () => toast.info(`点击了用户: ${item.nickname}`),
                message: () => toast.info(`发送消息给: ${item.nickname}`)
              })"
              empty-icon="i-heroicons-users"
              empty-title="暂无用户"
              empty-description="还没有用户数据"
              :empty-action="{ label: '添加用户', action: () => addDemoUser() }"
              @item-click="(user) => toast.info(`查看用户: ${user.nickname}`)"
              @search="handleDemoSearch"
            />
          </UCard>
        </section>

        <!-- SmartForm 展示 -->
        <section>
          <div class="mb-8">
            <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100 mb-2">
              SmartForm 智能表单
            </h2>
            <p class="text-slate-600 dark:text-slate-400">
              智能表单组件，支持多种字段类型和自动验证
            </p>
          </div>
          
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <!-- 用户信息表单 -->
            <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
              <template #header>
                <h3 class="text-lg font-semibold text-slate-900 dark:text-white">用户信息表单</h3>
              </template>
              
              <SmartForm
                :fields="userFormFields"
                :initial-data="userFormData"
                :loading="formLoading"
                :schema="userFormSchema"
                submit-label="保存用户信息"
                @submit="handleUserFormSubmit"
                @change="(field, value) => toast.info(`${field} 更改为: ${value}`)"
              />
            </UCard>
            
            <!-- 设置表单 -->
            <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
              <template #header>
                <h3 class="text-lg font-semibold text-slate-900 dark:text-white">设置表单</h3>
              </template>
              
              <SmartForm
                :fields="settingsFormFields"
                :initial-data="settingsFormData"
                :loading="formLoading"
                submit-label="保存设置"
                @submit="handleSettingsFormSubmit"
              />
            </UCard>
          </div>
        </section>

        <!-- 工具函数展示 -->
        <section>
          <div class="mb-8">
            <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100 mb-2">
              工具函数展示
            </h2>
            <p class="text-slate-600 dark:text-slate-400">
              项目中使用的通用工具函数
            </p>
          </div>
          
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <!-- 时间格式化 -->
            <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
              <template #header>
                <h3 class="text-lg font-semibold text-slate-900 dark:text-white flex items-center">
                  <UIcon name="i-heroicons-clock" class="w-5 h-5 mr-2" />
                  时间格式化
                </h3>
              </template>
              
              <div class="space-y-3">
                <div v-for="time in demoTimes" :key="time.label" class="flex justify-between">
                  <span class="text-sm text-slate-600 dark:text-slate-400">{{ time.label }}:</span>
                  <span class="text-sm font-medium">{{ formatTime(time.value) }}</span>
                </div>
              </div>
            </UCard>
            
            <!-- 距离格式化 -->
            <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
              <template #header>
                <h3 class="text-lg font-semibold text-slate-900 dark:text-white flex items-center">
                  <UIcon name="i-heroicons-map-pin" class="w-5 h-5 mr-2" />
                  距离格式化
                </h3>
              </template>
              
              <div class="space-y-3">
                <div v-for="distance in demoDistances" :key="distance.label" class="flex justify-between">
                  <span class="text-sm text-slate-600 dark:text-slate-400">{{ distance.label }}:</span>
                  <span class="text-sm font-medium">{{ formatDistance(distance.value) }}</span>
                </div>
              </div>
            </UCard>
            
            <!-- 文件大小格式化 -->
            <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
              <template #header>
                <h3 class="text-lg font-semibold text-slate-900 dark:text-white flex items-center">
                  <UIcon name="i-heroicons-document" class="w-5 h-5 mr-2" />
                  文件大小格式化
                </h3>
              </template>
              
              <div class="space-y-3">
                <div v-for="size in demoFileSizes" :key="size.label" class="flex justify-between">
                  <span class="text-sm text-slate-600 dark:text-slate-400">{{ size.label }}:</span>
                  <span class="text-sm font-medium">{{ formatFileSize(size.value) }}</span>
                </div>
              </div>
            </UCard>
          </div>
        </section>

        <!-- 状态管理展示 -->
        <section>
          <div class="mb-8">
            <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100 mb-2">
              异步状态管理
            </h2>
            <p class="text-slate-600 dark:text-slate-400">
              使用 useAsyncState 管理异步操作状态
            </p>
          </div>
          
          <UCard class="bg-white/60 dark:bg-slate-800/60 border border-slate-200/50 dark:border-slate-700/50">
            <div class="space-y-6">
              <div class="flex items-center gap-4">
                <UButton
                  @click="asyncState.execute()"
                  :loading="asyncState.loading.value"
                  color="primary"
                >
                  执行异步操作
                </UButton>
                
                <UButton
                  @click="asyncState.refresh()"
                  :loading="asyncState.loading.value"
                  variant="outline"
                >
                  刷新
                </UButton>
                
                <UButton
                  @click="asyncState.reset()"
                  variant="ghost"
                >
                  重置
                </UButton>
              </div>
              
              <div class="p-4 bg-slate-100 dark:bg-slate-700 rounded-lg">
                <div class="space-y-2">
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-medium">状态:</span>
                    <UBadge 
                      :color="asyncState.loading.value ? 'warning' : (asyncState.error.value ? 'error' : 'success')"
                      variant="soft"
                    >
                      {{ asyncState.loading.value ? '加载中' : (asyncState.error.value ? '错误' : '成功') }}
                    </UBadge>
                  </div>
                  
                  <div v-if="asyncState.data.value" class="text-sm">
                    <span class="font-medium">数据:</span>
                    {{ asyncState.data.value }}
                  </div>
                  
                  <div v-if="asyncState.error.value" class="text-sm text-red-600 dark:text-red-400">
                    <span class="font-medium">错误:</span>
                    {{ asyncState.error.value.message }}
                  </div>
                </div>
              </div>
            </div>
          </UCard>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { z } from 'zod'
import type { FormField } from '~/components/SmartForm.vue'
import type { UserResponse } from '~/types/api'
import { formatTime, formatDistance, formatFileSize } from '~/utils/formatters'

definePageMeta({
  title: '组件展示',
  description: '展示项目中的通用组件和设计系统',
  layout: 'dashboard'
})

// 基础状态
const toast = useSmartToast()
const demoLoading = ref(false)
const formLoading = ref(false)

// 演示数据
const demoUsers = ref<UserResponse[]>([
  {
    user_id: '1',
    nickname: '张三',
    avatar_url: '',
    is_temp: false,
    email: 'zhangsan@example.com'
  },
  {
    user_id: '2',
    nickname: '李四',
    avatar_url: '',
    is_temp: false,
    email: 'lisi@example.com'
  },
  {
    user_id: '3',
    nickname: '王五',
    avatar_url: '',
    is_temp: true,
    email: 'wangwu@example.com'
  }
])

// DataView 工具栏操作
const demoToolbarActions = [
  {
    key: 'refresh',
    label: '刷新',
    icon: 'i-heroicons-arrow-path',
    variant: 'outline' as const,
    action: () => {
      demoLoading.value = true
      setTimeout(() => {
        demoLoading.value = false
        toast.success('数据已刷新')
      }, 1000)
    }
  },
  {
    key: 'add',
    label: '添加用户',
    icon: 'i-heroicons-plus',
    color: 'primary' as const,
    action: () => addDemoUser()
  }
]

// 表单数据
const userFormData = reactive({
  nickname: '',
  email: '',
  bio: '',
  age: 18,
  gender: 'male',
  avatar: null
})

const settingsFormData = reactive({
  notifications: true,
  darkMode: false,
  language: 'zh-CN',
  autoSave: true
})

// 表单字段定义
const userFormFields: FormField[] = [
  {
    name: 'nickname',
    label: '昵称',
    type: 'text',
    placeholder: '请输入昵称',
    required: true,
    icon: 'i-heroicons-user'
  },
  {
    name: 'email',
    label: '邮箱',
    type: 'email',
    placeholder: '请输入邮箱',
    required: true,
    icon: 'i-heroicons-envelope'
  },
  {
    name: 'bio',
    label: '个人简介',
    type: 'textarea',
    placeholder: '介绍一下自己...',
    rows: 3
  },
  {
    name: 'age',
    label: '年龄',
    type: 'number',
    min: 1,
    max: 120,
    step: 1
  },
  {
    name: 'gender',
    label: '性别',
    type: 'select',
    options: [
      { label: '男', value: 'male' },
      { label: '女', value: 'female' },
      { label: '其他', value: 'other' }
    ],
    required: true
  },
  {
    name: 'avatar',
    label: '头像',
    type: 'file',
    accept: 'image/*',
    placeholder: '选择头像图片'
  }
]

const settingsFormFields: FormField[] = [
  {
    name: 'notifications',
    label: '启用通知',
    type: 'switch',
    description: '接收应用通知'
  },
  {
    name: 'darkMode',
    label: '深色模式',
    type: 'switch',
    description: '使用深色主题'
  },
  {
    name: 'language',
    label: '语言',
    type: 'select',
    options: [
      { label: '中文', value: 'zh-CN' },
      { label: 'English', value: 'en-US' },
      { label: '日本語', value: 'ja-JP' }
    ],
    required: true
  },
  {
    name: 'autoSave',
    label: '自动保存',
    type: 'switch',
    description: '自动保存用户数据'
  }
]

// 表单验证
const userFormSchema = z.object({
  nickname: z.string().min(2, '昵称至少2个字符').max(20, '昵称不超过20个字符'),
  email: z.string().email('请输入有效邮箱'),
  bio: z.string().max(200, '简介不超过200字符').optional(),
  age: z.number().min(1, '年龄必须大于0').max(120, '年龄不能超过120'),
  gender: z.enum(['male', 'female', 'other']),
  avatar: z.any().optional()
})

// 演示时间数据
const demoTimes = [
  { label: '刚刚', value: new Date().toISOString() },
  { label: '5分钟前', value: new Date(Date.now() - 5 * 60 * 1000).toISOString() },
  { label: '2小时前', value: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString() },
  { label: '3天前', value: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString() },
  { label: '1周前', value: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString() }
]

// 演示距离数据
const demoDistances = [
  { label: '很近', value: 50 },
  { label: '附近', value: 500 },
  { label: '1公里内', value: 800 },
  { label: '较远', value: 5000 },
  { label: '很远', value: 50000 }
]

// 演示文件大小数据
const demoFileSizes = [
  { label: '小文件', value: 1024 },
  { label: '图片', value: 1024 * 1024 * 2 },
  { label: '视频', value: 1024 * 1024 * 50 },
  { label: '大文件', value: 1024 * 1024 * 1024 * 2 },
  { label: '超大文件', value: 1024 * 1024 * 1024 * 10 }
]

// 异步状态管理演示
const mockAsyncOperation = async () => {
  await new Promise(resolve => setTimeout(resolve, 2000))
  if (Math.random() > 0.7) {
    throw new Error('模拟错误')
  }
  return `操作完成时间: ${new Date().toLocaleTimeString()}`
}

const asyncState = useAsyncState(mockAsyncOperation, null, {
  onSuccess: (data) => toast.success(`操作成功: ${data}`),
  onError: (error) => toast.error(`操作失败: ${error.message}`)
})

// 方法
const addDemoUser = () => {
  const newUser: UserResponse = {
    user_id: String(Date.now()),
    nickname: `用户${demoUsers.value.length + 1}`,
    avatar_url: '',
    is_temp: false,
    email: `user${demoUsers.value.length + 1}@example.com`
  }
  demoUsers.value.push(newUser)
  toast.success(`添加了新用户: ${newUser.nickname}`)
}

const handleDemoSearch = (query: string) => {
  toast.info(`搜索: ${query}`)
}

const handleUserFormSubmit = async (data: any) => {
  formLoading.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1500))
    toast.success('用户信息保存成功')
    console.log('用户表单数据:', data)
  } catch (error) {
    toast.error('保存失败')
  } finally {
    formLoading.value = false
  }
}

const handleSettingsFormSubmit = async (data: any) => {
  formLoading.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000))
    toast.success('设置保存成功')
    console.log('设置表单数据:', data)
  } catch (error) {
    toast.error('保存失败')
  } finally {
    formLoading.value = false
  }
}
</script>
