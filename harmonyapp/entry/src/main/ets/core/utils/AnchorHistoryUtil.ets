/**
 * 位置历史记录工具
 * 支持按分类管理历史（浏览锚点 / 创建选点等互不干扰）
 */

import { preferences } from '@kit.ArkData';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { PreferenceStores } from '../../constants/AppConstants';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'AnchorHistoryUtil';
const MAX_HISTORY = 10;

/** 历史记录项 */
export class AnchorHistoryItem {
  latitude: number = 0;
  longitude: number = 0;
  address: string = '';
  locationName: string = '';
  timestamp: number = 0;
}

export class AnchorHistoryUtil {
  private static prefInstance: preferences.Preferences | null = null;
  private static caches: Map<string, AnchorHistoryItem[]> = new Map();

  /** 初始化 Preferences（需要在应用启动后调用一次） */
  static init(context: Context): void {
    if (!AnchorHistoryUtil.prefInstance) {
      try {
        AnchorHistoryUtil.prefInstance = preferences.getPreferencesSync(context, { name: PreferenceStores.ANCHOR_HISTORY });
      } catch {
        hilog.warn(DOMAIN_NUMBER, TAG, '初始化 Preferences 失败');
      }
    }
  }

  private static getPreferencesInstance(): preferences.Preferences | null {
    return AnchorHistoryUtil.prefInstance;
  }

  private static prefKey(category: string): string {
    return `${category}_history`;
  }

  /** 获取历史列表。category 默认 'anchor'，创建选点用 'location_pick' */
  static getHistory(category: string = 'anchor'): AnchorHistoryItem[] {
    try {
      const cached = AnchorHistoryUtil.caches.get(category);
      if (cached && cached.length > 0) {
        return cached;
      }
      const pref = AnchorHistoryUtil.getPreferencesInstance();
      if (!pref) {
        return [];
      }
      const json = pref.getSync(AnchorHistoryUtil.prefKey(category), '[]') as string;
      const items: AnchorHistoryItem[] = JSON.parse(json) as AnchorHistoryItem[];
      AnchorHistoryUtil.caches.set(category, items);
      return items;
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, `读取历史失败: ${category}`);
      return [];
    }
  }

  /** 添加一条记录。category 默认 'anchor'，创建选点用 'location_pick' */
  static addToHistory(latitude: number, longitude: number, locationName: string = '', category: string = 'anchor'): void {
    try {
      const items = AnchorHistoryUtil.getHistory(category);
      const newItem = new AnchorHistoryItem();
      newItem.latitude = latitude;
      newItem.longitude = longitude;
      newItem.locationName = locationName;
      newItem.timestamp = Date.now();

      const filtered = items.filter((item: AnchorHistoryItem) => {
        const latDiff = Math.abs(item.latitude - newItem.latitude);
        const lngDiff = Math.abs(item.longitude - newItem.longitude);
        return latDiff > 0.0005 || lngDiff > 0.0005;
      });

      filtered.unshift(newItem);
      const trimmed = filtered.slice(0, MAX_HISTORY);

      AnchorHistoryUtil.caches.set(category, trimmed);
      const pref = AnchorHistoryUtil.getPreferencesInstance();
      if (pref) {
        pref.putSync(AnchorHistoryUtil.prefKey(category), JSON.stringify(trimmed));
        pref.flush();
      }
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, `保存历史失败: ${category}`);
    }
  }

  /** 清空指定分类的历史。category 默认 'anchor' */
  static clearHistory(category: string = 'anchor'): void {
    try {
      AnchorHistoryUtil.caches.set(category, []);
      const pref = AnchorHistoryUtil.getPreferencesInstance();
      if (pref) {
        pref.putSync(AnchorHistoryUtil.prefKey(category), '[]');
        pref.flush();
      }
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, `清空历史失败: ${category}`);
    }
  }
}
