// 聊天输入草稿（Web）：按「会话作用域 + id」持久化未发送文本，切换会话 / 刷新页面 /
// 从会话列表返回都能恢复。与 Android `ChatDrafts`（SharedPreferences）同语义：
// 发送成功、删除会话、退出登录时清理，避免把上一位登录用户的输入泄露给下一位。
//
// 只存文本：图片在选图后即上传并随消息发送，未发送图片无法跨会话恢复。
// 隐私模式 / 禁用存储时 localStorage 会抛错——草稿降级为不持久化，聊天本身不受影响。
const PREFIX = 'hotspot.chatDraft.';

/** 会话作用域：私聊按对端 user_id、群聊按 group_id 分键，避免同 id 串档。 */
export type ChatDraftScope = 'private' | 'group';

export function chatDraftKey(scope: ChatDraftScope, id: string): string {
	return `${scope}:${id}`;
}

function getStorage(): Storage | null {
	try {
		return typeof localStorage === 'undefined' ? null : localStorage;
	} catch {
		return null;
	}
}

export function loadChatDraft(key: string): string {
	if (!key) return '';
	try {
		return getStorage()?.getItem(PREFIX + key) ?? '';
	} catch {
		return '';
	}
}

/** 保存草稿；空白文本等同清空（不落盘），避免只有空格的草稿在下次进入时被恢复。 */
export function saveChatDraft(key: string, text: string): void {
	if (!key) return;
	try {
		const storage = getStorage();
		if (!storage) return;
		if (!text.trim()) {
			storage.removeItem(PREFIX + key);
			return;
		}
		storage.setItem(PREFIX + key, text);
	} catch {
		// 配额超限 / 存储不可用：静默降级，不打断输入。
	}
}

export function clearChatDraft(key: string): void {
	if (!key) return;
	try {
		getStorage()?.removeItem(PREFIX + key);
	} catch {
		// 忽略存储异常
	}
}

/** 退出登录 / 切换账号时清空本机全部聊天草稿（只动本功能前缀，不碰其他本地数据）。 */
export function clearAllChatDrafts(): void {
	try {
		const storage = getStorage();
		if (!storage) return;
		const keys: string[] = [];
		for (let i = 0; i < storage.length; i += 1) {
			const key = storage.key(i);
			if (key && key.startsWith(PREFIX)) keys.push(key);
		}
		for (const key of keys) storage.removeItem(key);
	} catch {
		// 忽略存储异常
	}
}
