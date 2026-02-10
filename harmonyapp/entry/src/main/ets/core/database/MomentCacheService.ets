/**
 * 瞬间缓存服务
 * 封装对 moments 表的操作
 */
import { RdbManager } from './RdbManager';
import { relationalStore, ValuesBucket } from '@kit.ArkData';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { MomentItem } from '../types/api';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG = 'MomentCacheService';

export class MomentCacheService {
  private static instance: MomentCacheService;
  private rdbManager = RdbManager.getInstance();

  private constructor() {}

  public static getInstance(): MomentCacheService {
    if (!MomentCacheService.instance) {
      MomentCacheService.instance = new MomentCacheService();
    }
    return MomentCacheService.instance;
  }

  /**
   * 缓存瞬间列表
   */
  public async cacheMoments(moments: MomentItem[]): Promise<void> {
    const store = this.rdbManager.getStore();
    if (!store) return;

    try {
      store.beginTransaction();
      
      // 简单策略：删除旧数据，插入新数据 (仅演示，实际应增量更新)
      // 注意：这里只删除前50条或者全部清空，视需求而定
      await store.executeSql('DELETE FROM moments');

      for (const moment of moments) {
        const valueBucket: ValuesBucket = {
          id: moment.moment_id,
          user_id: moment.user.user_id,
          nickname: moment.user.nickname,
          avatar_url: moment.user.avatar_url || '',
          description: moment.description,
          images: JSON.stringify(moment.images || []),
          tags: JSON.stringify(moment.tags || []),
          latitude: moment.location?.point?.latitude ?? 0,
          longitude: moment.location?.point?.longitude ?? 0,
          location_name: moment.location?.location_name ?? '',
          distance: moment.location?.distance ?? 0,
          likes_count: moment.likes_count,
          comments_count: moment.comments_count,
          created_at: moment.created_at,
          json_data: JSON.stringify(moment)
        };
        await store.insert('moments', valueBucket);
      }
      
      store.commit();
      hilog.info(DOMAIN_NUMBER, TAG, `Cached ${moments.length} moments`);
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Cache moments failed: ${JSON.stringify(err)}`);
      try {
        store.rollBack();
      } catch (e) {
        hilog.error(DOMAIN_NUMBER, TAG, `Rollback failed: ${JSON.stringify(e)}`);
      }
    }
  }

  /**
   * 获取缓存的瞬间列表
   */
  public async getCachedMoments(): Promise<MomentItem[]> {
    const store = this.rdbManager.getStore();
    if (!store) return [];

    try {
      const predicates = new relationalStore.RdbPredicates('moments');
      predicates.orderByDesc('created_at'); 
      
      const resultSet = await store.query(predicates);
      const items: MomentItem[] = [];

      while (resultSet.goToNextRow()) {
        const jsonData = resultSet.getString(resultSet.getColumnIndex('json_data'));
        if (jsonData) {
          try {
            items.push(JSON.parse(jsonData) as MomentItem);
          } catch (e) {
             // ignore parse error
          }
        }
      }
      
      resultSet.close();
      return items;
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Get cached moments failed: ${JSON.stringify(err)}`);
      return [];
    }
  }
}

