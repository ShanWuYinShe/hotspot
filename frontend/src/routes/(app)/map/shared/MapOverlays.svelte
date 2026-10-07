<script lang="ts">
	/**
	 * 地图数据覆盖层（桌面/移动共用）：当前位置点、热力桶徽标、图钉标记、
	 * 加载/错误提示条。两端差异全部走 props：
	 * - dotTop：当前位置点垂直位置（桌面 50% / 移动 46%，Figma Make 对齐）
	 * - dotPulse：移动端脉冲光环（halo scale/opacity 循环），桌面静态
	 * - chipTopClass：提示条顶部偏移（桌面 top-3 / 移动 top-16，给顶栏让位）
	 * 选中/下钻等交互语义由调用方传入（两端选中后的联动不同）。
	 */
	import { PIN_COLORS, PIN_TONES, type MapBounds, type MapHeatBucket } from './location-shared';
	import type { Pin } from './location-data';
	import MapPinMarker from './MapPinMarker.svelte';
	import MapHeatDot from '@/components/map/MapHeatDot.svelte';

	interface Props {
		pins: Pin[];
		bounds: MapBounds;
		heatBuckets: MapHeatBucket[];
		selectedPinId: string | number | null;
		loading: boolean;
		error: string | null;
		showDot: boolean;
		dotTop: string;
		dotPulse: boolean;
		chipTopClass: string;
		onSelectPin: (pin: Pin, e: MouseEvent) => void;
		onHeatSelect: (bucket: MapHeatBucket) => void;
		onRetry: () => void;
	}

	let {
		pins,
		bounds,
		heatBuckets,
		selectedPinId,
		loading,
		error,
		showDot,
		dotTop,
		dotPulse,
		chipTopClass,
		onSelectPin,
		onHeatSelect,
		onRetry
	}: Props = $props();
</script>

{#if showDot}
	<!-- Current location dot（移动端带脉冲光环：halo scale 1→1.8 / opacity 0.55→0 循环） -->
	<div style="position:absolute;left:50%;top:{dotTop};transform:translate(-50%,-50%);pointer-events:none;z-index:5;">
		<div class="relative rounded-full" style="width:18px;height:18px;background:{PIN_COLORS.user};border:var(--cst-border-width-heavy) solid var(--cst-surface-primary);box-shadow:var(--cst-marker-shadow) {PIN_TONES.user.glow};">
			<span class="{dotPulse ? 'map-pulse-halo' : ''} absolute rounded-full" style="inset:{dotPulse ? '-7px' : '-8px'};background:{PIN_TONES.user.bg};border:var(--cst-border-width-hairline) solid {PIN_TONES.user.border};"></span>
		</div>
	</div>
{/if}

<!-- Heatmap buckets（大视口 / 高密度时服务端返回热力统计；点徽标下钻到整格明细） -->
{#each heatBuckets as bucket (bucket.cell_token)}
	<MapHeatDot {bucket} {bounds} onSelect={onHeatSelect} />
{/each}

<!-- Pin markers -->
{#each pins as pin (pin.id)}
	<MapPinMarker
		{pin}
		{bounds}
		isSelected={selectedPinId === pin.id}
		onSelect={(e) => onSelectPin(pin, e)}
	/>
{/each}

{#if loading}
	<!-- 加载中的轻量指示 -->
	<div role="status" aria-live="polite" class="absolute {chipTopClass} left-1/2 -translate-x-1/2 z-20 rounded-full bg-surface-primary px-3 py-1 text-xs text-text-secondary shadow-raised-sm border border-border-default">
		加载地图数据…
	</div>
{:else if error}
	<!-- 加载失败：给出原因与重试，避免空白地图被误认为「附近什么都没有」 -->
	<div role="alert" class="absolute {chipTopClass} left-1/2 -translate-x-1/2 z-20 flex items-center gap-2 rounded-full bg-surface-primary px-3 py-1.5 text-xs text-status-danger shadow-raised-sm border border-border-default">
		<span>{error}</span>
		<button type="button" class="link-btn focus-ring rounded-sm text-accent-default" onclick={onRetry}>重试</button>
	</div>
{/if}
