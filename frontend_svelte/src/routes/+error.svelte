<!--
  根级错误页面
  当 load 函数抛出 error() 或发生未处理错误时显示
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { t } from '$lib/i18n';
  import Home from 'lucide-svelte/icons/home';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import TriangleAlert from 'lucide-svelte/icons/triangle-alert';

  const is404 = $derived(page.status === 404);
</script>

<div
  class="flex min-h-[60vh] flex-col items-center justify-center gap-6 px-4 text-center error-enter"
>
  <div
    class="flex h-20 w-20 items-center justify-center rounded-[28px]"
    style="background: var(--surface-bg); box-shadow: var(--shadow-inset); border: 1px solid var(--border-theme);"
  >
    <TriangleAlert size={36} color={is404 ? 'var(--text-secondary)' : 'var(--neu-error)'} />
  </div>

  <div class="space-y-2">
    <h1
      style="font-size: 36px; font-weight: 700; color: var(--text-primary); letter-spacing: -0.02em;"
    >
      {page.status}
    </h1>
    <p style="font-size: 16px; color: var(--text-secondary);">
      {#if is404}
        {t('errorPage.notFound')}
      {:else}
        {t('errorPage.defaultMessage')}
      {/if}
    </p>
  </div>

  <div class="flex gap-3">
    <button
      class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
      style="box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
      onclick={() => history.back()}
      aria-label={t('errorPage.backAria')}
    >
      <ArrowLeft size={16} />
      {t('errorPage.back')}
    </button>
    <button
      class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
      style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 4px 14px var(--glow);"
      onclick={() => goto('/')}
      aria-label={t('errorPage.homeAria')}
    >
      <Home size={16} />
      {t('errorPage.home')}
    </button>
  </div>
</div>
