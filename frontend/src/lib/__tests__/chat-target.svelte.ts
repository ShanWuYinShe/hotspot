// 测试用可变会话标识（runes 模块）：驱动 useChat 的 targetId/draftKey 变化，
// 供「切换会话」用例验证草稿隔离与失败重发状态不跨会话。
let current = $state('p1');

export const chatTarget = {
	get value() {
		return current;
	},
	set value(next: string) {
		current = next;
	}
};
