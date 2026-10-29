<script lang="ts">
	import { ArrowRight, Eye, EyeOff, LockKeyhole, Mail } from '@lucide/svelte';
	import { login, register, guest } from '@/stores/auth.svelte';
	import { requestPasswordReset, resetPassword, sendRegisterCode, getClientConfig } from '@/api';
	import Button from '@/components/ui/button.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
	import Input from '@/components/forms/Input.svelte';
	import {
		validateAuthFields,
		validateLoginId,
		type AuthFieldErrors,
		type AuthFieldName,
		type AuthMode
	} from '@/lib/auth-validation';

	// 全页 Auth 路由与 LoginPromptSheet 共用的认证表单。
	// 五模式：login / register / guest + 忘记密码子流程
	// （request_password_reset → reset_password）。校验规则对齐 harmonyapp：
	//   login_id 3–64，password 8–128 含字母+数字，nickname 1–30（访客可空，
	//   留空由服务端自动生成默认昵称）。
	//
	// 注册验证码（spec: auth/send_register_code）：login_id 为手机号（纯数字）或
	// 邮箱（含 '@'）时需先发送验证码验证所有权，提交时 verification_code 必填；
	// 用户名型不显示验证码输入、提交 null。
	//
	// 无障碍契约：每个字段独立校验，错误就近展示（role="alert" + aria-describedby），
	// 失败后自动聚焦首个错误字段；密码框提供显隐切换。

	type FieldName = AuthFieldName;
	type FieldErrors = AuthFieldErrors;

	let {
		compact = false,
		onSuccess,
		onModeChange,
		initialMode = 'login'
	}: {
		/** When true the form is rendered inside a compact sheet (no hero/header). */
		compact?: boolean;
		/** Called after a successful auth; the parent decides where to go. */
		onSuccess?: () => void;
		/** Initial mode. */
		initialMode?: AuthMode;
		/** 模式切换回调（登录/注册/访客…）：宿主据此切换品牌区文案语境。 */
		onModeChange?: (m: AuthMode) => void;
	} = $props();

	// initialMode 仅在挂载时生效（此后模式切换走 switchMode）：闭包读取一次初始值
	const getInitialMode = () => initialMode;
	let mode = $state<AuthMode>(getInitialMode());
	// 登录方式（对齐 Figma Make LoginPage 双 tab：验证码登录/密码登录）
	let loginId = $state('');
	let password = $state('');
	let nickname = $state('');
	let code = $state('');
	let newPassword = $state('');
	let resetLoginId = $state('');
	let showPassword = $state(false);
	// 条款链接：消费后台可配置的 terms/privacy url（第二轮审查 r85；未配置时降级纯文本）
	let termsUrl = $state('');
	let privacyUrl = $state('');
	$effect(() => {
		void getClientConfig()
			.then((res) => {
				if (res.code === 0 && res.data) {
					termsUrl = res.data.contact.terms_of_service_url || '';
					privacyUrl = res.data.contact.privacy_policy_url || '';
				}
			})
			.catch(() => {});
	});
	let submitting = $state(false);
	let error = $state<string | null>(null);
	let fieldErrors = $state<FieldErrors>({});

	// 注册验证码：手机号（纯数字）或邮箱（含 '@'）型 login_id 需先验证所有权
	const needsRegisterCode = $derived(
		mode === 'register' && (/^\d+$/.test(loginId.trim()) || loginId.includes('@'))
	);
	let codeCountdown = $state(0);
	let sendingCode = $state(false);
	let countdownTimer: ReturnType<typeof setInterval> | undefined;

	$effect(() => {
		// 组件卸载时清理倒计时定时器
		return () => {
			if (countdownTimer) clearInterval(countdownTimer);
		};
	});

	const sendRegisterCodeAndCountdown = async () => {
		if (codeCountdown > 0 || sendingCode) return;
		const idErr = validateLoginId(loginId);
		if (idErr) {
			fieldErrors = { ...fieldErrors, loginId: idErr };
			error = idErr;
			document.getElementById('auth-field-loginId')?.focus();
			return;
		}
		sendingCode = true;
		try {
			const res = await sendRegisterCode({ login_id: loginId, client_type: 'web' });
			if (res.code === 0) {
				// spec: 防枚举——无论 login_id 是否已注册均成功；60s 内禁止重发
				codeCountdown = 60;
				countdownTimer = setInterval(() => {
					codeCountdown -= 1;
					if (codeCountdown <= 0) {
						codeCountdown = 0;
						clearInterval(countdownTimer);
						countdownTimer = undefined;
					}
				}, 1000);
			} else {
				error = res.message ?? '验证码发送失败，请重试';
			}
		} catch {
			error = '网络错误，请稍后重试';
		} finally {
			sendingCode = false;
		}
	};

	const reset = () => {
		error = null;
		fieldErrors = {};
		submitting = false;
	};

	const switchMode = (m: AuthMode) => {
		mode = m;
		reset();
		onModeChange?.(m);
	};

	function validate(): FieldErrors {
		return validateAuthFields({
			mode,
			method: 'password' as const,
			loginId,
			password,
			nickname,
			code,
			newPassword,
			resetLoginId,
			needsRegisterCode
		});
	}

	function focusFirstInvalid(errs: FieldErrors): void {
		const first = (Object.keys(errs) as FieldName[])[0];
		if (!first) return;
		document.getElementById(`auth-field-${first}`)?.focus();
	}

	const clearFieldError = (name: FieldName) => {
		if (fieldErrors[name]) {
			fieldErrors = { ...fieldErrors, [name]: undefined };
		}
	};

	const handleSubmit = async () => {
		if (submitting) return;
		error = null;
		fieldErrors = {};

		// Client-side validation aligned with harmonyapp LoginPage.
		const errs = validate();
		if (Object.keys(errs).length > 0) {
			fieldErrors = errs;
			error = Object.values(errs)[0] ?? null;
			focusFirstInvalid(errs);
			return;
		}

		submitting = true;
		try {
			let res;
			if (mode === 'login') res = await login({ login_id: loginId, password });
			else if (mode === 'register') res = await register({ login_id: loginId, nickname, password, code: needsRegisterCode ? code.trim() : null });
			else if (mode === 'guest') res = await guest({ nickname: nickname.trim() || null });
			else if (mode === 'forgot') {
				res = await requestPasswordReset({ login_id: resetLoginId, client_type: 'web' });
				if (res.code === 0) {
					// Spec: always returns success (anti-enumeration) → move to reset step.
					// 内部步骤迁移必须同步 notify 宿主：/login 的页头文案取自 onModeChange
					// 回传的 mode，此前直接改 mode 会漏通知，导致表单已进入「账号+验证码+
					// 新密码」而卡片仍写「找回你的账号。／我们会发送验证码。」（走查 F1）。
					// 不走 switchMode 是因其会 reset() 清空字段，而 reset 步骤需要保留
					// resetLoginId 才能提交。
					mode = 'reset';
					onModeChange?.('reset');
					submitting = false;
					return;
				}
			} else if (mode === 'reset') {
				res = await resetPassword({
					login_id: resetLoginId,
					verification_code: code,
					new_password: newPassword,
					client_type: 'web',
					// spec ResetPasswordRequest：Web 端 device_id 为 null（服务端从 Cookie 读取）
					device_id: null
				});
			}
			if (res && res.code === 0) {
				onSuccess?.();
				return;
			}
			// Business failure: surface the server message (spec: message is for display).
			error = res?.message ?? '操作失败，请重试';
		} catch {
			error = '网络错误，请稍后重试';
		} finally {
			submitting = false;
		}
	};

	function submitLabel(m: AuthMode): string {
		switch (m) {
			case 'login': return '登录';
			case 'register': return '注册并登录';
			case 'guest': return '以访客身份继续';
			case 'forgot': return '发送验证码';
			case 'reset': return '重置密码并登录';
		}
	}
</script>

<form
	class="flex flex-col {compact ? 'gap-3' : 'gap-4'}"
	novalidate
	onsubmit={(e) => {
		e.preventDefault();
		void handleSubmit();
	}}
>
	{#if mode === 'register' || mode === 'guest'}
		<Input
			id="auth-field-nickname"
			label="昵称"
			value={nickname}
			onChange={(v) => {
				nickname = v;
				clearFieldError('nickname');
			}}
			placeholder={mode === 'guest' ? '昵称（可选，留空自动生成）' : '1–30 个字符'}
			disabled={submitting}
			autocomplete="nickname"
			hasError={!!fieldErrors.nickname}
			errorText={fieldErrors.nickname}
			reserveErrorSpace
		/>
	{/if}

	{#if mode === 'login' || mode === 'register'}
		<!-- 验证码登录接口 spec 暂无：不展示不可用选项（业务态不可伪装成可用功能）；
		     仅注册手机号/邮箱型需验证码（所有权验证），与密码共存 -->
		<Input
			id="auth-field-loginId"
			label="手机号、邮箱或账号"
			value={loginId}
			onChange={(v) => {
				loginId = v;
				clearFieldError('loginId');
			}}
			placeholder="请输入账号"
			disabled={submitting}
			autocomplete="username"
			hasError={!!fieldErrors.loginId}
			errorText={fieldErrors.loginId}
			reserveErrorSpace
		>
			{#snippet prefix()}
				<Mail class="size-4 shrink-0 text-accent-default" strokeWidth={1.75} />
			{/snippet}
		</Input>
		{#if needsRegisterCode}
			<!-- 验证码：注册手机号/邮箱型验证所有权（登录验证码接口 spec 暂无），与密码共存 -->
			<Input
				id="auth-field-code"
				label="验证码"
				value={code}
				onChange={(v) => {
					code = v;
					clearFieldError('code');
				}}
				placeholder="输入收到的验证码"
				disabled={submitting}
				autocomplete="one-time-code"
				hasError={!!fieldErrors.code}
				errorText={fieldErrors.code}
				reserveErrorSpace
			/>
			<Button
				type="button"
				variant="surface-accent"
				size="lg"
				class="w-full"
				disabled={codeCountdown > 0 || sendingCode}
				onclick={() => void sendRegisterCodeAndCountdown()}
			>
				{#if needsRegisterCode}
					{sendingCode ? '发送中…' : codeCountdown > 0 ? `${codeCountdown}s 后重发` : '发送验证码'}
				{:else}
					获取验证码
				{/if}
			</Button>
		{/if}
		{#if mode === 'login' || mode === 'register'}
			<Input
				id="auth-field-password"
				type={showPassword ? 'text' : 'password'}
				label="密码"
				value={password}
				onChange={(v) => {
					password = v;
					clearFieldError('password');
				}}
				placeholder="请输入密码"
				disabled={submitting}
				autocomplete="current-password"
				hasError={!!fieldErrors.password}
				errorText={fieldErrors.password}
				reserveErrorSpace
			>
				{#snippet prefix()}
					<LockKeyhole class="size-4 shrink-0 text-accent-default" strokeWidth={1.75} />
				{/snippet}
				{#snippet suffix()}
					<button
						type="button"
						class="hit-slop -m-2 shrink-0 p-2 focus-ring"
						aria-label={showPassword ? '隐藏密码' : '显示密码'}
						onclick={() => (showPassword = !showPassword)}
					>
						{#if showPassword}
							<EyeOff class="size-4 text-text-secondary" strokeWidth={1.75} />
						{:else}
							<Eye class="size-4 text-text-secondary" strokeWidth={1.75} />
						{/if}
					</button>
				{/snippet}
			</Input>
		{/if}
	{/if}

	{#if mode === 'forgot' || mode === 'reset'}
		<Input
			id="auth-field-resetLoginId"
			label="账号"
			value={resetLoginId}
			onChange={(v) => {
				resetLoginId = v;
				clearFieldError('resetLoginId');
			}}
			placeholder="注册时的手机号 / 邮箱"
			disabled={submitting}
			autocomplete="username"
			hasError={!!fieldErrors.resetLoginId}
			errorText={fieldErrors.resetLoginId}
			reserveErrorSpace
		/>
		{#if mode === 'reset'}
			<Input
				id="auth-field-code"
				label="验证码"
				value={code}
				onChange={(v) => {
					code = v;
					clearFieldError('code');
				}}
				placeholder="输入收到的验证码"
				disabled={submitting}
				autocomplete="one-time-code"
				hasError={!!fieldErrors.code}
				errorText={fieldErrors.code}
				reserveErrorSpace
			/>
			<Input
				id="auth-field-newPassword"
				type={showPassword ? 'text' : 'password'}
				label="新密码"
				value={newPassword}
				onChange={(v) => {
					newPassword = v;
					clearFieldError('newPassword');
				}}
				placeholder="8–128 位，含字母和数字"
				disabled={submitting}
				autocomplete="new-password"
				hasError={!!fieldErrors.newPassword}
				errorText={fieldErrors.newPassword}
				reserveErrorSpace
			>
				{#snippet suffix()}
					<button
						type="button"
						class="hit-slop -m-2 shrink-0 p-2 focus-ring"
						aria-label={showPassword ? '隐藏密码' : '显示密码'}
						onclick={() => (showPassword = !showPassword)}
					>
						{#if showPassword}
							<EyeOff class="size-4 text-text-secondary" strokeWidth={1.75} />
						{:else}
							<Eye class="size-4 text-text-secondary" strokeWidth={1.75} />
						{/if}
					</button>
				{/snippet}
			</Input>
		{/if}
	{/if}

	<!-- 表单级错误同样常驻占位：此前条件渲染，提示一出现就把下方「登录」按钮顶下去，
	     与字段错误叠加后表现为按钮上下跳（2026-10-21 录屏分析）。 -->
	<p class="text-xs text-status-danger min-h-4" role="alert">{error ?? ''}</p>

	<Button
		type="submit"
		variant="primary"
		size="lg"
		class="w-full"
		loading={submitting}
		disabled={submitting}
	>
		{submitting ? '处理中…' : submitLabel(mode)}
		{#if !submitting && (mode === 'login' || mode === 'register')}
			<ArrowRight class="size-4"  strokeWidth={1.75} />
		{/if}
	</Button>

	{#if mode === 'login' && !compact}
		<!-- 微信登录暂未开放：按本组件「不展示不可用选项」原则不下发点了必报错的死按钮，
		     原「其他方式」分隔线随微信按钮一并移除，仅保留条款行 -->
		<p class="mt-5 text-center text-xs leading-5 text-text-secondary">
			登录即表示你同意
			{#if termsUrl}<a href={termsUrl} target="_blank" rel="noopener" class="link-btn text-accent-default">服务条款</a>{:else}服务条款{/if}
			与
			{#if privacyUrl}<a href={privacyUrl} target="_blank" rel="noopener" class="link-btn text-accent-default">隐私政策</a>{:else}隐私政策{/if}。
		</p>
	{/if}

	<!-- Mode switcher -->
	<div class="flex flex-wrap items-center justify-center gap-3.5 mt-1" >
		{#if mode === 'register'}
			<button type="button" onclick={() => switchMode('login')} class="link-btn hit-slop-44 -my-2 py-2 text-sm focus-ring text-accent-default">
				已有账号？登录
			</button>
		{/if}
		{#if mode === 'login'}
			<button type="button" onclick={() => switchMode('register')} class="link-btn hit-slop-44 -my-2 py-2 text-sm focus-ring text-accent-default">
				注册新账号
			</button>
			<button type="button" onclick={() => switchMode('forgot')} class="link-btn hit-slop-44 -my-2 py-2 text-sm focus-ring text-accent-default">
				忘记密码
			</button>
		{/if}
		{#if mode === 'forgot' || mode === 'reset' || mode === 'guest'}
			<button type="button" onclick={() => switchMode('login')} class="link-btn hit-slop-44 -my-2 py-2 text-sm focus-ring text-accent-default">
				返回登录
			</button>
		{/if}
		{#if mode !== 'guest' && mode !== 'forgot' && mode !== 'reset'}
			<button type="button" onclick={() => switchMode('guest')} class="link-btn hit-slop-44 -my-2 py-2 text-sm focus-ring text-accent-default">
				随便逛逛（访客）
			</button>
		{/if}
	</div>
</form>
