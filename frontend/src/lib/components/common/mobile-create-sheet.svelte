<script lang="ts">
  import { t } from '$lib/i18n';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import * as Dialog from '$lib/components/ui/dialog';
  import Image from 'lucide-svelte/icons/image';
  import UsersRound from 'lucide-svelte/icons/users-round';

  function handleCreateMoment() {
    uiStore.openCreateMoment();
  }

  function handleCreateGroup() {
    uiStore.openCreateGroup();
  }

  const isMobileOrTablet = $derived(platformStore.isMobile || platformStore.isTablet);

  function onOpenChange(value: boolean) {
    if (!value) uiStore.closeCreateSheet();
  }
</script>

<Dialog.Root open={uiStore.showCreateSheet} {onOpenChange}>
  <Dialog.Portal>
    <Dialog.Overlay />
    <Dialog.Content variant="sheet" hideClose class="gap-4 px-5 pt-5">
      {#if isMobileOrTablet}
        <div class="mx-auto -mt-1 mb-2 h-1 w-9 rounded-full" style="background: var(--border-theme);"></div>
      {/if}
      <div class="text-center">
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
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
