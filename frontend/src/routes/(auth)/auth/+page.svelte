<script lang="ts">
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { untrack } from 'svelte';
  import { authStore } from '$lib/stores/auth.svelte';
  import { normalizeRedirectPath, uiStore } from '$lib/stores/ui.svelte';
  import AuthFormContent from '$lib/components/common/auth-form-content.svelte';
  import Shield from 'lucide-svelte/icons/shield';

  $effect(() => {
    if (browser && authStore.isAuthenticated) {
      untrack(() => {
        const redirect = page.url.searchParams.get('redirect');
        const target = normalizeRedirectPath(redirect) ?? '/';
        goto(target, { replaceState: true });
      });
    }
  });

  async function handleSuccess() {
    uiStore.onLoginSuccess();
    const redirect = page.url.searchParams.get('redirect');
    const target = normalizeRedirectPath(redirect) ?? '/';
    goto(target, { replaceState: true });
  }
</script>

<svelte:head>
  <title>登录 - HotSpot</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center page-pad">
  <div
    class="page-shell page-shell--form w-full rounded-[var(--radius-lg)] p-6"
    style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
  >
    <div class="mb-6 text-center">
      <div
        class="mx-auto mb-4 flex items-center justify-center rounded-full"
        style="width: 48px; height: 48px; box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
      >
        <Shield size={19} color="var(--brand)" />
      </div>
      <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">登录后继续</h1>
      <p class="text-body" style="color: var(--text-secondary); margin-top: 6px">
        使用你的账号、注册新账号或继续访客体验
      </p>
    </div>

    <AuthFormContent onSuccess={handleSuccess} />
  </div>
</div>
