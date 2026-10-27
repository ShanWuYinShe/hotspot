// 会话列表行级手势：移动端长按 500ms / 桌面右键 → 打开该行的会话操作菜单。
// 目的（独立设计评审 P2-24）：把「置顶/免打扰/删除」从每行常驻 ⋯ 收成一个菜单入口，
// 移动端列表不再 5 行 5 个 ⋯；桌面端仍由行内 hover 显现的 ⋯ 承担，右键为补充。
//
// 实现要点：
//   - 只用**外层包装**监听（挂在 ListContainer 之外，避免在 role=list 与 listitem
//     之间插入无语义节点触发 axe aria-required-children）；
//   - 通过 [role="listitem"] 定位行，再按兄弟顺序换取 chats 下标；
//   - 长按触发后吞掉随后的 click（capture 阶段），避免误触打开会话；
//   - 滚动（位移 > 10px）或抬手取消，不阻塞列表滚动。

export interface RowLongPressOptions {
	/** 行数（chats 顺序与 listitem 顺序一致） */
	count: number;
	/** 命中行后回调（入参为 chats 下标） */
	onMore: (index: number) => void;
	/** 长按阈值，默认 500ms */
	delay?: number;
}

const MOVE_TOLERANCE = 10;

export function rowLongPress(node: HTMLElement, initial: RowLongPressOptions) {
	let opts = initial;
	let timer: ReturnType<typeof setTimeout> | null = null;
	let fired = false;
	let startX = 0;
	let startY = 0;

	const rowIndexOf = (target: EventTarget | null): number => {
		const row = (target as HTMLElement | null)?.closest?.('[role="listitem"]');
		if (!row?.parentElement) return -1;
		const rows = Array.from(row.parentElement.children).filter(
			(el) => el.getAttribute('role') === 'listitem'
		);
		const idx = rows.indexOf(row);
		return idx >= 0 && idx < opts.count ? idx : -1;
	};

	const clear = () => {
		if (timer) {
			clearTimeout(timer);
			timer = null;
		}
	};

	const onPointerDown = (e: PointerEvent) => {
		fired = false;
		// 鼠标（含触控笔 hover 场景）走 hover/右键路径，不启用长按
		if (e.pointerType === 'mouse') return;
		const idx = rowIndexOf(e.target);
		if (idx < 0) return;
		startX = e.clientX;
		startY = e.clientY;
		clear();
		timer = setTimeout(() => {
			timer = null;
			fired = true;
			opts.onMore(idx);
		}, opts.delay ?? 500);
	};
	const onPointerMove = (e: PointerEvent) => {
		if (!timer) return;
		if (Math.abs(e.clientX - startX) > MOVE_TOLERANCE || Math.abs(e.clientY - startY) > MOVE_TOLERANCE) {
			clear();
		}
	};
	const onPointerEnd = () => clear();
	const onClickCapture = (e: Event) => {
		if (!fired) return;
		fired = false;
		e.preventDefault();
		e.stopPropagation();
	};
	const onContextMenu = (e: MouseEvent) => {
		const idx = rowIndexOf(e.target);
		if (idx < 0) return;
		e.preventDefault();
		opts.onMore(idx);
	};

	node.addEventListener('pointerdown', onPointerDown as EventListener, { passive: true });
	node.addEventListener('pointermove', onPointerMove as EventListener, { passive: true });
	node.addEventListener('pointerup', onPointerEnd);
	node.addEventListener('pointercancel', onPointerEnd);
	node.addEventListener('pointerleave', onPointerEnd);
	node.addEventListener('click', onClickCapture, true);
	node.addEventListener('contextmenu', onContextMenu);

	return {
		update(next: RowLongPressOptions) {
			opts = next;
		},
		destroy() {
			clear();
			node.removeEventListener('pointerdown', onPointerDown as EventListener);
			node.removeEventListener('pointermove', onPointerMove as EventListener);
			node.removeEventListener('pointerup', onPointerEnd);
			node.removeEventListener('pointercancel', onPointerEnd);
			node.removeEventListener('pointerleave', onPointerEnd);
			node.removeEventListener('click', onClickCapture, true);
			node.removeEventListener('contextmenu', onContextMenu);
		}
	};
}
