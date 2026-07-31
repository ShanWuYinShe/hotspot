// 地图共享逻辑（迁移自 React location.shared.tsx 的纯 TS 部分）。

import type { PinType } from './location-data';
import markerUserUrl from '@/assets/markers/marker_user.svg';
import markerGroupUrl from '@/assets/markers/marker_group.svg';
import markerMomentUrl from '@/assets/markers/marker_moment.svg';
import markerUserClusterUrl from '@/assets/markers/marker_user_cluster.svg';
import markerGroupClusterUrl from '@/assets/markers/marker_group_cluster.svg';
import markerMomentClusterUrl from '@/assets/markers/marker_moment_cluster.svg';

export {
	markerUserUrl,
	markerGroupUrl,
	markerMomentUrl,
	markerUserClusterUrl,
	markerGroupClusterUrl,
	markerMomentClusterUrl
};

export const SNAP_MIN = 148;
export const SNAP_MID = 318;
export const SNAP_MAX = 540;
export type SnapState = 'min' | 'mid' | 'max';
export const SNAP_HEIGHTS: Record<SnapState, number> = { min: SNAP_MIN, mid: SNAP_MID, max: SNAP_MAX };

export const PIN_COLORS: Record<PinType, string> = {
	user: 'var(--map-user)',
	group: 'var(--map-group)',
	post: 'var(--map-post)'
};
export const PIN_LABELS: Record<PinType, string> = { user: '用户', group: '群组', post: '动态' };
export const pinTone = (type: PinType, amount: number) =>
	`color-mix(in srgb, ${PIN_COLORS[type]} ${amount}%, transparent)`;
export const floatBtnStyle =
	'background:var(--neu-bg);border:1px solid var(--neu-border);box-shadow:0 2px 12px var(--neu-shadow-dark), 0 1px 3px var(--neu-shadow-dark);';

export interface MapBounds {
	minLat: number;
	maxLat: number;
	minLng: number;
	maxLng: number;
}

export function getMarkerUrl(type: PinType, count: number) {
	if (type === 'user') return count >= 3 ? markerUserClusterUrl : markerUserUrl;
	if (type === 'group') return count >= 3 ? markerGroupClusterUrl : markerGroupUrl;
	return count >= 3 ? markerMomentClusterUrl : markerMomentUrl;
}

export function buildMapData(center: { lat: number; lng: number }, zoom: number) {
	const bounds: MapBounds = {
		minLat: center.lat - zoom,
		maxLat: center.lat + zoom,
		minLng: center.lng - zoom,
		maxLng: center.lng + zoom
	};
	return {
		url: `https://www.openstreetmap.org/export/embed.html?bbox=${bounds.minLng},${bounds.minLat},${bounds.maxLng},${bounds.maxLat}&layer=mapnik`,
		bounds
	};
}
