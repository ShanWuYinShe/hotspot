<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import ThemeToggle from '$lib/components/common/theme-toggle.svelte';
  import User from 'lucide-svelte/icons/user';
  import Bell from 'lucide-svelte/icons/bell';
  import Shield from 'lucide-svelte/icons/shield';
  import Lock from 'lucide-svelte/icons/lock';
  import LogOut from 'lucide-svelte/icons/log-out';
  import Trash2 from 'lucide-svelte/icons/trash-2';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Share2 from 'lucide-svelte/icons/share-2';
  import ShieldAlert from 'lucide-svelte/icons/shield-alert';
  import Palette from 'lucide-svelte/icons/palette';
  import Check from 'lucide-svelte/icons/check';

  const settingsSections = [
    {
      title: '账户',
      items: [
        {
          icon: User,
          label: '个人资料',
          href: '/profile',
          description: '修改头像、昵称、简介',
          color: 'var(--neu-accent-primary)'
        },
        {
          icon: Lock,
          label: '修改密码',
          href: '/settings/password',
          description: '更改登录密码',
          color: 'var(--neu-warning)'
        },
        {
          icon: Shield,
          label: '会话管理',
          href: '/settings/sessions',
          description: '管理登录设备',
          color: 'var(--neu-accent-primary)'
        }
      ]
    },
    {
      title: '隐私',
      items: [
        {
          icon: Shield,
          label: '隐私设置',
          href: '/settings/privacy',
          description: '谁可以看到你的信息',
          color: 'var(--neu-fallback-text)'
        }
      ]
    },
    {
      title: '通知',
      items: [
        {
          icon: Bell,
          label: '通知设置',
          href: '/settings/notifications',
          description: '管理推送通知',
          color: 'var(--neu-accent-primary)'
        }
      ]
    },
    {
      title: '记录与分享',
      items: [
        {
          icon: Share2,
          label: '我的分享',
          href: '/settings/shares',
          description: '查看并撤销已创建分享',
          color: 'var(--neu-success)'
        },
        {
          icon: ShieldAlert,
          label: '我的举报记录',
          href: '/settings/reports',
          description: '跟踪举报处理进度',
          color: 'var(--neu-accent-primary)'
        }
      ]
    }
  ];

  const currentThemeFamily = $derived(themeStore.themeFamily);

  function handleLogout() {
    authStore.logout();
    toast.success('已退出登录');
  }

  function setThemeFamily(themeFamily: ThemeFamily) {
    themeStore.setThemeFamily(themeFamily);
  }
</script>

<svelte:head>
  <title>设置 - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-3xl">
  <!-- Header -->
  <div class="mb-5 px-5 pt-3">
    <h1 style="font-size: 22px; color: var(--neu-text);">设置</h1>
    <p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">
      管理账户、安全、隐私与外观
    </p>
  </div>

  <div class="space-y-3 px-5 pb-6">
    {#each settingsSections as section (section.title)}
      <!-- Section label -->
      <p
        class="px-1"
        style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--neu-text-muted); margin-top: 8px;"
      >
        {section.title}
      </p>
      <!-- Section card -->
      <div
        class="overflow-hidden rounded-[22px]"
        style="background: var(--neu-bg); box-shadow: 7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
      >
        {#each section.items as item, i (item.href)}
          {#if i > 0}
            <div class="mx-4" style="height: 1px; background: var(--neu-border);"></div>
          {/if}
          <a href={item.href} class="flex items-center gap-3 px-4 py-3.5 transition-colors">
            <div
              class="flex flex-shrink-0 items-center justify-center rounded-full"
              style="width: 36px; height: 36px; box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border);"
            >
              <item.icon size={15} strokeWidth={1.5} color={item.color} />
            </div>
            <div class="min-w-0 flex-1">
              <p style="font-size: 14px; color: var(--neu-text);">{item.label}</p>
              <p style="font-size: 11px; color: var(--neu-text-muted);">{item.description}</p>
            </div>
            <ChevronRight size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
          </a>
        {/each}
      </div>
    {/each}

    <!-- Appearance -->
    <p
      class="px-1"
      style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--neu-text-muted); margin-top: 8px;"
    >
      外观
    </p>
    <div
      class="overflow-hidden rounded-[22px]"
      style="background: var(--neu-bg); box-shadow: 7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
    >
      <!-- Dark mode toggle -->
      <div class="flex items-center justify-between px-4 py-3.5">
        <div class="flex items-center gap-3">
          <div
            class="flex flex-shrink-0 items-center justify-center rounded-full"
            style="width: 36px; height: 36px; box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border);"
          >
            <Palette size={15} strokeWidth={1.5} color="var(--neu-accent-primary)" />
          </div>
          <div>
            <p style="font-size: 14px; color: var(--neu-text);">{t('sidebar.toggleTheme')}</p>
            <p style="font-size: 11px; color: var(--neu-text-muted);">
              {t('sidebar.systemMode')} / {t('sidebar.light')} / {t('sidebar.dark')}
            </p>
          </div>
        </div>
        <ThemeToggle />
      </div>

      <div class="mx-4" style="height: 1px; background: var(--neu-border);"></div>

      <!-- Theme family picker -->
      <div class="grid grid-cols-3 gap-2.5 p-4">
        {#each THEME_FAMILY_OPTIONS as option (option.family)}
          <button
            type="button"
            class="relative rounded-[20px] px-3 py-4 text-left transition-all"
            style={currentThemeFamily === option.family
              ? 'background: var(--neu-accent-primary); box-shadow: 0 4px 14px var(--neu-glow);'
              : 'background: var(--neu-bg); box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border);'}
            onclick={() => setThemeFamily(option.family)}
          >
            <p
              style="font-size: 12px; font-weight: 600; color: {currentThemeFamily === option.family
                ? 'var(--neu-active-text)'
                : 'var(--neu-text)'};"
            >
              {t(option.labelKey)}
            </p>
            <p
              style="font-size: 11px; margin-top: 2px; color: {currentThemeFamily === option.family
                ? 'var(--neu-active-text)'
                : 'var(--neu-text-muted)'};"
            >
              {t(option.descriptionKey)}
            </p>
            {#if currentThemeFamily === option.family}
              <div class="absolute right-2 top-2">
                <Check size={14} color="var(--neu-active-text)" />
              </div>
            {/if}
          </button>
        {/each}
      </div>
    </div>

    <!-- Danger zone -->
    <p
      class="px-1"
      style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--neu-text-muted); margin-top: 8px;"
    >
      危险操作
    </p>
    <div
      class="overflow-hidden rounded-[22px]"
      style="background: var(--neu-bg); box-shadow: 7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
    >
      <button
        class="flex w-full items-center gap-3 px-4 py-3.5 text-left transition-colors"
        onclick={handleLogout}
      >
        <div
          class="flex flex-shrink-0 items-center justify-center rounded-full"
          style="width: 36px; height: 36px; box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border);"
        >
          <LogOut size={15} strokeWidth={1.5} color="var(--neu-error)" />
        </div>
        <span style="font-size: 14px; color: var(--neu-error);">退出登录</span>
      </button>
      <div class="mx-4" style="height: 1px; background: var(--neu-border);"></div>
      <button class="flex w-full items-center gap-3 px-4 py-3.5 text-left transition-colors">
        <div
          class="flex flex-shrink-0 items-center justify-center rounded-full"
          style="width: 36px; height: 36px; box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border);"
        >
          <Trash2 size={15} strokeWidth={1.5} color="var(--neu-error)" />
        </div>
        <span style="font-size: 14px; color: var(--neu-error);">删除账户</span>
      </button>
    </div>
  </div>
</div>
