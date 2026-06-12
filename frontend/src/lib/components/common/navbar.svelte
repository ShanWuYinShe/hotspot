<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { adminPermissionsStore } from '$lib/stores/admin-permissions.svelte';
  import { RippleButton } from '$lib/components/ui/interactive';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import ThemeToggle from './theme-toggle.svelte';
  import SearchModal from './search-modal.svelte';
  import NotificationDropdown from './notification-dropdown.svelte';
  import Search from 'lucide-svelte/icons/search';
  import User from 'lucide-svelte/icons/user';
  import Settings from 'lucide-svelte/icons/settings';
  import Shield from 'lucide-svelte/icons/shield';
  import LogOut from 'lucide-svelte/icons/log-out';
  import Menu from 'lucide-svelte/icons/menu';
  import Plus from 'lucide-svelte/icons/plus';
  import Image from 'lucide-svelte/icons/image';
  import UsersRound from 'lucide-svelte/icons/users-round';
  import Avatar from '$lib/components/common/avatar.svelte';

  interface Props {
    onMenuClick?: () => void;
    showMenu?: boolean;
  }

  let { onMenuClick, showMenu = true }: Props = $props();
  let showSearchModal = $state(false);

  function closeTopbarModals() {
    showSearchModal = false;
  }
  function openSearchModal() {
    closeTopbarModals();
    showSearchModal = true;
  }

  onMount(() => {
    const handleKeydown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        openSearchModal();
      }
    };
    window.addEventListener('keydown', handleKeydown);
    return () => window.removeEventListener('keydown', handleKeydown);
  });

  function handleLogout() {
    authStore.logout();
    goto('/');
  }
  function handleAuthEntry() {
    uiStore.openLoginModal();
  }
  function openCreateMoment() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal();
      return;
    }
    closeTopbarModals();
    uiStore.openCreateMoment();
  }
  function openCreateGroup() {
    if (!authStore.isAuthenticated) {
      uiStore.openLoginModal();
      return;
    }
    closeTopbarModals();
    uiStore.openCreateGroup();
  }
</script>

<header
  class="sticky top-0 z-50 w-full"
  style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme); padding-top: env(safe-area-inset-top, 0px);"
>
  <div class="mx-auto flex h-13 w-full items-center gap-1.5 px-3 sm:h-16 sm:gap-3 sm:px-4 md:px-6">
    {#if showMenu}
      <RippleButton
        variant="icon"
        size="icon"
        class="hidden shrink-0 md:hidden"
        onclick={onMenuClick}
        circular
        aria-label={t('nav.openMenu')}
      >
        <Menu class="h-5 w-5" />
      </RippleButton>
    {/if}

    <!-- Logo -->
    <a
      href="/"
      class="flex shrink-0 items-center gap-2 rounded-[12px] px-1.5 py-1 transition-all sm:px-2"
    >
      <div
        class="flex items-center justify-center rounded-[12px]"
        style="width: 34px; height: 34px; box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
      >
        <img src="/logo_32.png" alt="HotSpot" class="h-5 w-5 rounded-[8px] sm:h-5 sm:w-5" />
      </div>
      <span class="hidden text-lg font-bold sm:inline-block" style="color: var(--brand);"
        >HotSpot</span
      >
    </a>

    <!-- Right side -->
    <div class="ml-auto flex shrink-0 items-center gap-1 sm:gap-2">
      <RippleButton
        variant="icon"
        size="icon"
        class="h-9 w-9 sm:h-10 sm:w-10"
        aria-label={t('nav.openSearch')}
        onclick={openSearchModal}
        circular
      >
        <Search class="h-4 w-4 sm:h-5 sm:w-5" />
      </RippleButton>

      <div class="hidden md:block">
        <ThemeToggle />
      </div>

      {#if authStore.isAuthenticated}
        <div class="hidden md:block">
          <DropdownMenu.Root>
            <DropdownMenu.Trigger>
              {#snippet child({ props })}
                <RippleButton variant="icon" size="icon" class="h-10 w-10" {...props} circular>
                  <Plus class="h-5 w-5" />
                </RippleButton>
              {/snippet}
            </DropdownMenu.Trigger>
            <DropdownMenu.Content align="end">
              <DropdownMenu.Item onclick={openCreateMoment}>
                <Image class="mr-2 h-4 w-4" />
                {t('nav.postMoment')}
              </DropdownMenu.Item>
              <DropdownMenu.Item onclick={openCreateGroup}>
                <UsersRound class="mr-2 h-4 w-4" />
                {t('nav.createGroup')}
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Root>
        </div>

        <!-- Notifications Dropdown -->
        <NotificationDropdown />

        <!-- User menu -->
        <DropdownMenu.Root>
          <DropdownMenu.Trigger>
            {#snippet child({ props })}
              <button
                {...props}
                class="inline-flex items-center justify-center rounded-full p-0.5 outline-none sm:p-1"
              >
                <div
                  class="overflow-hidden rounded-full"
                  style="width: 32px; height: 32px; border: 2px solid var(--brand);"
                >
                  <Avatar
                    src={authStore.currentAccount?.avatar_url}
                    name={authStore.currentAccount?.nickname}
                    alt=""
                    size={28}
                    border={false}
                  />
                </div>
              </button>
            {/snippet}
          </DropdownMenu.Trigger>
          <DropdownMenu.Content class="w-56" align="end">
            <DropdownMenu.Label>
              <div class="flex flex-col space-y-1">
                <p style="font-size: 13px; font-weight: 500; color: var(--text-primary);">
                  {authStore.currentAccount?.nickname}
                </p>
                <p style="font-size: 11px; color: var(--text-secondary);">
                  ID: {authStore.currentUserId}
                </p>
              </div>
            </DropdownMenu.Label>
            <DropdownMenu.Separator />
            <DropdownMenu.Item>
              {#snippet child({ props })}
                <button
                  {...props}
                  class="flex w-full items-center"
                  onclick={() => goto('/profile')}
                >
                  <User class="mr-2 h-4 w-4" />
                  {t('nav.profile')}
                </button>
              {/snippet}
            </DropdownMenu.Item>
            <DropdownMenu.Item>
              {#snippet child({ props })}
                <button
                  {...props}
                  class="flex w-full items-center"
                  onclick={() => goto('/settings')}
                >
                  <Settings class="mr-2 h-4 w-4" />
                  {t('nav.settings')}
                </button>
              {/snippet}
            </DropdownMenu.Item>
            {#if adminPermissionsStore.isAdmin}
              <DropdownMenu.Item>
                {#snippet child({ props })}
                  <button
                    {...props}
                    class="flex w-full items-center"
                    onclick={() => goto('/admin')}
                  >
                    <Shield class="mr-2 h-4 w-4" />
                    {t('nav.adminPanel')}
                  </button>
                {/snippet}
              </DropdownMenu.Item>
            {/if}
            <DropdownMenu.Separator />
            <DropdownMenu.Item>
              {#snippet child({ props })}
                <button {...props} class="flex w-full items-center" onclick={handleLogout}>
                  <LogOut class="mr-2 h-4 w-4" />
                  {t('nav.logout')}
                </button>
              {/snippet}
            </DropdownMenu.Item>
          </DropdownMenu.Content>
        </DropdownMenu.Root>
      {:else}
        <RippleButton
          variant="icon"
          size="icon"
          class="h-9 w-9 sm:hidden"
          aria-label={t('nav.loginOrRegister')}
          onclick={handleAuthEntry}
          circular
        >
          <User class="h-4 w-4" />
        </RippleButton>
        <button
          class="hidden rounded-full px-4 py-1.5 outline-none focus-visible:ring-2 focus-visible:ring-[var(--glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-bg)] sm:inline-flex"
          style="background: var(--brand); color: var(--text-on-accent); font-size: 12px; font-weight: 500; box-shadow: 0 3px 10px var(--glow);"
          onclick={handleAuthEntry}
        >
          {t('nav.loginRegister')}
        </button>
      {/if}
    </div>
  </div>
</header>

<SearchModal bind:open={showSearchModal} />
