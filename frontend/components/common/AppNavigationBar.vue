<template>
  <nav v-if="shouldShow" class="app-navigation" :class="navigationClass">
    <NuxtLink v-for="(item, index) in regularNavItems" :key="item.path" :to="item.path" class="app-navigation__item"
      :class="{ 'app-navigation__item--active': isActive(item.path) }" @click="handleNavClick(item)">
      <div class="app-navigation__icon">
        <NBadge v-if="item.badge" :value="item.badge" :hidden="item.badge === 0" :type="item.badgeType || 'error'">
          <Icon :name="item.icon" size="24" />
        </NBadge>
        <Icon v-else :name="item.icon" size="24" />
      </div>
      <span class="app-navigation__label">{{ item.label }}</span>
    </NuxtLink>

    <!-- 用户切换器 -->
    <div class="app-navigation__user-switcher">
      <UserSwitcher size="small" @user-switched="handleUserSwitched" @user-added="handleUserAdded"
        @upgrade-user="handleUpgradeUser" />
    </div>
  </nav>
</template>

<script setup lang="ts">
import { NBadge } from 'naive-ui'
import UserSwitcher from './UserSwitcher.vue'
import { useRoute } from 'vue-router'

interface NavItem {
  path: string
  label: string
  icon: string
  badge?: number
  badgeType?: 'success' | 'warning' | 'info' | 'default' | 'error'
}

interface AppNavigationBarProps {
  items?: NavItem[]
  position?: 'bottom' | 'top' | 'left' | 'right'
  mode?: 'compact' | 'full' | 'auto'
}

const props = withDefaults(defineProps<AppNavigationBarProps>(), {
  position: 'bottom',
  mode: 'auto',
  items: () => [
    {
      path: '/',
      label: '首页',
      icon: 'mdi:home'
    },
    {
      path: '/group',
      label: '群组',
      icon: 'mdi:account-group'
    },
    {
      path: '/messages',
      label: '消息',
      icon: 'mdi:message'
    },
    {
      path: '/map',
      label: '地图',
      icon: 'mdi:map-marker'
    },
    {
      path: '/dynamic',
      label: '动态',
      icon: 'mdi:camera'
    },
    {
      path: '/settings',
      label: '设置',
      icon: 'mdi:cog'
    }
  ]
})

const emit = defineEmits<{
  'nav-click': [item: NavItem]
  'user-switched': [userId: number]
  'user-added': [userId: number]
}>()

const route = useRoute()

const navItems = computed(() => props.items)

// 分离常规导航项（不包含用户相关项）
const regularNavItems = computed(() => navItems.value)

// 简化的显示逻辑
const shouldShow = computed(() => {
  return true
})

// CSS 类名根据模式和位置
const navigationClass = computed(() => {
  return {
    [`app-navigation--${props.position}`]: true,
    'app-navigation--compact': props.mode === 'compact',
    'app-navigation--full': props.mode === 'full',
    'app-navigation--auto': props.mode === 'auto'
  }
})

const isActive = (path: string) => {
  if (path === '/') {
    return route.path === '/'
  }
  return route.path.startsWith(path)
}

const handleNavClick = (item: NavItem) => {
  emit('nav-click', item)
}

const handleUserSwitched = (userId: number) => {
  emit('user-switched', userId)
}

const handleUserAdded = (userId: number) => {
  emit('user-added', userId)
}

const handleUpgradeUser = () => {
  navigateTo('/auth?mode=upgrade')
}
</script>

<style scoped lang="scss">
@use '~/assets/styles/mixins.scss' as *;

.app-navigation {
  position: fixed;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-primary);
  display: flex;
  align-items: center;
  padding: 8px 0;
  z-index: var(--z-navigation);
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.1);

  // 位置相关样式
  &--bottom {
    bottom: 0;
    left: 0;
    right: 0;
    justify-content: space-around;
    border-top: 1px solid var(--color-border-primary);
    border-left: none;
    border-right: none;
    border-bottom: none;
    padding-bottom: calc(8px + env(safe-area-inset-bottom));
  }

  &--top {
    top: 0;
    left: 0;
    right: 0;
    justify-content: space-around;
    border-bottom: 1px solid var(--color-border-primary);
    border-left: none;
    border-right: none;
    border-top: none;
    padding-top: calc(8px + env(safe-area-inset-top));
  }

  &--left {
    top: 0;
    left: 0;
    bottom: 0;
    flex-direction: column;
    justify-content: flex-start;
    border-right: 1px solid var(--color-border-primary);
    border-top: none;
    border-bottom: none;
    border-left: none;
    padding: 8px 0;
    width: 80px;
  }

  &--right {
    top: 0;
    right: 0;
    bottom: 0;
    flex-direction: column;
    justify-content: flex-start;
    border-left: 1px solid var(--color-border-primary);
    border-top: none;
    border-bottom: none;
    border-right: none;
    padding: 8px 0;
    width: 80px;
  }

  &__item {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-decoration: none;
    color: var(--color-text-secondary);
    min-width: 48px;
    padding: 4px 8px;
    border-radius: 8px;
    transition: all 0.3s ease;
    cursor: pointer;

    &:hover {
      background: var(--color-bg-tertiary);
      color: var(--color-primary-500);
    }

    &--active {
      color: var(--color-primary-500);
      background: var(--color-primary-50);

      .app-navigation__icon {
        transform: scale(1.1);
      }
    }
  }

  &__icon {
    margin-bottom: 2px;
    transition: transform 0.3s ease;
  }

  &__label {
    font-size: 12px;
    font-weight: 500;
    line-height: 1;
  }

  // 用户切换器样式
  &__user-switcher {
    display: flex;
    align-items: center;
    padding: 4px 8px;

    :deep(.user-switcher) {
      .current-user-trigger {
        padding: 4px 8px;
        min-height: 40px;

        .user-avatar {
          width: 24px;
          height: 24px;

          .avatar-placeholder {
            font-size: 12px;
          }
        }

        .user-info {
          .username {
            font-size: 12px;
          }

          .user-type {
            font-size: 10px;
          }
        }
      }
    }
  }

  // 桌面端适配
  &--desktop {
    padding: 12px 0;

    .app-navigation__item {
      min-width: 60px;
      padding: 8px 12px;

      .app-navigation__icon {
        margin-bottom: 4px;
      }

      .app-navigation__label {
        font-size: 14px;
      }
    }

    .app-navigation__user-switcher {
      padding: 8px 12px;

      :deep(.user-switcher) {
        .current-user-trigger {
          padding: 6px 12px;
          min-height: 48px;

          .user-avatar {
            width: 32px;
            height: 32px;

            .avatar-placeholder {
              font-size: 14px;
            }
          }

          .user-info {
            .username {
              font-size: 14px;
            }

            .user-type {
              font-size: 12px;
            }
          }
        }
      }
    }
  }

  // 侧边导航特殊样式
  &--left,
  &--right {
    .app-navigation__item {
      margin-bottom: 8px;
      min-width: 56px;
      padding: 8px 4px;
    }

    .app-navigation__label {
      font-size: 10px;
      text-align: center;
    }

    .app-navigation__user-switcher {
      margin-top: auto;
      padding: 8px 4px;
    }
  }
}

/* 响应式显示控制 - 根据屏幕尺寸自动调整 */
.app-navigation--compact {
  @include respond-above(md) {
    display: none !important;
  }
}

.app-navigation--full {
  @include respond-below(md) {
    display: none !important;
  }
}

.app-navigation--auto {
  /* 自动模式，始终显示，通过默认样式处理 */
  display: flex;
}

// 适配小屏幕
@media (max-width: 360px) {
  .app-navigation--bottom,
  .app-navigation--top {
    padding: 6px 0;

    &.app-navigation--bottom {
      padding-bottom: calc(6px + env(safe-area-inset-bottom));
    }

    &.app-navigation--top {
      padding-top: calc(6px + env(safe-area-inset-top));
    }

    .app-navigation__item {
      min-width: 40px;
      padding: 2px 4px;
    }

    .app-navigation__icon {
      margin-bottom: 1px;
    }

    .app-navigation__label {
      font-size: 10px;
    }
  }
}

// 当导航项太多时自动调整
@media (max-width: 480px) {
  .app-navigation--bottom,
  .app-navigation--top {
    .app-navigation__item {
      min-width: 36px;
      padding: 4px 2px;
    }

    .app-navigation__label {
      font-size: 11px;
    }
  }
}

// 响应式断点
@media (max-width: 768px) {
  .app-navigation--desktop {
    display: none;
  }
}

// 深色模式适配
@media (prefers-color-scheme: dark) {
  .app-navigation {
    background: var(--color-bg-primary);
    border-color: var(--color-border-secondary);
    box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.3);
  }
}
</style>
