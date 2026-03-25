<script lang="ts">
	import { withMinimumLoadingTime, MIN_LOADING_ACTION } from '$lib/utils/loading';
	import { authService, usersService } from '$lib/api/services';
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

	// 表单状态
	let loginId = $state('');
	let password = $state('');
	let nickname = $state('');
	let confirmPassword = $state('');
	let guestNickname = $state('');

	let isLoading = $state(false);
	let showPassword = $state(false);
	let currentTab = $state('login');
	const PASSWORD_POLICY_TIP = '密码需为 8-128 位且同时包含字母和数字';
	const hasLetterAndDigit = (value: string) => /[A-Za-z]/.test(value) && /\d/.test(value);

	function handleClose() {
		uiStore.closeLoginModal();
		// 清空表单
		loginId = '';
		password = '';
		nickname = '';
		confirmPassword = '';
		guestNickname = '';
		currentTab = 'login';
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

		if (!isSuccess(statusRes) || !statusRes.data.is_valid) {
			return false;
		}
		if (!isSuccess(profileRes)) {
			return false;
		}

		const expiresAt =
			statusRes.data.expires_at ?? new Date(Date.now() + 15 * 60 * 1000).toISOString();

		authStore.login({
			user_id: profileRes.data.user_id,
			user_type: profileRes.data.user_type,
			access_token_expires_at: expiresAt
		});
		authStore.setProfile(profileRes.data);
		return true;
	}

	async function handleLogin() {
		const trimmedLoginId = loginId.trim();
		if (!trimmedLoginId || !password) {
			toast.error('请填写登录信息');
			return;
		}
		if (trimmedLoginId.length < 3 || trimmedLoginId.length > 64) {
			toast.error('登录账号长度需在 3-64 字符之间');
			return;
		}
		if (password.length < 8 || password.length > 128 || !hasLetterAndDigit(password)) {
			toast.error(PASSWORD_POLICY_TIP);
			return;
		}

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
					toast.success('登录成功');
					uiStore.onLoginSuccess();
					loginId = '';
					password = '';
				} else {
					toast.error('获取用户信息失败');
				}
			} else {
				toast.error(response.message || '登录失败');
			}
		} catch (error) {
			toast.error('登录失败，请稍后重试');
		} finally {
			isLoading = false;
		}
	}

	async function handleRegister() {
		const trimmedLoginId = loginId.trim();
		const trimmedNickname = nickname.trim();
		if (!trimmedLoginId || !password || !trimmedNickname) {
			toast.error('请填写完整信息');
			return;
		}
		if (trimmedLoginId.length < 3 || trimmedLoginId.length > 64) {
			toast.error('登录账号长度需在 3-64 字符之间');
			return;
		}
		if (trimmedNickname.length < 1 || trimmedNickname.length > 30) {
			toast.error('昵称长度需在 1-30 字符之间');
			return;
		}

		if (password !== confirmPassword) {
			toast.error('两次密码不一致');
			return;
		}

		if (password.length < 8 || password.length > 128 || !hasLetterAndDigit(password)) {
			toast.error(PASSWORD_POLICY_TIP);
			return;
		}

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
					toast.success('注册成功');
					uiStore.onLoginSuccess();
				} else {
					toast.error('获取用户信息失败');
				}
			} else {
				toast.error(response.message || '注册失败');
			}
		} catch (error) {
			toast.error('注册失败，请稍后重试');
		} finally {
			isLoading = false;
		}
	}

	async function handleGuestLogin() {
		const trimmedNickname = guestNickname.trim();
		if (!trimmedNickname) {
			toast.error('请输入昵称');
			return;
		}
		if (trimmedNickname.length > 30) {
			toast.error('昵称长度需在 1-30 字符之间');
			return;
		}

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
					toast.success('访客登录成功');
					uiStore.onLoginSuccess();
				} else {
					toast.error('获取用户信息失败');
				}
			} else {
				toast.error(response.message || '访客登录失败');
			}
		} catch (error) {
			toast.error('访客登录失败，请稍后重试');
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
				<div>欢迎来到 HotSpot</div>
			</Dialog.Title>
			<Dialog.Description>
				登录后即可使用完整功能
			</Dialog.Description>
		</Dialog.Header>

		<Tabs bind:value={currentTab} class="mt-4">
			<TabsList class="grid w-full grid-cols-3 rounded-3xl bg-background/75 p-1.5 ring-1 ring-border/60">
				<TabsTrigger value="login" class="rounded-2xl">登录</TabsTrigger>
				<TabsTrigger value="register" class="rounded-2xl">注册</TabsTrigger>
				<TabsTrigger value="guest" class="rounded-2xl">访客</TabsTrigger>
			</TabsList>

			<!-- 固定最小高度以防止跳动（需容纳注册表单 4 个输入框 + 按钮） -->
			<div class="min-h-[400px]">
				<!-- 登录 -->
				<TabsContent value="login">
				<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4 pt-4 animate-in fade-in-0 duration-200">
					<div class="space-y-2">
						<Label for="modal-login-id">账号</Label>
						<Input
							id="modal-login-id"
							type="text"
							placeholder="手机号或邮箱"
							autocomplete="username"
							bind:value={loginId}
							disabled={isLoading}
							class="h-11 rounded-2xl border-border/60 bg-background/70"
						/>
					</div>
					<div class="space-y-2">
						<Label for="modal-login-password">密码</Label>
						<div class="relative">
							<Input
								id="modal-login-password"
								type={showPassword ? 'text' : 'password'}
								placeholder="请输入密码"
								autocomplete="current-password"
								bind:value={password}
								disabled={isLoading}
								class="h-11 rounded-2xl border-border/60 bg-background/70"
							/>
							<Button
								type="button"
								variant="ghost"
								size="icon"
								class="absolute right-1 top-1 h-9 w-9 rounded-xl px-0"
								onclick={() => (showPassword = !showPassword)}
								aria-label={showPassword ? '隐藏密码' : '显示密码'}
							>
								{#if showPassword}
									<EyeOff class="h-4 w-4" />
								{:else}
									<Eye class="h-4 w-4" />
								{/if}
							</Button>
						</div>
					</div>
					<Button type="submit" class="h-11 w-full rounded-2xl" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						登录
					</Button>
				</form>
			</TabsContent>

			<!-- 注册 -->
			<TabsContent value="register">
				<form onsubmit={(e) => { e.preventDefault(); handleRegister(); }} class="space-y-4 pt-4 animate-in fade-in-0 duration-200">
					<div class="space-y-2">
						<Label for="modal-register-nickname">昵称</Label>
						<Input
							id="modal-register-nickname"
							type="text"
							placeholder="给自己起个名字"
							autocomplete="nickname"
							bind:value={nickname}
							disabled={isLoading}
							class="h-11 rounded-2xl border-border/60 bg-background/70"
						/>
					</div>
					<div class="space-y-2">
						<Label for="modal-register-id">账号</Label>
						<Input
							id="modal-register-id"
							type="text"
							placeholder="手机号或邮箱"
							autocomplete="username"
							bind:value={loginId}
							disabled={isLoading}
							class="h-11 rounded-2xl border-border/60 bg-background/70"
						/>
					</div>
					<div class="space-y-2">
						<Label for="modal-register-password">密码</Label>
						<Input
							id="modal-register-password"
							type="password"
							placeholder="至少8位，包含字母和数字"
							autocomplete="new-password"
							bind:value={password}
							disabled={isLoading}
							class="h-11 rounded-2xl border-border/60 bg-background/70"
						/>
					</div>
					<div class="space-y-2">
						<Label for="modal-register-confirm">确认密码</Label>
						<Input
							id="modal-register-confirm"
							type="password"
							placeholder="再次输入密码"
							autocomplete="new-password"
							bind:value={confirmPassword}
							disabled={isLoading}
							class="h-11 rounded-2xl border-border/60 bg-background/70"
						/>
					</div>
					<Button type="submit" class="h-11 w-full rounded-2xl" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						注册
					</Button>
				</form>
			</TabsContent>

			<!-- 访客 -->
			<TabsContent value="guest">
				<form onsubmit={(e) => { e.preventDefault(); handleGuestLogin(); }} class="space-y-4 pt-4 animate-in fade-in-0 duration-200">
					<div class="rounded-2xl border border-border/50 bg-background/65 p-3 text-sm text-muted-foreground">
						访客模式可快速体验，部分功能受限。
					</div>
					<div class="space-y-2">
						<Label for="modal-guest-nickname">昵称</Label>
						<Input
							id="modal-guest-nickname"
							type="text"
							placeholder="给自己起个名字"
							autocomplete="nickname"
							bind:value={guestNickname}
							disabled={isLoading}
							class="h-11 rounded-2xl border-border/60 bg-background/70"
						/>
					</div>
					<Button type="submit" class="h-11 w-full rounded-2xl" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						访客登录
					</Button>
				</form>
			</TabsContent>
			</div>
		</Tabs>
	</Dialog.Content>
</Dialog.Root>
