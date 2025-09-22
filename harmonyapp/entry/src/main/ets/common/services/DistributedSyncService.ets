/**
 * HarmonyOS 分布式数据同步服务
 * 利用 HarmonyOS 分布式能力实现跨设备数据同步
 */
import { distributedKVStore } from '@kit.ArkData';
import { BusinessError } from '@kit.BasicServicesKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';

export interface SyncData {
  userId: string;
  dataType: 'user_profile' | 'messages' | 'moments' | 'groups';
  data: any;
  timestamp: number;
  deviceId: string;
}

export class DistributedSyncService {
  private static instance: DistributedSyncService;
  private kvManager: distributedKVStore.KVManager | null = null;
  private kvStore: distributedKVStore.SingleKVStore | null = null;
  private readonly storeId = 'hotspot_distributed_store';

  private constructor() {
  }

  public static getInstance(): DistributedSyncService {
    if (!DistributedSyncService.instance) {
      DistributedSyncService.instance = new DistributedSyncService();
    }
    return DistributedSyncService.instance;
  }

  /**
   * 初始化分布式存储
   * @param context 从页面组件传入的 Context
   */
  public async initialize(context: common.Context): Promise<void> {
    try {
      // 正确的 KVManager 创建方式
      this.kvManager = distributedKVStore.createKVManager({
        context: context,
        bundleName: "com.lsyb.geotalk"
      });

      // 正确的 KVStore 选项配置
      const options = {
        createIfMissing: true,
        encrypt: true,
        backup: false,
        autoSync: true,
        securityLevel: distributedKVStore.SecurityLevel.S2,
        kvStoreType: distributedKVStore.KVStoreType.SINGLE_VERSION
      };

      // 获取 KVStore 实例
      this.kvStore = await this.kvManager.getKVStore(this.storeId, options) as distributedKVStore.SingleKVStore;

      // 设置同步监听
      this.setupSyncListener();

      hilog.info(0x0000, 'DistributedSync', '分布式存储初始化成功');
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'DistributedSync', `分布式存储初始化失败: ${err.message}`);
      throw error;
    }
  }

  /**
   * 同步用户数据到其他设备
   */
  public async syncUserData(syncData: SyncData): Promise<void> {
    if (!this.kvStore) {
      throw new Error('分布式存储未初始化');
    }

    try {
      const key = `${syncData.dataType}_${syncData.userId}_${syncData.timestamp}`;
      const value = JSON.stringify(syncData);

      await this.kvStore.put(key, value);

      hilog.info(0x0000, 'DistributedSync', `数据同步成功: ${key}`);
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'DistributedSync', `数据同步失败: ${err.message}`);
      throw error;
    }
  }

  /**
   * 从其他设备获取同步数据
   */
  public async getSyncData(dataType: string, userId: string): Promise<SyncData[]> {
    if (!this.kvStore) {
      throw new Error('分布式存储未初始化');
    }

    try {
      const prefix = `${dataType}_${userId}`;
      const entries = await this.kvStore.getEntries(prefix);

      const syncDataList: SyncData[] = [];
      for (const entry of entries) {
        try {
          const syncData = JSON.parse(entry.value.toString()) as SyncData;
          syncDataList.push(syncData);
        } catch (parseError) {
          hilog.warn(0x0000, 'DistributedSync', `解析同步数据失败: ${entry.key}`);
        }
      }

      // 按时间戳排序，最新的在前
      return syncDataList.sort((a, b) => b.timestamp - a.timestamp);
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'DistributedSync', `获取同步数据失败: ${err.message}`);
      throw error;
    }
  }

  /**
   * 清理过期的同步数据
   */
  public async cleanExpiredData(maxAge: number = 7 * 24 * 60 * 60 * 1000): Promise<void> {
    if (!this.kvStore) {
      return;
    }

    try {
      const allEntries = await this.kvStore.getEntries('');
      const currentTime = Date.now();

      for (const entry of allEntries) {
        try {
          const syncData = JSON.parse(entry.value.toString()) as SyncData;
          if (currentTime - syncData.timestamp > maxAge) {
            await this.kvStore.delete(entry.key);
            hilog.info(0x0000, 'DistributedSync', `清理过期数据: ${entry.key}`);
          }
        } catch (error) {
          // 如果解析失败，也删除这个条目
          await this.kvStore.delete(entry.key);
        }
      }
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'DistributedSync', `清理过期数据失败: ${err.message}`);
    }
  }

  /**
   * 销毁服务
   */
  public async destroy(): Promise<void> {
    try {
      if (this.kvStore) {
        // 移除事件监听器
        this.kvStore.off('dataChange');
        this.kvStore.off('syncComplete');
      }

      if (this.kvManager && this.kvStore) {
        // 关闭 KVStore
        await this.kvManager.closeKVStore(this.storeId);
        // 注意：通常不需要删除 KVStore，除非确实要清除所有数据
        // await this.kvManager.deleteKVStore(this.storeId);
      }

      this.kvStore = null;
      this.kvManager = null;

      hilog.info(0x0000, 'DistributedSync', '分布式同步服务已销毁');
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'DistributedSync', `销毁服务失败: ${err.message}`);
    }
  }

  /**
   * 设置同步监听器
   */
  private setupSyncListener(): void {
    if (!this.kvStore) {
      return;
    }

    try {
      // 监听数据变更事件
      this.kvStore.on('dataChange', distributedKVStore.SubscribeType.SUBSCRIBE_TYPE_REMOTE,
        (data: distributedKVStore.ChangeNotification) => {
          hilog.info(0x0000, 'DistributedSync',
            `收到远程数据变更: ${data.insertEntries.length} 条插入, ${data.updateEntries.length} 条更新`);

          // 处理远程数据变更
          this.handleRemoteDataChange(data);
        });

      // 监听同步完成事件
      this.kvStore.on('syncComplete', (data: Array<[string, number]>) => {
        hilog.info(0x0000, 'DistributedSync', `数据同步完成: ${JSON.stringify(data)}`);
      });
    } catch (error) {
      const err = error as BusinessError;
      hilog.error(0x0000, 'DistributedSync', `设置同步监听失败: ${err.message}`);
    }
  }

  /**
   * 处理远程数据变更
   */
  private handleRemoteDataChange(data: distributedKVStore.ChangeNotification): void {
    // 处理插入的数据
    for (const entry of data.insertEntries) {
      try {
        const syncData = JSON.parse(entry.value.toString()) as SyncData;
        this.processRemoteSyncData(syncData);
      } catch (error) {
        hilog.warn(0x0000, 'DistributedSync', `处理远程插入数据失败: ${entry.key}`);
      }
    }

    // 处理更新的数据
    for (const entry of data.updateEntries) {
      try {
        const syncData = JSON.parse(entry.value.toString()) as SyncData;
        this.processRemoteSyncData(syncData);
      } catch (error) {
        hilog.warn(0x0000, 'DistributedSync', `处理远程更新数据失败: ${entry.key}`);
      }
    }
  }

  /**
   * 处理远程同步数据
   */
  private processRemoteSyncData(syncData: SyncData): void {
    // 根据数据类型进行相应处理
    switch (syncData.dataType) {
      case 'user_profile':
        // 更新用户资料
        this.updateLocalUserProfile(syncData);
        break;
      case 'messages':
        // 同步消息
        this.syncMessages(syncData);
        break;
      case 'moments':
        // 同步动态
        this.syncMoments(syncData);
        break;
      case 'groups':
        // 同步群组
        this.syncGroups(syncData);
        break;
      default:
        hilog.warn(0x0000, 'DistributedSync', `未知的数据类型: ${syncData.dataType}`);
    }
  }

  /**
   * 更新本地用户资料
   */
  private updateLocalUserProfile(syncData: SyncData): void {
    // TODO: 实现用户资料更新逻辑
    hilog.info(0x0000, 'DistributedSync', `更新用户资料: ${syncData.userId}`);
  }

  /**
   * 同步消息
   */
  private syncMessages(syncData: SyncData): void {
    // TODO: 实现消息同步逻辑
    hilog.info(0x0000, 'DistributedSync', `同步消息: ${syncData.userId}`);
  }

  /**
   * 同步动态
   */
  private syncMoments(syncData: SyncData): void {
    // TODO: 实现动态同步逻辑
    hilog.info(0x0000, 'DistributedSync', `同步动态: ${syncData.userId}`);
  }

  /**
   * 同步群组
   */
  private syncGroups(syncData: SyncData): void {
    // TODO: 实现群组同步逻辑
    hilog.info(0x0000, 'DistributedSync', `同步群组: ${syncData.userId}`);
  }
}
