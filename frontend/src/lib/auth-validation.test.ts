import { describe, expect, it } from 'vitest';
import {
	validateAuthFields,
	validateLoginId,
	validateNickname,
	validatePassword,
	type AuthMode
} from './auth-validation';

describe('validateLoginId', () => {
	it('空与越界长度报错', () => {
		expect(validateLoginId('')).toBe('请输入账号');
		expect(validateLoginId('ab')).toBe('账号长度需 3–64');
		expect(validateLoginId('a'.repeat(65))).toBe('账号长度需 3–64');
	});
	it('3–64 合法', () => {
		expect(validateLoginId('abc')).toBeNull();
		expect(validateLoginId('a'.repeat(64))).toBeNull();
	});
});

describe('validatePassword', () => {
	it('长度与构成规则', () => {
		expect(validatePassword('')).toBe('请输入密码');
		expect(validatePassword('a1')).toBe('密码长度需 8–128');
		expect(validatePassword('abcdefgh')).toBe('密码需包含字母和数字');
		expect(validatePassword('12345678')).toBe('密码需包含字母和数字');
		expect(validatePassword('abcd1234')).toBeNull();
	});
});

describe('validateNickname', () => {
	it('必填与可选语义', () => {
		expect(validateNickname('')).toBe('请输入昵称');
		expect(validateNickname('', { optional: true })).toBeNull();
		expect(validateNickname('n'.repeat(31))).toBe('昵称长度需 1–30');
	});
});

describe('validateAuthFields', () => {
	const base = {
		method: 'password' as const,
		loginId: 'alice',
		password: 'abcd1234',
		nickname: 'alice',
		code: '',
		newPassword: '',
		resetLoginId: '',
		needsRegisterCode: false
	};
	const run = (mode: AuthMode, over: Partial<typeof base> = {}) =>
		validateAuthFields({ ...base, mode, ...over });

	it('login 密码方式校验账号+密码', () => {
		expect(run('login')).toEqual({});
		expect(run('login', { password: 'x' }).password).toBe('密码长度需 8–128');
	});

	it('login 验证码方式校验账号+验证码', () => {
		expect(run('login', { method: 'code', code: '  ' }).code).toBe('请输入验证码');
		expect(run('login', { method: 'code', code: '123456' })).toEqual({});
	});

	it('register 手机号/邮箱型需验证码，用户名型不需要', () => {
		expect(run('register', { loginId: '13800138000', needsRegisterCode: true }).code).toBe('请输入验证码');
		expect(run('register', { loginId: 'a@b.com', needsRegisterCode: true, code: '123456' })).toEqual({});
		expect(run('register', { loginId: 'username' })).toEqual({});
	});

	it('guest 昵称可空', () => {
		expect(run('guest', { nickname: '' })).toEqual({});
	});

	it('forgot 仅校验 resetLoginId；reset 额外校验验证码与新密码', () => {
		expect(run('forgot', { resetLoginId: '' }).resetLoginId).toBe('请输入账号');
		expect(run('reset', { resetLoginId: 'alice', code: '', newPassword: 'weak' }).code).toBe('请输入验证码');
		expect(run('reset', { resetLoginId: 'alice', code: '123', newPassword: 'nopassword' }).newPassword).toBe('密码需包含字母和数字');
	});
});
