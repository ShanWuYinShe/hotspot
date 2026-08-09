// AuthForm 契约测试：字段级校验、错误展示、密码显隐、登录成功调用。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import AuthForm from '../AuthForm.svelte';
import { login, register, guest } from '@/stores/auth.svelte';
import { requestPasswordReset, resetPassword } from '@/api';

vi.mock('@/stores/auth.svelte', () => ({
	login: vi.fn(),
	register: vi.fn(),
	guest: vi.fn()
}));
vi.mock('@/api', () => ({
	requestPasswordReset: vi.fn(),
	resetPassword: vi.fn()
}));

describe('AuthForm', () => {
	beforeEach(() => {
		vi.mocked(login).mockReset();
		vi.mocked(register).mockReset();
		vi.mocked(guest).mockReset();
		vi.mocked(requestPasswordReset).mockReset();
		vi.mocked(resetPassword).mockReset();
		vi.mocked(login).mockResolvedValue({ code: 0, data: {} });
	});

	it('空提交显示逐字段错误', async () => {
		render(AuthForm, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: '登录' }));
		expect((await screen.findAllByText('请输入账号')).length).toBeGreaterThan(0);
		expect(screen.getByText('请输入密码')).toBeTruthy();
		expect(login).not.toHaveBeenCalled();
	});

	it('注册模式校验昵称/密码', async () => {
		render(AuthForm, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: '注册新账号' }));
		await fireEvent.click(screen.getByRole('button', { name: '注册并登录' }));
		expect(await screen.findByText('请输入昵称')).toBeTruthy();
		expect(screen.getAllByText('请输入账号').length).toBeGreaterThan(0);
		expect(screen.getAllByText('请输入密码').length).toBeGreaterThan(0);
		expect(register).not.toHaveBeenCalled();
	});

	it('登录成功调用 login（client_type 由 store 补充）', async () => {
		render(AuthForm, { props: {} });
		await fireEvent.input(screen.getByLabelText('手机号、邮箱或账号'), { target: { value: 'alice' } });
		await fireEvent.input(screen.getByLabelText('密码'), { target: { value: 'pass1234' } });
		await fireEvent.click(screen.getByRole('button', { name: '登录' }));
		await waitFor(() => expect(login).toHaveBeenCalledWith({ login_id: 'alice', password: 'pass1234' }));
	});

	it('密码框支持显隐切换', async () => {
		render(AuthForm, { props: {} });
		const password = screen.getByLabelText('密码') as HTMLInputElement;
		expect(password.type).toBe('password');
		await fireEvent.click(screen.getByRole('button', { name: '显示密码' }));
		expect((screen.getByLabelText('密码') as HTMLInputElement).type).toBe('text');
	});
});
