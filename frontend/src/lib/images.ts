// 图片 URL 管线（Round 6）：统一 Unsplash 参数与 srcset 生成。
// 约束：只处理 images.unsplash.com；其它域名（含外链/本地/无效 URL）原样返回，禁止改写。
// 组件消费时配合 loading/decoding/fetchpriority 与渐显类使用。

const UNSPLASH_HOST = 'images.unsplash.com';
const DEFAULT_QUALITY = 78;

export type ImageOptions = {
	/** 目标显示宽度（px，1x 档） */
	width: number;
	/** 质量，默认 78 */
	quality?: number;
	/** crop：填满裁切；max：等比缩放到不超过 */
	fit?: 'crop' | 'max';
};

/** 是否是 Unsplash CDN 图片（唯一允许追加参数的域名）。 */
export function isUnsplashImage(url: string | null | undefined): boolean {
	if (!url) return false;
	try {
		return new URL(url).hostname === UNSPLASH_HOST;
	} catch {
		return false;
	}
}

/**
 * 生成优化后的图片 URL：
 * - images.unsplash.com：追加/覆盖 auto=format、fit、q、w 参数；
 * - 其它域名或无效 URL：原样返回。
 */
export function optimizedImageUrl(
	url: string | null | undefined,
	{ width, quality = DEFAULT_QUALITY, fit = 'crop' }: ImageOptions
): string {
	if (!url) return '';
	if (!isUnsplashImage(url)) return url;
	try {
		const parsed = new URL(url);
		parsed.searchParams.set('auto', 'format');
		parsed.searchParams.set('fit', fit);
		parsed.searchParams.set('q', String(quality));
		parsed.searchParams.set('w', String(Math.max(1, Math.round(width))));
		return parsed.toString();
	} catch {
		return url;
	}
}

/**
 * 生成 srcset 字符串（1x/2x 档由调用方给出实际宽度数组）。
 * 非 Unsplash 域名返回 undefined（调用方不渲染 srcset，浏览器回退 src）。
 */
export function buildImageSrcSet(
	url: string | null | undefined,
	widths: number[],
	quality = DEFAULT_QUALITY
): string | undefined {
	if (!url || !Array.isArray(widths) || widths.length === 0 || !isUnsplashImage(url)) return undefined;
	const entries = widths
		.filter((w) => Number.isFinite(w) && w > 0)
		.map((w) => `${optimizedImageUrl(url, { width: w, quality })} ${Math.round(w)}w`);
	return entries.length > 0 ? entries.join(', ') : undefined;
}


/** 尺寸档预设（组件按显示尺寸选档，避免散落魔法数）。
 *  widths 为 srcset 候选宽度（1x/2x 语义由浏览器按 DPR 选择）；quality 默认 78，full 档 85。 */
export const IMAGE_PRESETS = {
	thumb: { widths: [200, 400], quality: 78 },
	card: { widths: [400, 800], quality: 78 },
	cover: { widths: [800, 1600], quality: 80 },
	hero: { widths: [1200, 2400], quality: 82 },
	full: { widths: [1600, 2400], quality: 85 }
} as const;

export type ImagePresetName = keyof typeof IMAGE_PRESETS;

/** 头像档：按显示尺寸生成 2x/3x 两档（32→64/96、48→96/144、80→160/240），sizes=显示尺寸。 */
export function avatarImageWidths(displayPx: number): number[] {
	const px = Math.max(16, Math.round(displayPx));
	const two = Math.round(px * 2);
	const three = Math.round(px * 3);
	return px <= 56 ? [two, three] : [two];
}

/** 按预设生成 srcset；非 unsplash 返回 undefined（浏览器回退 src）。 */
export function presetSrcSet(url: string, preset: ImagePresetName): string | undefined {
	const p = IMAGE_PRESETS[preset];
	return buildImageSrcSet(url, [...p.widths], p.quality);
}

/** 按预设生成 1x URL（未传 width 时取预设最小档）。 */
export function presetImageUrl(url: string, preset: ImagePresetName, width?: number): string {
	const p = IMAGE_PRESETS[preset];
	return optimizedImageUrl(url, { width: width ?? p.widths[0], quality: p.quality });
}

/** 图片渐显 action：加载完成前 opacity-0，onload 后 opacity-1（0.22s，reduced-motion 由全局规则压为瞬时）。
 *  对本地/非 unsplash 图片同样生效；若图片已在缓存（complete）立即显示。 */
export function fadeImage(node: HTMLImageElement) {
	node.classList.add('img-fade');
	const show = () => node.classList.add('img-loaded');
	if (node.complete && node.naturalWidth > 0) show();
	node.addEventListener('load', show);
	return {
		destroy() {
			node.removeEventListener('load', show);
		}
	};
}
