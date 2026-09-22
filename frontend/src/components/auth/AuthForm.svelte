<script lang="ts">
	import { ArrowRight, Eye, EyeOff, LockKeyhole, Mail } from '@lucide/svelte';
	import { login, register, guest } from '@/stores/auth.svelte';
	import { requestPasswordReset, resetPassword, sendRegisterCode } from '@/api';
	import Button from '@/components/ui/button.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';
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
	let method = $state<'code' | 'password'>('password');
	let loginId = $state('');
	let password = $state('');
	let nickname = $state('');
	let code = $state('');
	let newPassword = $state('');
	let resetLoginId = $state('');
	let showPassword = $state(false);
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
			method,
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
			if (mode === 'login') {
				if (method === 'code') {
					// 验证码登录接口待后端提供（spec 暂无），UI 对齐 Figma Make 先行
					error = '验证码登录暂未开放，请使用密码登录';
					return;
				}
				res = await login({ login_id: loginId, password });
			}
			else if (mode === 'register') res = await register({ login_id: loginId, nickname, password, code: needsRegisterCode ? code.trim() : null });
			else if (mode === 'guest') res = await guest({ nickname: nickname.trim() || null });
			else if (mode === 'forgot') {
				res = await requestPasswordReset({ login_id: resetLoginId, client_type: 'web' });
				if (res.code === 0) {
					// Spec: always returns success (anti-enumeration) → move to reset step.
					mode = 'reset';
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
		<label class="flex flex-col gap-1.5" for="auth-field-nickname">
			<span class="text-xs tracking-field text-text-secondary" >昵称</span>
			<span class="flex items-center gap-2 input-surface rounded-md px-3 py-3" >
				<input
					id="auth-field-nickname"
					type="text"
					bind:value={nickname}
					placeholder={mode === 'guest' ? '昵称（可选，留空自动生成）' : '1–30 个字符'}
					disabled={submitting}
					autocomplete="nickname"
					aria-invalid={fieldErrors.nickname ? 'true' : undefined}
					aria-describedby={fieldErrors.nickname ? 'auth-field-nickname-error' : undefined}
					oninput={() => clearFieldError('nickname')}
					class="min-w-0 flex-1 bg-transparent text-sm focus-ring text-text-primary"
					
				/>
			</span>
			{#if fieldErrors.nickname}
				<p id="auth-field-nickname-error" class="text-xs text-status-danger"  role="alert">{fieldErrors.nickname}</p>
			{/if}
		</label>
	{/if}

	{#if mode === 'login' || mode === 'register'}
		{#if mode === 'login'}
			<!-- 登录方式选择：验证码登录接口 spec 暂无、后端未提供——不展示不可用选项，
			     避免用户填完表单才被告知「暂未开放」（业务态不可伪装成可用功能） -->
		{/if}
		<label class="flex flex-col gap-1.5" for="auth-field-loginId">
			<span class="text-xs font-medium tracking-field text-text-secondary">手机号、邮箱或账号</span>
			<span class="flex items-center gap-2 input-surface rounded-md px-3 py-3" >
				<Mail class="size-4 shrink-0 text-accent-default"   strokeWidth={1.75} />
				<input
					id="auth-field-loginId"
					type="text"
					bind:value={loginId}
					placeholder="请输入账号"
					disabled={submitting}
					autocomplete="username"
					aria-invalid={fieldErrors.loginId ? 'true' : undefined}
					aria-describedby={fieldErrors.loginId ? 'auth-field-loginId-error' : undefined}
					oninput={() => clearFieldError('loginId')}
					class="min-w-0 flex-1 bg-transparent text-sm focus-ring placeholder:text-text-secondary text-text-primary"
					
				/>
			</span>
			{#if fieldErrors.loginId}
				<p id="auth-field-loginId-error" class="text-xs text-status-danger"  role="alert">{fieldErrors.loginId}</p>
			{/if}
		</label>
			{#if (mode === 'login' && method === 'code') || needsRegisterCode}
				<!-- 验证码：登录验证码模式（发送接口暂未开放）或注册手机号/邮箱型（所有权验证）；
				     注册型与密码共存（注册始终需要密码），登录验证码模式替代密码 -->
				<label class="flex flex-col gap-1.5" for="auth-field-code">
					<span class="text-xs tracking-field text-text-secondary" >验证码</span>
					<span class="flex items-center gap-2 input-surface rounded-md px-3 py-3" >
						<input
							id="auth-field-code"
							type="text"
							bind:value={code}
							placeholder="输入收到的验证码"
							disabled={submitting}
							autocomplete="one-time-code"
							aria-invalid={fieldErrors.code ? 'true' : undefined}
							aria-describedby={fieldErrors.code ? 'auth-field-code-error' : undefined}
							oninput={() => clearFieldError('code')}
							class="min-w-0 flex-1 bg-transparent text-sm focus-ring placeholder:text-text-secondary text-text-primary"

						/>
					</span>
					{#if fieldErrors.code}
						<p id="auth-field-code-error" class="text-xs text-status-danger" role="alert">{fieldErrors.code}</p>
					{/if}
				</label>
				<Button
					type="button"
					variant="secondary"
					size="lg"
					class="w-full bg-accent-bg"
					disabled={needsRegisterCode && (codeCountdown > 0 || sendingCode)}
					onclick={() => {
						if (needsRegisterCode) void sendRegisterCodeAndCountdown();
						else error = '验证码发送功能即将开放，请使用密码登录';
					}}
				>
					{#if needsRegisterCode}
						{sendingCode ? '发送中…' : codeCountdown > 0 ? `${codeCountdown}s 后重发` : '发送验证码'}
					{:else}
						获取验证码
					{/if}
				</Button>
			{/if}
			{#if (mode === 'login' && method === 'password') || mode === 'register'}
			<label class="flex flex-col gap-1.5" for="auth-field-password">
				<span class="text-xs font-medium tracking-field text-text-secondary">密码</span>
				<span class="flex items-center gap-2 input-surface rounded-md px-3 py-3" >
					<LockKeyhole class="size-4 shrink-0 text-accent-default"   strokeWidth={1.75} />
					<input
						id="auth-field-password"
						type={showPassword ? 'text' : 'password'}
						bind:value={password}
						placeholder="请输入密码"
						disabled={submitting}
						autocomplete="current-password"
						aria-invalid={fieldErrors.password ? 'true' : undefined}
						aria-describedby={fieldErrors.password ? 'auth-field-password-error' : undefined}
						oninput={() => clearFieldError('password')}
						class="min-w-0 flex-1 bg-transparent text-sm focus-ring placeholder:text-text-secondary text-text-primary"
						
					/>
					<button
						type="button"
						class="hit-slop shrink-0 focus-ring"
						aria-label={showPassword ? '隐藏密码' : '显示密码'}
						onclick={() => (showPassword = !showPassword)}
					>
						{#if showPassword}
							<EyeOff class="size-4 text-text-secondary"  strokeWidth={1.75} />
						{:else}
							<Eye class="size-4 text-text-secondary"  strokeWidth={1.75} />
						{/if}
					</button>
				</span>
				{#if fieldErrors.password}
					<p id="auth-field-password-error" class="text-xs text-status-danger"  role="alert">{fieldErrors.password}</p>
				{/if}
			</label>
		{/if}
	{/if}

	{#if mode === 'forgot' || mode === 'reset'}
		<label class="flex flex-col gap-1.5" for="auth-field-resetLoginId">
			<span class="text-xs tracking-field text-text-secondary" >账号</span>
			<span class="flex items-center gap-2 input-surface rounded-md px-3 py-3" >
				<input
					id="auth-field-resetLoginId"
					type="text"
					bind:value={resetLoginId}
					placeholder="注册时的手机号 / 邮箱"
					disabled={submitting}
					autocomplete="username"
					aria-invalid={fieldErrors.resetLoginId ? 'true' : undefined}
					aria-describedby={fieldErrors.resetLoginId ? 'auth-field-resetLoginId-error' : undefined}
					oninput={() => clearFieldError('resetLoginId')}
					class="min-w-0 flex-1 bg-transparent text-sm focus-ring placeholder:text-text-secondary text-text-primary"
					
				/>
			</span>
			{#if fieldErrors.resetLoginId}
				<p id="auth-field-resetLoginId-error" class="text-xs text-status-danger"  role="alert">{fieldErrors.resetLoginId}</p>
			{/if}
		</label>
		{#if mode === 'reset'}
			<label class="flex flex-col gap-1.5" for="auth-field-code">
				<span class="text-xs tracking-field text-text-secondary" >验证码</span>
				<span class="flex items-center gap-2 input-surface rounded-md px-3 py-3" >
					<input
						id="auth-field-code"
						type="text"
						bind:value={code}
						placeholder="输入收到的验证码"
						disabled={submitting}
						autocomplete="one-time-code"
						aria-invalid={fieldErrors.code ? 'true' : undefined}
						aria-describedby={fieldErrors.code ? 'auth-field-code-error' : undefined}
						oninput={() => clearFieldError('code')}
						class="min-w-0 flex-1 bg-transparent text-sm focus-ring placeholder:text-text-secondary text-text-primary"
						
					/>
				</span>
				{#if fieldErrors.code}
					<p id="auth-field-code-error" class="text-xs text-status-danger"  role="alert">{fieldErrors.code}</p>
				{/if}
			</label>
			<label class="flex flex-col gap-1.5" for="auth-field-newPassword">
				<span class="text-xs tracking-field text-text-secondary" >新密码</span>
				<span class="flex items-center gap-2 input-surface rounded-md px-3 py-3" >
					<input
						id="auth-field-newPassword"
						type={showPassword ? 'text' : 'password'}
						bind:value={newPassword}
						placeholder="8–128 位，含字母和数字"
						disabled={submitting}
						autocomplete="new-password"
						aria-invalid={fieldErrors.newPassword ? 'true' : undefined}
						aria-describedby={fieldErrors.newPassword ? 'auth-field-newPassword-error' : undefined}
						oninput={() => clearFieldError('newPassword')}
						class="min-w-0 flex-1 bg-transparent text-sm focus-ring placeholder:text-text-secondary text-text-primary"
						
					/>
					<button
						type="button"
						class="hit-slop shrink-0 focus-ring"
						aria-label={showPassword ? '隐藏密码' : '显示密码'}
						onclick={() => (showPassword = !showPassword)}
					>
						{#if showPassword}
							<EyeOff class="size-4 text-text-secondary"  strokeWidth={1.75} />
						{:else}
							<Eye class="size-4 text-text-secondary"  strokeWidth={1.75} />
						{/if}
					</button>
				</span>
				{#if fieldErrors.newPassword}
					<p id="auth-field-newPassword-error" class="text-xs text-status-danger"  role="alert">{fieldErrors.newPassword}</p>
				{/if}
			</label>
		{/if}
	{/if}

	{#if error}
		<p class="text-xs text-status-danger"   role="alert">{error}</p>
	{/if}

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
		<p class="mt-5 text-center text-xs leading-5 text-text-secondary">登录即表示你同意服务条款与隐私政策。</p>
	{/if}

	<!-- Mode switcher -->
	<div class="flex flex-wrap items-center justify-center gap-3.5 mt-1" >
		{#if mode === 'register'}
			<button type="button" onclick={() => switchMode('login')} class="link-btn -my-1 py-1 text-sm focus-ring text-accent-default">
				已有账号？登录
			</button>
		{/if}
		{#if mode === 'login'}
			<button type="button" onclick={() => switchMode('register')} class="link-btn -my-1 py-1 text-sm focus-ring text-accent-default">
				注册新账号
			</button>
			<button type="button" onclick={() => switchMode('forgot')} class="link-btn -my-1 py-1 text-sm focus-ring text-accent-default">
				忘记密码
			</button>
		{/if}
		{#if mode === 'forgot' || mode === 'reset'}
			<button type="button" onclick={() => switchMode('login')} class="link-btn -my-1 py-1 text-sm focus-ring text-accent-default">
				返回登录
			</button>
		{/if}
		{#if mode !== 'guest' && mode !== 'forgot' && mode !== 'reset'}
			<button type="button" onclick={() => switchMode('guest')} class="link-btn -my-1 py-1 text-sm focus-ring text-accent-default">
				随便逛逛（访客）
			</button>
		{/if}
	</div>
</form>
