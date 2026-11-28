import { peekBootstrap } from '@/lib/bootstrap-cache';

/**
 * 瞬间正文长度上限：spec AppConfig.limits.moment_description_max_length（可配置）。
 * 权威来源 client_config（bootstrap 已缓存时同步可取），兜底常量必须与后端默认一致
 * （platform/appconfig/source.go，2000）。此前 Web 发布页硬编码 280，与
 * Android（读 limits）/后端（2000）不一致：同一段文字在 Web 被静默截断。
 */
export const FALLBACK_MOMENT_TEXT_LIMIT = 2000;

export function resolveMomentTextLimit(injected?: number | null): number {
	const configured = injected ?? peekBootstrap()?.client_config?.limits?.moment_description_max_length ?? null;
	return configured && configured > 0 ? configured : FALLBACK_MOMENT_TEXT_LIMIT;
}

/** 按上限截断（发布页追加话题/心情片段与输入时共用，避免各处各写 slice(0, 280)）。 */
export function clampMomentText(text: string, injected?: number | null): string {
	return text.slice(0, resolveMomentTextLimit(injected));
}
