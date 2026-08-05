import type { Component } from 'svelte';
import { Coffee, Music, Users, Star } from '@lucide/svelte';
import type { TrendingBundle } from '@/api/types';

// 首页/发现页共用的「附近活动」行数据：把 trending 热点动态映射为列表行填充
// （图标/配色按顺序轮换，与 Figma HomePage 活动列表契约一致）。

export interface Activity {
	id: number;
	icon: Component;
	title: string;
	sub: string;
	color: string;
	bg: string;
}

const ICONS = [Coffee, Music, Users, Star] as const;
const COLORS = [
	'var(--status-warning)',
	'var(--accent-default)',
	'var(--marker-group)',
	'var(--status-danger)'
] as const;
const BGS = [
	'color-mix(in srgb, var(--status-warning) 10%, transparent)',
	'var(--accent-soft)',
	'color-mix(in srgb, var(--marker-group) 10%, transparent)',
	'color-mix(in srgb, var(--status-danger) 10%, transparent)'
] as const;

export function toActivities(trending: TrendingBundle | null): Activity[] {
	if (!trending) return [];
	return (trending.hot_moments || []).map((a, i) => ({
		id: i + 1,
		icon: ICONS[i % 4],
		title: a.description ?? '',
		sub: a.location?.location_name ?? '',
		color: COLORS[i % 4],
		bg: BGS[i % 4]
	}));
}
