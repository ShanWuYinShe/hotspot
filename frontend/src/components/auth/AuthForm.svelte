<script lang="ts">
	import type { Snippet } from 'svelte';
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

	let mode = $state<AuthMode>(initialMode);
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
		if (mode === 'login') fieldErr = validateLoginId(loginId) ?? validatePassword(password);
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
			if (mode === 'login') res = await login({ login_id: loginId, password });
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
			<span style="font-size:11px;letter-spacing:0.08em;color:var(--neu-text-muted);">昵称</span>
			<input
				type="text"
				bind:value={nickname}
				placeholder={mode === 'guest' ? '给自己起个名字' : '1–30 个字符'}
				disabled={submitting}
				autocomplete="off"
				class="rounded-[12px] px-3.5 py-2.5 text-[15px] outline-none transition-shadow"
				style="background:var(--neu-bg);color:var(--neu-text);border:1px solid var(--neu-border);"
			/>
		</label>
	{/if}

	{#if mode === 'login' || mode === 'register'}
		<label class="flex flex-col" style="gap:6px;">
			<span style="font-size:11px;letter-spacing:0.08em;color:var(--neu-text-muted);">账号</span>
			<input
				type="text"
				bind:value={loginId}
				placeholder="手机号 / 邮箱 / 用户名"
				disabled={submitting}
				autocomplete="off"
				class="rounded-[12px] px-3.5 py-2.5 text-[15px] outline-none transition-shadow"
				style="background:var(--neu-bg);color:var(--neu-text);border:1px solid var(--neu-border);"
			/>
		</label>
		<label class="flex flex-col" style="gap:6px;">
			<span style="font-size:11px;letter-spacing:0.08em;color:var(--neu-text-muted);">密码</span>
			<input
				type="password"
				bind:value={password}
				placeholder="8–128 位，含字母和数字"
				disabled={submitting}
				autocomplete="current-password"
				class="rounded-[12px] px-3.5 py-2.5 text-[15px] outline-none transition-shadow"
				style="background:var(--neu-bg);color:var(--neu-text);border:1px solid var(--neu-border);"
			/>
		</label>
	{/if}

	{#if mode === 'forgot' || mode === 'reset'}
		<label class="flex flex-col" style="gap:6px;">
			<span style="font-size:11px;letter-spacing:0.08em;color:var(--neu-text-muted);">账号</span>
			<input
				type="text"
				bind:value={resetLoginId}
				placeholder="注册时的手机号 / 邮箱"
				disabled={submitting}
				autocomplete="off"
				class="rounded-[12px] px-3.5 py-2.5 text-[15px] outline-none transition-shadow"
				style="background:var(--neu-bg);color:var(--neu-text);border:1px solid var(--neu-border);"
			/>
		</label>
		{#if mode === 'reset'}
			<label class="flex flex-col" style="gap:6px;">
				<span style="font-size:11px;letter-spacing:0.08em;color:var(--neu-text-muted);">验证码</span>
				<input
					type="text"
					bind:value={code}
					placeholder="输入收到的验证码"
					disabled={submitting}
					autocomplete="off"
					class="rounded-[12px] px-3.5 py-2.5 text-[15px] outline-none transition-shadow"
					style="background:var(--neu-bg);color:var(--neu-text);border:1px solid var(--neu-border);"
				/>
			</label>
			<label class="flex flex-col" style="gap:6px;">
				<span style="font-size:11px;letter-spacing:0.08em;color:var(--neu-text-muted);">新密码</span>
				<input
					type="password"
					bind:value={newPassword}
					placeholder="8–128 位，含字母和数字"
					disabled={submitting}
					autocomplete="new-password"
					class="rounded-[12px] px-3.5 py-2.5 text-[15px] outline-none transition-shadow"
					style="background:var(--neu-bg);color:var(--neu-text);border:1px solid var(--neu-border);"
				/>
			</label>
		{/if}
	{/if}

	{#if error}
		<p style="font-size:12px;color:var(--semantic-hot);" role="alert">{error}</p>
	{/if}

	<button
		type="submit"
		class="neu-hover-btn w-full rounded-[14px] py-3 text-[15px] font-medium transition-opacity"
		style="background:var(--neu-accent-primary);color:white;opacity:{submitting ? 0.6 : 1};"
		disabled={submitting}
	>
		{submitting ? '处理中…' : submitLabel(mode)}
	</button>

	<!-- Mode switcher -->
	<div class="flex flex-wrap items-center justify-center" style="gap:14px;margin-top:4px;">
		{#if mode !== 'login' && mode !== 'forgot' && mode !== 'reset'}
			<button type="button" onclick={() => switchMode('login')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--neu-accent-primary);">
				已有账号？登录
			</button>
		{/if}
		{#if mode === 'login'}
			<button type="button" onclick={() => switchMode('register')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--neu-accent-primary);">
				注册新账号
			</button>
			<button type="button" onclick={() => switchMode('forgot')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--neu-accent-primary);">
				忘记密码
			</button>
		{/if}
		{#if mode === 'register' || mode === 'forgot' || mode === 'reset'}
			<button type="button" onclick={() => switchMode('login')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--neu-accent-primary);">
				返回登录
			</button>
		{/if}
		{#if mode !== 'guest' && mode !== 'forgot' && mode !== 'reset'}
			<button type="button" onclick={() => switchMode('guest')} class="text-[12px] transition-opacity hover:opacity-70" style="color:var(--neu-accent-primary);">
				随便逛逛（访客）
			</button>
		{/if}
	</div>
</form>
