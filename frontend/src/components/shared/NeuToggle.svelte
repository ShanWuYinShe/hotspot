<script lang="ts">
	// iOS 式开关——所有主题与模式下可用（迁移自 React SettingsPage.tsx 内 NeuToggle）。
	// OFF: 中性灰轨道 + 白色拇指（始终可见）
	// ON:  accent-primary 轨道 + 白色拇指

	let { on, onToggle }: { on: boolean; onToggle: () => void } = $props();
</script>

<button
	type="button"
	class="neu-hover-btn relative outline-none flex-shrink-0"
	role="switch"
	aria-checked={on}
	style="width:50px;height:28px;border-radius:14px;background:{on ? 'var(--neu-accent-primary)' : 'rgba(120, 120, 128, 0.22)'};transition:background 0.25s cubic-bezier(0.4,0,0.2,1);box-shadow:{on ? 'inset 0 1px 2px rgba(0,0,0,0.12)' : 'inset 0 1px 2px rgba(0,0,0,0.08), 0 0 0 1px rgba(120,120,128,0.15)'};"
	onclick={onToggle}
>
	<!-- Thumb — always white, always has a visible shadow -->
	<span
		class="absolute bg-[var(--neu-bg)]"
		style="top:3px;width:22px;height:22px;border-radius:11px;box-shadow:0 1px 4px rgba(0,0,0,0.26), 0 0.5px 1px rgba(0,0,0,0.14);left:{on ? 25 : 2}px;transition:left 0.25s cubic-bezier(0.4,0,0.2,1);"
	></span>
</button>
