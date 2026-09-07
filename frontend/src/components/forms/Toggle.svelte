<script lang="ts">
	// Toggle — iOS 式开关，所有主题与模式下可用。
	// 交互层由 bits-ui Switch 提供：role="switch" + 键盘空格切换 + 焦点管理。
	// 视觉契约对齐 Figma Make SwitchControl：轨道/滑块/按压反馈全部走语义 token
	// （--surface-toggle-off/thumb、--elevation-toggle-*、--motion-standard）。
	// 状态色必须类化（inline background 会压制 hover/active 反馈）：
	//   hover 差异化（M3 switch 规范）：on → 轨道 accent 光晕抬档；off → 滑块放大预览。
	// 根按钮透明 padding 扩展热区至 ≥44px（p-2 -m-2 不占布局流）。
	import { Switch } from 'bits-ui';

	let {
		on,
		onToggle,
		label,
		disabled = false,
		compact = false
	}: {
		on: boolean;
		onToggle: () => void;
		label?: string;
		disabled?: boolean;
		compact?: boolean;
	} = $props();

	const width = $derived(compact ? 48 : 50);
	const thumbOffset = $derived(compact ? 22 : 23);
</script>

<Switch.Root
	class="group hover-btn focus-ring relative -m-2 shrink-0 rounded-full bg-transparent p-2
		disabled:cursor-not-allowed disabled:opacity-faint"
	checked={on}
	{disabled}
	onCheckedChange={(next) => {
		if (next !== on) onToggle();
	}}
	aria-label={label}
>
	<span
		class="relative block rounded-full transition-[background-color,box-shadow] duration-standard ease-standard
			{on
			? 'bg-accent-default shadow-toggle-on group-hover:shadow-raised-sm'
			: 'bg-surface-toggle-off shadow-toggle-off'}"
		style="width:{width}px;height:28px;"
	>
		<span
			class="absolute rounded-full bg-surface-toggle-thumb shadow-toggle-thumb
				transition-[left,transform] duration-standard ease-bounce
				group-active:scale-90 {on ? '' : 'group-hover:scale-110'}"
			style="top:3px;width:22px;height:22px;left:{on ? thumbOffset : 2}px;"
		></span>
	</span>
</Switch.Root>
