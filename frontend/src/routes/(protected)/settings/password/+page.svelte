<script lang="ts">
  import { authService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import { toast } from '$lib/stores/toast.svelte';
  import { t } from '$lib/i18n';
  import PageHeader from '$lib/components/layout/page-header.svelte';
  import Loader2 from 'lucide-svelte/icons/loader-2';
  import { validateField } from '$lib/utils/form-validation';
  import type { FieldRule } from '$lib/utils/form-validation';

  let oldPassword = $state('');
  let newPassword = $state('');
  let confirmPassword = $state('');
  let isSaving = $state(false);
  let touched = $state<Record<string, boolean>>({});

  const hasLetterAndDigit = (value: string) => /[A-Za-z]/.test(value) && /\d/.test(value);

  const oldPasswordRules = $derived<FieldRule[]>([
    { required: true, requiredMessage: t('settings.currentPasswordRequired') }
  ]);
  const newPasswordRules = $derived<FieldRule[]>([
    { required: true, requiredMessage: t('settings.newPasswordRequired') },
    { minLength: 8, minLengthMessage: t('settings.newPasswordMinLength') },
    { maxLength: 128, maxLengthMessage: t('settings.newPasswordMaxLength') },
    {
      custom: (v) =>
        v && !hasLetterAndDigit(v) ? t('settings.newPasswordNeedsLettersAndDigits') : undefined
    }
  ]);
  const confirmPasswordRules = $derived<FieldRule[]>([
    { required: true, requiredMessage: t('settings.confirmPasswordRequired') },
    {
      custom: (v) => (v && v !== newPassword ? t('settings.passwordMismatch') : undefined)
    }
  ]);

  let oldPasswordError = $derived(
    touched['oldPassword'] ? validateField(oldPassword, oldPasswordRules) : undefined
  );
  let newPasswordError = $derived(
    touched['newPassword'] ? validateField(newPassword, newPasswordRules) : undefined
  );
  let confirmPasswordError = $derived(
    touched['confirmPassword'] ? validateField(confirmPassword, confirmPasswordRules) : undefined
  );

  function markTouched(field: string) {
    touched = { ...touched, [field]: true };
  }

  function markAllTouched() {
    touched = { oldPassword: true, newPassword: true, confirmPassword: true };
  }

  function hasErrors(): boolean {
    const nextOldPasswordError = validateField(oldPassword, oldPasswordRules);
    const nextNewPasswordError = validateField(newPassword, newPasswordRules);
    const nextConfirmPasswordError = validateField(confirmPassword, confirmPasswordRules);
    return !!nextOldPasswordError || !!nextNewPasswordError || !!nextConfirmPasswordError;
  }

  async function handleSubmit() {
    markAllTouched();
    if (hasErrors()) return;

    isSaving = true;
    try {
      const response = await authService.changePassword({
        old_password: oldPassword,
        new_password: newPassword
      });
      if (isSuccess(response)) {
        toast.success(t('settings.passwordUpdated'));
        oldPassword = '';
        newPassword = '';
        confirmPassword = '';
        touched = {};
      } else {
        toast.error(response.message || t('settings.passwordChangeFailed'));
      }
    } catch {
      toast.error(t('settings.passwordChangeFailed'));
    } finally {
      isSaving = false;
    }
  }
</script>

<svelte:head>
  <title>{t('settings.passwordPageTitle')}</title>
</svelte:head>

<div class="page-pad">
  <div class="page-shell page-shell--form space-y-5">
    <PageHeader
      title={t('settings.passwordHeader')}
      description={t('settings.passwordHeaderDesc')}
      backHref="/settings"
      backLabel={t('settings.backToSettings')}
    />

    <div
      class="rounded-[var(--radius-lg)] p-5"
      style="background: var(--surface-bg); box-shadow: var(--shadow-lg); border: 1px solid var(--border-theme);"
    >
      <p
        class="text-body-lg"
        style="font-weight: 600; color: var(--text-primary); margin-bottom: 16px"
      >
        {t('settings.accountSecurity')}
      </p>

      <form
        onsubmit={(e) => {
          e.preventDefault();
          handleSubmit();
        }}
        class="space-y-4"
      >
        <div class="space-y-2">
          <label for="old-password" class="text-body" style="color: var(--text-secondary)"
            >{t('settings.currentPassword')}</label
          >
          <input
            id="old-password"
            type="password"
            bind:value={oldPassword}
            onblur={() => markTouched('oldPassword')}
            class="input-base w-full rounded-[var(--radius-md)] px-4 py-2.5 outline-none"
            style="border: 1px solid {oldPasswordError
              ? 'var(--theme-error)'
              : 'var(--border-theme)'}; background: var(--surface-bg); color: var(--text-primary);"
          />
          {#if oldPasswordError}
            <p class="text-meta" style="color: var(--theme-error)">{oldPasswordError}</p>
          {/if}
        </div>
        <div class="space-y-2">
          <label for="new-password" class="text-body" style="color: var(--text-secondary)"
            >{t('settings.newPassword')}</label
          >
          <input
            id="new-password"
            type="password"
            bind:value={newPassword}
            onblur={() => markTouched('newPassword')}
            class="input-base w-full rounded-[var(--radius-md)] px-4 py-2.5 outline-none"
            style="border: 1px solid {newPasswordError
              ? 'var(--theme-error)'
              : 'var(--border-theme)'}; background: var(--surface-bg); color: var(--text-primary);"
          />
          {#if newPasswordError}
            <p class="text-meta" style="color: var(--theme-error)">{newPasswordError}</p>
          {/if}
        </div>
        <div class="space-y-2">
          <label for="confirm-password" class="text-body" style="color: var(--text-secondary)"
            >{t('settings.confirmNewPassword')}</label
          >
          <input
            id="confirm-password"
            type="password"
            bind:value={confirmPassword}
            onblur={() => markTouched('confirmPassword')}
            class="input-base w-full rounded-[var(--radius-md)] px-4 py-2.5 outline-none"
            style="border: 1px solid {confirmPasswordError
              ? 'var(--theme-error)'
              : 'var(--border-theme)'}; background: var(--surface-bg); color: var(--text-primary);"
          />
          {#if confirmPasswordError}
            <p class="text-meta" style="color: var(--theme-error)">{confirmPasswordError}</p>
          {/if}
        </div>
        <button
          type="submit"
          class="flex h-11 w-full items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
          style="background: var(--brand); color: var(--text-on-accent); box-shadow: var(--shadow-brand-btn);"
          disabled={isSaving}
        >
          {#if isSaving}
            <Loader2 size={16} class="mr-2 animate-spin" />
          {/if}
          {t('settings.saveChanges')}
        </button>
      </form>
    </div>
  </div>
</div>
