<script lang="ts">
	import { withMinimumLoadingTime, MIN_LOADING_ACTION } from '$lib/utils/loading';
	import { authService, usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { t } from '$lib/i18n';
	import { authStore } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Eye from 'lucide-svelte/icons/eye';
	import EyeOff from 'lucide-svelte/icons/eye-off';
	import { validateField } from '$lib/utils/form-validation';
	import type { FieldRule } from '$lib/utils/form-validation';

	type SuccessHandler = () => void | Promise<void>;

	let { onSuccess = async () => {} }: { onSuccess?: SuccessHandler } = $props();

	let loginId = $state('');
	let password = $state('');
	let nickname = $state('');
	let confirmPassword = $state('');
	let guestNickname = $state('');

	let isLoading = $state(false);
	let showPassword = $state(false);
	let currentTab = $state('login');
	let touched = $state<Record<string, boolean>>({});
	const hasLetterAndDigit = (value: string) => /[A-Za-z]/.test(value) && /\d/.test(value);

	const loginIdRules: FieldRule[] = [
		{ required: true, requiredMessage: '请输入账号' },
		{ minLength: 3, minLengthMessage: '账号需 3-64 个字符' },
		{ maxLength: 64, maxLengthMessage: '账号需 3-64 个字符' }
	];
	const passwordRules: FieldRule[] = [
		{ required: true, requiredMessage: '请输入密码' },
		{ minLength: 8, minLengthMessage: '密码需为 8-128 位且同时包含字母和数字' },
		{ maxLength: 128, maxLengthMessage: '密码需为 8-128 位且同时包含字母和数字' },
		{
			custom: (v) =>
				v && !hasLetterAndDigit(v) ? '密码需为 8-128 位且同时包含字母和数字' : undefined
		}
	];
	const nicknameRules: FieldRule[] = [
		{ required: true, requiredMessage: '请输入昵称' },
		{ maxLength: 30, maxLengthMessage: '昵称最多 30 个字符' }
	];
	const confirmPasswordRules: FieldRule[] = [
		{ required: true, requiredMessage: '请确认密码' },
		{ custom: (v) => (v && v !== password ? '两次密码不一致' : undefined) }
	];
	const guestNicknameRules: FieldRule[] = [
		{ required: true, requiredMessage: '请输入昵称' },
		{ maxLength: 30, maxLengthMessage: '昵称最多 30 个字符' }
	];

	let loginIdError = $derived(
		touched['loginId'] ? validateField(loginId, loginIdRules) : undefined
	);
	let passwordError = $derived(
		touched['password'] ? validateField(password, passwordRules) : undefined
	);
	let nicknameError = $derived(
		touched['nickname'] ? validateField(nickname, nicknameRules) : undefined
	);
	let confirmPasswordError = $derived(
		touched['confirmPassword']
			? validateField(confirmPassword, confirmPasswordRules)
			: undefined
	);
	let guestNicknameError = $derived(
		touched['guestNickname'] ? validateField(guestNickname, guestNicknameRules) : undefined
	);

	function markTouched(field: string) {
		touched = { ...touched, [field]: true };
	}

	function resetForm() {
		loginId = '';
		password = '';
		nickname = '';
		confirmPassword = '';
		guestNickname = '';
		currentTab = 'login';
		touched = {};
		showPassword = false;
	}

	async function completeAuthFlow(): Promise<boolean> {
		const [statusRes, profileRes] = await Promise.all([
			authService.checkStatus(),
			usersService.getMyProfile()
		]);

		if (!isSuccess(statusRes) || !statusRes.data?.is_valid) {
			return false;
		}
		if (!isSuccess(profileRes) || !profileRes.data) {
			return false;
		}

		const expiresAt =
			statusRes.data.expires_at ?? new Date(Date.now() + 15 * 60 * 1000).toISOString();
		const profile = profileRes.data;

		authStore.login({
			user_id: profile.user_id,
			user_type: profile.user_type,
			access_token_expires_at: expiresAt
		});
		authStore.setProfile(profile);
		return true;
	}

	async function finishSuccess(successMessage: string) {
		toast.success(successMessage);
		await onSuccess();
		resetForm();
	}

	async function handleLogin() {
		touched = { ...touched, loginId: true, password: true };
		if (loginIdError || passwordError) return;

		const trimmedLoginId = loginId.trim();
		isLoading = true;
		try {
			const response = await withMinimumLoadingTime(
				() =>
					authService.login({
						login_id: trimmedLoginId,
						password,
						client_type: 'web'
					}),
				MIN_LOADING_ACTION
			);

			if (response.code === 0) {
				if (await completeAuthFlow()) {
					await finishSuccess(t('auth.loginSuccess'));
				} else {
					toast.error(t('auth.getProfileFailed'));
				}
			} else {
				toast.error(response.message || t('auth.loginFailed'));
			}
		} catch {
			toast.error(t('auth.loginFailedRetry'));
		} finally {
			isLoading = false;
		}
	}

	async function handleRegister() {
		touched = { ...touched, loginId: true, password: true, nickname: true, confirmPassword: true };
		if (loginIdError || passwordError || nicknameError || confirmPasswordError) return;

		const trimmedLoginId = loginId.trim();
		const trimmedNickname = nickname.trim();
		isLoading = true;
		try {
			const response = await withMinimumLoadingTime(
				() =>
					authService.register({
						login_id: trimmedLoginId,
						nickname: trimmedNickname,
						password,
						client_type: 'web'
					}),
				MIN_LOADING_ACTION
			);

			if (response.code === 0) {
				if (await completeAuthFlow()) {
					await finishSuccess(t('auth.registerSuccess'));
				} else {
					toast.error(t('auth.getProfileFailed'));
				}
			} else {
				toast.error(response.message || t('auth.registerFailed'));
			}
		} catch {
			toast.error(t('auth.registerFailedRetry'));
		} finally {
			isLoading = false;
		}
	}

	async function handleGuestLogin() {
		touched = { ...touched, guestNickname: true };
		if (guestNicknameError) return;

		const trimmedNickname = guestNickname.trim();
		isLoading = true;
		try {
			const response = await withMinimumLoadingTime(
				() =>
					authService.guest({
						nickname: trimmedNickname,
						client_type: 'web'
					}),
				MIN_LOADING_ACTION
			);

			if (response.code === 0) {
				if (await completeAuthFlow()) {
					await finishSuccess(t('auth.guestLoginSuccess'));
				} else {
					toast.error(t('auth.getProfileFailed'));
				}
			} else {
				toast.error(response.message || t('auth.guestLoginFailed'));
			}
		} catch {
			toast.error(t('auth.guestLoginFailedRetry'));
		} finally {
			isLoading = false;
		}
	}
</script>

<div class="mt-4">
	<!-- Tab bar -->
	<div class="grid w-full grid-cols-3 rounded-[24px] p-1.5" style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);">
		{#each [{ value: 'login', label: t('auth.login') }, { value: 'register', label: t('auth.register') }, { value: 'guest', label: t('auth.guest') }] as tab (tab.value)}
			<button
				type="button"
				class="relative rounded-[18px] px-3 py-2 outline-none transition-all"
				style="background: {currentTab === tab.value ? 'var(--neu-accent-primary)' : 'transparent'}; color: {currentTab === tab.value ? 'var(--neu-active-text)' : 'var(--neu-text-muted)'}; font-size: 13px; font-weight: {currentTab === tab.value ? 600 : 400}; box-shadow: {currentTab === tab.value ? '0 2px 8px var(--neu-glow)' : 'none'};"
				onclick={() => (currentTab = tab.value)}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<div class="min-h-[400px]">
		{#if currentTab === 'login'}
			<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4 pt-4">
				<div class="space-y-2">
					<label for="auth-login-id" style="font-size: 12px; color: var(--neu-text);">{t('auth.account')}</label>
					<input
						id="auth-login-id"
						type="text"
						placeholder={t('auth.phoneOrEmail')}
						autocomplete="username"
						bind:value={loginId}
						disabled={isLoading}
						onblur={() => markTouched('loginId')}
						class="rounded-[14px] px-4 py-2.5 outline-none w-full"
						style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid {loginIdError ? 'var(--neu-error)' : 'var(--neu-border)'}; font-size: 13px;"
					/>
					{#if loginIdError}
						<p style="font-size: 12px; color: var(--neu-error);">{loginIdError}</p>
					{/if}
				</div>
				<div class="space-y-2">
					<label for="auth-login-password" style="font-size: 12px; color: var(--neu-text);">{t('auth.password')}</label>
					<div class="relative">
						<input
							id="auth-login-password"
							type={showPassword ? 'text' : 'password'}
							placeholder={t('auth.enterPassword')}
							autocomplete="current-password"
							bind:value={password}
							disabled={isLoading}
							onblur={() => markTouched('password')}
							class="rounded-[14px] px-4 py-2.5 outline-none w-full pr-10"
							style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid {passwordError ? 'var(--neu-error)' : 'var(--neu-border)'}; font-size: 13px;"
						/>
						<button
							type="button"
							class="absolute right-1 top-1 flex h-9 w-9 items-center justify-center rounded-[12px] outline-none"
							style="background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
							onclick={() => (showPassword = !showPassword)}
							aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
						>
							{#if showPassword}
								<EyeOff style="width: 16px; height: 16px; color: var(--neu-text);" />
							{:else}
								<Eye style="width: 16px; height: 16px; color: var(--neu-text);" />
							{/if}
						</button>
					</div>
					{#if passwordError}
						<p style="font-size: 12px; color: var(--neu-error);">{passwordError}</p>
					{/if}
				</div>
				<button type="submit" class="rounded-full px-5 py-2.5 outline-none w-full" style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 14px; box-shadow: 0 3px 10px var(--neu-glow);" disabled={isLoading}>
					{#if isLoading}
						<Loader2 class="mr-2 inline-block" style="width: 16px; height: 16px;" />
					{/if}
					{t('auth.login')}
				</button>
			</form>
		{:else if currentTab === 'register'}
			<form onsubmit={(e) => { e.preventDefault(); handleRegister(); }} class="space-y-4 pt-4">
				<div class="space-y-2">
					<label for="auth-register-nickname" style="font-size: 12px; color: var(--neu-text);">{t('auth.nickname')}</label>
					<input
						id="auth-register-nickname"
						type="text"
						placeholder={t('auth.giveYourselfAName')}
						autocomplete="nickname"
						bind:value={nickname}
						disabled={isLoading}
						onblur={() => markTouched('nickname')}
						class="rounded-[14px] px-4 py-2.5 outline-none w-full"
						style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid {nicknameError ? 'var(--neu-error)' : 'var(--neu-border)'}; font-size: 13px;"
					/>
					{#if nicknameError}
						<p style="font-size: 12px; color: var(--neu-error);">{nicknameError}</p>
					{/if}
				</div>
				<div class="space-y-2">
					<label for="auth-register-id" style="font-size: 12px; color: var(--neu-text);">{t('auth.account')}</label>
					<input
						id="auth-register-id"
						type="text"
						placeholder={t('auth.phoneOrEmail')}
						autocomplete="username"
						bind:value={loginId}
						disabled={isLoading}
						onblur={() => markTouched('loginId')}
						class="rounded-[14px] px-4 py-2.5 outline-none w-full"
						style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid {loginIdError ? 'var(--neu-error)' : 'var(--neu-border)'}; font-size: 13px;"
					/>
					{#if loginIdError}
						<p style="font-size: 12px; color: var(--neu-error);">{loginIdError}</p>
					{/if}
				</div>
				<div class="space-y-2">
					<label for="auth-register-password" style="font-size: 12px; color: var(--neu-text);">{t('auth.password')}</label>
					<input
						id="auth-register-password"
						type="password"
						placeholder={t('auth.atLeast8Chars')}
						autocomplete="new-password"
						bind:value={password}
						disabled={isLoading}
						onblur={() => markTouched('password')}
						class="rounded-[14px] px-4 py-2.5 outline-none w-full"
						style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid {passwordError ? 'var(--neu-error)' : 'var(--neu-border)'}; font-size: 13px;"
					/>
					{#if passwordError}
						<p style="font-size: 12px; color: var(--neu-error);">{passwordError}</p>
					{/if}
				</div>
				<div class="space-y-2">
					<label for="auth-register-confirm" style="font-size: 12px; color: var(--neu-text);">{t('auth.confirmPassword')}</label>
					<input
						id="auth-register-confirm"
						type="password"
						placeholder={t('auth.enterPasswordAgain')}
						autocomplete="new-password"
						bind:value={confirmPassword}
						disabled={isLoading}
						onblur={() => markTouched('confirmPassword')}
						class="rounded-[14px] px-4 py-2.5 outline-none w-full"
						style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid {confirmPasswordError ? 'var(--neu-error)' : 'var(--neu-border)'}; font-size: 13px;"
					/>
					{#if confirmPasswordError}
						<p style="font-size: 12px; color: var(--neu-error);">{confirmPasswordError}</p>
					{/if}
				</div>
				<button type="submit" class="rounded-full px-5 py-2.5 outline-none w-full" style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 14px; box-shadow: 0 3px 10px var(--neu-glow);" disabled={isLoading}>
					{#if isLoading}
						<Loader2 class="mr-2 inline-block" style="width: 16px; height: 16px;" />
					{/if}
					{t('auth.register')}
				</button>
			</form>
		{:else if currentTab === 'guest'}
			<form onsubmit={(e) => { e.preventDefault(); handleGuestLogin(); }} class="space-y-4 pt-4">
				<div class="rounded-[24px] p-3" style="background: var(--neu-bg); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); font-size: 13px; color: var(--neu-text-muted);">
					{t('auth.guestModeHint')}
				</div>
				<div class="space-y-2">
					<label for="auth-guest-nickname" style="font-size: 12px; color: var(--neu-text);">{t('auth.nickname')}</label>
					<input
						id="auth-guest-nickname"
						type="text"
						placeholder={t('auth.giveYourselfAName')}
						autocomplete="nickname"
						bind:value={guestNickname}
						disabled={isLoading}
						onblur={() => markTouched('guestNickname')}
						class="rounded-[14px] px-4 py-2.5 outline-none w-full"
						style="background: var(--neu-bg); color: var(--neu-text); box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid {guestNicknameError ? 'var(--neu-error)' : 'var(--neu-border)'}; font-size: 13px;"
					/>
					{#if guestNicknameError}
						<p style="font-size: 12px; color: var(--neu-error);">{guestNicknameError}</p>
					{/if}
				</div>
				<button type="submit" class="rounded-full px-5 py-2.5 outline-none w-full" style="background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 14px; box-shadow: 0 3px 10px var(--neu-glow);" disabled={isLoading}>
					{#if isLoading}
						<Loader2 class="mr-2 inline-block" style="width: 16px; height: 16px;" />
					{/if}
					{t('auth.guestLogin')}
				</button>
			</form>
		{/if}
	</div>
</div>
