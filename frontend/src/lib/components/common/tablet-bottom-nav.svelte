<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { interactive } from '$lib/actions/interactive';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import { bottomTabs, isNavActive, type BottomTab } from '$lib/config/navigation';
  import Plus from 'lucide-svelte/icons/plus';
  import X from 'lucide-svelte/icons/x';
  import LogIn from 'lucide-svelte/icons/log-in';

  // 未登录时隐藏需要登录的整页功能，用「登录」入口占位（与移动端一致）。
  const loginTab: BottomTab = {
    key: 'login',
    labelKey: 'nav.login',
    icon: LogIn,
    href: '/auth'
  };
  const visibleTabs = $derived(
    authStore.isAuthenticated
      ? bottomTabs
      : [...bottomTabs.filter((tab) => !tab.requiresAuth), loginTab]
  );
  const navItems = $derived(visibleTabs);
  const currentPath = $derived(page.url.pathname);

  function isActive(tab: BottomTab) {
    return isNavActive(tab.href, tab.exact, currentPath);
  }

  function navigate(tab: BottomTab) {
    if (!tab.href) return;
    if (tab.key === 'login') {
      uiStore.openLoginModalWithRedirect({});
      return;
    }
    void goto(tab.href);
  }

  // 与移动端一致：底栏「消息」角标合并未读消息 + 未读通知。
  const inboxBadge = $derived(
    authStore.isAuthenticated ? messageStore.totalUnreadCount + notificationStore.unreadCount : 0
  );

  function tabBadge(tab: BottomTab): number {
    return tab.badge === 'inbox' ? inboxBadge : 0;
  }

  // Shared sliding pill — glides (and resizes) between tabs like the React
  // reference's Framer Motion `layoutId` indicator, instead of fading per tab.
  let barEl = $state<HTMLDivElement | undefined>();
  let tabEls = $state<(HTMLButtonElement | undefined)[]>([]);
  let pill = $state({ x: 0, y: 4, w: 0, h: 0 });
  const activeIndex = $derived(navItems.findIndex((it) => isActive(it)));
  const pillVisible = $derived(activeIndex >= 0 && pill.w > 0);

  function measurePill() {
    const el = activeIndex >= 0 ? tabEls[activeIndex] : undefined;
    if (!barEl || !el) return;
    pill = {
      x: el.offsetLeft + 6,
      y: el.offsetTop + 4,
      w: Math.max(0, el.offsetWidth - 12),
      h: Math.max(0, el.offsetHeight - 8)
    };
  }

  $effect(() => {
    // Re-measure when the active tab changes or the route updates. Never write
    // `pill` here — doing so while reading it would re-trigger this effect and
    // blow the update depth (effect_update_depth_exceeded).
    activeIndex;
    currentPath;
    if (activeIndex < 0) return;
    const raf = requestAnimationFrame(measurePill);
    const onResize = () => measurePill();
    window.addEventListener('resize', onResize);
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener('resize', onResize);
    };
  });
</script>

<div
  class="flex w-full shrink-0 justify-center"
  style="padding: 8px 24px; padding-bottom: max(env(safe-area-inset-bottom, 16px), 16px); background: transparent;"
>
  <div
    bind:this={barEl}
    class="flex items-center"
    style="position: relative; width: 100%; max-width: 520px; border-radius: var(--radius-pill); background: var(--glass-bg); backdrop-filter: blur(var(--theme-glass-blur)) saturate(var(--theme-glass-saturate)); -webkit-backdrop-filter: blur(var(--theme-glass-blur)) saturate(var(--theme-glass-saturate)); box-shadow: var(--theme-glass-shadow); border: 1px solid var(--glass-border); padding: 8px 16px; gap: 6px;"
  >
    <span
      aria-hidden="true"
      style="position: absolute; left: {pill.x}px; top: {pill.y}px; width: {pill.w}px; height: {pill.h}px; border-radius: 20px; background: var(--brand); opacity: {pillVisible
        ? 0.14
        : 0}; pointer-events: none; z-index: 0; will-change: left, width; transition: left 0.42s cubic-bezier(0.34, 1.32, 0.46, 1), width 0.42s cubic-bezier(0.34, 1.32, 0.46, 1), top 0.42s cubic-bezier(0.34, 1.32, 0.46, 1), height 0.42s cubic-bezier(0.34, 1.32, 0.46, 1), opacity 0.22s ease;"
    ></span>
    {#each navItems.slice(0, 2) as item, i (item.key)}
      <button
        bind:this={tabEls[i]}
        use:interactive={{ type: 'btn' }}
        class="relative flex flex-col items-center justify-center outline-none select-none transition-transform duration-150 active:scale-95"
        style="flex: 1; min-width: 0; padding: 9px 0; gap: 5px;"
        onclick={() => navigate(item)}
        aria-label={t(item.labelKey)}
        aria-current={isActive(item) ? 'page' : undefined}
      >
        <span
          style="position: relative; z-index: 1; display: flex; align-items: center; justify-content: center;"
        >
          <item.icon
            size={24}
            strokeWidth={isActive(item) ? 2.1 : 1.5}
            color={isActive(item) ? 'var(--brand)' : 'var(--text-secondary)'}
            style="transition: color 0.22s ease;"
          />
        </span>
        <span
          class="text-caption"
          style="line-height: 1; color: {isActive(item)
            ? 'var(--brand)'
            : 'var(--text-secondary)'}; font-weight: {isActive(item)
            ? 600
            : 400}; transition: color 0.22s ease; letter-spacing: 0.02em; position: relative; z-index: 1"
        >
          {t(item.labelKey)}
        </span>
      </button>
    {/each}

    <!-- Center FAB -->
    <div class="flex shrink-0 items-center justify-center" style="width: 64px;">
      <button
        use:interactive={{ type: 'btn' }}
        class="flex items-center justify-center rounded-full outline-none select-none"
        style="width: 48px; height: 48px; background: {uiStore.showCreateSheet
          ? 'var(--theme-text-muted)'
          : 'var(--brand)'}; box-shadow: {uiStore.showCreateSheet
          ? 'none'
          : 'var(--theme-fab-glow)'}; transition: background 0.22s, box-shadow 0.22s;"
        onclick={() => uiStore.openCreateSheet()}
        aria-label={uiStore.showCreateSheet ? '关闭' : '发布'}
      >
        {#if uiStore.showCreateSheet}
          <span style="display: flex; animation: fabIconIn 0.18s ease-out both;">
            <X size={19} strokeWidth={2.2} color="var(--text-on-accent)" />
          </span>
        {:else}
          <span style="display: flex; animation: fabIconIn 0.18s ease-out both;">
            <Plus size={19} strokeWidth={2.2} color="var(--text-on-accent)" />
          </span>
        {/if}
      </button>
    </div>

    {#each navItems.slice(2) as item, i (item.key)}
      <button
        bind:this={tabEls[i + 2]}
        use:interactive={{ type: 'btn' }}
        class="relative flex flex-col items-center justify-center outline-none select-none transition-transform duration-150 active:scale-95"
        style="flex: 1; min-width: 0; padding: 9px 0; gap: 5px;"
        onclick={() => navigate(item)}
        aria-label={t(item.labelKey)}
        aria-current={isActive(item) ? 'page' : undefined}
      >
        <span
          style="position: relative; z-index: 1; display: flex; align-items: center; justify-content: center;"
        >
          <item.icon
            size={24}
            strokeWidth={isActive(item) ? 2.1 : 1.5}
            color={isActive(item) ? 'var(--brand)' : 'var(--text-secondary)'}
            style="transition: color 0.22s ease;"
          />
          {#if tabBadge(item) > 0}
            <span
              class="text-caption absolute flex items-center justify-center rounded-full"
              style="top: -4px; right: -6px; height: 20px; min-width: 20px; padding: 0 6px; background: var(--brand); color: var(--text-on-accent); font-weight: 600; line-height: 20px"
            >
              {tabBadge(item) > 99 ? '99+' : tabBadge(item)}
            </span>
          {/if}
        </span>
        <span
          class="text-caption"
          style="line-height: 1; color: {isActive(item)
            ? 'var(--brand)'
            : 'var(--text-secondary)'}; font-weight: {isActive(item)
            ? 600
            : 400}; transition: color 0.22s ease; letter-spacing: 0.02em; position: relative; z-index: 1"
        >
          {t(item.labelKey)}
        </span>
      </button>
    {/each}
  </div>
</div>
