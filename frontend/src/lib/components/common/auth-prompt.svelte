<script lang="ts">
  import { uiStore } from '$lib/stores/ui.svelte';
  import { authStore } from '$lib/stores/auth.svelte';
  import Lock from 'lucide-svelte/icons/lock';

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
      <Lock size={20} />
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
    z-index: var(--z-overlay);
    background: transparent;
    border: none;
    cursor: default;
  }

  .auth-prompt-card {
    position: fixed;
    bottom: max(24px, env(safe-area-inset-bottom, 24px));
    left: 50%;
    transform: translateX(-50%);
    z-index: var(--z-modal);
    width: calc(100vw - 48px);
    max-width: 420px;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 16px;
    border-radius: var(--radius-lg);
    background: var(--surface-bg);
    border: 1px solid var(--border-theme);
    box-shadow: var(--shadow-lg);
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
    border-radius: var(--radius-sm);
    background: var(--surface-bg);
    border: 1px solid var(--border-theme);
    box-shadow:
      4px 4px 9px var(--shadow-dark),
      -4px -4px 9px var(--shadow-light);
    color: var(--brand);
  }

  .auth-prompt-content {
    flex: 1;
    min-width: 0;
  }

  .auth-prompt-title {
    font-size: var(--text-body-lg);
    font-weight: 600;
    color: var(--text-primary);
  }

  .auth-prompt-desc {
    font-size: var(--text-meta);
    color: var(--text-secondary);
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
    border-radius: var(--radius-pill);
    border: 1px solid var(--border-theme);
    background: var(--surface-bg);
    font-size: var(--text-body);
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.15s;
  }

  .auth-prompt-dismiss:hover {
    color: var(--text-primary);
  }

  .auth-prompt-login {
    height: 34px;
    padding: 0 18px;
    border-radius: var(--radius-pill);
    border: none;
    background: var(--brand);
    font-size: var(--text-body);
    font-weight: 600;
    color: var(--text-on-accent);
    box-shadow: var(--shadow-brand-btn);
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
