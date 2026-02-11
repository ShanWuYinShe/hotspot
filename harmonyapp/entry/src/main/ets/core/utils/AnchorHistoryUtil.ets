/**
 * 浏览锚点历史记录工具
 * 管理用户设置过的浏览锚点位置历史
 */

import { preferences } from '@kit.ArkData';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'AnchorHistoryUtil';
const PREF_KEY = 'anchor_history';
const MAX_HISTORY = 10;

/** 锚点历史记录项 */
export class AnchorHistoryItem {
  latitude: number = 0;
  longitude: number = 0;
  address: string = '';
  locationName: string = '';
  timestamp: number = 0;
}

export class AnchorHistoryUtil {
  private static prefInstance: preferences.Preferences | null = null;
  private static historyCache: AnchorHistoryItem[] = [];

  /** 初始化 Preferences（需要在应用启动后调用一次） */
  static init(context: Context): void {
    if (!AnchorHistoryUtil.prefInstance) {
      try {
        AnchorHistoryUtil.prefInstance = preferences.getPreferencesSync(context, { name: 'anchor_history' });
      } catch {
        hilog.warn(DOMAIN_NUMBER, TAG, '初始化 Preferences 失败');
      }
    }
  }

  private static getPreferencesInstance(): preferences.Preferences | null {
    return AnchorHistoryUtil.prefInstance;
  }

  /** 获取锚点历史列表 */
  static getHistory(): AnchorHistoryItem[] {
    try {
      if (AnchorHistoryUtil.historyCache.length > 0) {
        return AnchorHistoryUtil.historyCache;
      }
      const pref = AnchorHistoryUtil.getPreferencesInstance();
      if (!pref) {
        return [];
      }
      const json = pref.getSync(PREF_KEY, '[]') as string;
      const items: AnchorHistoryItem[] = JSON.parse(json) as AnchorHistoryItem[];
      AnchorHistoryUtil.historyCache = items;
      return items;
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '读取锚点历史失败');
      return [];
    }
  }

  /** 添加一条锚点记录 */
  static addToHistory(latitude: number, longitude: number, locationName: string = ''): void {
    try {
      const items = AnchorHistoryUtil.getHistory();
      const newItem = new AnchorHistoryItem();
      newItem.latitude = latitude;
      newItem.longitude = longitude;
      newItem.locationName = locationName;
      newItem.timestamp = Date.now();

      // 去重：如果距离非常近（约 50m 以内）则替换
      const filtered = items.filter((item: AnchorHistoryItem) => {
        const latDiff = Math.abs(item.latitude - newItem.latitude);
        const lngDiff = Math.abs(item.longitude - newItem.longitude);
        return latDiff > 0.0005 || lngDiff > 0.0005;
      });

      filtered.unshift(newItem);
      const trimmed = filtered.slice(0, MAX_HISTORY);

      AnchorHistoryUtil.historyCache = trimmed;
      const pref = AnchorHistoryUtil.getPreferencesInstance();
      if (pref) {
        pref.putSync(PREF_KEY, JSON.stringify(trimmed));
        pref.flush();
      }
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '保存锚点历史失败');
    }
  }

  /** 清空历史 */
  static clearHistory(): void {
    try {
      AnchorHistoryUtil.historyCache = [];
      const pref = AnchorHistoryUtil.getPreferencesInstance();
      if (pref) {
        pref.putSync(PREF_KEY, '[]');
        pref.flush();
      }
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '清空锚点历史失败');
    }
  }
}
