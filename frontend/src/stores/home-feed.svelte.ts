// 首页信息流分段状态（桌面端灵动岛与移动端内容区共用同一份状态）。
// 桌面：通过 HomeDesktop 注册默认常驻灵动岛（headerIsland），支持在展开面板中切换流；
// 移动/平板：分段器在页内品牌头下方。两条路径共用本 store，避免双份状态。

export type HomeFeedTab = 'discover' | 'following';

let activeTab = $state<HomeFeedTab>('discover');
// 桌面端：分段器滚出视口后显示顶栏灵动岛（handoff），回到顶部再隐藏。
let islandVisible = $state(false);

export const homeFeed = {
	get activeTab() {
		return activeTab;
	},
	setTab(tab: HomeFeedTab) {
		activeTab = tab;
	},
	get islandVisible() {
		return islandVisible;
	},
	setIslandVisible(visible: boolean) {
		islandVisible = visible;
	},
	/** 回到首页时重置为「为你推荐」，与改造前页内组件卸载重建的行为一致。 */
	reset() {
		activeTab = 'discover';
		islandVisible = false;
	}
};
