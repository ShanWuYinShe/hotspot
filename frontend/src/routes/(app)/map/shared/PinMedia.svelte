<script lang="ts">
	import type { Pin } from './location-data';
	import { PIN_TONES, markerSvg } from './location-shared';

	// 图钉媒体展示（地图三处共用：桌面面板列表行 / 移动 Sheet 卡 / 桌面选中详情）：
	// 用户 → 头像 + 在线角标；动态 → 图片缩略图；群组/其他 → 类型 marker（PIN_TONES 淡底）。

	let {
		pin,
		size = 42,
		markerWidth = 20,
		markerHeight = 27,
		selected = false
	}: {
		pin: Pin;
		size?: number;
		markerWidth?: number;
		markerHeight?: number;
		/** 所在列表行处于选中态（accent 背景）时传入：marker 底换实色，
		    避免 type 色与 accent 背景同色隐形（user 色 = accent 色）。 */
		selected?: boolean;
	} = $props();
</script>

{#if pin.type === 'user' && pin.avatar}
	<div class="relative shrink-0">
		<img src={pin.avatar} alt={pin.name || pin.title || ''} class="rounded-full object-cover" style="width:{size}px;height:{size}px;" />
		{#if pin.online}
			<span class="absolute bottom-0 right-0 rounded-full bg-status-online" style="width:10px;height:10px;border:2px solid var(--cst-surface-primary);"></span>
		{/if}
	</div>
{:else if pin.type === 'moment' && pin.image}
	<img src={pin.image} alt={pin.title || pin.name || ''} class="rounded-sm object-cover shrink-0" style="width:{size}px;height:{size}px;" />
{:else}
	<div
		class="flex items-center justify-center rounded-sm shrink-0"
		style="width:{size}px;height:{size}px;background:{selected ? 'var(--cst-surface-primary)' : PIN_TONES[pin.type].bg};"
	>
		
		{@html markerSvg(pin.type, pin.count ?? 1, { w: markerWidth, h: markerHeight })}
	</div>
{/if}
