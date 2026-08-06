<script lang="ts">
	import type { Pin } from './location-data';
	import { pinTone, getMarkerUrl } from './location-shared';

	// 图钉媒体展示（地图三处共用：桌面面板列表行 / 移动 Sheet 卡 / 桌面选中详情）：
	// 用户 → 头像 + 在线角标；动态 → 图片缩略图；群组/其他 → 类型 marker（pinTone 底色）。

	let {
		pin,
		size = 42,
		markerWidth = 20,
		markerHeight = 27
	}: {
		pin: Pin;
		size?: number;
		markerWidth?: number;
		markerHeight?: number;
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
	<img src={pin.image} alt={pin.title || pin.name || ''} class="rounded-tight object-cover shrink-0" style="width:{size}px;height:{size}px;" />
{:else}
	<div
		class="flex items-center justify-center rounded-small shrink-0"
		style="width:{size}px;height:{size}px;background:{pinTone(pin.type, 12)};"
	>
		<img src={getMarkerUrl(pin.type, pin.count ?? 1)} alt={pin.type} style="width:{markerWidth}px;height:{markerHeight}px;" />
	</div>
{/if}
