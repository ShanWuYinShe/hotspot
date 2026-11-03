import type { Snippet } from 'svelte';

// PageIsland（原 header-island）——桌面端顶栏中央「页面锚点岛」的状态单例。
//
// 目标：把「会随滚动离开视口、但用户随时可能需要」的常驻信息（标题/身份/状态/行动）
// 固定到 52px 顶栏中央，用户自由滚动也不丢失上下文。
//
// 注册契约：
//   - 每个页面（或详情页头）用一个 token 注册；同一时刻只有最后注册者生效（active）。
//   - 页面自己负责判定「锚点是否已离开视口」，并通过 setVisible 同步岛的显隐；
//     store 只存状态与内容，不做滚动监听（监听归 PageHeader/DetailHeader/action）。
//   - release 只释放自己的 token，避免路由切换时旧页面把新页面的岛关掉。
//
// 内容模型（三段式）：[leading: 返回/头像] · [title + meta] · [trailing: ≤2 行动]

export interface AuthorInfo {
	id?: string;
	name: string;
	avatar?: string | null;
	location?: string | null;
	time?: string | null;
}

export interface PageIslandConfig {
	/** 主标题（页面名 / 作者名 / 群名 / 会话对象） */
	title: string;
	/** 副信息一行：计数 / 位置 / 状态 / 当前 tab（可选） */
	meta?: string;
	/** 带头像的身份信息（详情页作者/群/用户），与 title 二选一使用 */
	author?: AuthorInfo;
	/** 左侧返回按钮（详情页等） */
	onBack?: () => void;
	backLabel?: string;
	/** 点击头像/身份区（作者名片的既有交互） */
	onAuthorClick?: () => void;
	/** 点击标题回到锚点；缺省由顶栏滚动回页首 */
	onTitleClick?: () => void;
	/** 右侧行动区（写操作/入口）：每页最多 2 个控件 */
	trailing?: Snippet;
}

/** 注册令牌：页面/页头各自持有，只允许当前 token 更新与关闭。 */
export type IslandToken = symbol;

let currentToken: IslandToken | null = null;

class PageIslandStore {
	/** 是否已有页面注册（路由切换时用于避免残留） */
	active = $state(false);
	/** 锚点是否已离开视口（由注册方判定） */
	visible = $state(false);

	title = $state('');
	meta = $state('');
	author = $state<AuthorInfo | undefined>(undefined);
	onBack = $state<(() => void) | undefined>(undefined);
	backLabel = $state('返回');
	onAuthorClick = $state<(() => void) | undefined>(undefined);
	onTitleClick = $state<(() => void) | undefined>(undefined);
	trailing = $state<Snippet | undefined>(undefined);

	register(token: IslandToken, config: PageIslandConfig) {
		currentToken = token;
		this.applyConfig(config);
		this.active = true;
		this.visible = false;
	}

	/** 仅当前持有者能更新（路由切换后旧组件的 effect 更新会被忽略）。 */
	update(token: IslandToken, config: Partial<PageIslandConfig>) {
		if (token !== currentToken) return;
		if (config.title !== undefined) this.title = config.title;
		if (config.meta !== undefined) this.meta = config.meta;
		if (config.author !== undefined) this.author = config.author;
		if (config.onBack !== undefined) this.onBack = config.onBack;
		if (config.backLabel !== undefined) this.backLabel = config.backLabel;
		if (config.onAuthorClick !== undefined) this.onAuthorClick = config.onAuthorClick;
		if (config.onTitleClick !== undefined) this.onTitleClick = config.onTitleClick;
		if (config.trailing !== undefined) this.trailing = config.trailing;
	}

	/** 锚点显隐（滚动触发方调用）；非当前持有者忽略。 */
	setVisible(token: IslandToken, visible: boolean) {
		if (token !== currentToken) return;
		this.visible = visible;
	}

	/** 释放自身 token：只有当前持有者释放时才真正清空状态。 */
	release(token: IslandToken) {
		if (token !== currentToken) return;
		currentToken = null;
		this.reset();
	}

	private applyConfig(config: PageIslandConfig) {
		this.title = config.title;
		this.meta = config.meta ?? '';
		this.author = config.author;
		this.onBack = config.onBack;
		this.backLabel = config.backLabel ?? '返回';
		this.onAuthorClick = config.onAuthorClick;
		this.onTitleClick = config.onTitleClick;
		this.trailing = config.trailing;
	}

	private reset() {
		this.active = false;
		this.visible = false;
		this.title = '';
		this.meta = '';
		this.author = undefined;
		this.onBack = undefined;
		this.backLabel = '返回';
		this.onAuthorClick = undefined;
		this.onTitleClick = undefined;
		this.trailing = undefined;
	}
}

export const headerIsland = new PageIslandStore();
