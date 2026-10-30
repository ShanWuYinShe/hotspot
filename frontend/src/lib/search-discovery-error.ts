// 搜索页发现态失败聚合的纯决策逻辑（DOM/请求无关，便于单测）。
//
// 发现态由三个独立请求拼成：热门话题+热门动态（同一响应）、人物推荐、推荐群组。
// 任一失败都不得静默转空态（布局契约：失败必须展示并可重试），呈现分三档：
// - none  ：全部成功（各区块按「有数据才渲染 / 诚实空态」的既有规则处理）；
// - inline：部分失败，或全部失败但页面仍有「最近搜索」可看 → 内联错误条 + 重试，
//           保留已成功的区块，不把整页换成错误块；
// - full  ：全部失败且没有任何本地内容 → 整块 LoadError + 重试。

export interface DiscoveryFailureFlags {
	moments: boolean;
	users: boolean;
	groups: boolean;
}

export type DiscoveryErrorView = 'none' | 'inline' | 'full';

export function resolveDiscoveryErrorView(failed: DiscoveryFailureFlags, hasRecent: boolean): DiscoveryErrorView {
	const count = Number(failed.moments) + Number(failed.users) + Number(failed.groups);
	if (count === 0) return 'none';
	if (count === 3 && !hasRecent) return 'full';
	return 'inline';
}
