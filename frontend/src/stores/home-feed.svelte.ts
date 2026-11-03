// 首页信息流分段状态（桌面端灵动岛与内容区共用同一份状态）。
// 桌面：分段器渲染在顶栏中央灵动岛（DesktopLayout → HomeFeedIsland）；
// 移动/平板：分段器仍在页内品牌头下方。两条路径共用本 store，避免双份状态。

export type HomeFeedTab = 'discover' | 'following';

let activeTab = $state<HomeFeedTab>('discover');

export const homeFeed = {
	get activeTab() {
		return activeTab;
	},
	setTab(tab: HomeFeedTab) {
		activeTab = tab;
	},
	/** 回到首页时重置为「为你推荐」，与改造前页内组件卸载重建的行为一致。 */
	reset() {
		activeTab = 'discover';
	}
};
