<script lang="ts">
  import type { Snippet } from 'svelte';
  import { browser } from '$app/environment';
  import { page } from '$app/state';
  import { t } from '$lib/i18n';
  import { isSubPage } from '$lib/utils/sub-page';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import DesktopHeader from '$lib/components/layout/desktop-header.svelte';
  import Sidebar from '$lib/components/common/sidebar.svelte';
  import MobileBottomNav from '$lib/components/common/mobile-bottom-nav.svelte';
  import TabletBottomNav from '$lib/components/common/tablet-bottom-nav.svelte';

  interface Props {
    children: Snippet;
    /** 桌面侧边栏覆盖（默认主侧边栏）。 */
    sidebar?: Snippet;
    /** 移动 / 平板内容区顶部的二级导航（如管理后台分区标签）。 */
    subnav?: Snippet;
    /** 需要登录才能查看内容；未登录时展示统一的登录提示。 */
    requireAuth?: boolean;
  }

  let { children, sidebar, subnav, requireAuth = false }: Props = $props();

  const isSubPageView = $derived(isSubPage(page.url.pathname));

  $effect(() => {
    if (browser && authStore.initialized && !authStore.isAuthenticated && !authStore.isRestoring) {
      uiStore.saveRedirectUrl();
    }
  });
</script>

{#snippet restoring()}
  <div class="flex flex-col items-center justify-center gap-3 py-20">
    <div
      class="h-8 w-8 animate-spin rounded-full"
      style="border: 4px solid var(--border-theme); border-top-color: var(--brand);"
    ></div>
    <p style="font-size: 13px; color: var(--text-secondary);">{t('auth.restoringSession')}</p>
  </div>
{/snippet}

{#snippet loginRequired()}
  <div class="flex flex-col items-center justify-center py-20 text-center">
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
        <path d="M7 11V7a5 5 0 0 1 10 0v4" />
      </svg>
    </div>
    <h2 class="mb-2 text-xl font-semibold" style="color: var(--text-primary);">
      {t('auth.loginRequired')}
    </h2>
    <p class="mb-6" style="color: var(--text-secondary);">{t('auth.loginRequiredDescription')}</p>
    <button
      class="rounded-[22px] px-8 py-3.5 font-medium transition-all"
      style="background: var(--brand); color: var(--text-on-accent); box-shadow: 0 3px 10px var(--glow);"
      onclick={() => uiStore.openLoginModalWithRedirect({})}
    >
      {t('common.loginNow')}
    </button>
  </div>
{/snippet}

{#snippet gated()}
  {#if !requireAuth || authStore.isAuthenticated}
    {@render children()}
  {:else if authStore.isRestoring}
    {@render restoring()}
  {:else}
    {@render loginRequired()}
  {/if}
{/snippet}

{#if platformStore.isMobile}
  <!-- Mobile: full-screen content + edge-to-edge bottom nav -->
  <div
    class="flex h-[100dvh] w-full flex-col overflow-hidden transition-colors duration-500"
    style="background-color: var(--surface-bg); color: var(--text-primary);"
  >
    <main class="mobile-app-main no-scrollbar scroll-smooth page-enter">
      {#if subnav}{@render subnav()}{/if}
      <div class="mobile-shell-content" class:mobile-shell-content--no-nav={isSubPageView}>
        {@render gated()}
      </div>
    </main>
    {#if !isSubPageView}
      <MobileBottomNav />
    {/if}
  </div>
{:else if platformStore.isTablet}
  <!-- Tablet: full-width single-column content + floating glass pill bottom nav -->
  <div
    class="flex h-[100dvh] w-full flex-col overflow-hidden transition-colors duration-500"
    style="background-color: var(--surface-bg); color: var(--text-primary);"
  >
    <main
      class="page-enter no-scrollbar flex-1 scroll-smooth overflow-x-hidden overflow-y-auto"
      style="background-color: var(--surface-bg);"
    >
      {#if subnav}{@render subnav()}{/if}
      {@render gated()}
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
    <DesktopHeader />

    <!-- Body: sidebar + main content -->
    <div class="flex flex-1 gap-4 overflow-hidden p-4">
      {#if sidebar}{@render sidebar()}{:else}<Sidebar />{/if}

      <!-- Main content card -->
      <main class="desktop-main-card page-enter">
        <div class="h-full overflow-y-auto no-scrollbar">
          {@render gated()}
        </div>
      </main>
    </div>
  </div>
{/if}
