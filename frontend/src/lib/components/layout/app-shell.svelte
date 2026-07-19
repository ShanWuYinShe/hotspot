<script lang="ts">
  import type { Snippet } from 'svelte';
  import { browser } from '$app/environment';
  import { page } from '$app/state';
  import { t } from '$lib/i18n';
  import { isSubPage } from '$lib/utils/sub-page';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { Button } from '$lib/components/ui/button';
  import DesktopHeader from '$lib/components/layout/desktop-header.svelte';
  import Sidebar from '$lib/components/common/sidebar.svelte';
  import MobileBottomNav from '$lib/components/common/mobile-bottom-nav.svelte';
  import TabletBottomNav from '$lib/components/common/tablet-bottom-nav.svelte';
  import Lock from 'lucide-svelte/icons/lock';

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

  // 宽屏平板（横屏 / 大尺寸）的 chat 采用 split-pane：会话列表常驻左栏，会话详情显示在右栏。
  // 此时会话详情不应再被当作"全屏子页"而隐藏底栏——否则用户进入会话后会丢失全局导航。
  const isWideChatPane = $derived(
    platformStore.isTablet && platformStore.isWide && page.url.pathname.startsWith('/chat')
  );

  const isSubPageView = $derived(isSubPage(page.url.pathname) && !isWideChatPane);

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
    <p class="text-body" style="color: var(--text-secondary)">{t('auth.restoringSession')}</p>
  </div>
{/snippet}

{#snippet loginRequired()}
  <div class="flex flex-col items-center justify-center py-20 text-center">
    <div class="mb-4 rounded-[var(--radius-lg)] surface-flat p-6">
      <Lock size={48} color="var(--text-secondary)" strokeWidth={2} />
    </div>
    <h2 class="mb-2 text-xl font-semibold" style="color: var(--text-primary);">
      {t('auth.loginRequired')}
    </h2>
    <p class="mb-6" style="color: var(--text-secondary);">{t('auth.loginRequiredDescription')}</p>
    <Button
      variant="default"
      size="lg"
      onclick={() => uiStore.openLoginModalWithRedirect({})}
    >
      {t('common.loginNow')}
    </Button>
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
      style="background-color: var(--surface-bg); padding-bottom: max(80px, env(safe-area-inset-bottom, 24px));"
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
        {@render gated()}
      </main>
    </div>
  </div>
{/if}
