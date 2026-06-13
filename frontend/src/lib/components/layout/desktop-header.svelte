<script lang="ts">
  import { t } from '$lib/i18n';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { mode, toggleMode } from 'mode-watcher';
  import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';
  import Search from 'lucide-svelte/icons/search';
  import Sun from 'lucide-svelte/icons/sun';
  import Moon from 'lucide-svelte/icons/moon';
  import Bell from 'lucide-svelte/icons/bell';
  import Settings from 'lucide-svelte/icons/settings';
  import Palette from 'lucide-svelte/icons/palette';
  import Check from 'lucide-svelte/icons/check';

  const currentThemeFamily = $derived(themeStore.themeFamily);
  const currentMode = $derived(mode.current);

  function selectThemeFamily(family: ThemeFamily) {
    themeStore.setThemeFamily(family);
  }
</script>

<!-- Desktop top header — 52px (Figma Make), shared by all route groups -->
<header class="desktop-header">
  <!-- Logo -->
  <a href="/" class="mr-3 flex cursor-pointer items-center gap-2 select-none">
    <img src="/logo_32.png" alt="HotSpot" class="h-7 w-7 rounded-full object-cover" />
    <span class="tracking-tight" style="font-size: 17px; color: var(--text-primary);">HotSpot</span>
  </a>

  <div class="flex-1"></div>

  <!-- Right actions -->
  <div class="flex items-center gap-0.5">
    <button
      class="desktop-header__btn"
      aria-label={t('nav.openSearch')}
      onclick={() => uiStore.openSearchModal()}
    >
      <Search size={17} color="var(--text-secondary)" />
    </button>
    <button class="desktop-header__btn" aria-label={t('sidebar.toggleTheme')} onclick={toggleMode}>
      {#if currentMode === 'dark'}
        <Sun size={17} color="var(--text-secondary)" />
      {:else}
        <Moon size={17} color="var(--text-secondary)" />
      {/if}
    </button>
    <DropdownMenu.Root>
      <DropdownMenu.Trigger>
        {#snippet child({ props })}
          <button {...props} class="desktop-header__btn" aria-label={t('sidebar.themeFamily')}>
            <Palette size={17} color="var(--text-secondary)" />
          </button>
        {/snippet}
      </DropdownMenu.Trigger>
      <DropdownMenu.Content class="w-44" align="end">
        <DropdownMenu.Label>{t('sidebar.themeFamily')}</DropdownMenu.Label>
        <DropdownMenu.Separator />
        {#each THEME_FAMILY_OPTIONS as option (option.family)}
          <DropdownMenu.Item onclick={() => selectThemeFamily(option.family)}>
            <span class="flex-1">{t(option.labelKey)}</span>
            {#if currentThemeFamily === option.family}
              <Check size={15} />
            {/if}
          </DropdownMenu.Item>
        {/each}
      </DropdownMenu.Content>
    </DropdownMenu.Root>
    <button
      class="mx-1.5 flex items-center justify-center rounded-full outline-none transition-shadow duration-200"
      style="width: 32px; height: 32px; background: var(--brand); box-shadow: 0 4px 12px var(--glow);"
      onclick={() => uiStore.openCreateSheet()}
      aria-label={t('nav.publish')}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="15"
        height="15"
        viewBox="0 0 24 24"
        fill="none"
        stroke="var(--text-on-accent)"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"><path d="M5 12h14" /><path d="M12 5v14" /></svg
      >
    </button>
    <button
      class="desktop-header__btn"
      aria-label={t('sidebar.notifications')}
      onclick={() => void goto('/notifications')}
    >
      <Bell size={17} color="var(--text-secondary)" />
    </button>
    <button
      class="desktop-header__btn"
      aria-label={t('sidebar.settingsLabel')}
      onclick={() => void goto('/settings')}
    >
      <Settings size={17} color="var(--text-secondary)" />
    </button>
    {#if authStore.isAuthenticated}
      <button
        type="button"
        class="relative ml-1.5 cursor-pointer"
        style="width: 32px; height: 32px; padding: 0; border: none; background: none;"
        aria-label={t('nav.profile')}
        onclick={() => void goto('/profile')}
      >
        <Avatar
          src={authStore.currentAccount?.avatar_url || ''}
          name={authStore.currentAccount?.nickname || 'Profile'}
          alt={authStore.currentAccount?.nickname || 'Profile'}
          size={32}
        />
        {#if notificationStore.unreadCount > 0}
          <div class="desktop-header__badge" style="top: -3px; right: -3px;">
            {notificationStore.unreadCount}
          </div>
        {/if}
      </button>
    {/if}
  </div>
</header>
