<script lang="ts">
	// Toggle — iOS 式开关，所有主题与模式下可用。
	// 交互层由 bits-ui Switch 提供：role="switch" + 键盘空格切换 + 焦点管理）。
	// 视觉契约对齐 Figma Make SwitchControl：轨道/滑块/按压反馈全部走语义 token
	// （--surface-toggle-off/thumb、--elevation-toggle-*、--motion-standard）。
	import { Switch } from 'bits-ui';

	let {
		on,
		onToggle,
		label,
		compact = false
	}: {
		on: boolean;
		onToggle: () => void;
		label?: string;
		compact?: boolean;
	} = $props();

	const width = $derived(compact ? 48 : 50);
	const thumbOffset = $derived(compact ? 22 : 23);
</script>

<Switch.Root
	class="hover-btn relative outline-none shrink-0"
	checked={on}
	onCheckedChange={(next) => {
		if (next !== on) onToggle();
	}}
	aria-label={label}
	style="width:{width}px;height:28px;border-radius:14px;background:{on ? 'var(--cst-accent-default)' : 'var(--cst-surface-toggle-off)'};transition:background var(--cst-motion-standard) var(--cst-ease-standard);box-shadow:{on ? 'var(--cst-elevation-toggle-on)' : 'var(--cst-elevation-toggle-off)'};"
>
	<span
		class="absolute"
		style="top:3px;width:22px;height:22px;border-radius:11px;background:var(--cst-surface-toggle-thumb);box-shadow:var(--cst-elevation-toggle-thumb);left:{on ? thumbOffset : 2}px;transition:left var(--cst-motion-standard) var(--cst-ease-standard);"
	></span>
</Switch.Root>
