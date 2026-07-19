<script lang="ts">
	  import { page } from '$app/state';
	  import { goto } from '$app/navigation';
	  import { authService } from '$lib/api/services';
	  import { isSuccess } from '$lib/api/client';
	  import { toast } from '$lib/stores/toast.svelte';
	  import { t } from '$lib/i18n';
  import { authStore } from '$lib/stores/auth.svelte';
  import { usersService } from '$lib/api/services';
  import { validateField } from '$lib/utils/form-validation';
  import { withMinimumLoadingTime, MIN_LOADING_ACTION } from '$lib/utils/loading';
  import type { FieldRule } from '$lib/utils/form-validation';
  import Shield from 'lucide-svelte/icons/shield';
  import Spinner from '$lib/components/common/spinner.svelte';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import Check from 'lucide-svelte/icons/check';

  type Step = 'request' | 'reset' | 'done';

  let step = $state<Step>('request');
  let loginId = $state(page.url.searchParams.get('login_id') ?? '');
  let verificationCode = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let isLoading = $state(false);
  let touched = $state<Record<string, boolean>>({});

  const loginIdRules: FieldRule[] = [
    { required: true, requiredMessage: '请输入账号' },
    { minLength: 3, minLengthMessage: '账号格式不正确' },
    { maxLength: 64, maxLengthMessage: '账号格式不正确' }
  ];
  const codeRules: FieldRule[] = [
    { required: true, requiredMessage: '请输入验证码' }
  ];
  const passwordRules: FieldRule[] = [
    { required: true, requiredMessage: '请输入新密码' },
    { minLength: 8, minLengthMessage: '密码需为 8-128 位且同时包含字母和数字' },
    { maxLength: 128, maxLengthMessage: '密码需为 8-128 位且同时包含字母和数字' },
    {
      custom: (v) =>
        v && !(/[A-Za-z]/.test(v) && /\d/.test(v)) ? '密码需为 8-128 位且同时包含字母和数字' : undefined
    }
  ];
  const confirmRules: FieldRule[] = [
    { required: true, requiredMessage: '请确认密码' },
    { custom: (v) => (v && v !== newPassword ? '两次密码不一致' : undefined) }
  ];

  let loginIdError = $derived(touched['loginId'] ? validateField(loginId, loginIdRules) : undefined);
  let codeError = $derived(touched['code'] ? validateField(verificationCode, codeRules) : undefined);
  let passwordError = $derived(touched['password'] ? validateField(newPassword, passwordRules) : undefined);
  let confirmError = $derived(touched['confirm'] ? validateField(confirmPassword, confirmRules) : undefined);

  function markTouched(field: string) {
    touched = { ...touched, [field]: true };
  }

  async function handleRequest() {
    touched = { ...touched, loginId: true };
    if (loginIdError) return;

    isLoading = true;
    try {
      const response = await withMinimumLoadingTime(
        () => authService.requestPasswordReset({ login_id: loginId.trim(), client_type: 'web' }),
        MIN_LOADING_ACTION
      );
      if (response.code === 0) {
        step = 'reset';
        toast.success(t('resetPassword.codeSent'));
      } else {
        toast.error(response.message || t('resetPassword.sendCodeFailed'));
      }
    } catch {
      toast.error(t('resetPassword.networkError'));
    } finally {
      isLoading = false;
    }
  }

  async function handleReset() {
    touched = { ...touched, code: true, password: true, confirm: true };
    if (codeError || passwordError || confirmError) return;

    isLoading = true;
    try {
      const response = await withMinimumLoadingTime(
        () =>
          authService.resetPassword({
            login_id: loginId.trim(),
            verification_code: verificationCode.trim(),
            new_password: newPassword,
            client_type: 'web'
          }),
        MIN_LOADING_ACTION
      );
      if (response.code === 0) {
        // 重置成功后自动登录，补全本地状态
        const [statusRes, profileRes] = await Promise.all([
          authService.checkStatus(),
          usersService.getMyProfile()
        ]);
        if (isSuccess(statusRes) && statusRes.data?.is_valid && isSuccess(profileRes) && profileRes.data) {
          authStore.login({
            user_id: profileRes.data.user_id,
            user_type: profileRes.data.user_type,
            access_token_expires_at:
              statusRes.data.expires_at ?? new Date(Date.now() + 15 * 60 * 1000).toISOString()
          });
          authStore.setProfile(profileRes.data);
        }
        step = 'done';
      } else {
        toast.error(response.message || t('resetPassword.resetFailed'));
      }
    } catch {
      toast.error(t('resetPassword.networkError'));
    } finally {
      isLoading = false;
    }
  }
</script>

<svelte:head>
  <title>{step === 'request' ? '重置密码' : step === 'reset' ? '设置新密码' : '密码已重置'} - HotSpot</title>
</svelte:head>

<div class="flex min-h-screen items-center justify-center page-pad">
  <div
    class="page-shell page-shell--form w-full rounded-[var(--radius-lg)] p-6"
    style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
  >
    <div class="mb-6 text-center">
      <div
        class="mx-auto mb-4 flex items-center justify-center rounded-full"
        style="width: 48px; height: 48px; border: 1px solid var(--border-theme);"
      >
        {#if step === 'done'}
          <Check size={19} color="var(--brand)" />
        {:else}
          <Shield size={19} color="var(--brand)" />
        {/if}
      </div>

      {#if step === 'request'}
        <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">重置密码</h1>
        <p class="text-body" style="color: var(--text-secondary); margin-top: 6px">
          输入你的注册邮箱或手机号，我们将发送验证码
        </p>
      {:else if step === 'reset'}
        <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">设置新密码</h1>
        <p class="text-body" style="color: var(--text-secondary); margin-top: 6px">
          验证码已发送至 {loginId}，请查收
        </p>
      {:else}
        <h1 class="text-h1" style="font-weight: 600; color: var(--text-primary)">密码已重置</h1>
        <p class="text-body" style="color: var(--text-secondary); margin-top: 6px">
          你的密码已成功修改，已自动登录
        </p>
      {/if}
    </div>

    {#if step === 'request'}
      <form
        onsubmit={(e) => { e.preventDefault(); handleRequest(); }}
        class="space-y-4"
      >
        <div class="space-y-2">
          <label for="reset-login-id" class="text-meta" style="color: var(--text-primary)">邮箱或手机号</label>
          <input
            id="reset-login-id"
            type="text"
            placeholder="输入注册时使用的邮箱或手机号"
            autocomplete="username"
            bind:value={loginId}
            disabled={isLoading}
            onblur={() => markTouched('loginId')}
            class="text-body input-base rounded-[var(--radius-md)] px-4 py-2.5 outline-none w-full"
            style="background: var(--surface-bg); color: var(--text-primary); border: 1px solid {loginIdError ? 'var(--error)' : 'var(--border-theme)'}"
          />
          {#if loginIdError}
            <p class="text-meta" style="color: var(--error)">{loginIdError}</p>
          {/if}
        </div>
        <button
          type="submit"
          class="text-body-lg rounded-full px-5 py-2.5 outline-none w-full transition-[transform,box-shadow] duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.98] focus-visible:ring-[2px] focus-visible:ring-[var(--glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-bg)]"
          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
          disabled={isLoading}
        >
          {#if isLoading}
            <Spinner size="md" class="mr-2" />
          {/if}
          发送验证码
        </button>
      </form>
    {:else if step === 'reset'}
      <form
        onsubmit={(e) => { e.preventDefault(); handleReset(); }}
        class="space-y-4"
      >
        <div class="space-y-2">
          <label for="reset-code" class="text-meta" style="color: var(--text-primary)">验证码</label>
          <input
            id="reset-code"
            type="text"
            placeholder="输入收到的验证码"
            autocomplete="one-time-code"
            bind:value={verificationCode}
            disabled={isLoading}
            onblur={() => markTouched('code')}
            class="text-body input-base rounded-[var(--radius-md)] px-4 py-2.5 outline-none w-full"
            style="background: var(--surface-bg); color: var(--text-primary); border: 1px solid {codeError ? 'var(--error)' : 'var(--border-theme)'}"
          />
          {#if codeError}
            <p class="text-meta" style="color: var(--error)">{codeError}</p>
          {/if}
        </div>
        <div class="space-y-2">
          <label for="reset-password" class="text-meta" style="color: var(--text-primary)">新密码</label>
          <input
            id="reset-password"
            type="password"
            placeholder="至少 8 位，包含字母和数字"
            autocomplete="new-password"
            bind:value={newPassword}
            disabled={isLoading}
            onblur={() => markTouched('password')}
            class="text-body input-base rounded-[var(--radius-md)] px-4 py-2.5 outline-none w-full"
            style="background: var(--surface-bg); color: var(--text-primary); border: 1px solid {passwordError ? 'var(--error)' : 'var(--border-theme)'}"
          />
          {#if passwordError}
            <p class="text-meta" style="color: var(--error)">{passwordError}</p>
          {/if}
        </div>
        <div class="space-y-2">
          <label for="reset-confirm" class="text-meta" style="color: var(--text-primary)">确认新密码</label>
          <input
            id="reset-confirm"
            type="password"
            placeholder="再次输入新密码"
            autocomplete="new-password"
            bind:value={confirmPassword}
            disabled={isLoading}
            onblur={() => markTouched('confirm')}
            class="text-body input-base rounded-[var(--radius-md)] px-4 py-2.5 outline-none w-full"
            style="background: var(--surface-bg); color: var(--text-primary); border: 1px solid {confirmError ? 'var(--error)' : 'var(--border-theme)'}"
          />
          {#if confirmError}
            <p class="text-meta" style="color: var(--error)">{confirmError}</p>
          {/if}
        </div>
        <button
          type="submit"
          class="text-body-lg rounded-full px-5 py-2.5 outline-none w-full transition-[transform,box-shadow] duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.98] focus-visible:ring-[2px] focus-visible:ring-[var(--glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-bg)]"
          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
          disabled={isLoading}
        >
          {#if isLoading}
            <Spinner size="md" class="mr-2" />
          {/if}
          重置密码
        </button>
      </form>
    {:else}
      <div class="space-y-4">
        <button
          class="text-body-lg rounded-full px-5 py-2.5 outline-none w-full transition-[transform,box-shadow] duration-150 hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.98] focus-visible:ring-[2px] focus-visible:ring-[var(--glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-bg)]"
          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn)"
          onclick={() => goto('/')}
        >
          返回首页
        </button>
        <button
          class="text-body-lg rounded-full px-5 py-2.5 outline-none w-full transition-[transform,box-shadow] duration-150 active:scale-[0.98] focus-visible:ring-[2px] focus-visible:ring-[var(--glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-bg)]"
          style="background: var(--surface-bg); color: var(--text-primary); border: 1px solid var(--border-theme);"
          onclick={() => goto('/auth')}
        >
          返回登录
        </button>
      </div>
    {/if}

    {#if step !== 'done'}
      <div class="mt-6 text-center">
        <a
          href="/auth"
          class="text-body inline-flex items-center gap-1 outline-none hover:underline"
          style="color: var(--text-secondary)"
        >
          <ArrowLeft size={14} strokeWidth={1.5} />
          返回登录
        </a>
      </div>
    {/if}
  </div>
</div>
