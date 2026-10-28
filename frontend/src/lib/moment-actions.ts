import { favoriteMoment, setReaction } from '@/api';
import { showToast } from '@/stores/toast.svelte';
import { ROUTES } from '@/lib/routes';

// 瞬间卡「点赞 / 收藏 / 分享」的共享动作实现（R14 卡片交互契约）。
//
// 为什么抽到这里：首页信息流卡（HomeFeedItem）与网格卡（MomentCard）此前各写一套——
// 信息流卡的图标只是装饰 div（点了只会打开详情），网格卡是真按钮；点赞/收藏的失败
// 提示、乐观回滚语义也不一致。卡片交互契约要求「同类卡片同一套行为」，因此把
// 请求 + 失败 toast 收敛为唯一实现，乐观状态由调用方（组件本地或 feed store）持有。
//
// 注意：本模块不做登录拦截（requireAuth 属于 UI 层，由卡片组件在点击时先行判定）。

/** 点赞/取消点赞；返回是否成功（调用方据此回滚乐观状态）。 */
export async function setMomentLiked(momentId: string, nextLiked: boolean, emoji = '👍'): Promise<boolean> {
	try {
		const res = await setReaction({ target_type: 'moment', target_id: momentId, emoji, active: nextLiked });
		if (res.code !== 0) throw new Error(res.message || '操作失败');
		return true;
	} catch (err) {
		showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		return false;
	}
}

/** 收藏/取消收藏；返回是否成功（调用方据此回滚乐观状态）。 */
export async function setMomentSaved(momentId: string, nextSaved: boolean): Promise<boolean> {
	try {
		const res = await favoriteMoment({ moment_id: momentId, active: nextSaved });
		if (res.code !== 0) throw new Error(res.message || '操作失败');
		return true;
	} catch (err) {
		showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		return false;
	}
}

/**
 * 分享瞬间：优先系统分享（navigator.share），否则复制链接并提示。
 * 所有卡片共用同一文案与失败提示（避免有的卡复制、有的卡静默）。
 */
export async function shareMoment(opts: { momentId: string; authorName?: string; text?: string }): Promise<void> {
	const url = window.location.origin + ROUTES.moment(opts.momentId);
	const title = `${opts.authorName ?? 'HotSpot 用户'}的瞬间`;
	try {
		if (navigator.share) {
			await navigator.share({ title, text: opts.text, url });
			return;
		}
		await navigator.clipboard?.writeText(url);
		showToast('瞬间链接已复制到剪贴板', 'success');
	} catch {
		// 用户取消系统分享（AbortError）不是错误，不提示；其余失败按复制兜底。
		try {
			await navigator.clipboard?.writeText(url);
			showToast('瞬间链接已复制到剪贴板', 'success');
		} catch {
			showToast('分享失败，请重试', 'danger');
		}
	}
}
