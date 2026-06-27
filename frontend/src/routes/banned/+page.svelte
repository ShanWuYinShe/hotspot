<!--
  账号封禁页
  hooks.client.ts 在收到后端 10201（账号封禁）时跳转至此，携带 ?reason=...
  封禁账号不显示应用外壳，用户需重新登录或申诉
-->
<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth.svelte';
  import { uiStore } from '$lib/stores/ui.svelte';
  import ShieldBan from 'lucide-svelte/icons/shield-ban';
  import Home from 'lucide-svelte/icons/home';

  const reason = $derived(page.url.searchParams.get('reason')?.trim() || '');

  function goLogin() {
    // 封禁后可能仍有其它未封禁账户，统一走登录流程
    uiStore.openLoginModalWithRedirect({ url: '/' });
  }
</script>

<svelte:head>
  <title>账号封禁 - HotSpot</title>
</svelte:head>

<div
  class="flex min-h-[80vh] flex-col items-center justify-center gap-6 px-4 text-center banned-enter"
>
  <div
    class="flex h-20 w-20 items-center justify-center rounded-[var(--radius-button)]"
    style="background: var(--surface-bg); box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme);"
  >
    <ShieldBan size={36} color="var(--theme-error)" />
  </div>

  <div class="space-y-2">
    <h1
      style="font-size: var(--text-display-lg); font-weight: 700; color: var(--text-primary); letter-spacing: -0.02em;"
    >
      账号已被封禁
    </h1>
    <p class="text-title" style="color: var(--text-secondary)">
      你的账号因违反社区规范被封禁，相关功能已被限制。
    </p>
    {#if reason}
      <p
        class="mx-auto max-w-md rounded-2xl px-4 py-3 text-body"
        style="background: var(--surface-bg); border: 1px solid var(--border-theme); color: var(--text-secondary);"
      >
        {reason}
      </p>
    {/if}
  </div>

  <div class="flex gap-3">
    {#if authStore.accounts.length > 0}
      <button
        class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
        style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
        onclick={goLogin}
        aria-label="切换账号登录"
      >
        切换账号
      </button>
    {/if}
    <button
      class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
      style="box-shadow: var(--shadow-sm); border: 1px solid var(--border-theme); color: var(--text-secondary); background: var(--surface-bg);"
      onclick={() => goto('/')}
      aria-label="返回首页"
    >
      <Home size={16} />
      返回首页
    </button>
  </div>
</div>

<style>
  .banned-enter {
    animation: banned-enter 0.3s ease-out;
  }
  @keyframes banned-enter {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
</style>
