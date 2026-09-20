// 真实位置上报（spec api_spec_user.md → users/update_location）。
//
// 与「浏览锚点」是两个概念，勿混：
//   - 真实位置（本模块）：我在哪 —— 服务端 users.latitude/longitude/s2_cell_id，
//     用于被别人发现、展示大致位置，属隐私数据；
//   - 浏览锚点（map-auto-locate）：我在看哪 —— 仅作为请求参数 reference_point，
//     影响距离计算基准，不写回服务端用户行。
//
// 上报策略（与用户确认）：首次定位成功即上报。同一 ~110m 网格内不重复上报
// （省请求、避免与后端 S2 写入放大）；失败静默（上报属后台行为，不该用错误提示
// 打断交互），下次定位成功自动重试；未登录不发请求（/api/v1 需要身份）。

import { updateLocation } from '@/api';
import { loadDefaultLocation } from '@/lib/default-location';

/** 上报过的最小网格键；同格不重复上报。 */
const GRID_KEY = 'hotspot:location-reported';
/** 网格粒度：与 nearby 进度 key 同口径（3 位小数 ≈ 110m）。 */
const GRID_DECIMALS = 3;
/**
 * 名称兜底文案。契约（spec api_spec_types.md → Location）：location_name 必填，
 * 「无名称时使用 ClientConfig.defaults.default_location.location_name」——
 * 故正常路径回退默认位置名，仅当配置也拿不到时才用本文案。
 */
const FALLBACK_NAME = '未知位置';

export interface ReportedFix {
	lat: number;
	lng: number;
	/** 位置名（可选）：调用方已知名称时给出；纯 GPS 定位时无名称。 */
	name?: string;
}

function gridKey(lat: number, lng: number): string {
	return `${lat.toFixed(GRID_DECIMALS)},${lng.toFixed(GRID_DECIMALS)}`;
}

function readReported(): string | null {
	try {
		return localStorage.getItem(GRID_KEY);
	} catch {
		return null;
	}
}

function writeReported(key: string): void {
	try {
		localStorage.setItem(GRID_KEY, key);
	} catch {
		/* 存储不可用不影响上报本身 */
	}
}

/**
 * 上报当前位置。命中同一网格时跳过（返回 false = 未发请求）。
 * 全程不抛错：调用方无需 try/catch，也不应因此改变界面状态。
 * @param isLoggedIn 未登录直接跳过（该端点在 /api/v1，需要身份）
 */
export async function reportLocation(fix: ReportedFix, isLoggedIn: boolean): Promise<boolean> {
	if (!isLoggedIn) return false;
	const { lat, lng } = fix;
	if (!Number.isFinite(lat) || !Number.isFinite(lng)) return false;
	if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return false;

	const key = gridKey(lat, lng);
	if (readReported() === key) return false;

	// 名称优先级：调用方给定 > 默认位置名（契约要求）> 兜底文案。
	// loadDefaultLocation 是模块级缓存，命中时无网络开销。
	let name = fix.name?.trim() ?? '';
	if (!name) {
		try {
			name = (await loadDefaultLocation())?.location_name ?? '';
		} catch {
			/* 配置不可用则走兜底文案 */
		}
	}

	try {
		const res = await updateLocation({
			location: {
				point: { latitude: lat, longitude: lng },
				location_name: name || FALLBACK_NAME
			}
		});
		if (res.code !== 0) return false;
		writeReported(key);
		return true;
	} catch {
		return false;
	}
}

/** 测试/调试用：清除上报去重标记。 */
export function resetReportedLocation(): void {
	try {
		localStorage.removeItem(GRID_KEY);
	} catch {
		/* ignore */
	}
}
