// 菜单/弹窗/瞬态面板的 Escape + 外部点击关闭生命周期（迁移自 React useDismissibleLayer）。
// 组件内用 $effect 挂载：
//   $effect(() => attachDismissibleLayer({ open, onDismiss, root, closeOnOutsidePress }));
// $effect 返回 cleanup 即自动卸载。

export type DismissibleOptions = {
	open: boolean;
	onDismiss: () => void;
	root?: HTMLElement | null;
	closeOnOutsidePress?: boolean;
};

export function attachDismissibleLayer({
	open,
	onDismiss,
	root = null,
	closeOnOutsidePress = false
}: DismissibleOptions): () => void {
	if (!open) return () => {};

	const onKeyDown = (event: KeyboardEvent) => {
		if (event.key === 'Escape') onDismiss();
	};
	const onPointerDown = (event: PointerEvent) => {
		if (closeOnOutsidePress && root && !root.contains(event.target as Node)) onDismiss();
	};
	window.addEventListener('keydown', onKeyDown);
	if (closeOnOutsidePress) window.addEventListener('pointerdown', onPointerDown);
	return () => {
		window.removeEventListener('keydown', onKeyDown);
		if (closeOnOutsidePress) window.removeEventListener('pointerdown', onPointerDown);
	};
}
