<template>
  <ClientOnly>
    <div class="user-switcher">
      <NDropdown trigger="click" placement="bottom-start" :options="dropdownOptions" @select="handleSelect"
        class="user-dropdown">
        <div class="current-user-trigger">
          <div class="user-avatar">
            <img v-if="currentUser?.avatar_url" :src="currentUser.avatar_url" :alt="currentUser.nickname"
              class="avatar-img" />
            <div v-else class="avatar-placeholder">
              {{ (currentUser?.nickname || '?').charAt(0).toUpperCase() }}
            </div>
          </div>
          <div class="user-info">
            <div class="username">
              <ClientOnly fallback="未登录">
                {{ currentUser?.nickname || '未登录' }}
              </ClientOnly>
            </div>
            <div class="user-type">
              <ClientOnly>
                <span v-if="currentUser?.is_temporary" class="temp-badge">临时用户</span>
                <span v-else class="regular-badge">正式用户</span>
                <template #fallback>
                  <span class="regular-badge">用户</span>
                </template>
              </ClientOnly>
            </div>
          </div>
          <NIcon class="dropdown-arrow">
            <ArrowDown />
          </NIcon>
        </div>
      </NDropdown>

      <!-- 登录对话框 -->
      <NModal v-model:show="showLoginDialog" preset="dialog" title="登录账户" :close-on-esc="false" :mask-closable="false">
        <NForm :model="loginForm" label-width="80px">
          <NFormItem label="用户名">
            <NInput v-model:value="loginForm.login_id" placeholder="请输入用户名" />
          </NFormItem>
          <NFormItem label="密码">
            <NInput v-model:value="loginForm.password" type="password" placeholder="请输入密码" @keyup.enter="handleLogin" />
          </NFormItem>
        </NForm>

        <template #action>
          <NButton @click="showLoginDialog = false">取消</NButton>
          <NButton type="primary" :loading="isLoading" @click="handleLogin">
            登录
          </NButton>
        </template>
      </NModal>

      <!-- 创建临时用户对话框 -->
      <NModal v-model:show="showTempUserDialog" preset="dialog" title="创建临时用户" :close-on-esc="false"
        :mask-closable="false">
        <NForm :model="tempUserForm" label-width="80px">
          <NFormItem label="昵称">
            <NInput v-model:value="tempUserForm.nickname" placeholder="请输入昵称（可选）" @keyup.enter="handleCreateTempUser" />
          </NFormItem>
        </NForm>

        <template #action>
          <NButton @click="showTempUserDialog = false">取消</NButton>
          <NButton type="primary" :loading="isLoading" @click="handleCreateTempUser">
            创建
          </NButton>
        </template>
      </NModal>
    </div>

    <template #fallback>
      <div class="user-switcher-loading">
        <div class="loading-placeholder">
          <div class="placeholder-avatar"></div>
          <div class="placeholder-text">
            <div class="placeholder-line"></div>
            <div class="placeholder-line small"></div>
          </div>
        </div>
      </div>
    </template>
  </ClientOnly>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import {
  NDropdown,
  NIcon,
  NButton,
  NModal,
  NInput,
  NForm,
  NFormItem,
  useMessage,
  useDialog
} from 'naive-ui'
import { ArrowDown, Add as Plus, Person as User, Settings as Setting, LogOut, Close } from '@vicons/ionicons5'
import { useUserManagerStore } from '~/stores/userManager'
import { useAuthStore } from '~/stores/auth'
import type { LoginRequest } from '~/types/api/auth'

// Props & Emits
interface Props {
  size?: 'small' | 'default' | 'large'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'default'
})

const emit = defineEmits<{
  'user-switched': [userId: number]
  'user-added': [userId: number]
  'upgrade-user': []
}>()

// Stores
const userManager = useUserManagerStore()
const authStore = useAuthStore()

// API实例
const dialog = useDialog()
const message = useMessage()

// 状态
const showLoginDialog = ref(false)
const showTempUserDialog = ref(false)
const isLoading = ref(false)

// 表单数据
const loginForm = ref<LoginRequest>({
  login_id: '',
  password: ''
})

const tempUserForm = ref({
  nickname: ''
})

// 计算属性
const currentUser = computed(() => userManager.currentUser)

const otherUsers = computed(() => {
  if (!currentUser.value) return userManager.recentUsers

  return userManager.recentUsers.filter(user =>
    user.user_id !== currentUser.value?.user_id
  )
})

// 下拉菜单选项
const dropdownOptions = computed(() => {
  const options = []

  // 当前用户信息
  if (currentUser.value) {
    options.push({
      type: 'group',
      label: '当前用户',
      key: 'current',
      children: [
        {
          label: `${currentUser.value.nickname} (${currentUser.value.is_temporary ? '临时' : '正式'})`,
          key: 'current-info',
          disabled: true
        }
      ]
    })

    if (currentUser.value.is_temporary) {
      options.push({
        label: '升级账户',
        key: 'upgrade',
        icon: () => h(NIcon, { style: { color: 'var(--color-warning-500)' } }, { default: () => h(User) }),
        props: {
          style: { color: 'var(--color-text-primary)' }
        }
      })
    }
  }

  // 其他用户
  if (otherUsers.value.length > 0) {
    options.push({
      type: 'group',
      label: '切换账户',
      key: 'switch',
      children: otherUsers.value.map(user => ({
        label: `${user.nickname} (${user.is_temporary ? '临时' : '正式'})`,
        key: `switch-${user.user_id}`,
        icon: () => h(NIcon, { style: { color: 'var(--color-primary-500)' } }, { default: () => h(User) }),
        props: {
          style: { color: 'var(--color-text-primary)' }
        }
      }))
    })
  }

  // 分隔线
  if (options.length > 0) {
    options.push({
      type: 'divider',
      key: 'divider1'
    })
  }

  // 操作菜单
  options.push(
    {
      label: '添加账户',
      key: 'add-account',
      icon: () => h(NIcon, { style: { color: 'var(--color-primary-500)' } }, { default: () => h(Plus) }),
      props: {
        style: { color: 'var(--color-text-primary)' }
      }
    },
    {
      label: '新建临时用户',
      key: 'create-temp',
      icon: () => h(NIcon, { style: { color: 'var(--color-success-500)' } }, { default: () => h(User) }),
      props: {
        style: { color: 'var(--color-text-primary)' }
      }
    },
    {
      type: 'divider',
      key: 'divider2'
    },
    {
      label: '个人设置',
      key: 'profile',
      icon: () => h(NIcon, { style: { color: 'var(--color-info-500)' } }, { default: () => h(Setting) }),
      props: {
        style: { color: 'var(--color-text-primary)' }
      }
    },
    {
      label: '退出登录',
      key: 'logout',
      icon: () => h(NIcon, { style: { color: 'var(--color-error-500)' } }, { default: () => h(LogOut) }),
      props: {
        style: { color: 'var(--color-error-500)' }
      }
    }
  )

  return options
})

// 方法
const formatLastUsed = (lastUsed: string) => {
  const now = new Date()
  const date = new Date(lastUsed)
  const diffMs = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays === 0) return '今天'
  if (diffDays === 1) return '昨天'
  if (diffDays < 7) return `${diffDays}天前`
  if (diffDays < 30) return `${Math.floor(diffDays / 7)}周前`
  return `${Math.floor(diffDays / 30)}个月前`
}

const handleSelect = async (key: string) => {
  try {
    if (key.startsWith('switch-')) {
      const userId = parseInt(key.replace('switch-', ''))
      await userManager.switchToUser(userId)
      emit('user-switched', userId)
      message.success('切换用户成功')
    } else {
      switch (key) {
        case 'upgrade':
          emit('upgrade-user')
          break

        case 'add-account':
          showLoginDialog.value = true
          break

        case 'create-temp':
          showTempUserDialog.value = true
          break

        case 'profile':
          await navigateTo('/profile')
          break

        case 'logout':
          await handleLogout()
          break
      }
    }
  } catch (error: any) {
    message.error(error.message || '操作失败')
  }
}

const handleLogin = async () => {
  if (!loginForm.value.login_id || !loginForm.value.password) {
    message.warning('请填写完整的登录信息')
    return
  }

  isLoading.value = true

  try {
    const response = await userManager.addUserByLogin(loginForm.value)

    emit('user-added', response.user_id)
    message.success('登录成功')

    // 重置表单
    loginForm.value = { login_id: '', password: '' }
    showLoginDialog.value = false
  } catch (error: any) {
    message.error(error.message || '登录失败')
  } finally {
    isLoading.value = false
  }
}

const handleCreateTempUser = async () => {
  isLoading.value = true

  try {
    const response = await userManager.createTempUserAccount(tempUserForm.value.nickname)

    emit('user-added', response.user_id)
    message.success('创建临时用户成功')

    // 重置表单
    tempUserForm.value = { nickname: '' }
    showTempUserDialog.value = false
  } catch (error: any) {
    message.error(error.message || '创建失败')
  } finally {
    isLoading.value = false
  }
}

const handleLogout = async () => {
  const d = dialog.warning({
    title: '确认退出',
    content: '确定要退出当前账户吗？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        if (currentUser.value) {
          await userManager.removeAccount(currentUser.value.user_id)
        }
        message.success('退出成功')
        await navigateTo('/auth')
      } catch (error: any) {
        message.error(error.message || '退出失败')
      }
    }
  })
}
</script>

<style scoped lang="scss">
.user-switcher {
  position: relative;

  .current-user-trigger {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    padding: var(--spacing-2) var(--spacing-3);
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border-primary);
    border-radius: var(--radius-lg);
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      border-color: var(--color-primary-500);
      background: var(--color-primary-50);
    }
  }

  .user-avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    overflow: hidden;
    border: 2px solid var(--color-border-secondary);

    .avatar-img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .avatar-placeholder {
      width: 100%;
      height: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--color-primary-100);
      color: var(--color-primary-600);
      font-weight: 600;
      font-size: 14px;
    }
  }

  .user-info {
    flex: 1;
    min-width: 0;

    .username {
      font-size: 14px;
      font-weight: 500;
      color: var(--color-text-primary);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .user-type {
      font-size: 12px;
      color: var(--color-text-secondary);
      margin-top: 2px;
    }
  }

  .dropdown-arrow {
    color: var(--color-text-secondary);
    transition: transform 0.2s ease;
  }
}

// 徽章样式
.temp-badge {
  background: var(--color-warning-100);
  color: var(--color-warning-600);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 500;
}

.regular-badge {
  background: var(--color-success-100);
  color: var(--color-success-600);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 500;
}

// 加载状态
.user-switcher-loading {
  .loading-placeholder {
    display: flex;
    align-items: center;
    gap: var(--spacing-2);
    padding: var(--spacing-2) var(--spacing-3);

    .placeholder-avatar {
      width: 32px;
      height: 32px;
      border-radius: 50%;
      background: var(--color-bg-tertiary);
      animation: pulse 1.5s ease-in-out infinite;
    }

    .placeholder-text {
      flex: 1;

      .placeholder-line {
        height: 12px;
        background: var(--color-bg-tertiary);
        border-radius: 4px;
        animation: pulse 1.5s ease-in-out infinite;
        margin-bottom: 4px;

        &.small {
          width: 60%;
          height: 10px;
          margin-bottom: 0;
        }
      }
    }
  }
}

@keyframes pulse {

  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.5;
  }
}

// 响应式
@media (max-width: 768px) {
  .user-switcher {
    .current-user-trigger {
      padding: var(--spacing-2);

      .user-info {
        .username {
          font-size: 13px;
        }

        .user-type {
          font-size: 11px;
        }
      }
    }
  }
}

// 深色模式适配
@media (prefers-color-scheme: dark) {
  .user-switcher {
    .current-user-trigger {
      background: var(--color-bg-primary);
      border-color: var(--color-border-secondary);

      &:hover {
        background: var(--color-bg-tertiary);
      }
    }
  }
}

// 为下拉菜单添加自定义样式
:deep(.n-dropdown-menu) {
  .n-dropdown-option {
    color: var(--color-text-primary) !important;

    &:hover {
      background-color: var(--color-bg-hover) !important;
      color: var(--color-text-primary) !important;
    }

    &.n-dropdown-option--disabled {
      color: var(--color-text-tertiary) !important;
    }
  }

  .n-dropdown-group .n-dropdown-group-header {
    color: var(--color-text-secondary) !important;
  }
}

// 修复菜单项图标颜色
:deep(.n-dropdown-option .n-icon) {
  color: inherit !important;
}

// 确保hover状态下的文字可见性
:deep(.n-dropdown-option:hover) {
  .n-dropdown-option__label {
    color: var(--color-text-primary) !important;
  }

  .n-icon {
    color: var(--color-primary-500) !important;
  }
}

// 针对不同状态的颜色优化
:deep(.n-dropdown-option) {
  &[data-key="upgrade"] {
    &:hover {
      background-color: var(--color-warning-50) !important;
      color: var(--color-warning-700) !important;

      .n-icon {
        color: var(--color-warning-600) !important;
      }
    }
  }

  &[data-key="logout"] {
    &:hover {
      background-color: var(--color-error-50) !important;
      color: var(--color-error-700) !important;

      .n-icon {
        color: var(--color-error-600) !important;
      }
    }
  }

  &[data-key^="switch-"] {
    &:hover {
      background-color: var(--color-primary-50) !important;
      color: var(--color-primary-700) !important;

      .n-icon {
        color: var(--color-primary-600) !important;
      }
    }
  }
}
</style>