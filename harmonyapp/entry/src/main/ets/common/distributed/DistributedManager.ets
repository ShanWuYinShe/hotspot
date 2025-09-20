/**
 * HarmonyOS分布式能力管理器
 * 集成HarmonyOS分布式数据管理、设备发现、跨设备协同等能力
 */

import distributedKVStore from '@ohos.data.distributedKVStore';
import deviceManager from '@ohos.distributedDeviceManager';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { GetHistoryResponse } from '../types/BackendApiTypes';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'DistributedManager';

export interface DeviceInfo {
  deviceId: string;
  deviceName: string;
  deviceType: number;
  networkId: string;
  isOnline: boolean;
}

export interface DistributedData {
  key: string;
  value: string | number | boolean | Uint8Array;
  timestamp: number;
}

export interface SyncProgress {
  deviceId: string;
  status: 'syncing' | 'completed' | 'failed';
  progress: number;
  error?: string;
}

export class DistributedManager {
  private static instance: DistributedManager | null = null;
  private context: common.UIAbilityContext | null = null;
  private kvManager: distributedKVStore.KVManager | null = null;
  private kvStore: distributedKVStore.DeviceKVStore | null = null;
  private deviceManager: deviceManager.DeviceManager | null = null;
  private trustedDevices: DeviceInfo[] = [];
  private syncCallbacks: Map<string, (progress: SyncProgress) => void> = new Map();

  // 分布式存储配置
  private readonly STORE_ID = 'geotalk_distributed_store';
  private readonly APP_ID = 'com.geotalk.harmonyos';

  private constructor() {}

  static getInstance(): DistributedManager {
    if (!DistributedManager.instance) {
      DistributedManager.instance = new DistributedManager();
    }
    return DistributedManager.instance;
  }

  /**
   * 初始化分布式管理器
   */
  async initialize(context: common.UIAbilityContext): Promise<void> {
    try {
      this.context = context;
      
      // 初始化分布式数据管理
      await this.initDistributedKVStore();
      
      // 初始化设备管理
      await this.initDeviceManager();
      
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 分布式管理器初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 分布式管理器初始化失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 初始化分布式键值存储
   */
  private async initDistributedKVStore(): Promise<void> {
    try {
      if (!this.context) {
        throw new Error('Context未初始化');
      }

      // 创建KVManager
      const kvManagerConfig: distributedKVStore.KVManagerConfig = {
        context: this.context,
        bundleName: this.context.abilityInfo.bundleName
      };

      this.kvManager = distributedKVStore.createKVManager(kvManagerConfig);

      // 创建分布式KV存储
      const options: distributedKVStore.Options = {
        createIfMissing: true,
        encrypt: false,
        backup: false,
        autoSync: true,
        securityLevel: distributedKVStore.SecurityLevel.S1,
        kvStoreType: distributedKVStore.KVStoreType.DEVICE_COLLABORATION
      };

      this.kvStore = await this.kvManager.getKVStore(this.STORE_ID, options) as distributedKVStore.DeviceKVStore;

      // 监听数据变化
      this.kvStore.on('dataChange', distributedKVStore.SubscribeType.SUBSCRIBE_TYPE_ALL, (data) => {
        this.handleDataChange(data);
      });

      // 监听同步完成事件
      this.kvStore.on('syncComplete', (data) => {
        this.handleSyncComplete(data);
      });

      hilog.info(DOMAIN_NUMBER, TAG, '✅ 分布式KV存储初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 分布式KV存储初始化失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 初始化设备管理
   */
  private async initDeviceManager(): Promise<void> {
    try {
      // 创建设备管理器实例
      this.deviceManager = deviceManager.createDeviceManager(this.APP_ID);

      // 监听设备状态变化
      this.deviceManager.on('deviceStateChange', (data) => {
        this.handleDeviceStateChange(data);
      });

      // 监听设备发现
      this.deviceManager.on('discoverSuccess', (data) => {
        this.handleDeviceDiscovered(data);
      });

      // 获取可信设备列表
      await this.refreshTrustedDevices();

      hilog.info(DOMAIN_NUMBER, TAG, '✅ 设备管理器初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 设备管理器初始化失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 存储分布式数据
   */
  async putData(key: string, value: string | number | boolean | Uint8Array): Promise<void> {
    try {
      if (!this.kvStore) {
        throw new Error('分布式存储未初始化');
      }

      await this.kvStore.put(key, value);
      hilog.info(DOMAIN_NUMBER, TAG, `✅ 数据存储成功: ${key}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 数据存储失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 获取分布式数据
   */
  async getData(key: string): Promise<string | number | boolean | Uint8Array | undefined> {
    try {
      if (!this.kvStore) {
        throw new Error('分布式存储未初始化');
      }

      const value = await this.kvStore.get(key);
      hilog.info(DOMAIN_NUMBER, TAG, `✅ 数据获取成功: ${key}`);
      return value;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 数据获取失败: ${JSON.stringify(error)}`);
      return undefined;
    }
  }

  /**
   * 删除分布式数据
   */
  async deleteData(key: string): Promise<void> {
    try {
      if (!this.kvStore) {
        throw new Error('分布式存储未初始化');
      }

      await this.kvStore.delete(key);
      hilog.info(DOMAIN_NUMBER, TAG, `✅ 数据删除成功: ${key}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 数据删除失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 同步消息到其他设备
   */
  async syncMessages(messages: GetHistoryResponse[], targetDeviceIds?: string[]): Promise<void> {
    try {
      if (!this.kvStore) {
        throw new Error('分布式存储未初始化');
      }

      // 将消息存储到分布式KV存储
      for (const message of messages) {
        const key = `message_${message.message_id}`;
        const value = JSON.stringify(message);
        await this.putData(key, value);
      }

      // 触发同步到指定设备
      const deviceIds = targetDeviceIds || this.trustedDevices.map(d => d.deviceId);
      if (deviceIds.length > 0) {
        await this.kvStore.sync(deviceIds, distributedKVStore.SyncMode.PUSH_PULL);
        hilog.info(DOMAIN_NUMBER, TAG, `✅ 消息同步已触发，目标设备: ${deviceIds.length}个`);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 消息同步失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 获取可信设备列表
   */
  async getTrustedDevices(): Promise<DeviceInfo[]> {
    return [...this.trustedDevices];
  }

  /**
   * 刷新可信设备列表
   */
  async refreshTrustedDevices(): Promise<void> {
    try {
      if (!this.deviceManager) {
        throw new Error('设备管理器未初始化');
      }

      const devices = this.deviceManager.getAvailableDeviceListSync();
      this.trustedDevices = devices.map(device => ({
        deviceId: device.deviceId,
        deviceName: device.deviceName,
        deviceType: device.deviceType,
        networkId: device.networkId || '',
        isOnline: true
      }));

      hilog.info(DOMAIN_NUMBER, TAG, `✅ 可信设备列表已更新: ${this.trustedDevices.length}个设备`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 刷新设备列表失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 开始设备发现
   */
  async startDeviceDiscovery(): Promise<void> {
    try {
      if (!this.deviceManager) {
        throw new Error('设备管理器未初始化');
      }

      const discoverParam = {
        subscribeId: Date.now(),
        mode: 0xAA, // 主动发现模式
        medium: 0, // 自动选择发现媒介
        freq: 2, // 高频发现
        isSameAccount: false,
        isWakeRemote: true,
        capability: 0
      };

      this.deviceManager.startDiscovering(discoverParam);
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 设备发现已开始');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 开始设备发现失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 停止设备发现
   */
  async stopDeviceDiscovery(): Promise<void> {
    try {
      if (!this.deviceManager) {
        return;
      }

      this.deviceManager.stopDiscovering();
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 设备发现已停止');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 停止设备发现失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 注册同步进度回调
   */
  registerSyncCallback(deviceId: string, callback: (progress: SyncProgress) => void): void {
    this.syncCallbacks.set(deviceId, callback);
  }

  /**
   * 注销同步进度回调
   */
  unregisterSyncCallback(deviceId: string): void {
    this.syncCallbacks.delete(deviceId);
  }

  /**
   * 处理数据变化事件
   */
  private handleDataChange(data: distributedKVStore.ChangeNotification): void {
    hilog.info(DOMAIN_NUMBER, TAG, `数据变化事件: ${data.insertEntries?.length || 0}个新增, ${data.updateEntries?.length || 0}个更新, ${data.deleteEntries?.length || 0}个删除`);
    
    // 这里可以通知应用层数据发生了变化
    // 例如：更新UI、刷新消息列表等
  }

  /**
   * 处理同步完成事件
   */
  private handleSyncComplete(data: Array<[string, number]>): void {
    for (const [deviceId, result] of data) {
      const callback = this.syncCallbacks.get(deviceId);
      if (callback) {
        const progress: SyncProgress = {
          deviceId,
          status: result === 0 ? 'completed' : 'failed',
          progress: 100,
          error: result !== 0 ? `同步失败，错误码: ${result}` : undefined
        };
        callback(progress);
      }
      
      hilog.info(DOMAIN_NUMBER, TAG, `设备 ${deviceId} 同步${result === 0 ? '成功' : '失败'}`);
    }
  }

  /**
   * 处理设备状态变化
   */
  private handleDeviceStateChange(data: deviceManager.DeviceStateChange): void {
    hilog.info(DOMAIN_NUMBER, TAG, `设备状态变化: ${data.deviceId} - ${data.action}`);
    
    // 更新可信设备列表
    this.refreshTrustedDevices();
  }

  /**
   * 处理设备发现事件
   */
  private handleDeviceDiscovered(data: deviceManager.DiscoverySuccess): void {
    hilog.info(DOMAIN_NUMBER, TAG, `发现新设备: ${data.device?.deviceName}`);
    
    // 可以在这里处理新发现的设备
  }

  /**
   * 清理资源
   */
  async cleanup(): Promise<void> {
    try {
      await this.stopDeviceDiscovery();
      
      if (this.kvStore) {
        this.kvStore.off('dataChange');
        this.kvStore.off('syncComplete');
      }
      
      if (this.deviceManager) {
        this.deviceManager.off('deviceStateChange');
        this.deviceManager.off('discoverSuccess');
        deviceManager.releaseDeviceManager(this.deviceManager);
      }
      
      this.syncCallbacks.clear();
      this.trustedDevices = [];
      this.context = null;
      
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 分布式管理器清理完成');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 分布式管理器清理失败: ${JSON.stringify(error)}`);
    }
  }
}
