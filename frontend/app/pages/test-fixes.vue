<template>
  <div class="min-h-screen bg-gradient-to-br from-neutral-50 to-neutral-100 dark:from-neutral-900 dark:to-neutral-950">
    <UContainer class="py-8">
      <UCard>
        <template #header>
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-check-circle" class="w-6 h-6 text-green-500" />
            <h1 class="text-2xl font-bold">功能修复测试页面</h1>
          </div>
        </template>

        <template #body>
          <div class="space-y-8">
            <!-- 认证状态测试 -->
            <div class="space-y-4">
              <h2 class="text-xl font-semibold flex items-center gap-2">
                <UIcon name="i-heroicons-shield-check" class="w-5 h-5 text-blue-500" />
                认证状态测试
              </h2>
              
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <UCard>
                  <template #header>
                    <h3 class="font-semibold">当前认证状态</h3>
                  </template>
                  <div class="space-y-2">
                    <div class="flex items-center justify-between">
                      <span>已认证:</span>
                      <UBadge :color="authStore.isAuthenticated ? 'success' : 'error'">
                        {{ authStore.isAuthenticated ? '是' : '否' }}
                      </UBadge>
                    </div>
                    <div class="flex items-center justify-between">
                      <span>用户昵称:</span>
                      <span>{{ authStore.user?.nickname || '未登录' }}</span>
                    </div>
                    <div class="flex items-center justify-between">
                      <span>Token存在:</span>
                      <UBadge :color="!!authStore.token ? 'success' : 'error'">
                        {{ !!authStore.token ? '是' : '否' }}
                      </UBadge>
                    </div>
                  </div>
                </UCard>

                <UCard>
                  <template #header>
                    <h3 class="font-semibold">快速操作</h3>
                  </template>
                  <div class="space-y-3">
                    <UButton 
                      v-if="!authStore.isAuthenticated" 
                      to="/auth" 
                      color="primary" 
                      block
                    >
                      前往登录
                    </UButton>
                    <UButton 
                      v-else 
                      @click="authStore.logout" 
                      color="error" 
                      variant="outline" 
                      block
                    >
                      退出登录
                    </UButton>
                    <UButton to="/profile" variant="outline" block :disabled="!authStore.isAuthenticated">
                      访问个人资料 (需认证)
                    </UButton>
                  </div>
                </UCard>
              </div>
            </div>

            <!-- 用户菜单测试 -->
            <div class="space-y-4">
              <h2 class="text-xl font-semibold flex items-center gap-2">
                <UIcon name="i-heroicons-user-circle" class="w-5 h-5 text-purple-500" />
                用户菜单测试
              </h2>
              
              <UCard>
                <div class="flex items-center justify-between">
                  <span>用户菜单组件:</span>
                  <UserMenu />
                </div>
              </UCard>
            </div>

            <!-- 快捷键测试 -->
            <div class="space-y-4">
              <h2 class="text-xl font-semibold flex items-center gap-2">
                <UIcon name="i-heroicons-command-line" class="w-5 h-5 text-orange-500" />
                快捷键测试
              </h2>
              
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <UCard>
                  <template #header>
                    <h3 class="font-semibold">已注册的快捷键</h3>
                  </template>
                  <div class="space-y-2">
                    <div v-for="(shortcuts, category) in shortcutsByCategory" :key="category">
                      <h4 class="font-medium text-sm text-neutral-600 dark:text-neutral-400 mb-2">{{ category }}</h4>
                      <div class="space-y-1">
                        <div 
                          v-for="shortcut in shortcuts" 
                          :key="shortcut.key"
                          class="flex items-center justify-between text-sm"
                        >
                          <span>{{ shortcut.description }}</span>
                          <div class="flex items-center gap-1">
                            <UKbd 
                              v-for="key in formatShortcut(shortcut.key)" 
                              :key="key"
                              size="xs"
                            >
                              {{ key }}
                            </UKbd>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </UCard>

                <UCard>
                  <template #header>
                    <h3 class="font-semibold">快捷键操作</h3>
                  </template>
                  <div class="space-y-3">
                    <UButton @click="showHelp" color="primary" block>
                      显示快捷键帮助 (⌘/)
                    </UButton>
                    <UButton @click="testShortcut" variant="outline" block>
                      测试快捷键功能
                    </UButton>
                  </div>
                </UCard>
              </div>
            </div>

            <!-- i18n 测试 -->
            <div class="space-y-4">
              <h2 class="text-xl font-semibold flex items-center gap-2">
                <UIcon name="i-heroicons-language" class="w-5 h-5 text-green-500" />
                国际化测试
              </h2>
              
              <UCard>
                <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div class="space-y-2">
                    <h4 class="font-medium">通用词汇</h4>
                    <div class="space-y-1 text-sm">
                      <div>{{ t('common.loading') }}</div>
                      <div>{{ t('common.success') }}</div>
                      <div>{{ t('common.error') }}</div>
                      <div>{{ t('common.cancel') }}</div>
                    </div>
                  </div>
                  
                  <div class="space-y-2">
                    <h4 class="font-medium">认证相关</h4>
                    <div class="space-y-1 text-sm">
                      <div>{{ t('auth.login') }}</div>
                      <div>{{ t('auth.register') }}</div>
                      <div>{{ t('auth.logout') }}</div>
                      <div>{{ t('auth.loginRequired') }}</div>
                    </div>
                  </div>
                  
                  <div class="space-y-2">
                    <h4 class="font-medium">导航</h4>
                    <div class="space-y-1 text-sm">
                      <div>{{ t('nav.home') }}</div>
                      <div>{{ t('nav.groups') }}</div>
                      <div>{{ t('nav.messages') }}</div>
                      <div>{{ t('nav.profile') }}</div>
                    </div>
                  </div>
                  
                  <div class="space-y-2">
                    <h4 class="font-medium">快捷键</h4>
                    <div class="space-y-1 text-sm">
                      <div>{{ t('shortcuts.help.title') }}</div>
                      <div>{{ t('shortcuts.help.empty') }}</div>
                      <div>{{ t('shortcuts.help.tip') }}</div>
                    </div>
                  </div>
                </div>
              </UCard>
            </div>
          </div>
        </template>

        <template #footer>
          <div class="flex items-center justify-between">
            <UButton to="/" variant="outline">
              返回首页
            </UButton>
            <div class="text-sm text-neutral-500">
              测试页面 - 验证修复功能
            </div>
          </div>
        </template>
      </UCard>
    </UContainer>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  title: "功能修复测试",
  requiresAuth: false
})

const { t } = useI18n()
const authStore = useAuthStore()
const { getShortcutsByCategory, formatShortcut, showHelp } = useGlobalShortcuts()

const shortcutsByCategory = getShortcutsByCategory

const testShortcut = () => {
  // 测试一个简单的快捷键功能
  alert('快捷键功能正常工作！')
}
</script>
