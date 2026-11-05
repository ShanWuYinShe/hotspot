import type { Snippet } from 'svelte';
import type { IslandRect } from '@/lib/island-morph';
import { publishHandoff } from '@/lib/island-handoff';

// PageIsland（原 header-island）——桌面端顶栏中央「页面锚点岛」的状态单例。
//
// 目标：把「会随滚动离开视口、但用户随时可能需要」的常驻信息（标题/身份/状态/行动）
// 固定到 52px 顶栏中央，用户自由滚动也不丢失上下文。
//
// 注册契约：
//   - 每个页面（或详情页头）用一个 token 注册；同一时刻只有最后注册者生效（active）。
//   - 页面用 setHandoff 上报「标题 → 岛」的滚动交接进度 p（0→1）；store 只存状态与内容，
//     不做滚动监听（监听归 PageHeader/DetailHeader）。
//   - release 只释放自己的 token，避免路由切换时旧页面把新页面的岛关掉；
//     注册/释放都会把交接归零 —— 路由切换时旧岛直接消失，不会再出现「飞回上一页标题」的幽灵动画。
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
	/** 交接是否已开始（p>0）：控制岛是否挂载。 */
	visible = $state(false);
	/** 交接是否已基本落位（p>0.55，滞回）：用于页头 inert / 无障碍不可达。 */
	settled = $state(false);

	/**
	 * 当前页面阅读进度（0–100）。由唯一滚动承载区 RoutedContentViewport 写入，
	 * 顶栏岛底部进度条消费（首页信息流岛与通用页面岛共用同一进度）。
	 * 与注册 token 无关：首页不注册岛，但同样需要进度。
	 */
	progress = $state(0);

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
		this.settled = false;
		// 新页面接管：把上一页残留的交接进度归零（路由切换不再有幽灵回飞）。
		publishHandoff({ p: 0, source: null });
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

	/**
	 * 滚动交接进度（滚动触发方每帧调用）；非当前持有者忽略。
	 * p=0 标题完整可见、岛不存在；p=1 标题已进顶栏、岛完全落位。
	 * source 为锚点当前矩形（非吸顶页头每帧跟手更新；吸顶页头为固定快照）。
	 */
	setHandoff(token: IslandToken, p: number, source: IslandRect | null) {
		if (token !== currentToken) return;
		const next = Math.min(1, Math.max(0, Number.isFinite(p) ? p : 0));
		const visible = next > 0.001;
		const settled = next > 0.55;
		if (visible !== this.visible) this.visible = visible;
		if (settled !== this.settled) this.settled = settled;
		publishHandoff({ p: next, source });
	}

	/** 阅读进度（滚动 owner 调用；与 token 无关，首页岛同样消费）。 */
	setProgress(value: number) {
		this.progress = Math.min(100, Math.max(0, Number.isFinite(value) ? value : 0));
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
		this.settled = false;
		publishHandoff({ p: 0, source: null });
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