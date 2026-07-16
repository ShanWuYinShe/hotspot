<script lang="ts">
  import { t } from '$lib/i18n';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { fade, fly } from 'svelte/transition';
  import { backOut, cubicIn } from 'svelte/easing';
  import Image from 'lucide-svelte/icons/image';
  import UsersRound from 'lucide-svelte/icons/users-round';

  interface Props {}

  let {}: Props = $props();

  function handleCreateMoment() {
    uiStore.openCreateMoment();
  }

  function handleCreateGroup() {
    uiStore.openCreateGroup();
  }

  const isMobileOrTablet = $derived(platformStore.isMobile || platformStore.isTablet);
  const show = $derived(uiStore.showCreateSheet);
</script>

{#if show}
  <!-- Backdrop -->
  <button
    class="fixed inset-0 z-[var(--z-overlay)] cursor-default"
    style="background: var(--overlay); backdrop-filter: blur(3px);"
    onclick={() => uiStore.closeCreateSheet()}
    aria-label="关闭"
    transition:fade={{ duration: 220 }}
  ></button>

  <!-- Sheet -->
  <div
    class="fixed z-[var(--z-modal)] bottom-0 left-0 right-0 overflow-hidden rounded-t-[28px] px-5 pb-[calc(1.25rem+env(safe-area-inset-bottom,0px))] pt-5"
    style="background: var(--surface-bg); border-top: 1px solid var(--border-theme); box-shadow: 0 -8px 40px rgba(0,0,0,0.18), 0 -2px 0 var(--border-theme);"
    in:fly={{ y: 320, duration: 440, easing: backOut }}
    out:fly={{ y: 320, duration: 240, easing: cubicIn }}
  >
    {#if isMobileOrTablet}
      <div class="mx-auto mb-4 h-1 w-9 rounded-full" style="background: var(--border-theme);"></div>
    {/if}
    <div class="mb-5 text-center">
      <h2 class="text-title" style="font-weight: 600; color: var(--text-primary)">
        {t('nav.createSheetTitle')}
      </h2>
      <p class="text-meta" style="color: var(--text-secondary); margin-top: 4px">
        {t('nav.createSheetDescription')}
      </p>
    </div>

    <div class="grid gap-3">
      <button
        class="flex h-16 items-center gap-3 rounded-[var(--radius-md)] px-4 text-left outline-none active:scale-95 transition-transform duration-100"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
        onclick={handleCreateMoment}
      >
        <div
          class="flex items-center justify-center rounded-full"
          style="width: 38px; height: 38px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
        >
          <Image size={16} strokeWidth={1.5} color="var(--brand)" />
        </div>
        <div>
          <p class="text-body" style="font-weight: 600; color: var(--text-primary)">
            {t('nav.postMoment')}
          </p>
          <p class="text-caption" style="color: var(--text-secondary)">
            {t('nav.createMomentHint')}
          </p>
        </div>
      </button>

      <button
        class="flex h-16 items-center gap-3 rounded-[var(--radius-md)] px-4 text-left outline-none active:scale-95 transition-transform duration-100"
        style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
        onclick={handleCreateGroup}
      >
        <div
          class="flex items-center justify-center rounded-full"
          style="width: 38px; height: 38px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
        >
          <UsersRound size={16} strokeWidth={1.5} color="var(--brand)" />
        </div>
        <div>
          <p class="text-body" style="font-weight: 600; color: var(--text-primary)">
            {t('nav.createGroup')}
          </p>
          <p class="text-caption" style="color: var(--text-secondary)">
            {t('nav.createGroupHint')}
          </p>
        </div>
      </button>
    </div>
  </div>
{/if}
