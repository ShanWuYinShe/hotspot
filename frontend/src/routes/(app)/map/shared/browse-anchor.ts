// 浏览锚点（Browse Anchor）纯逻辑：地图页「设为浏览锚点」的语义与文案。
//
// 语义红线：浏览锚点 = 附近页距离/推荐所用的基准位置（localStorage 持久化），
// 它**不等于**地图当前视口中心；用户可在“未设置”时回退服务端默认位置。
// 地图页必须让两者同时可见，避免“拖到别处一点按钮就悄悄改了基准”的误解。
import type { MapPoint } from '@/lib/map-auto-locate';
import { MY_LOCATION_NAME, UNNAMED_LOCATION_NAME } from './location-name';

/** 判定「地图中心已≈锚点」的容差：拖拽/逆地理抖动不应让 CTA 在两种状态间跳。 */
export const ANCHOR_SAME_TOLERANCE_METERS = 50;

const EARTH_RADIUS_M = 6_371_000;

function toRadians(deg: number): number {
	return (deg * Math.PI) / 180;
}

/** Haversine 距离（米）：锚点/中心都是 WGS-84 近似坐标，城市尺度误差可忽略。 */
export function anchorDistanceMeters(a: { lat: number; lng: number }, b: { lat: number; lng: number }): number {
	const dLat = toRadians(b.lat - a.lat);
	const dLng = toRadians(b.lng - a.lng);
	const lat1 = toRadians(a.lat);
	const lat2 = toRadians(b.lat);
	const h = Math.sin(dLat / 2) ** 2 + Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLng / 2) ** 2;
	return 2 * EARTH_RADIUS_M * Math.asin(Math.min(1, Math.sqrt(h)));
}

/** 中心与锚点是否视为同一位置（用于 CTA 的“已是当前锚点”态）。 */
export function isSameAnchorPoint(
	center: { lat: number; lng: number } | null,
	anchor: MapPoint | null,
	toleranceMeters = ANCHOR_SAME_TOLERANCE_METERS
): boolean {
	if (!center || !anchor) return false;
	return anchorDistanceMeters(center, anchor) <= toleranceMeters;
}

/**
 * 确认写入锚点时的名称：优先选中地点名，其次当前中心地名。
 * 「未命名区域 / 我的位置」不是可持久化的地点名（前者信息缺失、后者会冻结时间语义），
 * 返回 undefined，锚点展示时回退「自定义位置」。
 */
export function resolveAnchorName(
	selectedName?: string | null,
	centerName?: string | null
): string | undefined {
	const selected = selectedName?.trim();
	if (selected) return selected;
	const center = centerName?.trim();
	if (!center || center === UNNAMED_LOCATION_NAME || center === MY_LOCATION_NAME) return undefined;
	return center;
}

/** 锚点展示名：历史锚点可能没有名称（纯拖拽写入），统一回退到「自定义位置」。 */
export function anchorDisplayName(anchor: MapPoint | null): string {
	const name = anchor?.name?.trim();
	return name || '自定义位置';
}

/**
 * 底部主 CTA 文案：明确「以什么为锚点」，而不是含糊的“确认”。
 * 注意：中心已是锚点时也保持主按钮文案（不出现「当前浏览锚点」disabled 态）——
 * 用户随时可以重复确认/更新，是否已设置由锚点卡与地图标记表达。
 */
export function anchorActionLabel(selectedLocationName?: string | null): string {
	if (selectedLocationName?.trim()) return '以此地点为浏览锚点';
	return '将地图中心设为浏览锚点';
}

/** 锚点用途提示（首次设置/未设置时展示，说明为什么值得点一下）。 */
export function anchorPurposeHint(hasAnchor: boolean): string {
	return hasAnchor ? '附近页以此处为距离与推荐基准' : '设置后，附近页按这里计算距离与推荐';
}
