// 文本复制统一封装：优先 Clipboard API（安全上下文 + 权限允许），
// 非安全上下文/权限被拒/API 缺失时回退到 execCommand('copy')（保留用户手势内执行）。
// 返回是否复制成功，调用方据此给成功/失败反馈。
export async function copyText(text: string): Promise<boolean> {
	if (!text) return false;
	try {
		if (navigator.clipboard?.writeText) {
			await navigator.clipboard.writeText(text);
			return true;
		}
	} catch {
		// 回退到 execCommand
	}
	try {
		const textarea = document.createElement('textarea');
		textarea.value = text;
		textarea.setAttribute('readonly', '');
		textarea.style.position = 'fixed';
		textarea.style.opacity = '0';
		document.body.appendChild(textarea);
		textarea.select();
		const copied = document.execCommand('copy');
		document.body.removeChild(textarea);
		return copied;
	} catch {
		return false;
	}
}
