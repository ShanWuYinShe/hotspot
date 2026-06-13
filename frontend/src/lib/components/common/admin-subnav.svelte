<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import { adminNavItems, isNavActive, type NavItem } from '$lib/config/navigation';

  const currentPathname = $derived(page.url.pathname);

  function isActive(item: NavItem): boolean {
    return isNavActive(item.href, item.exact, currentPathname);
  }
</script>

<!-- Admin section tabs — horizontal scrollable strip for tablet / mobile -->
<div
  class="sticky top-0 z-10 no-scrollbar overflow-x-auto"
  style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
>
  <div class="flex w-max items-center gap-1.5 px-4 py-2.5">
    {#each adminNavItems as item (item.href)}
      <button
        class="flex flex-shrink-0 items-center gap-1.5 rounded-full px-3 py-1.5 outline-none transition-colors"
        style="font-size: 13px; font-weight: {isActive(item) ? 600 : 400}; background: {isActive(
          item
        )
          ? 'var(--brand)'
          : 'var(--theme-accent-soft)'}; color: {isActive(item)
          ? 'var(--text-on-accent)'
          : 'var(--text-secondary)'};"
        onclick={() => void goto(item.href)}
      >
        <item.icon size={15} strokeWidth={isActive(item) ? 2 : 1.6} />
        {t(item.labelKey)}
      </button>
    {/each}
  </div>
</div>
