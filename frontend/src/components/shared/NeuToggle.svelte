<script lang="ts">
	// iOS 式开关——所有主题与模式下可用（迁移自旧版 SettingsPage 内 NeuToggle，
	// 交互层由 bits-ui Switch 提供：role="switch" + 键盘空格切换 + 焦点管理）。
	// OFF: 中性灰轨道 + 白色拇指（始终可见）
	// ON:  accent-primary 轨道 + 白色拇指
	import { Switch } from 'bits-ui';

	let { on, onToggle }: { on: boolean; onToggle: () => void } = $props();
</script>

<Switch.Root
	class="neu-hover-btn relative outline-none flex-shrink-0"
	checked={on}
	onCheckedChange={(next) => {
		if (next !== on) onToggle();
	}}
	style="width:50px;height:28px;border-radius:14px;background:{on ? 'var(--neu-accent-primary)' : 'rgba(120, 120, 128, 0.22)'};transition:background 0.25s cubic-bezier(0.4,0,0.2,1), filter 0.18s ease;box-shadow:{on ? 'inset 0 1px 2px rgba(0,0,0,0.12)' : 'inset 0 1px 2px rgba(0,0,0,0.08), 0 0 0 1px rgba(120,120,128,0.15)'};"
>
	<span
		class="absolute bg-[var(--neu-bg)]"
		style="top:3px;width:22px;height:22px;border-radius:11px;box-shadow:0 1px 4px rgba(0,0,0,0.26), 0 0.5px 1px rgba(0,0,0,0.14);left:{on ? 25 : 2}px;transition:left 0.25s cubic-bezier(0.4,0,0.2,1);"
	></span>
</Switch.Root>
