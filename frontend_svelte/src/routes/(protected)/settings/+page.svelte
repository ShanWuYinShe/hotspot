<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import ThemeToggle from '$lib/components/common/theme-toggle.svelte';
  import User from 'lucide-svelte/icons/user';
  import Bell from 'lucide-svelte/icons/bell';
  import Shield from 'lucide-svelte/icons/shield';
  import Lock from 'lucide-svelte/icons/lock';
  import LogOut from 'lucide-svelte/icons/log-out';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Share2 from 'lucide-svelte/icons/share-2';
  import ShieldAlert from 'lucide-svelte/icons/shield-alert';
  import Palette from 'lucide-svelte/icons/palette';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';

  const PREVIEW_ACCENT: Record<ThemeFamily, string> = {
    vibrant: '#6366f1',
    forest: '#3D7A55',
    neu: '#D4845A'
  };

  const THEME_NAMES: Record<ThemeFamily, string> = {
    vibrant: '活力',
    forest: '自然',
    neu: '质感'
  };

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

  const notificationItems = [
    { key: 'like', label: '点赞通知' },
    { key: 'follow', label: '关注通知' },
    { key: 'comment', label: '评论通知' },
    { key: 'nearby', label: '附近动态' }
  ];

  let notifications = $state<Record<string, boolean>>({
    like: true,
    follow: true,
    comment: true,
    nearby: false
  });

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
  <!-- Header with back button -->
  <div
    class="sticky top-0 z-10 flex items-center gap-3 px-5 py-3"
    style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
  >
    <button
      type="button"
      class="flex flex-shrink-0 items-center justify-center rounded-full"
      style="width: 38px; height: 38px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
      onclick={() => history.back()}
      aria-label="返回"
    >
      <ArrowLeft size={16} color="var(--neu-text-muted)" strokeWidth={1.5} />
    </button>
    <h1 class="text-[18px] tracking-tight" style="color: var(--neu-text);">设置</h1>
  </div>

  <div class="space-y-3 px-5 pb-6">
    <!-- User card -->
    {#if authStore.currentAccount}
      <div
        class="flex items-center gap-3 rounded-[22px] px-4 py-3.5"
        style="background: var(--neu-bg); box-shadow: 7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
      >
        <div
          class="flex-shrink-0 overflow-hidden rounded-full"
          style="width: 50px; height: 50px; padding: 2px; border: 1px solid var(--neu-border); box-shadow: var(--neu-shadow-sm);"
        >
          {#if authStore.currentAccount.avatar_url}
            <img
              src={authStore.currentAccount.avatar_url}
              alt={authStore.currentAccount.nickname}
              class="h-full w-full rounded-full object-cover"
            />
          {:else}
            <div
              class="flex h-full w-full items-center justify-center rounded-full"
              style="background: var(--neu-accent-primary);"
            >
              <User size={20} color="var(--neu-active-text)" />
            </div>
          {/if}
        </div>
        <div class="min-w-0">
          <p style="font-size: 15px; color: var(--neu-text); font-weight: 500;">
            {authStore.currentAccount.nickname || '用户'}
          </p>
          <p style="font-size: 11px; color: var(--neu-text-muted);">
            @{authStore.currentAccount.user_id.slice(0, 8)}
          </p>
        </div>
      </div>
    {/if}

    <!-- Theme picker (standalone section, matching Figma Make) -->
    <p
      class="px-1"
      style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--neu-text-muted); margin-top: 8px;"
    >
      主题风格
    </p>
    <div class="flex gap-2.5">
      {#each THEME_FAMILY_OPTIONS as option (option.family)}
        {@const accent = PREVIEW_ACCENT[option.family]}
        {@const isSelected = currentThemeFamily === option.family}
        <button
          type="button"
          class="relative flex flex-1 flex-col items-center gap-3 rounded-[20px] py-4 text-center transition-all"
          style="background: var(--neu-bg); {isSelected
            ? 'box-shadow: 0 0 0 2.5px ' +
              accent +
              ', 5px 5px 12px rgba(0,0,0,0.10); border: 1.5px solid ' +
              accent +
              ';'
            : 'box-shadow: 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);'}"
          onclick={() => setThemeFamily(option.family)}
        >
          <!-- Mini phone frame preview -->
          <div
            style="width: 60px; height: 80px; border-radius: 12px; background: var(--neu-bg); border: 1px solid {accent}22; box-shadow: 0 2px 8px rgba(0,0,0,0.12); position: relative; overflow: hidden;"
          >
            <div
              style="position: absolute; bottom: 0; left: 0; right: 0; display: flex; justify-content: space-around; align-items: center; height: 20px; background: var(--neu-bg); border-top: 1px solid {accent}30;"
            >
              {#each [0, 1, 2] as i (i)}
                <div
                  style="width: {i === 1 ? '14px' : '10px'}; height: {i === 1
                    ? '14px'
                    : '8px'}; border-radius: {i === 1 ? '7px' : '4px'}; background: {i === 1
                    ? accent
                    : `${accent}55`};"
                ></div>
              {/each}
            </div>
            <div style="display: flex; flex-direction: column; gap: 6px; padding: 12px 8px 0 8px;">
              {#each [80, 60, 70] as w, i (i)}
                <div
                  style="width: {w}%; height: 5px; border-radius: 3px; background: {i === 0
                    ? `${accent}90`
                    : `${accent}35`};"
                ></div>
              {/each}
            </div>
          </div>
          <div style="display: flex; flex-direction: column; align-items: center; gap: 2px;">
            <span style="font-size: 12px; color: {accent};">{THEME_NAMES[option.family]}</span>
            <span style="font-size: 10px; text-align: center; line-height: 1.4; color: {accent}99;">
              {#if option.family === 'vibrant'}霓虹活力{:else if option.family === 'forest'}森林极简{:else}柔和质感{/if}
            </span>
          </div>
          {#if isSelected}
            <div
              style="display: flex; align-items: center; justify-content: center; border-radius: 50%; width: 20px; height: 20px; background: {accent};"
            >
              <svg
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="#FFFFFF"
                stroke-width="3"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <polyline points="20 6 9 17 4 12"></polyline>
              </svg>
            </div>
          {/if}
        </button>
      {/each}
    </div>

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
      <div class="flex items-center justify-between px-4 py-3.5">
        <div class="flex items-center gap-3">
          <div
            class="flex flex-shrink-0 items-center justify-center rounded-full"
            style="width: 36px; height: 36px; box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border);"
          >
            <Palette size={15} strokeWidth={1.5} color="var(--neu-accent-primary)" />
          </div>
          <div>
            <p style="font-size: 14px; color: var(--neu-text);">深色模式</p>
            <p style="font-size: 11px; color: var(--neu-text-muted);">跟随系统 / 浅色 / 深色</p>
          </div>
        </div>
        <ThemeToggle />
      </div>
    </div>

    {#each settingsSections as section (section.title)}
      <p
        class="px-1"
        style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--neu-text-muted); margin-top: 8px;"
      >
        {section.title}
      </p>
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

    <!-- Notification toggles -->
    <p
      class="px-1"
      style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--neu-text-muted); margin-top: 8px;"
    >
      通知偏好
    </p>
    <div
      class="overflow-hidden rounded-[22px]"
      style="background: var(--neu-bg); box-shadow: 7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
    >
      {#each notificationItems as item, i (item.key)}
        {#if i > 0}
          <div class="mx-4" style="height: 1px; background: var(--neu-border);"></div>
        {/if}
        <div class="flex items-center justify-between px-4 py-3.5">
          <span style="font-size: 14px; color: var(--neu-text);">{item.label}</span>
          <button
            type="button"
            role="switch"
            aria-checked={notifications[item.key]}
            class="relative flex-shrink-0"
            style="width: 50px; height: 28px; border-radius: 14px; border: none; cursor: pointer; transition: background 0.25s ease; background: {notifications[
              item.key
            ]
              ? 'var(--neu-accent-primary)'
              : 'rgba(120,120,128,0.22)'};"
            onclick={() => (notifications[item.key] = !notifications[item.key])}
          >
            <span
              style="position: absolute; top: 3px; width: 22px; height: 22px; border-radius: 11px; background: #FFFFFF; box-shadow: 0 1px 4px rgba(0,0,0,0.26); transition: transform 0.25s ease; left: {notifications[
                item.key
              ]
                ? '25px'
                : '3px'};"
            ></span>
          </button>
        </div>
      {/each}
    </div>

    <!-- Log Out -->
    <div class="mt-5">
      <button
        class="flex w-full items-center gap-3.5 rounded-[22px] px-4 py-3.5 text-left transition-colors"
        style="background: var(--neu-bg); box-shadow: 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
        onclick={handleLogout}
      >
        <div
          class="flex flex-shrink-0 items-center justify-center rounded-full"
          style="width: 36px; height: 36px; box-shadow: var(--neu-shadow-sm); border: 1px solid var(--neu-border);"
        >
          <LogOut size={15} strokeWidth={1.5} color="var(--neu-accent-primary)" />
        </div>
        <span style="font-size: 14px; color: var(--neu-accent-primary);">退出登录</span>
      </button>
    </div>
  </div>
</div>
