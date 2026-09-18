import { createShare } from '@/api';
import { ROUTES } from '@/lib/routes';

// 分享链接复制 —— share/create（spec api_spec_share.md）+ 剪贴板复制 + 降级链的
// 统一封装（用户主页 / 动态详情等入口共用，避免各页手写同一套 try/catch）。
//
// 流程：share/create 生成分享码 → 复制 /share/[code] 落地页链接；
// createShare 失败（未登录/网络异常/资源不可访问）时降级为复制页面原始链接，
// 保持「分享」操作在离线/匿名场景下依然可用。

export type ShareTarget =
	| { type: 'user'; id: string }
	| { type: 'group'; id: string }
	| { type: 'moment'; id: string };

export type ShareCopyResult = 'shared' | 'copied' | 'failed';

function fallbackUrl(target: ShareTarget): string {
	const route =
		target.type === 'user' ? ROUTES.user(target.id) : target.type === 'group' ? ROUTES.group(target.id) : ROUTES.moment(target.id);
	return `${window.location.origin}${route}`;
}

async function writeClipboard(text: string): Promise<void> {
	if (!navigator.clipboard?.writeText) throw new Error('Clipboard API unavailable');
	await navigator.clipboard.writeText(text);
}

export async function copyShareLink(target: ShareTarget): Promise<ShareCopyResult> {
	// 空 id 直接失败（路由构造无意义）
	if (!target.id) return 'failed';
	try {
		const payload =
			target.type === 'user'
				? ({ shared_user_id: target.id } as const)
				: target.type === 'group'
					? ({ shared_group_id: target.id } as const)
					: ({ shared_moment_id: target.id } as const);
		const res = await createShare({ share_type: target.type, payload, expires_in_seconds: null });
		if (res.code !== 0 || !res.data?.share_code) throw new Error(res.message || 'share/create failed');
		await writeClipboard(`${window.location.origin}/share/${res.data.share_code}`);
		return 'shared';
	} catch {
		// 降级：分享码创建失败时回退为复制页面原始链接（既有行为）
		try {
			await writeClipboard(fallbackUrl(target));
			return 'copied';
		} catch {
			return 'failed';
		}
	}
}
