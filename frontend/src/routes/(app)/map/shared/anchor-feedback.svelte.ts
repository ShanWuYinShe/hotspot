// 浏览锚点确认按钮的反馈状态机（桌面/移动共用）。
//
// 设计：点击「将地图中心设为浏览锚点」后，反馈就发生在按钮本体上，
// 不再用右上角 toast 打断操作：
//   idle → pending（1.0s，spinner）→ success（1.4s，✓ 已生效）→ idle（恢复主按钮）
// 失败：pending → error（1.8s，按钮内提示 + 可重试）→ idle。
// 用户确认后的稳定态永远是主按钮（不出现灰色 disabled），
// 「是否已设置」由锚点卡与地图标记表达；中心=锚点时点击 = 重复确认。

export type AnchorFeedbackState = 'idle' | 'pending' | 'success' | 'error';

type Timer = ReturnType<typeof setTimeout>;

export function createAnchorFeedback() {
	let state = $state<AnchorFeedbackState>('idle');
	let timer: Timer | null = null;

	function clearTimer() {
		if (timer) {
			clearTimeout(timer);
			timer = null;
		}
	}

	/**
	 * 执行一次确认：action 返回 false 或抛错 → 进入 error。
	 * 写入本身是同步的（localStorage），pending/success 是纯反馈动画，
	 * 不阻塞地图拖拽/缩放。
	 */
	function run(action: () => boolean) {
		if (state === 'pending') return;
		clearTimer();
		state = 'pending';
		let ok = false;
		try {
			ok = action();
		} catch {
			ok = false;
		}
		timer = setTimeout(() => {
			timer = null;
			state = ok ? 'success' : 'error';
			timer = setTimeout(
				() => {
					timer = null;
					state = 'idle';
				},
				ok ? 1400 : 1800
			);
		}, 1000);
	}

	function reset() {
		clearTimer();
		state = 'idle';
	}

	function destroy() {
		clearTimer();
	}

	return {
		get state() {
			return state;
		},
		run,
		reset,
		destroy
	};
}
