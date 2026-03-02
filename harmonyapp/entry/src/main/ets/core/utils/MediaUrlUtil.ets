import { API_BASE_URL } from '../../constants/AppConstants';

const LOCAL_MEDIA_HOSTS: string[] = ['localhost', '127.0.0.1', 'hotspot_minio', 'minio'];

interface ParsedAbsoluteUrl {
  host: string;
  pathAndQuery: string;
}

export class MediaUrlUtil {
  private static readonly DEFAULT_ORIGIN: string = MediaUrlUtil.extractOrigin(API_BASE_URL);

  static normalize(url: string | null | undefined, preferredOrigin?: string): string {
    if (typeof url !== 'string') {
      return '';
    }
    const trimmed = url.trim();
    if (!trimmed) {
      return '';
    }

    const origin = MediaUrlUtil.normalizeOrigin(preferredOrigin) || MediaUrlUtil.DEFAULT_ORIGIN;
    if (!origin) {
      return trimmed;
    }

    if (trimmed.startsWith('/s3/')) {
      return `${origin}${trimmed}`;
    }

    if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
      return MediaUrlUtil.rewriteLocalAbsoluteUrl(trimmed, origin);
    }

    return trimmed;
  }

  static extractOrigin(fullUrl: string): string {
    const normalized = (fullUrl || '').trim();
    const protocolIndex = normalized.indexOf('://');
    if (protocolIndex < 0) {
      return '';
    }
    const hostStart = protocolIndex + 3;
    const pathStart = normalized.indexOf('/', hostStart);
    const origin = pathStart > 0 ? normalized.substring(0, pathStart) : normalized;
    return MediaUrlUtil.normalizeOrigin(origin);
  }

  private static normalizeOrigin(origin?: string): string {
    if (!origin) {
      return '';
    }
    const trimmed = origin.trim();
    if (!trimmed) {
      return '';
    }
    return trimmed.endsWith('/') ? trimmed.substring(0, trimmed.length - 1) : trimmed;
  }

  private static parseAbsoluteUrl(url: string): ParsedAbsoluteUrl | null {
    const protocolIndex = url.indexOf('://');
    if (protocolIndex < 0) {
      return null;
    }
    const hostStart = protocolIndex + 3;
    const slashIndex = url.indexOf('/', hostStart);
    if (slashIndex < 0) {
      return null;
    }
    const hostPort = url.substring(hostStart, slashIndex).toLowerCase();
    if (!hostPort) {
      return null;
    }
    const host = hostPort.split(':')[0];
    const pathAndQuery = url.substring(slashIndex);
    return {
      host,
      pathAndQuery: pathAndQuery || '/'
    };
  }

  private static rewriteLocalAbsoluteUrl(url: string, origin: string): string {
    const parsed = MediaUrlUtil.parseAbsoluteUrl(url);
    if (!parsed) {
      return url;
    }
    if (LOCAL_MEDIA_HOSTS.indexOf(parsed.host) < 0) {
      return url;
    }

    let pathAndQuery = parsed.pathAndQuery;
    if (pathAndQuery.startsWith('/s3/')) {
      return `${origin}${pathAndQuery}`;
    }
    if (pathAndQuery.startsWith('/hotspot/')) {
      pathAndQuery = `/s3${pathAndQuery}`;
      return `${origin}${pathAndQuery}`;
    }
    return url;
  }
}
