<script lang="ts">
	import type { Snippet } from 'svelte';
	import { ArrowRight, LockKeyhole, Mail, MessageCircle } from '@lucide/svelte';
	import { login, register, guest } from '@/stores/auth.svelte';
	import { requestPasswordReset, resetPassword } from '@/api';

	// 全页 Auth 路由与 LoginPromptSheet 共用的认证表单。
	// 五模式：login / register / guest + 忘记密码子流程
	// （request_password_reset → reset_password）。校验规则对齐 harmonyapp：
	//   login_id 3–64，password 8–128 含字母+数字，nickname 1–30。

	export type AuthMode = 'login' | 'register' | 'guest' | 'forgot' | 'reset';

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

	// initialMode 仅在挂载时生效（此后模式切换走 switchMode），此处有意只读取一次初始值。
	function initialModeValue(): AuthMode {
		return initialMode;
	}
	let mode = $state<AuthMode>(initialModeValue());
	// 登录方式（对齐 Figma Make LoginPage 双 tab：验证码登录/密码登录）
	let method = $state<'code' | 'password'>('password');
	let loginId = $state('');
	let password = $state('');
	let nickname = $state('');
	let code = $state('');
	let newPassword = $state('');
	let submitting = $state(false);
	let error = $state<string | null>(null);
	let resetLoginId = $state('');

	const reset = () => {
		error = null;
		submitting = false;
	};

	const switchMode = (m: AuthMode) => {
		mode = m;
		reset();
	};

	const handleSubmit = async () => {
		error = null;

		// Client-side validation aligned with harmonyapp LoginPage.
		let fieldErr: string | null = null;
		if (mode === 'login') {
			fieldErr =
				method === 'code'
					? validateLoginId(loginId) ?? (code.trim() ? null : '请输入验证码')
					: validateLoginId(loginId) ?? validatePassword(password);
		}
		else if (mode === 'register') fieldErr = validateLoginId(loginId) ?? validateNickname(nickname) ?? validatePassword(password);
		else if (mode === 'guest') fieldErr = validateNickname(nickname);
		else if (mode === 'forgot') fieldErr = validateLoginId(resetLoginId);
		else if (mode === 'reset') fieldErr = validateLoginId(resetLoginId) ?? (code ? null : '请输入验证码') ?? validatePassword(newPassword);
		if (fieldErr) {
			error = fieldErr;
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
	class="flex flex-col"
	style="gap:{compact ? 12 : 16}px;"
	onsubmit={(e) => {
		e.preventDefault();
		void handleSubmit();
	}}
>
	{#if mode === 'register' || mode === 'guest'}
		<label class="flex flex-col" style="gap:6px;">
			<span style="font-size:11px;letter-spacing:0.08em;color:var(--text-secondary);">昵称</span>
			<span class="flex items-center gap-2 border border-border-default px-3 py-3" style="background:var(--surface-primary);">
				<input
					type="text"
					bind:value={nickname}
					placeholder={mode === 'guest' ? '给自己起个名字' : '1–30 个字符'}
					disabled={submitting}
					autocomplete="off"
					class="min-w-0 flex-1 bg-transparent text-sm outline-none"
					style="color:var(--text-primary);"
				/>
			</span>
		</label>
	{/if}

	{#if mode === 'login' || mode === 'register'}
		{#if mode === 'login'}
			<!-- 登录方式双 tab（对齐 Figma Make LoginPage：容器 bg-secondary 无圆角，active 仅 bg-card） -->
			<div class="grid grid-cols-2 border border-border-default p-1" style="background:color-mix(in srgb, var(--accent-default) 6%, var(--surface-primary));">
				{#each ([['code', '验证码登录'], ['password', '密码登录']] as ['code' | 'password', string][]) as [id, label]}
					<button
						type="button"
						aria-pressed={method === id}
						onclick={() => (method = id)}
						class="px-3 py-2 text-xs outline-none transition-colors"
						style="background:{method === id ? 'var(--surface-primary)' : 'transparent'};color:{method === id ? 'var(--text-primary)' : 'var(--text-secondary)'};"
					>
						{label}
					</button>
				{/each}
			</div>
		{/if}
		<label class="flex flex-col" style="gap:6px;">
			<span style="font-size:11px;font-weight:500;color:var(--text-secondary);">手机号、邮箱或账号</span>
			<span class="flex items-center gap-2 border border-border-default px-3 py-3" style="background:var(--surface-primary);">
				<Mail style="width:16px;height:16px;color:var(--accent-default);flex-shrink:0;" strokeWidth={1.8} />
				<input
					type="text"
					bind:value={loginId}
					placeholder="请输入账号"
					disabled={submitting}
					autocomplete="off"
					class="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-text-secondary"
					style="color:var(--text-primary);"
				/>
			</span>
		</label>
		{#if mode === 'login' && method === 'code'}
			<!-- 验证码登录：验证码输入 + 获取验证码 -->
			<label class="flex flex-col" style="gap:6px;">
				<span style="font-size:11px;letter-spacing:0.08em;color:var(--text-secondary);">验证码</span>
				<span class="flex items-center gap-2 border border-border-default px-3 py-3" style="background:var(--surface-primary);">
					<input
						type="text"
						bind:value={code}
						placeholder="输入收到的验证码"
						disabled={submitting}
						autocomplete="off"
						class="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-text-secondary"
						style="color:var(--text-primary);"
					/>
				</span>
			</label>
			<button
				type="button"
				class="hover-btn w-full border border-border-default px-3 py-3 text-sm text-text-primary outline-none transition-colors"
				style="background:color-mix(in srgb, var(--accent-default) 6%, var(--surface-primary));"
				onclick={() => (error = '验证码发送功能即将开放，请使用密码登录')}
			>
				获取验证码
			</button>
		{:else if mode === 'login' || mode === 'register'}
			<label class="flex flex-col" style="gap:6px;">
				<span style="font-size:11px;font-weight:500;color:var(--text-secondary);">密码</span>
				<span class="flex items-center gap-2 border border-border-default px-3 py-3" style="background:var(--surface-primary);">
					<LockKeyhole style="width:16px;height:16px;color:var(--accent-default);flex-shrink:0;" strokeWidth={1.8} />
					<input
						type="password"
						bind:value={password}
						placeholder="请输入密码"
						disabled={submitting}
						autocomplete="current-password"
						class="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-text-secondary"
						style="color:var(--text-primary);"
					/>
				</span>
			</label>
		{/if}
	{/if}

	{#if mode === 'forgot' || mode === 'reset'}
		<label class="flex flex-col" style="gap:6px;">
			<span style="font-size:11px;letter-spacing:0.08em;color:var(--text-secondary);">账号</span>
			<span class="flex items-center gap-2 border border-border-default px-3 py-3" style="background:var(--surface-primary);">
				<input
					type="text"
					bind:value={resetLoginId}
					placeholder="注册时的手机号 / 邮箱"
					disabled={submitting}
					autocomplete="off"
					class="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-text-secondary"
					style="color:var(--text-primary);"
				/>
			</span>
		</label>
		{#if mode === 'reset'}
			<label class="flex flex-col" style="gap:6px;">
				<span style="font-size:11px;letter-spacing:0.08em;color:var(--text-secondary);">验证码</span>
				<span class="flex items-center gap-2 border border-border-default px-3 py-3" style="background:var(--surface-primary);">
					<input
						type="text"
						bind:value={code}
						placeholder="输入收到的验证码"
						disabled={submitting}
						autocomplete="off"
						class="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-text-secondary"
						style="color:var(--text-primary);"
					/>
				</span>
			</label>
			<label class="flex flex-col" style="gap:6px;">
				<span style="font-size:11px;letter-spacing:0.08em;color:var(--text-secondary);">新密码</span>
				<span class="flex items-center gap-2 border border-border-default px-3 py-3" style="background:var(--surface-primary);">
					<input
						type="password"
						bind:value={newPassword}
						placeholder="8–128 位，含字母和数字"
						disabled={submitting}
						autocomplete="new-password"
						class="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-text-secondary"
						style="color:var(--text-primary);"
					/>
				</span>
			</label>
		{/if}
	{/if}

	{#if error}
		<p style="font-size:12px;color:var(--status-danger);" role="alert">{error}</p>
	{/if}

	<button
		type="submit"
		class="hover-btn flex w-full items-center justify-center gap-2 px-4 py-3 text-sm font-medium transition-opacity"
		style="background:var(--accent-default);color:white;opacity:{submitting ? 0.6 : 1};"
		disabled={submitting}
	>
		{submitting ? '处理中…' : submitLabel(mode)}
		{#if !submitting && mode === 'login'}
			<ArrowRight style="width:16px;height:16px;" strokeWidth={1.8} />
		{/if}
	</button>

	{#if mode === 'login' && !compact}
		<!-- 其他方式（对齐 Figma Make LoginPage） -->
		<div class="flex items-center gap-3 text-[11px] text-text-secondary">
			<span class="h-px flex-1" style="background:var(--border-default);"></span>
			其他方式
			<span class="h-px flex-1" style="background:var(--border-default);"></span>
		</div>
		<button
			type="button"
			class="hover-btn mt-4 flex w-full items-center justify-center gap-2 border border-border-default px-3 py-2.5 text-xs text-text-primary outline-none transition-colors"
			style="background:var(--surface-primary);"
			onclick={() => (error = '微信登录暂未开放，请使用密码登录')}
		>
			<MessageCircle style="width:16px;height:16px;color:var(--accent-default);" strokeWidth={1.8} />
			使用微信继续
		</button>
		<p class="mt-5 text-center text-[11px] leading-5 text-text-secondary">登录即表示你同意服务条款与隐私政策。</p>
	{/if}

	<!-- Mode switcher -->
	<div class="flex flex-wrap items-center justify-center" style="gap:14px;margin-top:4px;">
		{#if mode !== 'login' && mode !== 'forgot' && mode !== 'reset'}
			<button type="button" onclick={() => switchMode('login')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--accent-default);">
				已有账号？登录
			</button>
		{/if}
		{#if mode === 'login'}
			<button type="button" onclick={() => switchMode('register')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--accent-default);">
				注册新账号
			</button>
			<button type="button" onclick={() => switchMode('forgot')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--accent-default);">
				忘记密码
			</button>
		{/if}
		{#if mode === 'register' || mode === 'forgot' || mode === 'reset'}
			<button type="button" onclick={() => switchMode('login')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--accent-default);">
				返回登录
			</button>
		{/if}
		{#if mode !== 'guest' && mode !== 'forgot' && mode !== 'reset'}
			<button type="button" onclick={() => switchMode('guest')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--accent-default);">
				随便逛逛（访客）
			</button>
		{/if}
	</div>
</form>
