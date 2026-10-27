/**
 * ImageUrlUtil —— 图片 URL 优化（R6，与 web `lib/images.ts` 同规则）。
 *
 * 仅对 images.unsplash.com 追加/覆盖尺寸参数：auto=format&fit=crop&q=<quality>&w=<width>；
 * 其他域名（含本地代理/头像接口）原样返回，禁止破坏外链。
 * 调用顺序：MediaUrlUtil.normalize(url) 先做相对路径/本地域名改写，再 optimized(...)。
 */
export class ImageUrlUtil {
  private static readonly UNSPLASH_HOST: string = 'images.unsplash.com';
  private static readonly DEFAULT_QUALITY: number = 78;

  /** 仅 unsplash 追加/覆盖 width/quality/fit/auto；已有同名参数覆盖，保留其他 query 与 hash */
  static optimized(url: string | null | undefined, width: number, quality: number = ImageUrlUtil.DEFAULT_QUALITY): string {
    if (typeof url !== 'string') {
      return '';
    }
    const trimmed: string = url.trim();
    if (!trimmed) {
      return '';
    }
    const host: string = ImageUrlUtil.extractHost(trimmed);
    if (host !== ImageUrlUtil.UNSPLASH_HOST) {
      return trimmed;
    }
    const hashIndex: number = trimmed.indexOf('#');
    const fragment: string = hashIndex >= 0 ? trimmed.substring(hashIndex) : '';
    const withoutHash: string = hashIndex >= 0 ? trimmed.substring(0, hashIndex) : trimmed;
    const queryIndex: number = withoutHash.indexOf('?');
    const basePath: string = queryIndex >= 0 ? withoutHash.substring(0, queryIndex) : withoutHash;
    const rawQuery: string = queryIndex >= 0 ? withoutHash.substring(queryIndex + 1) : '';
    const kept: string[] = [];
    if (rawQuery) {
      const pairs: string[] = rawQuery.split('&');
      for (let i = 0; i < pairs.length; i++) {
        const pair: string = pairs[i];
        if (!pair) {
          continue;
        }
        const eq: number = pair.indexOf('=');
        const key: string = (eq >= 0 ? pair.substring(0, eq) : pair).trim().toLowerCase();
        if (key === 'w' || key === 'q' || key === 'fit' || key === 'auto') {
          continue;
        }
        kept.push(pair);
      }
    }
    const safeWidth: number = Math.max(1, Math.round(width));
    const safeQuality: number = Math.max(1, Math.min(100, Math.round(quality)));
    kept.push(`auto=format`);
    kept.push(`fit=crop`);
    kept.push(`q=${safeQuality}`);
    kept.push(`w=${safeWidth}`);
    return `${basePath}?${kept.join('&')}${fragment}`;
  }

  static isUnsplash(url: string | null | undefined): boolean {
    if (typeof url !== 'string' || !url.trim()) {
      return false;
    }
    return ImageUrlUtil.extractHost(url.trim()) === ImageUrlUtil.UNSPLASH_HOST;
  }

  private static extractHost(url: string): string {
    const protocolIndex: number = url.indexOf('://');
    if (protocolIndex < 0) {
      return '';
    }
    const hostStart: number = protocolIndex + 3;
    const slashIndex: number = url.indexOf('/', hostStart);
    const hostPort: string = (slashIndex >= 0 ? url.substring(hostStart, slashIndex) : url.substring(hostStart)).toLowerCase();
    return hostPort.split(':')[0];
  }
}
