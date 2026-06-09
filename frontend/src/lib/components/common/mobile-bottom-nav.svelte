<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { interactive } from '$lib/actions/interactive';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Compass from 'lucide-svelte/icons/compass';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import User from 'lucide-svelte/icons/user';
  import Plus from 'lucide-svelte/icons/plus';
  import X from 'lucide-svelte/icons/x';

  const navItems = [
    { href: '/', label: t('nav.nearby'), icon: MapPin },
    { href: '/discover', label: t('nav.discover'), icon: Compass },
    { href: '/chat', label: t('nav.messagesTab'), icon: MessageCircle, requiresAuth: true },
    { href: '/profile', label: t('nav.me'), icon: User, requiresAuth: true }
  ];

  const currentPath = $derived(page.url.pathname);

  function isActive(href: string) {
    if (href === '/') return currentPath === '/';
    if (href === '/profile') return currentPath === '/profile' || currentPath === '/settings';
    return currentPath === href || currentPath.startsWith(`${href}/`);
  }

  function navigate(item: { href: string; requiresAuth?: boolean }) {
    if (item.requiresAuth && !authStore.isAuthenticated) {
      uiStore.openLoginModalWithRedirect({ url: item.href });
      return;
    }
    void goto(item.href);
  }

  const messageBadge = $derived(authStore.isAuthenticated ? messageStore.totalUnreadCount : 0);
  const profileBadge = $derived(authStore.isAuthenticated ? notificationStore.unreadCount : 0);

  // Shared sliding pill — mirrors the React reference's Framer Motion `layoutId`
  // behaviour so the active indicator glides between tabs instead of fading per tab.
  const SLIDER_W = 52;
  let rowEl = $state<HTMLDivElement | undefined>();
  let tabEls = $state<(HTMLButtonElement | undefined)[]>([]);
  let sliderX = $state(0);
  let sliderTop = $state(6);
  const activeIndex = $derived(navItems.findIndex((it) => isActive(it.href)));

  function measureSlider() {
    const el = tabEls[activeIndex];
    if (!rowEl || !el) return;
    sliderX = el.offsetLeft + el.offsetWidth / 2 - SLIDER_W / 2;
    sliderTop = el.offsetTop + 6;
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
      class="mobile-bottom-nav__slider {activeIndex >= 0 ? 'visible' : ''}"
      style="width: {SLIDER_W}px; top: {sliderTop}px; transform: translateX({sliderX}px);"
      aria-hidden="true"
    ></span>
    {#each navItems.slice(0, 2) as item, i (item.href)}
      <button
        bind:this={tabEls[i]}
        use:interactive={{ type: 'btn' }}
        class="mobile-bottom-nav__tab {isActive(item.href) ? 'active' : ''}"
        onclick={() => navigate(item)}
        aria-label={item.label}
      >
        <span
          class="mobile-bottom-nav__tab-icon"
          style="position: relative; z-index: 1; display: flex; align-items: center; justify-content: center;"
        >
          <item.icon
            size={22}
            strokeWidth={isActive(item.href) ? 2.1 : 1.5}
            color={isActive(item.href) ? 'var(--brand)' : 'var(--text-secondary)'}
            style="transition: color 0.22s ease;"
          />
        </span>
        <span
          class="mobile-bottom-nav__tab-label"
          style="color: {isActive(item.href)
            ? 'var(--brand)'
            : 'var(--text-secondary)'}; transition: color 0.22s ease;"
        >
          {item.label}
        </span>
        <span class="mobile-bottom-nav__tab-indicator"></span>
      </button>
    {/each}

    <!-- Center FAB -->
    <div class="flex flex-col items-center" style="width: 68px;">
      <button
        use:interactive={{ type: 'btn' }}
        class="mobile-bottom-nav__fab"
        style="background: {uiStore.showCreateSheet
          ? 'var(--neu-text-muted)'
          : 'var(--brand)'}; box-shadow: {uiStore.showCreateSheet
          ? 'none'
          : 'var(--neu-fab-glow)'}; transition: background 0.22s ease, box-shadow 0.22s ease;"
        onclick={() => uiStore.openCreateSheet()}
        aria-label={uiStore.showCreateSheet ? '关闭' : '发布'}
      >
        {#if uiStore.showCreateSheet}
          <span style="display: flex; animation: fabIconIn 0.18s ease-out both;">
            <X size={20} strokeWidth={2.2} color="var(--text-on-accent)" />
          </span>
        {:else}
          <span style="display: flex; animation: fabIconIn 0.18s ease-out both;">
            <Plus size={20} strokeWidth={2.2} color="var(--text-on-accent)" />
          </span>
        {/if}
      </button>
    </div>

    {#each navItems.slice(2) as item, i (item.href)}
      <button
        bind:this={tabEls[i + 2]}
        use:interactive={{ type: 'btn' }}
        class="mobile-bottom-nav__tab {isActive(item.href) ? 'active' : ''}"
        onclick={() => navigate(item)}
        aria-label={item.label}
      >
        <span
          class="mobile-bottom-nav__tab-icon"
          style="position: relative; z-index: 1; display: flex; align-items: center; justify-content: center;"
        >
          <item.icon
            size={22}
            strokeWidth={isActive(item.href) ? 2.1 : 1.5}
            color={isActive(item.href) ? 'var(--brand)' : 'var(--text-secondary)'}
            style="transition: color 0.22s ease;"
          />
          {#if item.href === '/chat' && messageBadge > 0}
            <span class="mobile-bottom-nav__badge">{messageBadge > 99 ? '99+' : messageBadge}</span>
          {:else if item.href === '/profile' && profileBadge > 0}
            <span class="mobile-bottom-nav__badge">{profileBadge > 99 ? '99+' : profileBadge}</span>
          {/if}
        </span>
        <span
          class="mobile-bottom-nav__tab-label"
          style="color: {isActive(item.href)
            ? 'var(--brand)'
            : 'var(--text-secondary)'}; transition: color 0.22s ease;"
        >
          {item.label}
        </span>
        <span class="mobile-bottom-nav__tab-indicator"></span>
      </button>
    {/each}
  </div>
</nav>
