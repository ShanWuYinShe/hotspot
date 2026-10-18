// AuthForm 契约测试：字段级校验、错误展示、密码显隐、登录成功调用、
// 注册验证码（手机号/邮箱型）、访客昵称可选。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import AuthForm from '../AuthForm.svelte';
import { login, register, guest } from '@/stores/auth.svelte';
import { requestPasswordReset, resetPassword, sendRegisterCode, getClientConfig } from '@/api';

vi.mock('@/stores/auth.svelte', () => ({
	login: vi.fn(),
	register: vi.fn(),
	guest: vi.fn()
}));
vi.mock('@/api', () => ({
	requestPasswordReset: vi.fn(),
	resetPassword: vi.fn(),
	sendRegisterCode: vi.fn(),
	getClientConfig: vi.fn()
}));

describe('AuthForm', () => {
	beforeEach(() => {
		vi.mocked(login).mockReset();
		vi.mocked(register).mockReset();
		vi.mocked(guest).mockReset();
		vi.mocked(requestPasswordReset).mockReset();
		vi.mocked(resetPassword).mockReset();
		vi.mocked(sendRegisterCode).mockReset();
		vi.mocked(login).mockResolvedValue({ code: 0, data: {} });
		vi.mocked(register).mockResolvedValue({ code: 0, data: {} });
		vi.mocked(guest).mockResolvedValue({ code: 0, data: {} });
		vi.mocked(sendRegisterCode).mockResolvedValue({ code: 0, data: undefined });
		vi.mocked(getClientConfig).mockResolvedValue({
			code: 0,
			data: { contact: { terms_of_service_url: '', privacy_policy_url: '' } }
		} as never);
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

	it('手机号型注册显示验证码输入，空验证码提交报错', async () => {
		render(AuthForm, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: '注册新账号' }));
		await fireEvent.input(screen.getByLabelText('手机号、邮箱或账号'), { target: { value: '13800138000' } });
		await fireEvent.input(screen.getByLabelText('昵称'), { target: { value: '小鹿' } });
		await fireEvent.input(screen.getByLabelText('密码'), { target: { value: 'pass1234' } });
		// 手机号（纯数字）型 login_id → 显示验证码输入与发送按钮
		expect(screen.getByLabelText('验证码')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '注册并登录' }));
		expect((await screen.findAllByText('请输入验证码')).length).toBeGreaterThan(0);
		expect(register).not.toHaveBeenCalled();
	});

	it('用户名型注册不显示验证码输入，code 传 null', async () => {
		render(AuthForm, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: '注册新账号' }));
		await fireEvent.input(screen.getByLabelText('手机号、邮箱或账号'), { target: { value: 'lumos' } });
		await fireEvent.input(screen.getByLabelText('昵称'), { target: { value: '小鹿' } });
		await fireEvent.input(screen.getByLabelText('密码'), { target: { value: 'pass1234' } });
		expect(screen.queryByLabelText('验证码')).toBeNull();
		await fireEvent.click(screen.getByRole('button', { name: '注册并登录' }));
		await waitFor(() =>
			expect(register).toHaveBeenCalledWith({ login_id: 'lumos', nickname: '小鹿', password: 'pass1234', code: null })
		);
	});

	it('手机号型注册点发送验证码调用 sendRegisterCode', async () => {
		render(AuthForm, { props: {} });
		await fireEvent.click(screen.getByRole('button', { name: '注册新账号' }));
		await fireEvent.input(screen.getByLabelText('手机号、邮箱或账号'), { target: { value: 'lumos@example.com' } });
		await fireEvent.click(screen.getByRole('button', { name: '发送验证码' }));
		await waitFor(() =>
			expect(sendRegisterCode).toHaveBeenCalledWith({ login_id: 'lumos@example.com', client_type: 'web' })
		);
	});

	it('访客空昵称可提交（nickname 传 null）', async () => {
		render(AuthForm, { props: { initialMode: 'guest' } });
		expect(screen.queryByText('请输入昵称')).toBeNull();
		await fireEvent.click(screen.getByRole('button', { name: '以访客身份继续' }));
		await waitFor(() => expect(guest).toHaveBeenCalledWith({ nickname: null }));
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

	// 回归守卫（走查 F1）：/login 的卡片文案取自 onModeChange 回传的 mode。内部 forgot→reset
	// 步骤迁移若只改内部 mode 而不通知宿主，表单已进入「账号+验证码+新密码」而页头仍停在
	// 「找回你的账号。/ 我们会发送验证码。」。
	it('忘记密码发送成功后回调 onModeChange(reset)', async () => {
		vi.mocked(requestPasswordReset).mockResolvedValue({ code: 0, data: undefined } as never);
		const onModeChange = vi.fn();
		render(AuthForm, { props: { onModeChange } });
		await fireEvent.click(screen.getByRole('button', { name: '忘记密码' }));
		await fireEvent.input(screen.getByPlaceholderText('注册时的手机号 / 邮箱'), { target: { value: '13800000000' } });
		await fireEvent.click(screen.getByRole('button', { name: '发送验证码' }));
		await waitFor(() => expect(onModeChange).toHaveBeenCalledWith('reset'));
	});
});
