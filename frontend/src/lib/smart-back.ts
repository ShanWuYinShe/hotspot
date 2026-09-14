import { goto } from '$app/navigation';
import { ROUTES } from '@/lib/routes';

// smartBack — 站内返回：有历史可退时 history.back()，否则 goto fallback。
// 裸 history.back() 在分享链接/深链直达（history.length === 1）时无处可退，
// 会直接退出站点；各页 onBack/操作后返回统一走这里（手势滑动返回除外，
// 那是平台手势语义）。
// 注意不用 document.referrer 判断：SPA 内导航不更新 referrer，用它会把站内
// 流转误判成深链（如列表进详情再返回会被送去首页而非上一页）。
export function smartBack(fallback: string = ROUTES.home): void {
	if (typeof window !== 'undefined' && window.history.length > 1) {
		history.back();
	} else {
		void goto(fallback);
	}
}
