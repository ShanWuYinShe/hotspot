<script lang="ts">
	import { withMinimumLoadingTime, MIN_LOADING_ACTION } from '$lib/utils/loading';
	import { authService, usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { t } from '$lib/i18n';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Eye from 'lucide-svelte/icons/eye';
	import EyeOff from 'lucide-svelte/icons/eye-off';
	import { validateField } from '$lib/utils/form-validation';
	import type { FieldRule } from '$lib/utils/form-validation';

	// 表单状态
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

	// 验证规则
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

	// 内联错误
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

	function handleClose() {
		uiStore.closeLoginModal();
		// 清空表单
		loginId = '';
		password = '';
		nickname = '';
		confirmPassword = '';
		guestNickname = '';
		currentTab = 'login';
		touched = {};
	}

	/**
	 * 完成认证后的通用流程
	 * spec: Web 端 auth 响应体为空，需通过 check_status + getMyProfile 获取用户信息
	 */
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
				// spec: Web 端 body 为空，token 通过 Cookie 设置
				if (await completeAuthFlow()) {
					toast.success(t('auth.loginSuccess'));
					uiStore.onLoginSuccess();
					loginId = '';
					password = '';
				} else {
					toast.error(t('auth.getProfileFailed'));
				}
			} else {
				toast.error(response.message || t('auth.loginFailed'));
			}
		} catch (error) {
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
					toast.success(t('auth.registerSuccess'));
					uiStore.onLoginSuccess();
				} else {
					toast.error(t('auth.getProfileFailed'));
				}
			} else {
				toast.error(response.message || t('auth.registerFailed'));
			}
		} catch (error) {
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
					toast.success(t('auth.guestLoginSuccess'));
					uiStore.onLoginSuccess();
				} else {
					toast.error(t('auth.getProfileFailed'));
				}
			} else {
				toast.error(response.message || t('auth.guestLoginFailed'));
			}
		} catch (error) {
			toast.error(t('auth.guestLoginFailedRetry'));
		} finally {
			isLoading = false;
		}
	}

</script>

<Dialog.Root
	open={uiStore.showLoginModal}
	onOpenChange={(open) => {
		if (!open) {
			handleClose();
		}
	}}
>
	<Dialog.Content class="sm:max-w-md rounded-3xl border border-border/60 bg-card/95 shadow-xl backdrop-blur-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-3">
				<div class="flex h-10 w-10 items-center justify-center rounded-2xl border border-border/50 bg-primary/10 text-primary">
					<MapPin class="h-5 w-5" />
				</div>
				<div>{t('auth.welcome')}</div>
			</Dialog.Title>
			<Dialog.Description>
				{t('auth.loginSubtitle')}
			</Dialog.Description>
		</Dialog.Header>

		<Tabs bind:value={currentTab} class="mt-4">
			<TabsList class="grid w-full grid-cols-3 rounded-3xl bg-background/75 p-1.5 ring-1 ring-border/60">
				<TabsTrigger value="login" class="rounded-2xl">{t('auth.login')}</TabsTrigger>
				<TabsTrigger value="register" class="rounded-2xl">{t('auth.register')}</TabsTrigger>
				<TabsTrigger value="guest" class="rounded-2xl">{t('auth.guest')}</TabsTrigger>
			</TabsList>

			<!-- 固定最小高度以防止跳动（需容纳注册表单 4 个输入框 + 按钮） -->
			<div class="min-h-[400px]">
				<!-- 登录 -->
				<TabsContent value="login">
				<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4 pt-4 animate-in fade-in-0 duration-200">
					<div class="space-y-2">
						<Label for="modal-login-id">{t('auth.account')}</Label>
						<Input
							id="modal-login-id"
							type="text"
							placeholder={t('auth.phoneOrEmail')}
							autocomplete="username"
							bind:value={loginId}
							disabled={isLoading}
							onblur={() => markTouched('loginId')}
							class="h-11 rounded-2xl border-border/60 bg-background/70 {loginIdError ? 'border-destructive focus-visible:ring-destructive' : ''}"
						/>
						{#if loginIdError}
							<p class="text-xs text-destructive">{loginIdError}</p>
						{/if}
					</div>
					<div class="space-y-2">
						<Label for="modal-login-password">{t('auth.password')}</Label>
						<div class="relative">
							<Input
								id="modal-login-password"
								type={showPassword ? 'text' : 'password'}
								placeholder={t('auth.enterPassword')}
								autocomplete="current-password"
								bind:value={password}
								disabled={isLoading}
								onblur={() => markTouched('password')}
								class="h-11 rounded-2xl border-border/60 bg-background/70 {passwordError ? 'border-destructive focus-visible:ring-destructive' : ''}"
							/>
							<Button
								type="button"
								variant="ghost"
								size="icon"
								class="absolute right-1 top-1 h-9 w-9 rounded-xl px-0"
								onclick={() => (showPassword = !showPassword)}
								aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
							>
								{#if showPassword}
									<EyeOff class="h-4 w-4" />
								{:else}
									<Eye class="h-4 w-4" />
								{/if}
							</Button>
						</div>
						{#if passwordError}
							<p class="text-xs text-destructive">{passwordError}</p>
						{/if}
					</div>
					<Button type="submit" class="h-11 w-full rounded-2xl" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						{t('auth.login')}
					</Button>
				</form>
			</TabsContent>

			<!-- 注册 -->
			<TabsContent value="register">
				<form onsubmit={(e) => { e.preventDefault(); handleRegister(); }} class="space-y-4 pt-4 animate-in fade-in-0 duration-200">
					<div class="space-y-2">
						<Label for="modal-register-nickname">{t('auth.nickname')}</Label>
						<Input
							id="modal-register-nickname"
							type="text"
							placeholder={t('auth.giveYourselfAName')}
							autocomplete="nickname"
							bind:value={nickname}
							disabled={isLoading}
							onblur={() => markTouched('nickname')}
							class="h-11 rounded-2xl border-border/60 bg-background/70 {nicknameError ? 'border-destructive focus-visible:ring-destructive' : ''}"
						/>
						{#if nicknameError}
							<p class="text-xs text-destructive">{nicknameError}</p>
						{/if}
					</div>
					<div class="space-y-2">
						<Label for="modal-register-id">{t('auth.account')}</Label>
						<Input
							id="modal-register-id"
							type="text"
							placeholder={t('auth.phoneOrEmail')}
							autocomplete="username"
							bind:value={loginId}
							disabled={isLoading}
							onblur={() => markTouched('loginId')}
							class="h-11 rounded-2xl border-border/60 bg-background/70 {loginIdError ? 'border-destructive focus-visible:ring-destructive' : ''}"
						/>
						{#if loginIdError}
							<p class="text-xs text-destructive">{loginIdError}</p>
						{/if}
					</div>
					<div class="space-y-2">
						<Label for="modal-register-password">{t('auth.password')}</Label>
						<Input
							id="modal-register-password"
							type="password"
							placeholder={t('auth.atLeast8Chars')}
							autocomplete="new-password"
							bind:value={password}
							disabled={isLoading}
							onblur={() => markTouched('password')}
							class="h-11 rounded-2xl border-border/60 bg-background/70 {passwordError ? 'border-destructive focus-visible:ring-destructive' : ''}"
						/>
						{#if passwordError}
							<p class="text-xs text-destructive">{passwordError}</p>
						{/if}
					</div>
					<div class="space-y-2">
						<Label for="modal-register-confirm">{t('auth.confirmPassword')}</Label>
						<Input
							id="modal-register-confirm"
							type="password"
							placeholder={t('auth.enterPasswordAgain')}
							autocomplete="new-password"
							bind:value={confirmPassword}
							disabled={isLoading}
							onblur={() => markTouched('confirmPassword')}
							class="h-11 rounded-2xl border-border/60 bg-background/70 {confirmPasswordError ? 'border-destructive focus-visible:ring-destructive' : ''}"
						/>
						{#if confirmPasswordError}
							<p class="text-xs text-destructive">{confirmPasswordError}</p>
						{/if}
					</div>
					<Button type="submit" class="h-11 w-full rounded-2xl" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						{t('auth.register')}
					</Button>
				</form>
			</TabsContent>

			<!-- 访客 -->
			<TabsContent value="guest">
				<form onsubmit={(e) => { e.preventDefault(); handleGuestLogin(); }} class="space-y-4 pt-4 animate-in fade-in-0 duration-200">
					<div class="rounded-2xl border border-border/50 bg-background/65 p-3 text-sm text-muted-foreground">
						{t('auth.guestModeHint')}
					</div>
					<div class="space-y-2">
						<Label for="modal-guest-nickname">{t('auth.nickname')}</Label>
						<Input
							id="modal-guest-nickname"
							type="text"
							placeholder={t('auth.giveYourselfAName')}
							autocomplete="nickname"
							bind:value={guestNickname}
							disabled={isLoading}
							onblur={() => markTouched('guestNickname')}
							class="h-11 rounded-2xl border-border/60 bg-background/70 {guestNicknameError ? 'border-destructive focus-visible:ring-destructive' : ''}"
						/>
						{#if guestNicknameError}
							<p class="text-xs text-destructive">{guestNicknameError}</p>
						{/if}
					</div>
					<Button type="submit" class="h-11 w-full rounded-2xl" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						{t('auth.guestLogin')}
					</Button>
				</form>
			</TabsContent>
			</div>
		</Tabs>
	</Dialog.Content>
</Dialog.Root>
