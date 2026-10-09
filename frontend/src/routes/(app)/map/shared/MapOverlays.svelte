<script lang="ts">
	/**
	 * 地图数据覆盖层（桌面/移动共用）：当前位置点、热力桶徽标、图钉标记、
	 * 加载/错误提示条。两端差异全部走 props：
	 * - myLocation：我的位置（GPS fix，WGS-84）；蓝点按地理坐标投影，随缩放/平移
	 *   钉在同一地理点（语义红线：蓝点=我的位置，不是地图中心）
	 * - dotPulse：移动端脉冲光环（halo scale/opacity 循环），桌面静态
	 * - chipTopClass：提示条顶部偏移（桌面 top-3 / 移动 top-16，给顶栏让位）
	 * 选中/下钻等交互语义由调用方传入（两端选中后的联动不同）。
	 */
	import { type MapBounds, type MapHeatBucket } from './location-shared';
	import type { Pin } from './location-data';
	import MapPinMarker from './MapPinMarker.svelte';
	import MapHeatDot from '@/components/map/MapHeatDot.svelte';
	import MapMyLocationDot from './MapMyLocationDot.svelte';

	interface Props {
		pins: Pin[];
		bounds: MapBounds;
		heatBuckets: MapHeatBucket[];
		selectedPinId: string | number | null;
		loading: boolean;
		error: string | null;
		/** 我的位置（WGS-84 GPS fix）；null = 尚未定位成功，不渲染蓝点 */
		myLocation: { lat: number; lng: number } | null;
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
		myLocation,
		dotPulse,
		chipTopClass,
		onSelectPin,
		onHeatSelect,
		onRetry
	}: Props = $props();
</script>

{#if myLocation}
	<!-- 我的位置蓝点（按地理坐标投影，缩放/平移不漂移；移动端带脉冲光环） -->
	<MapMyLocationDot lat={myLocation.lat} lng={myLocation.lng} {bounds} pulse={dotPulse} />
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
	<div role="alert" class="pointer-events-auto absolute {chipTopClass} left-1/2 -translate-x-1/2 z-20 flex items-center gap-2 rounded-full bg-surface-primary px-3 py-1.5 text-xs text-status-danger shadow-raised-sm border border-border-default">
		<span>{error}</span>
		<button type="button" class="link-btn focus-ring rounded-sm text-accent-default" onclick={onRetry}>重试</button>
	</div>
{/if}
