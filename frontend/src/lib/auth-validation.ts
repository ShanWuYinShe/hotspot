	// 认证表单校验（AuthForm 与未来其他入口共用；对齐 harmonyapp LoginPage）：
	// login_id 3–64，password 8–128 含字母+数字，nickname 1–30（访客可空）。
	export type AuthMode = 'login' | 'register' | 'guest' | 'forgot' | 'reset';
	export type AuthFieldName = 'loginId' | 'password' | 'nickname' | 'code' | 'newPassword' | 'resetLoginId';
	export type AuthFieldErrors = Partial<Record<AuthFieldName, string>>;

	export function validateLoginId(v: string): string | null {
		if (!v) return '请输入账号';
		if (v.length < 3 || v.length > 64) return '账号长度需 3–64';
		return null;
	}

	export function validatePassword(v: string): string | null {
		if (!v) return '请输入密码';
		if (v.length < 8 || v.length > 128) return '密码长度需 8–128';
		if (!/[a-zA-Z]/.test(v) || !/\d/.test(v)) return '密码需包含字母和数字';
		return null;
	}

	export function validateNickname(v: string, opts: { optional?: boolean } = {}): string | null {
		if (!v) return opts.optional ? null : '请输入昵称';
		if (v.length < 1 || v.length > 30) return '昵称长度需 1–30';
		return null;
	}

	// 按模式聚合校验：返回非空错误表（空对象即通过）。
	// needsRegisterCode：注册时 login_id 为手机号（纯数字）或邮箱（含 '@'），
	// 提交需携带验证码（spec auth/send_register_code 所有权验证）。
	export function validateAuthFields(input: {
		mode: AuthMode;
		method: 'code' | 'password';
		loginId: string;
		password: string;
		nickname: string;
		code: string;
		newPassword: string;
		resetLoginId: string;
		needsRegisterCode: boolean;
	}): AuthFieldErrors {
		const errs: AuthFieldErrors = {};
		const { mode, method } = input;
		if (mode === 'login') {
			errs.loginId = validateLoginId(input.loginId) ?? undefined;
			if (method === 'code') errs.code = input.code.trim() ? undefined : '请输入验证码';
			else errs.password = validatePassword(input.password) ?? undefined;
		} else if (mode === 'register') {
			errs.loginId = validateLoginId(input.loginId) ?? undefined;
			errs.nickname = validateNickname(input.nickname) ?? undefined;
			errs.password = validatePassword(input.password) ?? undefined;
			// 手机号/邮箱型 login_id 必须携带验证码（所有权验证）；用户名型不校验
			if (input.needsRegisterCode) errs.code = input.code.trim() ? undefined : '请输入验证码';
		} else if (mode === 'guest') {
			// 访客昵称可选：留空由服务端自动生成默认昵称
			errs.nickname = validateNickname(input.nickname, { optional: true }) ?? undefined;
		} else if (mode === 'forgot') {
			errs.resetLoginId = validateLoginId(input.resetLoginId) ?? undefined;
		} else if (mode === 'reset') {
			errs.resetLoginId = validateLoginId(input.resetLoginId) ?? undefined;
			errs.code = input.code.trim() ? undefined : '请输入验证码';
			errs.newPassword = validatePassword(input.newPassword) ?? undefined;
		}
		return Object.fromEntries(Object.entries(errs).filter(([, v]) => v)) as AuthFieldErrors;
	}
