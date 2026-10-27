// 消息页深链落地与分栏可用性的纯函数判定（R13 从 +page.svelte 抽出，便于单测）。
import { ROUTES } from '@/lib/routes';

/**
 * 是否启用 Master-Detail 分栏。
 * 桌面与宽平板（≥960px，如横屏 iPad）双栏；窄平板（820 级纵向）/移动保持单栏。
 */
export function canUseSplitPane(view: { platform: string; isWideTablet: boolean }): boolean {
	return view.platform === 'desktop' || view.isWideTablet;
}

/**
 * 不可分栏时，把 ?peer=/?group= 深链落到单栏推入页。
 * 背景：桌面/宽平板生成的深链（如从通知、历史记录、分享链接进入）在窄平板/手机上
 * 若无人接管就只会停在列表页——表现为「会话面板空白」的死胡同。返回 null 表示无需跳转。
 */
export function resolvePushChatHref(
	queryPeer: string | null,
	queryGroup: string | null
): string | null {
	if (queryGroup) return ROUTES.groupMessage(queryGroup);
	if (queryPeer) return ROUTES.message(queryPeer);
	return null;
}
