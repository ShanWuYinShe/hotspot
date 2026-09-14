<script lang="ts">
	import { Plus, Minus } from '@lucide/svelte';
	import { MAP_MIN_ZOOM, MAP_MAX_ZOOM } from './map-projection';

	// 地图缩放控件（+/− 两连钮）：桌面/移动共用，位置与容器外观由调用方经
	// class/style 透传，保持两端视觉不变。
	// 分段圆角由按钮自身承担，不用 overflow-hidden 裁剪（会连带裁掉键盘焦点环）。
	// 按钮不用 hover-btn：分段组合控件中点击 0.94 缩放会让高亮块与整卡脱开，
	// hover/active 填充由全局 button 规则承载。

	let {
		zoom = $bindable(),
		class: className = '',
		style = ''
	}: { zoom: number; class?: string; style?: string } = $props();
</script>

<div class="flex flex-col rounded-lg {className}" {style}>
	<button
		type="button"
		class="flex items-center justify-center size-9 rounded-t-lg focus-ring text-text-primary disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-faint"
		style="border-bottom:var(--cst-border-width-hairline) solid var(--cst-border-default);"
		aria-label="放大"
		disabled={zoom >= MAP_MAX_ZOOM}
		onclick={() => (zoom = Math.min(MAP_MAX_ZOOM, zoom + 1))}
	>
		<Plus class="size-4" strokeWidth={1.75} />
	</button>
	<button
		type="button"
		class="flex items-center justify-center size-9 rounded-b-lg focus-ring text-text-primary disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-faint"
		aria-label="缩小"
		disabled={zoom <= MAP_MIN_ZOOM}
		onclick={() => (zoom = Math.max(MAP_MIN_ZOOM, zoom - 1))}
	>
		<Minus class="size-4" strokeWidth={1.75} />
	</button>
</div>
