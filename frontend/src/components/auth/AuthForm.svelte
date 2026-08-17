<script lang="ts">
	import { ArrowRight, Eye, EyeOff, LockKeyhole, Mail, MessageCircle } from '@lucide/svelte';
	import { login, register, guest } from '@/stores/auth.svelte';
	import { requestPasswordReset, resetPassword } from '@/api';
	import Button from '@/components/ui/button.svelte';
	import TagPill from '@/components/forms/TagPill.svelte';

	// 全页 Auth 路由与 LoginPromptSheet 共用的认证表单。
	// 五模式：login / register / guest + 忘记密码子流程
	// （request_password_reset → reset_password）。校验规则对齐 harmonyapp：
	//   login_id 3–64，password 8–128 含字母+数字，nickname 1–30。
	//
	// 无障碍契约：每个字段独立校验，错误就近展示（role="alert" + aria-describedby），
	// 失败后自动聚焦首个错误字段；密码框提供显隐切换。

	export type AuthMode = 'login' | 'register' | 'guest' | 'forgot' | 'reset';

	type FieldName = 'loginId' | 'password' | 'nickname' | 'code' | 'newPassword' | 'resetLoginId';
	type FieldErrors = Partial<Record<FieldName, string>>;

	let {
		compact = false,
		onSuccess,
		initialMode = 'login'
	}: {
		/** When true the form is rendered inside a compact sheet (no hero/header). */
		compact?: boolean;
		/** Called after a successful auth; the parent decides where to go. */
		onSuccess?: () => void;
		/** Initial mode. */
		initialMode?: AuthMode;
	} = $props();

	function validateLoginId(v: string): string | null {
		if (!v) return '请输入账号';
		if (v.length < 3 || v.length > 64) return '账号长度需 3–64';
		return null;
	}
	function validatePassword(v: string): string | null {
		if (!v) return '请输入密码';
		if (v.length < 8 || v.length > 128) return '密码长度需 8–128';
		if (!/[a-zA-Z]/.test(v) || !/\d/.test(v)) return '密码需包含字母和数字';
		return null;
	}
	function validateNickname(v: string): string | null {
		if (!v) return '请输入昵称';
		if (v.length < 1 || v.length > 30) return '昵称长度需 1–30';
		return null;
	}

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

	const reset = () => {
		error = null;
		fieldErrors = {};
		submitting = false;
	};

	const switchMode = (m: AuthMode) => {
		mode = m;
		reset();
	};

	function validate(): FieldErrors {
		const errs: FieldErrors = {};
		if (mode === 'login') {
			errs.loginId = validateLoginId(loginId) ?? undefined;
			if (method === 'code') errs.code = code.trim() ? undefined : '请输入验证码';
			else errs.password = validatePassword(password) ?? undefined;
		} else if (mode === 'register') {
			errs.loginId = validateLoginId(loginId) ?? undefined;
			errs.nickname = validateNickname(nickname) ?? undefined;
			errs.password = validatePassword(password) ?? undefined;
		} else if (mode === 'guest') {
			errs.nickname = validateNickname(nickname) ?? undefined;
		} else if (mode === 'forgot') {
			errs.resetLoginId = validateLoginId(resetLoginId) ?? undefined;
		} else if (mode === 'reset') {
			errs.resetLoginId = validateLoginId(resetLoginId) ?? undefined;
			errs.code = code.trim() ? undefined : '请输入验证码';
			errs.newPassword = validatePassword(newPassword) ?? undefined;
		}
		return Object.fromEntries(Object.entries(errs).filter(([, v]) => v)) as FieldErrors;
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
			else if (mode === 'register') res = await register({ login_id: loginId, nickname, password });
			else if (mode === 'guest') res = await guest({ nickname });
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
					client_type: 'web'
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
					placeholder={mode === 'guest' ? '给自己起个名字' : '1–30 个字符'}
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
			<!-- 登录方式选择：表单内标签统一走 TagPill（对齐 AGENTS 表单 chips 契约） -->
			<div class="flex flex-wrap gap-2">
				{#each ([['code', '验证码登录'], ['password', '密码登录']] as ['code' | 'password', string][]) as [id, label]}
					<TagPill active={method === id} onClick={() => (method = id)}>{label}</TagPill>
				{/each}
			</div>
		{/if}
		<label class="flex flex-col gap-1.5" for="auth-field-loginId">
			<span class="text-xs font-medium text-text-secondary"  >手机号、邮箱或账号</span>
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
		{#if mode === 'login' && method === 'code'}
			<!-- 验证码登录：验证码输入 + 获取验证码 -->
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
			<Button
				type="button"
				variant="secondary"
				size="lg"
				class="w-full bg-accent-bg"
				onclick={() => (error = '验证码发送功能即将开放，请使用密码登录')}
			>
				获取验证码
			</Button>
		{:else if mode === 'login' || mode === 'register'}
			<label class="flex flex-col gap-1.5" for="auth-field-password">
				<span class="text-xs font-medium text-text-secondary"  >密码</span>
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
		{#if !submitting && mode === 'login'}
			<ArrowRight class="size-4"  strokeWidth={1.75} />
		{/if}
	</Button>

	{#if mode === 'login' && !compact}
		<!-- 其他方式（对齐 Figma Make LoginPage） -->
		<div class="flex items-center gap-3 text-xs text-text-secondary">
			<span class="h-px flex-1 bg-border-default"></span>
			其他方式
			<span class="h-px flex-1 bg-border-default"></span>
		</div>
		<Button
			type="button"
			variant="secondary"
			size="md"
			class="mt-4 w-full"
			onclick={() => (error = '微信登录暂未开放，请使用密码登录')}
		>
			<MessageCircle class="size-4 text-accent-default"   strokeWidth={1.75} />
			使用微信继续
		</Button>
		<p class="mt-5 text-center text-xs leading-5 text-text-secondary">登录即表示你同意服务条款与隐私政策。</p>
	{/if}

	<!-- Mode switcher -->
	<div class="flex flex-wrap items-center justify-center gap-3.5 mt-1" >
		{#if mode !== 'login' && mode !== 'forgot' && mode !== 'reset'}
			<button type="button" onclick={() => switchMode('login')} class="link-btn text-sm focus-ring text-accent-default" >
				已有账号？登录
			</button>
		{/if}
		{#if mode === 'login'}
			<button type="button" onclick={() => switchMode('register')} class="link-btn text-sm focus-ring text-accent-default" >
				注册新账号
			</button>
			<button type="button" onclick={() => switchMode('forgot')} class="link-btn text-sm focus-ring text-accent-default" >
				忘记密码
			</button>
		{/if}
		{#if mode === 'register' || mode === 'forgot' || mode === 'reset'}
			<button type="button" onclick={() => switchMode('login')} class="link-btn text-sm focus-ring text-accent-default" >
				返回登录
			</button>
		{/if}
		{#if mode !== 'guest' && mode !== 'forgot' && mode !== 'reset'}
			<button type="button" onclick={() => switchMode('guest')} class="link-btn text-sm focus-ring text-accent-default" >
				随便逛逛（访客）
			</button>
		{/if}
	</div>
</form>
