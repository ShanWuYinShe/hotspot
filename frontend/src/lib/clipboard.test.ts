import { describe, it, expect, vi, afterEach } from 'vitest';
import { copyText } from './clipboard';

const originalClipboard = navigator.clipboard;

function stubClipboard(writeText: () => Promise<void>) {
	Object.defineProperty(navigator, 'clipboard', {
		value: { writeText },
		configurable: true
	});
}

afterEach(() => {
	Object.defineProperty(navigator, 'clipboard', { value: originalClipboard, configurable: true });
	vi.restoreAllMocks();
});

describe('copyText', () => {
	it('Clipboard API 可用时写入成功', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		stubClipboard(writeText);
		await expect(copyText('audit_123')).resolves.toBe(true);
		expect(writeText).toHaveBeenCalledWith('audit_123');
	});

	it('Clipboard API 被拒时回退 execCommand 成功', async () => {
		stubClipboard(vi.fn().mockRejectedValue(new Error('denied')));
		const execCommand = vi.fn().mockReturnValue(true);
		(document as unknown as { execCommand: typeof execCommand }).execCommand = execCommand;
		await expect(copyText('audit_456')).resolves.toBe(true);
		expect(execCommand).toHaveBeenCalledWith('copy');
	});

	it('两种方式都失败时返回 false', async () => {
		stubClipboard(vi.fn().mockRejectedValue(new Error('denied')));
		const execCommand = vi.fn().mockReturnValue(false);
		(document as unknown as { execCommand: typeof execCommand }).execCommand = execCommand;
		await expect(copyText('audit_789')).resolves.toBe(false);
	});

	it('空文本直接返回 false，不触碰剪贴板', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		stubClipboard(writeText);
		await expect(copyText('')).resolves.toBe(false);
		expect(writeText).not.toHaveBeenCalled();
	});
});
