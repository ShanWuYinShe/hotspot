/**
 * 小组件数据服务
 * 向 HarmonyOS 桌面小组件提供数据更新
 */

import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'WidgetDataService';

export class WidgetDataService {
  private static instance: WidgetDataService | null = null;
  private statsCache: Map<string, number> = new Map();

  static getInstance(): WidgetDataService {
    if (!WidgetDataService.instance) {
      WidgetDataService.instance = new WidgetDataService();
    }
    return WidgetDataService.instance;
  }

  /** 更新统计数据 */
  updateStats(key: string, value: number): void {
    this.statsCache.set(key, value);
    hilog.debug(DOMAIN_NUMBER, TAG, `更新统计: ${key}=${value}`);
  }

  /** 获取统计数据 */
  getStats(key: string): number {
    return this.statsCache.get(key) ?? 0;
  }
}
