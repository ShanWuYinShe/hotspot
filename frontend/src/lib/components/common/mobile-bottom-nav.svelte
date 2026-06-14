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

  const navItems = bottomTabs;
  const currentPath = $derived(page.url.pathname);

  function isActive(tab: BottomTab) {
    return isNavActive(tab.href, tab.exact, currentPath);
  }

  function navigate(tab: BottomTab) {
    if (!tab.href) return;
    if (tab.requiresAuth && !authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ url: tab.href });
      return;
    }
    void goto(tab.href);
  }

  // 通知在移动端归属于「消息」收件箱（页内通过铃铛进入通知页），
  // 因此底栏「消息」角标合并未读消息 + 未读通知。
  const inboxBadge = $derived(
    authStore.isAuthenticated ? messageStore.totalUnreadCount + notificationStore.unreadCount : 0
  );

  function tabBadge(tab: BottomTab): number {
    return tab.badge === 'inbox' ? inboxBadge : 0;
  }

  // Shared sliding pill — mirrors the React reference's Framer Motion `layoutId`
  // behaviour so the active indicator glides between tabs instead of fading per tab.
  let rowEl = $state<HTMLDivElement | undefined>();
  let tabEls = $state<(HTMLButtonElement | undefined)[]>([]);
  let pill = $state({ x: 0, y: 0, w: 0, h: 0 });
  const activeIndex = $derived(navItems.findIndex((it) => isActive(it)));
  const pillVisible = $derived(activeIndex >= 0 && pill.w > 0);

  function measureSlider() {
    const el = tabEls[activeIndex];
    if (!rowEl || !el) return;
    pill = {
      x: el.offsetLeft + 4,
      y: el.offsetTop + 4,
      w: Math.max(0, el.offsetWidth - 8),
      h: Math.max(0, el.offsetHeight - 8)
    };
  }

  $effect(() => {
    // Re-measure whenever the active tab changes or the route updates.
    activeIndex;
    currentPath;
    if (activeIndex < 0) return;
    const raf = requestAnimationFrame(measureSlider);
    const onResize = () => measureSlider();
    window.addEventListener('resize', onResize);
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener('resize', onResize);
    };
  });
</script>

<nav class="mobile-bottom-nav" aria-label="Mobile navigation">
  <div
    bind:this={rowEl}
    class="flex items-center justify-between px-2"
    style="position: relative; padding-top: 8px; padding-bottom: max(20px, env(safe-area-inset-bottom, 20px));"
  >
    <span
      class="mobile-bottom-nav__slider {pillVisible ? 'visible' : ''}"
      style="left: {pill.x}px; top: {pill.y}px; width: {pill.w}px; height: {pill.h}px;"
      aria-hidden="true"
    ></span>
    {#each navItems.slice(0, 2) as item, i (item.key)}
      <button
        bind:this={tabEls[i]}
        use:interactive={{ type: 'btn' }}
        class="mobile-bottom-nav__tab {isActive(item) ? 'active' : ''}"
        onclick={() => navigate(item)}
        aria-label={t(item.labelKey)}
        aria-current={isActive(item) ? 'page' : undefined}
      >
        <span
          class="mobile-bottom-nav__tab-icon"
          style="position: relative; z-index: 1; display: flex; align-items: center; justify-content: center;"
        >
          <item.icon
            size={22}
            strokeWidth={isActive(item) ? 2.1 : 1.5}
            color={isActive(item) ? 'var(--brand)' : 'var(--text-secondary)'}
            style="transition: color 0.22s ease;"
          />
        </span>
        <span
          class="mobile-bottom-nav__tab-label"
          style="color: {isActive(item)
            ? 'var(--brand)'
            : 'var(--text-secondary)'}; transition: color 0.22s ease;"
        >
          {t(item.labelKey)}
        </span>
        <span class="mobile-bottom-nav__tab-indicator"></span>
      </button>
    {/each}

    <!-- Center FAB -->
    <div class="flex flex-col items-center" style="width: 68px;">
      <button
        use:interactive={{ type: 'btn' }}
        class="mobile-bottom-nav__fab {uiStore.showCreateSheet ? 'is-open' : ''}"
        onclick={() => uiStore.openCreateSheet()}
        aria-label={uiStore.showCreateSheet ? '关闭' : '发布'}
      >
        <span class="mobile-bottom-nav__fab-glyph">
          {#if uiStore.showCreateSheet}
            <span style="display: flex; animation: fabIconIn 0.2s ease-out both;">
              <X size={20} strokeWidth={2.2} color="var(--text-on-accent)" />
            </span>
          {:else}
            <span style="display: flex; animation: fabIconIn 0.2s ease-out both;">
              <Plus size={20} strokeWidth={2.2} color="var(--text-on-accent)" />
            </span>
          {/if}
        </span>
      </button>
    </div>

    {#each navItems.slice(2) as item, i (item.key)}
      <button
        bind:this={tabEls[i + 2]}
        use:interactive={{ type: 'btn' }}
        class="mobile-bottom-nav__tab {isActive(item) ? 'active' : ''}"
        onclick={() => navigate(item)}
        aria-label={t(item.labelKey)}
        aria-current={isActive(item) ? 'page' : undefined}
      >
        <span
          class="mobile-bottom-nav__tab-icon"
          style="position: relative; z-index: 1; display: flex; align-items: center; justify-content: center;"
        >
          <item.icon
            size={22}
            strokeWidth={isActive(item) ? 2.1 : 1.5}
            color={isActive(item) ? 'var(--brand)' : 'var(--text-secondary)'}
            style="transition: color 0.22s ease;"
          />
          {#if tabBadge(item) > 0}
            <span class="mobile-bottom-nav__badge"
              >{tabBadge(item) > 99 ? '99+' : tabBadge(item)}</span
            >
          {/if}
        </span>
        <span
          class="mobile-bottom-nav__tab-label"
          style="color: {isActive(item)
            ? 'var(--brand)'
            : 'var(--text-secondary)'}; transition: color 0.22s ease;"
        >
          {t(item.labelKey)}
        </span>
        <span class="mobile-bottom-nav__tab-indicator"></span>
      </button>
    {/each}
  </div>
</nav>
