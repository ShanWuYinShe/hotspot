<script lang="ts">
	import { authService, usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { locationStore } from '$lib/stores/location.svelte';
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

	async function handleLogin() {
		if (!loginId || !password) {
			toast.error('请填写登录信息');
			return;
		}

		isLoading = true;
		try {
			const response = await authService.login({
				login_id: loginId,
				password,
				client_type: 'web'
			});

			if (isSuccess(response)) {
				authStore.login(response.data);
				await syncLocationAfterAuth();
				toast.success('登录成功');
				uiStore.onLoginSuccess();
				// 清空表单
				loginId = '';
				password = '';
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
		if (!loginId || !password || !nickname) {
			toast.error('请填写完整信息');
			return;
		}

		if (password !== confirmPassword) {
			toast.error('两次密码不一致');
			return;
		}

		if (password.length < 6) {
			toast.error('密码长度不能少于6位');
			return;
		}

		isLoading = true;
		try {
			const response = await authService.register({
				login_id: loginId,
				password,
				nickname,
				client_type: 'web'
			});

			if (isSuccess(response)) {
				authStore.login(response.data);
				await syncLocationAfterAuth();
				toast.success('注册成功');
				uiStore.onLoginSuccess();
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

		isLoading = true;
		try {
			const response = await authService.guest({
				nickname: trimmedNickname,
				client_type: 'web'
			});

			if (isSuccess(response)) {
				authStore.login(response.data);
				await syncLocationAfterAuth();
				toast.success('访客登录成功');
				uiStore.onLoginSuccess();
			} else {
				toast.error(response.message || '访客登录失败');
			}
		} catch (error) {
			toast.error('访客登录失败，请稍后重试');
		} finally {
			isLoading = false;
		}
	}

	async function syncLocationAfterAuth() {
		const current = locationStore.current;
		if (!current?.location_name) return;
		try {
			const response = await usersService.updateLocation({ location: current });
			if (!isSuccess(response)) {
				// 静默失败，避免打扰登录流程
			}
		} catch {
			// ignore
		}
	}
</script>

<Dialog.Root open={uiStore.showLoginModal} onOpenChange={(open) => !open && handleClose()}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<MapPin class="h-5 w-5 text-primary" />
				欢迎来到 HotSpot
			</Dialog.Title>
			<Dialog.Description>
				登录后即可使用完整功能
			</Dialog.Description>
		</Dialog.Header>

		<Tabs bind:value={currentTab} class="mt-4">
			<TabsList class="grid w-full grid-cols-3">
				<TabsTrigger value="login">登录</TabsTrigger>
				<TabsTrigger value="register">注册</TabsTrigger>
				<TabsTrigger value="guest">访客</TabsTrigger>
			</TabsList>

			<!-- 固定最小高度以防止跳动 -->
			<div class="min-h-[320px]">
				<!-- 登录 -->
				<TabsContent value="login">
				<form onsubmit={(e) => { e.preventDefault(); handleLogin(); }} class="space-y-4 pt-4">
					<div class="space-y-2">
						<Label for="modal-login-id">账号</Label>
						<Input
							id="modal-login-id"
							type="text"
							placeholder="手机号或邮箱"
							autocomplete="username"
							bind:value={loginId}
							disabled={isLoading}
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
							/>
							<Button
								type="button"
								variant="ghost"
								size="icon"
								class="absolute right-0 top-0 h-full px-3"
								onclick={() => (showPassword = !showPassword)}
							>
								{#if showPassword}
									<EyeOff class="h-4 w-4" />
								{:else}
									<Eye class="h-4 w-4" />
								{/if}
							</Button>
						</div>
					</div>
					<Button type="submit" class="w-full" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						登录
					</Button>
				</form>
			</TabsContent>

			<!-- 注册 -->
			<TabsContent value="register">
				<form onsubmit={(e) => { e.preventDefault(); handleRegister(); }} class="space-y-4 pt-4">
					<div class="space-y-2">
						<Label for="modal-register-nickname">昵称</Label>
						<Input
							id="modal-register-nickname"
							type="text"
							placeholder="给自己起个名字"
							autocomplete="nickname"
							bind:value={nickname}
							disabled={isLoading}
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
						/>
					</div>
					<div class="space-y-2">
						<Label for="modal-register-password">密码</Label>
						<Input
							id="modal-register-password"
							type="password"
							placeholder="至少6位"
							autocomplete="new-password"
							bind:value={password}
							disabled={isLoading}
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
						/>
					</div>
					<Button type="submit" class="w-full" disabled={isLoading}>
						{#if isLoading}
							<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						注册
					</Button>
				</form>
			</TabsContent>

			<!-- 访客 -->
			<TabsContent value="guest">
				<form onsubmit={(e) => { e.preventDefault(); handleGuestLogin(); }} class="space-y-4 pt-4">
					<div class="rounded-lg bg-muted p-3 text-sm text-muted-foreground">
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
						/>
					</div>
					<Button type="submit" class="w-full" disabled={isLoading}>
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
