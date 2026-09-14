import { goto } from '$app/navigation';
import { ROUTES } from '@/lib/routes';

// smartBack — 站内返回：有站内历史时 history.back()，否则 goto fallback。
// 裸 history.back() 在分享链接/深链直达时无站内历史，会直接退出站点；
// 各页 onBack/操作后返回统一走这里（手势滑动返回除外，那是平台手势语义）。
export function smartBack(fallback: string = ROUTES.home): void {
	if (
		typeof window !== 'undefined' &&
		window.history.length > 1 &&
		document.referrer.startsWith(window.location.origin)
	) {
		history.back();
	} else {
		void goto(fallback);
	}
}
