<script lang="ts">
  import { browser } from '$app/environment';
  import { page } from '$app/state';
  import { t } from '$lib/i18n';
  import { goto } from '$app/navigation';
  import { isSubPage } from '$lib/utils/sub-page';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { mode, toggleMode } from 'mode-watcher';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import MobileBottomNav from '$lib/components/common/mobile-bottom-nav.svelte';
  import TabletBottomNav from '$lib/components/common/tablet-bottom-nav.svelte';
  import Search from 'lucide-svelte/icons/search';
  import Sun from 'lucide-svelte/icons/sun';
  import Moon from 'lucide-svelte/icons/moon';
  import Bell from 'lucide-svelte/icons/bell';
  import Settings from 'lucide-svelte/icons/settings';
  import Palette from 'lucide-svelte/icons/palette';
  import Check from 'lucide-svelte/icons/check';
  import { notificationStore } from '$lib/stores/notification.svelte';
  import Sidebar from '$lib/components/common/sidebar.svelte';
  import Avatar from '$lib/components/common/avatar.svelte';

  let { children } = $props();

  const isSubPageView = $derived(isSubPage(page.url.pathname));
  const currentThemeFamily = $derived(themeStore.themeFamily);

  function selectThemeFamily(family: ThemeFamily) {
    themeStore.setThemeFamily(family);
  }

  $effect(() => {
    if (browser && authStore.initialized && !authStore.isAuthenticated && !authStore.isRestoring) {
      uiStore.saveRedirectUrl();
    }
  });

  const currentMode = $derived(mode.current);
</script>

{#if platformStore.isMobile}
  <!-- Mobile: full-screen content + edge-to-edge bottom nav -->
  <div
    class="flex h-[100dvh] w-full flex-col overflow-hidden transition-colors duration-500"
    style="background-color: var(--surface-bg); color: var(--text-primary);"
  >
    <main class="mobile-app-main no-scrollbar scroll-smooth">
      <div class="mobile-shell-content" class:mobile-shell-content--no-nav={isSubPageView}>
        {#if authStore.isAuthenticated}
          {@render children()}
        {:else if authStore.isRestoring}
          <div class="flex flex-col items-center justify-center gap-3 py-20">
            <div
              class="h-8 w-8 animate-spin rounded-full"
              style="border: 4px solid var(--border-theme); border-top-color: var(--brand);"
            ></div>
            <p style="font-size: 13px; color: var(--text-secondary);">
              {t('auth.restoringSession')}
            </p>
          </div>
        {:else}
          <div class="flex flex-col items-center justify-center py-20">
            <div class="mb-4 rounded-[22px] neu-flat p-6">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="48"
                height="48"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                style="color: var(--text-secondary);"
              >
                <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
                <path d="M7 11V7a5 0 0 1 10 0v4" />
              </svg>
            </div>
            <h2 class="mb-2 text-xl font-semibold" style="color: var(--text-primary);">
              {t('auth.loginRequired')}
            </h2>
            <p class="mb-6" style="color: var(--text-secondary);">
              {t('auth.loginRequiredDescription')}
            </p>
            <button
              class="rounded-[22px] px-8 py-3.5 font-medium transition-all"
              style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 3px 10px var(--glow);"
              onclick={() => uiStore.openLoginModalWithRedirect({})}
            >
              {t('common.loginNow')}
            </button>
          </div>
        {/if}
      </div>
    </main>
    {#if !isSubPageView}
      <MobileBottomNav />
    {/if}
  </div>
{:else if platformStore.isTablet}
  <!-- Tablet: full-width content + floating glass pill bottom nav -->
  <div
    class="flex h-[100dvh] w-full flex-col overflow-hidden transition-colors duration-500"
    style="background-color: var(--surface-bg); color: var(--text-primary);"
  >
    <main
      class="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar"
      style="background-color: var(--surface-bg);"
    >
      <div style="width: 100%;">
        {#if authStore.isAuthenticated}
          {@render children()}
        {:else if authStore.isRestoring}
          <div class="flex flex-col items-center justify-center gap-3 py-20">
            <div
              class="h-8 w-8 animate-spin rounded-full"
              style="border: 4px solid var(--border-theme); border-top-color: var(--brand);"
            ></div>
            <p style="font-size: 13px; color: var(--text-secondary);">
              {t('auth.restoringSession')}
            </p>
          </div>
        {:else}
          <div class="flex flex-col items-center justify-center py-20">
            <div class="mb-4 rounded-[22px] neu-flat p-6">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="48"
                height="48"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                style="color: var(--text-secondary);"
              >
                <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
                <path d="M7 11V7a5 0 0 1 10 0v4" />
              </svg>
            </div>
            <h2 class="mb-2 text-xl font-semibold" style="color: var(--text-primary);">
              {t('auth.loginRequired')}
            </h2>
            <p class="mb-6" style="color: var(--text-secondary);">
              {t('auth.loginRequiredDescription')}
            </p>
            <button
              class="rounded-[22px] px-8 py-3.5 font-medium transition-all"
              style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 3px 10px var(--glow);"
              onclick={() => uiStore.openLoginModalWithRedirect({})}
            >
              {t('common.loginNow')}
            </button>
          </div>
        {/if}
      </div>
    </main>
    {#if !isSubPageView}
      <TabletBottomNav />
    {/if}
  </div>
{:else}
  <!-- Desktop: header + sidebar + main card (≥1280px) -->
  <div
    class="flex h-screen w-screen flex-col overflow-hidden transition-colors duration-500"
    style="background-color: var(--page-bg); color: var(--text-primary);"
  >
    <!-- Header — 52px -->
    <header class="desktop-header">
      <!-- Logo -->
      <a href="/" class="mr-3 flex cursor-pointer items-center gap-2 select-none">
        <img src="/logo_32.png" alt="HotSpot" class="h-7 w-7 rounded-full object-cover" />
        <span class="tracking-tight" style="font-size: 17px; color: var(--text-primary);"
          >HotSpot</span
        >
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
        <button
          class="desktop-header__btn"
          aria-label={t('sidebar.toggleTheme')}
          onclick={toggleMode}
        >
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
            aria-label="个人主页"
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

    <!-- Body: sidebar + main content -->
    <div class="flex flex-1 gap-4 overflow-hidden p-4">
      <!-- Sidebar -->
      <Sidebar />

      <!-- Main content card -->
      <main class="desktop-main-card">
        <div class="h-full overflow-y-auto no-scrollbar">
          {#if authStore.isAuthenticated}
            {@render children()}
          {:else if authStore.isRestoring}
            <div class="flex flex-col items-center justify-center gap-3 py-20">
              <div
                class="h-8 w-8 animate-spin rounded-full"
                style="border: 4px solid var(--border-theme); border-top-color: var(--brand);"
              ></div>
              <p style="font-size: 13px; color: var(--text-secondary);">
                {t('auth.restoringSession')}
              </p>
            </div>
          {:else}
            <div class="flex flex-col items-center justify-center py-20">
              <h2 class="mb-2 text-xl font-semibold" style="color: var(--text-primary);">
                {t('auth.loginRequired')}
              </h2>
              <p class="mb-6" style="color: var(--text-secondary);">
                {t('auth.loginRequiredDescription')}
              </p>
              <button
                class="rounded-[22px] px-8 py-3.5 font-medium"
                style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 3px 10px var(--glow);"
                onclick={() => uiStore.openLoginModalWithRedirect({})}
              >
                {t('common.loginNow')}
              </button>
            </div>
          {/if}
        </div>
      </main>
    </div>
  </div>
{/if}
