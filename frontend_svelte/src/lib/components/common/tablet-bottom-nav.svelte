<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import { messageStore } from '$lib/stores/message.svelte';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Compass from 'lucide-svelte/icons/compass';
  import MessageCircle from 'lucide-svelte/icons/message-circle';
  import User from 'lucide-svelte/icons/user';
  import Plus from 'lucide-svelte/icons/plus';

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

  const HIDDEN_ROUTES = [
    '/settings',
    '/location',
    '/notifications',
    '/profile/edit',
    '/search',
    '/nearby'
  ];
  const HIDDEN_PREFIXES = ['/messages/', '/user/', '/post/', '/events/'];

  function isSubPage(): boolean {
    return (
      HIDDEN_ROUTES.some((r) => currentPath === r || currentPath.startsWith(r + '/')) ||
      HIDDEN_PREFIXES.some((p) => currentPath.startsWith(p))
    );
  }
</script>

{#if !isSubPage()}
  <div
    class="flex w-full shrink-0 justify-center"
    style="padding: 8px 24px; padding-bottom: max(env(safe-area-inset-bottom, 16px), 16px); background: transparent;"
  >
    <div
      class="flex items-center"
      style="width: 100%; max-width: 520px; border-radius: var(--radius-pill); background: var(--glass-bg); backdrop-filter: blur(var(--neu-glass-blur)) saturate(var(--neu-glass-saturate)); -webkit-backdrop-filter: blur(var(--neu-glass-blur)) saturate(var(--neu-glass-saturate)); box-shadow: var(--neu-glass-shadow); border: 1px solid var(--glass-border); padding: 8px 16px; gap: 6px;"
    >
      {#each navItems.slice(0, 2) as item (item.href)}
        <button
          class="relative flex flex-col items-center justify-center outline-none select-none transition-transform duration-150 active:scale-95"
          style="flex: 1; min-width: 0; padding: 9px 0; gap: 5px;"
          onclick={() => navigate(item)}
          aria-label={item.label}
          aria-current={isActive(item.href) ? 'page' : undefined}
        >
          {#if isActive(item.href)}
            <span
              style="position: absolute; inset: 4px 6px; border-radius: 20px; background: var(--brand); opacity: 0.14; transition: background 0.22s;"
            ></span>
          {/if}
          <span
            style="position: relative; z-index: 1; display: flex; align-items: center; justify-content: center;"
          >
            <item.icon
              size={24}
              strokeWidth={isActive(item.href) ? 2.1 : 1.5}
              color={isActive(item.href) ? 'var(--brand)' : 'var(--text-secondary)'}
              style="transition: color 0.22s ease;"
            />
          </span>
          <span
            style="font-size: 11px; line-height: 1; color: {isActive(item.href)
              ? 'var(--brand)'
              : 'var(--text-secondary)'}; font-weight: {isActive(item.href)
              ? 600
              : 400}; transition: color 0.22s ease; letter-spacing: 0.02em; position: relative; z-index: 1;"
          >
            {item.label}
          </span>
        </button>
      {/each}

      <!-- Center FAB -->
      <div class="flex shrink-0 items-center justify-center" style="width: 64px;">
        <button
          class="flex items-center justify-center rounded-full outline-none select-none"
          style="width: 48px; height: 48px; background: var(--brand); box-shadow: var(--neu-fab-glow); transition: background 0.22s, box-shadow 0.22s;"
          onclick={() => uiStore.openCreateSheet()}
          aria-label={t('nav.publish')}
        >
          <Plus size={19} strokeWidth={2.2} color="var(--text-on-accent)" />
        </button>
      </div>

      {#each navItems.slice(2) as item (item.href)}
        <button
          class="relative flex flex-col items-center justify-center outline-none select-none transition-transform duration-150 active:scale-95"
          style="flex: 1; min-width: 0; padding: 9px 0; gap: 5px;"
          onclick={() => navigate(item)}
          aria-label={item.label}
          aria-current={isActive(item.href) ? 'page' : undefined}
        >
          {#if isActive(item.href)}
            <span
              style="position: absolute; inset: 4px 6px; border-radius: 20px; background: var(--brand); opacity: 0.14; transition: background 0.22s;"
            ></span>
          {/if}
          <span
            style="position: relative; z-index: 1; display: flex; align-items: center; justify-content: center;"
          >
            <item.icon
              size={24}
              strokeWidth={isActive(item.href) ? 2.1 : 1.5}
              color={isActive(item.href) ? 'var(--brand)' : 'var(--text-secondary)'}
              style="transition: color 0.22s ease;"
            />
            {#if item.href === '/chat' && messageBadge > 0}
              <span
                class="absolute flex items-center justify-center rounded-full"
                style="top: -4px; right: -6px; height: 20px; min-width: 20px; padding: 0 6px; background: var(--brand); color: var(--text-on-accent); font-size: 11px; font-weight: 600; line-height: 20px;"
              >
                {messageBadge > 99 ? '99+' : messageBadge}
              </span>
            {:else if item.href === '/profile' && profileBadge > 0}
              <span
                class="absolute flex items-center justify-center rounded-full"
                style="top: -4px; right: -6px; height: 20px; min-width: 20px; padding: 0 6px; background: var(--brand); color: var(--text-on-accent); font-size: 11px; font-weight: 600; line-height: 20px;"
              >
                {profileBadge > 99 ? '99+' : profileBadge}
              </span>
            {/if}
          </span>
          <span
            style="font-size: 11px; line-height: 1; color: {isActive(item.href)
              ? 'var(--brand)'
              : 'var(--text-secondary)'}; font-weight: {isActive(item.href)
              ? 600
              : 400}; transition: color 0.22s ease; letter-spacing: 0.02em; position: relative; z-index: 1;"
          >
            {item.label}
          </span>
        </button>
      {/each}
    </div>
  </div>
{/if}
