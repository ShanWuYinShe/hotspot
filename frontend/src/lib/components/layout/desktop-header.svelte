<script lang="ts">
  import { t } from '$lib/i18n';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { mode, toggleMode } from 'mode-watcher';
  import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import Avatar from '$lib/components/common/avatar.svelte';
  import Search from 'lucide-svelte/icons/search';
  import Sun from 'lucide-svelte/icons/sun';
  import Moon from 'lucide-svelte/icons/moon';
  import Settings from 'lucide-svelte/icons/settings';
  import Palette from 'lucide-svelte/icons/palette';
  import Check from 'lucide-svelte/icons/check';
  import Plus from 'lucide-svelte/icons/plus';

  const currentThemeFamily = $derived(themeStore.themeFamily);
  const currentMode = $derived(mode.current);

  function selectThemeFamily(family: ThemeFamily) {
    themeStore.setThemeFamily(family);
  }
</script>

<!-- Desktop top header — 52px，跨所有路由组共用 -->
<header class="desktop-header">
  <!-- Logo -->
  <a href="/" class="mr-3 flex cursor-pointer items-center gap-2 select-none">
    <img src="/favicon-32x32.png" alt="HotSpot" class="h-7 w-7 rounded-full object-cover" />
    <span class="text-title tracking-tight" style="color: var(--text-primary)">HotSpot</span>
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
        <Moon size={17} color="var(--text-secondary)" />
      {:else}
        <Sun size={17} color="var(--text-secondary)" />
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
      style="width: 36px; height: 36px; background: var(--brand); box-shadow: var(--shadow-brand-btn);"
      onclick={() => uiStore.openCreateSheet()}
      aria-label={t('nav.publish')}
    >
      <Plus size={15} color="var(--text-on-accent)" strokeWidth={2.5} />
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
        class="ml-1.5 desktop-header__btn"
        style="width: 36px; height: 36px; padding: 0;"
        aria-label={t('nav.profile')}
        onclick={() => void goto('/profile')}
      >
        <Avatar
          src={authStore.currentAccount?.avatar_url || ''}
          name={authStore.currentAccount?.nickname || 'Profile'}
          alt={authStore.currentAccount?.nickname || 'Profile'}
          size={36}
        />
      </button>
    {:else}
      <button
        type="button"
        class="text-body ml-1.5 rounded-full px-4 font-medium transition-all duration-150 hover:brightness-110 active:scale-95"
        style="height: 36px; background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
        onclick={() => uiStore.openLoginModalWithRedirect({})}
      >
        {t('nav.loginRegister')}
      </button>
    {/if}
  </div>
</header>
