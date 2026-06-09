<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
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
  import Moon from 'lucide-svelte/icons/moon';

  const PREVIEW_ACCENT: Record<ThemeFamily, string> = {
    indigo: '#6366f1',
    forest: '#3D7A55',
    sand: '#D4845A'
  };

  const THEME_NAMES: Record<ThemeFamily, string> = {
    indigo: '活力',
    forest: '自然',
    sand: '质感'
  };

  const settingsSections = [
    {
      title: '账户',
      icon: User,
      items: [
        {
          icon: User,
          label: '个人资料',
          href: '/profile',
          description: '修改头像、昵称、简介',
          color: 'var(--brand)'
        },
        {
          icon: Lock,
          label: '修改密码',
          href: '/settings/password',
          description: '更改登录密码',
          color: 'var(--theme-warning)'
        },
        {
          icon: Shield,
          label: '会话管理',
          href: '/settings/sessions',
          description: '管理登录设备',
          color: 'var(--brand)'
        }
      ]
    },
    {
      title: '隐私',
      icon: Shield,
      items: [
        {
          icon: Shield,
          label: '隐私设置',
          href: '/settings/privacy',
          description: '谁可以看到你的信息',
          color: 'var(--text-tertiary)'
        }
      ]
    },
    {
      title: '通知',
      icon: Bell,
      items: [
        {
          icon: Bell,
          label: '通知设置',
          href: '/settings/notifications',
          description: '管理推送通知',
          color: 'var(--brand)'
        }
      ]
    },
    {
      title: '记录与分享',
      icon: Share2,
      items: [
        {
          icon: Share2,
          label: '我的分享',
          href: '/settings/shares',
          description: '查看并撤销已创建分享',
          color: 'var(--theme-success)'
        },
        {
          icon: ShieldAlert,
          label: '我的举报记录',
          href: '/settings/reports',
          description: '跟踪举报处理进度',
          color: 'var(--brand)'
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

  let activeSection = $state<string>('appearance');

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

{#if platformStore.isDesktop}
  <!-- Desktop: two-column split layout -->
  <div class="flex h-full">
    <!-- Left navigation panel -->
    <div
      class="flex flex-shrink-0 flex-col overflow-y-auto no-scrollbar"
      style="width: 260px; border-right: 1px solid var(--border-theme); padding: 20px 16px;"
    >
      <!-- User card -->
      {#if authStore.currentAccount}
        <div
          class="flex items-center gap-3 rounded-[18px] px-3.5 py-3"
          style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); margin-bottom: 16px;"
        >
          <div
            class="flex-shrink-0 overflow-hidden rounded-full"
            style="width: 42px; height: 42px; padding: 2px; border: 1px solid var(--border-theme);"
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
                style="background: var(--brand);"
              >
                <User size={18} color="var(--text-on-accent)" />
              </div>
            {/if}
          </div>
          <div class="min-w-0">
            <p style="font-size: 14px; color: var(--text-primary); font-weight: 500;">
              {authStore.currentAccount.nickname || '用户'}
            </p>
            <p style="font-size: 11px; color: var(--text-secondary);">
              @{authStore.currentAccount.user_id.slice(0, 8)}
            </p>
          </div>
        </div>
      {/if}

      <!-- Navigation items -->
      <div class="flex flex-col gap-1">
        <button
          type="button"
          class="flex items-center gap-2.5 rounded-[14px] px-3 py-2.5 text-left transition-all"
          style="background: {activeSection === 'appearance'
            ? 'var(--brand)'
            : 'transparent'}; color: {activeSection === 'appearance'
            ? 'var(--text-on-accent)'
            : 'var(--text-primary)'};"
          onclick={() => (activeSection = 'appearance')}
        >
          <Palette size={16} strokeWidth={1.5} />
          <span style="font-size: 13px; font-weight: 500;">外观与主题</span>
        </button>

        {#each settingsSections as section (section.title)}
          <button
            type="button"
            class="flex items-center gap-2.5 rounded-[14px] px-3 py-2.5 text-left transition-all"
            style="background: {activeSection === section.title
              ? 'var(--brand)'
              : 'transparent'}; color: {activeSection === section.title
              ? 'var(--text-on-accent)'
              : 'var(--text-primary)'};"
            onclick={() => (activeSection = section.title)}
          >
            <section.icon size={16} strokeWidth={1.5} />
            <span style="font-size: 13px; font-weight: 500;">{section.title}</span>
          </button>
        {/each}

        <button
          type="button"
          class="flex items-center gap-2.5 rounded-[14px] px-3 py-2.5 text-left transition-all"
          style="background: {activeSection === 'notifications'
            ? 'var(--brand)'
            : 'transparent'}; color: {activeSection === 'notifications'
            ? 'var(--text-on-accent)'
            : 'var(--text-primary)'};"
          onclick={() => (activeSection = 'notifications')}
        >
          <Bell size={16} strokeWidth={1.5} />
          <span style="font-size: 13px; font-weight: 500;">通知偏好</span>
        </button>

        <!-- Logout -->
        <div style="margin-top: 12px; height: 1px; background: var(--border-theme);"></div>
        <button
          type="button"
          class="flex items-center gap-2.5 rounded-[14px] px-3 py-2.5 text-left transition-all"
          style="color: var(--brand);"
          onclick={handleLogout}
        >
          <LogOut size={16} strokeWidth={1.5} />
          <span style="font-size: 13px; font-weight: 500;">退出登录</span>
        </button>
      </div>
    </div>

    <!-- Right content panel -->
    <div class="flex-1 overflow-y-auto no-scrollbar" style="padding: 28px 32px;">
      {#if activeSection === 'appearance'}
        <!-- Appearance section -->
        <div class="mb-6">
          <h2
            style="font-size: 18px; font-weight: 600; color: var(--text-primary); margin-bottom: 4px;"
          >
            外观与主题
          </h2>
          <p style="font-size: 12px; color: var(--text-secondary);">选择你喜欢的视觉风格</p>
        </div>

        <!-- Theme picker -->
        <div class="mb-6">
          <p
            style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 10px;"
          >
            主题风格
          </p>
          <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px;">
            {#each THEME_FAMILY_OPTIONS as option (option.family)}
              {@const accent = PREVIEW_ACCENT[option.family]}
              {@const isSelected = currentThemeFamily === option.family}
              <button
                type="button"
                class="relative flex flex-col items-center gap-3 rounded-[20px] py-4 text-center transition-all"
                class:surface-flat={!isSelected}
                style="background: var(--surface-bg); {isSelected
                  ? 'box-shadow: 0 0 0 2.5px ' +
                    accent +
                    ', 5px 5px 12px rgba(0,0,0,0.10); border: 1.5px solid ' +
                    accent +
                    ';'
                  : ''}"
                onclick={() => setThemeFamily(option.family)}
              >
                <div
                  style="width: 60px; height: 80px; border-radius: 12px; background: var(--surface-bg); border: 1px solid {accent}22; box-shadow: var(--shadow-md); position: relative; overflow: hidden;"
                >
                  <div
                    style="position: absolute; bottom: 0; left: 0; right: 0; display: flex; justify-content: space-around; align-items: center; height: 20px; background: var(--surface-bg); border-top: 1px solid {accent}30;"
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
                  <div
                    style="display: flex; flex-direction: column; gap: 6px; padding: 12px 8px 0 8px;"
                  >
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
                  <span style="font-size: 12px; color: {accent};">{THEME_NAMES[option.family]}</span
                  >
                  <span
                    style="font-size: 10px; text-align: center; line-height: 1.4; color: {accent}99;"
                  >
                    {#if option.family === 'indigo'}霓虹活力{:else if option.family === 'forest'}森林极简{:else}柔和质感{/if}
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
        </div>

        <!-- Dark mode toggle -->
        <div>
          <p
            style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 10px;"
          >
            深色模式
          </p>
          <div
            class="flex items-center justify-between rounded-[18px] px-4 py-3.5"
            style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
          >
            <div class="flex items-center gap-3">
              <div
                class="flex flex-shrink-0 items-center justify-center rounded-full"
                style="width: 36px; height: 36px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
              >
                <Moon size={15} strokeWidth={1.5} color="var(--brand)" />
              </div>
              <div>
                <p style="font-size: 14px; color: var(--text-primary);">深色模式</p>
                <p style="font-size: 11px; color: var(--text-secondary);">跟随系统 / 浅色 / 深色</p>
              </div>
            </div>
            <ThemeToggle />
          </div>
        </div>
      {:else if activeSection === 'notifications'}
        <!-- Notifications section -->
        <div class="mb-6">
          <h2
            style="font-size: 18px; font-weight: 600; color: var(--text-primary); margin-bottom: 4px;"
          >
            通知偏好
          </h2>
          <p style="font-size: 12px; color: var(--text-secondary);">管理你收到的通知类型</p>
        </div>
        <div
          class="overflow-hidden rounded-[18px]"
          style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
        >
          {#each notificationItems as item, i (item.key)}
            {#if i > 0}
              <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
            {/if}
            <div class="flex items-center justify-between px-4 py-3.5">
              <span style="font-size: 14px; color: var(--text-primary);">{item.label}</span>
              <button
                type="button"
                role="switch"
                aria-checked={notifications[item.key]}
                class="relative flex-shrink-0"
                style="width: 50px; height: 28px; border-radius: 14px; border: none; cursor: pointer; transition: background 0.25s ease; background: {notifications[
                  item.key
                ]
                  ? 'var(--brand)'
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
      {:else}
        <!-- Dynamic section based on activeSection -->
        {@const section = settingsSections.find((s) => s.title === activeSection)}
        {#if section}
          <div class="mb-6">
            <h2
              style="font-size: 18px; font-weight: 600; color: var(--text-primary); margin-bottom: 4px;"
            >
              {section.title}
            </h2>
          </div>
          <div
            class="overflow-hidden rounded-[18px]"
            style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
          >
            {#each section.items as item, i (item.href)}
              {#if i > 0}
                <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
              {/if}
              <a href={item.href} class="flex items-center gap-3 px-4 py-3.5 transition-colors">
                <div
                  class="flex flex-shrink-0 items-center justify-center rounded-full"
                  style="width: 36px; height: 36px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
                >
                  <item.icon size={15} strokeWidth={1.5} color={item.color} />
                </div>
                <div class="min-w-0 flex-1">
                  <p style="font-size: 14px; color: var(--text-primary);">{item.label}</p>
                  <p style="font-size: 11px; color: var(--text-secondary);">{item.description}</p>
                </div>
                <ChevronRight size={15} strokeWidth={1.5} color="var(--text-secondary)" />
              </a>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  </div>
{:else}
  <!-- Mobile / Tablet: single column layout -->
  <div class="mx-auto max-w-3xl">
    <!-- Header with back button -->
    <div
      class="sticky top-0 z-10 flex items-center gap-3 px-5 py-3"
      style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
    >
      <button
        type="button"
        class="surface-flat flex flex-shrink-0 items-center justify-center rounded-full outline-none transition-transform duration-150 active:scale-[0.88]"
        style="width: 36px; height: 36px;"
        onclick={() => history.back()}
        aria-label="返回"
      >
        <ArrowLeft size={15} strokeWidth={1.5} color="var(--text-secondary)" />
      </button>
      <h1 class="text-[18px] tracking-tight" style="color: var(--text-primary);">设置</h1>
    </div>

    <div class="space-y-3 px-5 pb-6">
      <!-- User card -->
      {#if authStore.currentAccount}
        <div class="surface-flat flex items-center gap-3 rounded-[22px] px-4 py-3.5">
          <div
            class="flex-shrink-0 overflow-hidden rounded-full"
            style="width: 50px; height: 50px; padding: 2px; border: 1px solid var(--border-theme); box-shadow: var(--shadow-sm);"
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
                style="background: var(--brand);"
              >
                <User size={20} color="var(--text-on-accent)" />
              </div>
            {/if}
          </div>
          <div class="min-w-0">
            <p style="font-size: 15px; color: var(--text-primary); font-weight: 500;">
              {authStore.currentAccount.nickname || '用户'}
            </p>
            <p style="font-size: 11px; color: var(--text-secondary);">
              @{authStore.currentAccount.user_id.slice(0, 8)}
            </p>
          </div>
        </div>
      {/if}

      <!-- Theme picker (standalone section, matching Figma Make) -->
      <p
        class="px-1"
        style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-top: 8px;"
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
            class:surface-flat={!isSelected}
            style="background: var(--surface-bg); {isSelected
              ? 'box-shadow: 0 0 0 2.5px ' +
                accent +
                ', 5px 5px 12px rgba(0,0,0,0.10); border: 1.5px solid ' +
                accent +
                ';'
              : ''}"
            onclick={() => setThemeFamily(option.family)}
          >
            <!-- Mini phone frame preview -->
            <div
              style="width: 60px; height: 80px; border-radius: 12px; background: var(--surface-bg); border: 1px solid {accent}22; box-shadow: var(--shadow-md); position: relative; overflow: hidden;"
            >
              <div
                style="position: absolute; bottom: 0; left: 0; right: 0; display: flex; justify-content: space-around; align-items: center; height: 20px; background: var(--surface-bg); border-top: 1px solid {accent}30;"
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
              <div
                style="display: flex; flex-direction: column; gap: 6px; padding: 12px 8px 0 8px;"
              >
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
              <span
                style="font-size: 10px; text-align: center; line-height: 1.4; color: {accent}99;"
              >
                {#if option.family === 'indigo'}霓虹活力{:else if option.family === 'forest'}森林极简{:else}柔和质感{/if}
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
        style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-top: 8px;"
      >
        外观
      </p>
      <div class="surface-flat overflow-hidden rounded-[22px]">
        <div class="flex items-center justify-between px-4 py-3.5">
          <div class="flex items-center gap-3">
            <div
              class="flex flex-shrink-0 items-center justify-center rounded-full"
              style="width: 36px; height: 36px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
            >
              <Palette size={15} strokeWidth={1.5} color="var(--brand)" />
            </div>
            <div>
              <p style="font-size: 14px; color: var(--text-primary);">深色模式</p>
              <p style="font-size: 11px; color: var(--text-secondary);">跟随系统 / 浅色 / 深色</p>
            </div>
          </div>
          <ThemeToggle />
        </div>
      </div>

      {#each settingsSections as section (section.title)}
        <p
          class="px-1"
          style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-top: 8px;"
        >
          {section.title}
        </p>
        <div class="overflow-hidden rounded-[22px]" style="">
          {#each section.items as item, i (item.href)}
            {#if i > 0}
              <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
            {/if}
            <a href={item.href} class="flex items-center gap-3 px-4 py-3.5 transition-colors">
              <div
                class="flex flex-shrink-0 items-center justify-center rounded-full"
                style="width: 36px; height: 36px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
              >
                <item.icon size={15} strokeWidth={1.5} color={item.color} />
              </div>
              <div class="min-w-0 flex-1">
                <p style="font-size: 14px; color: var(--text-primary);">{item.label}</p>
                <p style="font-size: 11px; color: var(--text-secondary);">{item.description}</p>
              </div>
              <ChevronRight size={15} strokeWidth={1.5} color="var(--text-secondary)" />
            </a>
          {/each}
        </div>
      {/each}

      <!-- Notification toggles -->
      <p
        class="px-1"
        style="font-size: 10px; letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-top: 8px;"
      >
        通知偏好
      </p>
      <div class="surface-flat overflow-hidden rounded-[22px]">
        {#each notificationItems as item, i (item.key)}
          {#if i > 0}
            <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
          {/if}
          <div class="flex items-center justify-between px-4 py-3.5">
            <span style="font-size: 14px; color: var(--text-primary);">{item.label}</span>
            <button
              type="button"
              role="switch"
              aria-checked={notifications[item.key]}
              class="relative flex-shrink-0"
              style="width: 50px; height: 28px; border-radius: 14px; border: none; cursor: pointer; transition: background 0.25s ease; background: {notifications[
                item.key
              ]
                ? 'var(--brand)'
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
          class="surface-flat flex w-full items-center gap-3.5 rounded-[22px] px-4 py-3.5 text-left transition-colors"
          onclick={handleLogout}
        >
          <div
            class="flex flex-shrink-0 items-center justify-center rounded-full"
            style="width: 36px; height: 36px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
          >
            <LogOut size={15} strokeWidth={1.5} color="var(--brand)" />
          </div>
          <span style="font-size: 14px; color: var(--brand);">退出登录</span>
        </button>
      </div>
    </div>
  </div>
{/if}
