<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { adminNavItems, isNavActive, type NavItem } from '$lib/config/navigation';
  import Home from 'lucide-svelte/icons/home';

  const currentPathname = $derived(page.url.pathname);

  function isActive(item: NavItem): boolean {
    return isNavActive(item.href, item.exact, currentPathname);
  }
</script>

<!-- Admin sidebar — same visual language as the main sidebar (192px, desktop-sidebar__nav-item) -->
<aside class="flex flex-shrink-0 flex-col" style="width: 192px;">
  <div class="flex min-h-0 flex-1 flex-col overflow-y-auto no-scrollbar">
    <!-- Back to app -->
    <button class="desktop-sidebar__nav-item" onclick={() => void goto('/')}>
      <Home size={17} strokeWidth={1.6} />
      <span class="text-sm">{t('sidebar.home')}</span>
    </button>

    <!-- Divider -->
    <div class="my-3" style="height: 1px; background: var(--border-theme);"></div>

    <!-- Section label -->
    <p
      class="text-xs mb-2 px-3"
      style="font-weight: 600; letter-spacing: 0.12em; text-transform: uppercase; color: var(--text-secondary); opacity: 0.65"
    >
      {t('sidebar.adminPanel')}
    </p>

    <!-- Admin nav -->
    <div class="flex flex-col" style="gap: 2px;">
      {#each adminNavItems as item (item.href)}
        <button
          class="desktop-sidebar__nav-item {isActive(item) ? 'active' : ''}"
          onclick={() => void goto(item.href)}
        >
          <item.icon size={17} strokeWidth={isActive(item) ? 2 : 1.6} />
          <span class="text-sm" class:font-medium={isActive(item)}>
            {t(item.labelKey)}
          </span>
        </button>
      {/each}
    </div>
  </div>
</aside>
