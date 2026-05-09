<script lang="ts">
  import { uiStore } from '$lib/stores/ui.svelte';
  import { authStore } from '$lib/stores/auth.svelte';

  function handleLogin() {
    uiStore.navigateToAuth();
    uiStore.closeAuthPrompt();
  }

  function handleDismiss() {
    uiStore.closeAuthPrompt();
  }
</script>

{#if uiStore.showAuthPrompt && !authStore.isAuthenticated}
  <!-- 遮罩 -->
  <button class="auth-prompt-overlay" onclick={handleDismiss} aria-label="关闭登录提示"></button>

  <!-- 提示卡片 -->
  <div class="auth-prompt-card" role="alertdialog" aria-modal="true" aria-label="请先登录">
    <div class="auth-prompt-icon">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="20"
        height="20"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
        <path d="M7 11V7a5 0 0 1 10 0v4" />
      </svg>
    </div>
    <div class="auth-prompt-content">
      <p class="auth-prompt-title">请先登录</p>
      <p class="auth-prompt-desc">登录后即可使用完整功能</p>
    </div>
    <div class="auth-prompt-actions">
      <button class="auth-prompt-dismiss" onclick={handleDismiss}> 稍后 </button>
      <button class="auth-prompt-login" onclick={handleLogin}> 去登录 </button>
    </div>
  </div>
{/if}

<style>
  .auth-prompt-overlay {
    position: fixed;
    inset: 0;
    z-index: 90;
    background: transparent;
    border: none;
    cursor: default;
  }

  .auth-prompt-card {
    position: fixed;
    bottom: 24px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 91;
    width: calc(100vw - 48px);
    max-width: 420px;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 16px;
    border-radius: 22px;
    background: var(--neu-bg);
    border: 1px solid var(--neu-border);
    box-shadow: var(--neu-shadow-lg);
    animation: slideUp 0.25s cubic-bezier(0.22, 1, 0.36, 1);
  }

  @keyframes slideUp {
    from {
      opacity: 0;
      transform: translateX(-50%) translateY(16px);
    }
    to {
      opacity: 1;
      transform: translateX(-50%) translateY(0);
    }
  }

  .auth-prompt-icon {
    flex-shrink: 0;
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 14px;
    background: var(--neu-bg);
    border: 1px solid var(--neu-border);
    box-shadow:
      4px 4px 9px var(--neu-shadow-dark),
      -4px -4px 9px var(--neu-shadow-light);
    color: var(--neu-accent-primary);
  }

  .auth-prompt-content {
    flex: 1;
    min-width: 0;
  }

  .auth-prompt-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--neu-text);
  }

  .auth-prompt-desc {
    font-size: 12px;
    color: var(--neu-text-muted);
    margin-top: 2px;
  }

  .auth-prompt-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }

  .auth-prompt-dismiss {
    height: 34px;
    padding: 0 14px;
    border-radius: 999px;
    border: 1px solid var(--neu-border);
    background: var(--neu-bg);
    font-size: 13px;
    font-weight: 500;
    color: var(--neu-text-muted);
    cursor: pointer;
    transition: all 0.15s;
  }

  .auth-prompt-dismiss:hover {
    color: var(--neu-text);
  }

  .auth-prompt-login {
    height: 34px;
    padding: 0 18px;
    border-radius: 999px;
    border: none;
    background: var(--neu-accent-primary);
    font-size: 13px;
    font-weight: 600;
    color: var(--neu-active-text);
    box-shadow: 0 3px 10px var(--neu-glow);
    cursor: pointer;
    transition: all 0.15s;
  }

  .auth-prompt-login:hover {
    opacity: 0.9;
  }

  .auth-prompt-login:active {
    transform: scale(0.97);
  }
</style>
