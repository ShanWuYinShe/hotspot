import { describe, it, expect } from 'vitest';
import { buildImageSrcSet, isUnsplashImage, optimizedImageUrl } from '../images';

const UNSPLASH = 'https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=600';

describe('optimizedImageUrl', () => {
	it('unsplash：追加 auto/format/fit/q/w 参数并覆盖已有 w', () => {
		const out = new URL(optimizedImageUrl(UNSPLASH, { width: 800 }));
		expect(out.hostname).toBe('images.unsplash.com');
		expect(out.searchParams.get('auto')).toBe('format');
		expect(out.searchParams.get('fit')).toBe('crop');
		expect(out.searchParams.get('q')).toBe('78');
		expect(out.searchParams.get('w')).toBe('800');
	});

	it('unsplash：覆盖已有 q/fit，保留其它业务参数', () => {
		const url = 'https://images.unsplash.com/photo-x?fit=max&q=20&ixid=abc&w=200';
		const out = new URL(optimizedImageUrl(url, { width: 400, quality: 60, fit: 'crop' }));
		expect(out.searchParams.get('fit')).toBe('crop');
		expect(out.searchParams.get('q')).toBe('60');
		expect(out.searchParams.get('w')).toBe('400');
		expect(out.searchParams.get('ixid')).toBe('abc');
	});

	it('非 unsplash 域名原样返回', () => {
		const url = 'https://example.com/a.png?w=100';
		expect(optimizedImageUrl(url, { width: 800 })).toBe(url);
	});

	it('无效 URL / 空字符串原样返回', () => {
		expect(optimizedImageUrl('', { width: 800 })).toBe('');
		expect(optimizedImageUrl('not a url', { width: 800 })).toBe('not a url');
	});
});

describe('buildImageSrcSet', () => {
	it('unsplash 按宽度生成 400w/800w', () => {
		const out = buildImageSrcSet(UNSPLASH, [400, 800]);
		expect(out).toBeTruthy();
		const parts = out!.split(', ');
		expect(parts).toHaveLength(2);
		const first = parts[0].split(' ')[0];
		const second = parts[1].split(' ')[0];
		expect(new URL(first).searchParams.get('w')).toBe('400');
		expect(new URL(second).searchParams.get('w')).toBe('800');
		expect(parts[0].endsWith(' 400w')).toBe(true);
		expect(parts[1].endsWith(' 800w')).toBe(true);
	});

	it('非 unsplash 返回 undefined', () => {
		expect(buildImageSrcSet('https://example.com/a.png', [400, 800])).toBeUndefined();
	});

	it('空宽度数组返回 undefined', () => {
		expect(buildImageSrcSet(UNSPLASH, [])).toBeUndefined();
	});
});

describe('isUnsplashImage', () => {
	it('识别 unsplash 与其它域名', () => {
		expect(isUnsplashImage(UNSPLASH)).toBe(true);
		expect(isUnsplashImage('https://example.com/x.png')).toBe(false);
		expect(isUnsplashImage('bad')).toBe(false);
	});
});
