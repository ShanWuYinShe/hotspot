/**
 * 统一权限管理器
 * 根据需求文档要求实现统一的权限管理服务
 */

import { abilityAccessCtrl, Permissions } from '@kit.AbilityKit';
import { geoLocationManager } from '@kit.LocationKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'PermissionManager';

/**
 * 权限状态枚举
 */
export enum PermissionStatus {
  GRANTED = 'granted',           // 已授权
  DENIED = 'denied',             // 已拒绝
  NOT_REQUESTED = 'not_requested', // 未请求
  PERMANENTLY_DENIED = 'permanently_denied' // 永久拒绝
}

/**
 * 权限信息接口
 */
export interface PermissionInfo {
  permission: Permissions;
  status: PermissionStatus;
  reason: string;
  isRequired: boolean;
  requestCount: number;
  lastRequestTime?: number;
}

/**
 * 权限配置接口
 */
interface PermissionConfig {
  reason: string;
  isRequired: boolean;
}

/**
 * 权限请求结果接口
 */
export interface CustomPermissionRequestResult {
  permission: Permissions;
  granted: boolean;
  status: PermissionStatus;
  message: string;
}

/**
 * 批量权限请求结果接口
 */
export interface BatchPermissionResult {
  allGranted: boolean;
  results: CustomPermissionRequestResult[];
  grantedPermissions: Permissions[];
  deniedPermissions: Permissions[];
}

export class PermissionManager {
  private static instance: PermissionManager | null = null;
  private context: common.UIAbilityContext | null = null;
  private atManager: abilityAccessCtrl.AtManager | null = null;
  private permissionCache: Map<Permissions, PermissionInfo> = new Map();

  // 权限配置映射
  private readonly permissionConfig: Map<Permissions, PermissionConfig> = new Map();

  private initPermissionConfig(): void {
    this.permissionConfig.set('ohos.permission.LOCATION', { reason: '用于获取您的位置信息，发现附近的用户和群组', isRequired: true });
    this.permissionConfig.set('ohos.permission.APPROXIMATELY_LOCATION', { reason: '用于获取大概位置信息，提供基于位置的服务', isRequired: false });
    this.permissionConfig.set('ohos.permission.INTERNET', { reason: '用于网络通信，获取最新数据', isRequired: true });
    this.permissionConfig.set('ohos.permission.GET_NETWORK_INFO', { reason: '用于检测网络状态，优化用户体验', isRequired: true });
    this.permissionConfig.set('ohos.permission.CAMERA', { reason: '用于拍摄照片和视频，分享生活瞬间', isRequired: false });
    this.permissionConfig.set('ohos.permission.READ_MEDIA', { reason: '用于读取相册中的照片和视频', isRequired: false });
    this.permissionConfig.set('ohos.permission.WRITE_MEDIA', { reason: '用于保存照片和视频到相册', isRequired: false });
    this.permissionConfig.set('ohos.permission.VIBRATE', { reason: '用于消息通知震动提醒', isRequired: false });
    this.permissionConfig.set('ohos.permission.MICROPHONE', { reason: '用于录制语音消息', isRequired: false });
  }

  private constructor() {
    this.atManager = abilityAccessCtrl.createAtManager();
    this.initPermissionConfig();
  }

  static getInstance(): PermissionManager {
    if (!PermissionManager.instance) {
      PermissionManager.instance = new PermissionManager();
    }
    return PermissionManager.instance;
  }

  /**
   * 初始化权限管理器
   */
  async initialize(context: common.UIAbilityContext): Promise<void> {
    try {
      this.context = context;
      hilog.info(DOMAIN_NUMBER, TAG, '权限管理器初始化成功');
      
      // 预加载权限状态
      await this.preloadPermissionStatus();
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `权限管理器初始化失败: ${JSON.stringify(error)}`);
      throw new Error('权限管理器初始化失败');
    }
  }

  /**
   * 预加载所有权限状态
   */
  private async preloadPermissionStatus(): Promise<void> {
    if (!this.context || !this.atManager) {
      return;
    }

    const permissionEntries = Array.from(this.permissionConfig.entries());
    for (let i = 0; i < permissionEntries.length; i++) {
      const permission = permissionEntries[i][0];
      const config = permissionEntries[i][1];
      try {
        const status = await this.checkPermissionStatus(permission);
        const permissionInfo: PermissionInfo = {
          permission,
          status,
          reason: config.reason,
          isRequired: config.isRequired,
          requestCount: 0
        };
        this.permissionCache.set(permission, permissionInfo);
      } catch (error) {
        hilog.warn(DOMAIN_NUMBER, TAG, `预加载权限状态失败: ${permission}, error: ${JSON.stringify(error)}`);
      }
    }
  }

  /**
   * 检查单个权限状态
   */
  async checkPermissionStatus(permission: Permissions): Promise<PermissionStatus> {
    if (!this.context || !this.atManager) {
      throw new Error('权限管理器未初始化');
    }

    try {
      const grantStatus = await this.atManager.checkAccessToken(this.context.applicationInfo.accessTokenId, permission);
      
      switch (grantStatus) {
        case abilityAccessCtrl.GrantStatus.PERMISSION_GRANTED:
          return PermissionStatus.GRANTED;
        case abilityAccessCtrl.GrantStatus.PERMISSION_DENIED:
          return PermissionStatus.DENIED;
        default:
          return PermissionStatus.NOT_REQUESTED;
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `检查权限状态失败: ${permission}, error: ${JSON.stringify(error)}`);
      return PermissionStatus.DENIED;
    }
  }

  /**
   * 检查权限是否已授权
   */
  async isPermissionGranted(permission: Permissions): Promise<boolean> {
    // 先检查缓存
    const cachedInfo = this.permissionCache.get(permission);
    if (cachedInfo && cachedInfo.status === PermissionStatus.GRANTED) {
      return true;
    }

    // 实时检查
    const status = await this.checkPermissionStatus(permission);
    
    // 更新缓存
    if (cachedInfo) {
      cachedInfo.status = status;
    }

    return status === PermissionStatus.GRANTED;
  }

  /**
   * 请求单个权限
   */
  async requestPermission(permission: Permissions): Promise<CustomPermissionRequestResult> {
    if (!this.context || !this.atManager) {
      throw new Error('权限管理器未初始化');
    }

    try {
      // 先检查是否已经授权
      const currentStatus = await this.checkPermissionStatus(permission);
      if (currentStatus === PermissionStatus.GRANTED) {
        return {
          permission,
          granted: true,
          status: PermissionStatus.GRANTED,
          message: '权限已授权'
        };
      }

      // 更新请求计数
      const permissionInfo = this.permissionCache.get(permission);
      if (permissionInfo) {
        permissionInfo.requestCount++;
        permissionInfo.lastRequestTime = Date.now();
      }

      // 请求权限
      const requestResult = await this.atManager.requestPermissionsFromUser(this.context, [permission]);
      const authResult = requestResult.authResults[0];
      const granted = authResult === abilityAccessCtrl.GrantStatus.PERMISSION_GRANTED;

      const newStatus = granted ? PermissionStatus.GRANTED : PermissionStatus.DENIED;
      
      // 更新缓存
      if (permissionInfo) {
        permissionInfo.status = newStatus;
      }

      hilog.info(DOMAIN_NUMBER, TAG, `权限请求结果: ${permission}, granted: ${granted}`);

      return {
        permission,
        granted,
        status: newStatus,
        message: granted ? '权限授权成功' : '权限被拒绝'
      };
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `请求权限失败: ${permission}, error: ${JSON.stringify(error)}`);
      return {
        permission,
        granted: false,
        status: PermissionStatus.DENIED,
        message: `权限请求失败: ${error}`
      };
    }
  }

  /**
   * 批量请求权限
   */
  async requestPermissions(permissions: Permissions[]): Promise<BatchPermissionResult> {
    const results: CustomPermissionRequestResult[] = [];
    const grantedPermissions: Permissions[] = [];
    const deniedPermissions: Permissions[] = [];

    for (const permission of permissions) {
      try {
        const result = await this.requestPermission(permission);
        results.push(result);
        
        if (result.granted) {
          grantedPermissions.push(permission);
        } else {
          deniedPermissions.push(permission);
        }
      } catch (error) {
        hilog.error(DOMAIN_NUMBER, TAG, `批量请求权限失败: ${permission}, error: ${JSON.stringify(error)}`);
        results.push({
          permission,
          granted: false,
          status: PermissionStatus.DENIED,
          message: `权限请求失败: ${error}`
        });
        deniedPermissions.push(permission);
      }
    }

    const allGranted = deniedPermissions.length === 0;

    return {
      allGranted,
      results,
      grantedPermissions,
      deniedPermissions
    };
  }

  /**
   * 请求必需权限
   */
  async requestRequiredPermissions(): Promise<BatchPermissionResult> {
    const requiredPermissions: Permissions[] = [];
    
    const permissionEntries = Array.from(this.permissionConfig.entries());
    for (let i = 0; i < permissionEntries.length; i++) {
      const permission = permissionEntries[i][0];
      const config = permissionEntries[i][1];
      if (config.isRequired) {
        requiredPermissions.push(permission);
      }
    }

    return await this.requestPermissions(requiredPermissions);
  }

  /**
   * 检查必需权限是否都已授权
   */
  async areRequiredPermissionsGranted(): Promise<boolean> {
    const permissionEntries = Array.from(this.permissionConfig.entries());
    for (let i = 0; i < permissionEntries.length; i++) {
      const permission = permissionEntries[i][0];
      const config = permissionEntries[i][1];
      if (config.isRequired) {
        const granted = await this.isPermissionGranted(permission);
        if (!granted) {
          return false;
        }
      }
    }
    return true;
  }

  /**
   * 获取权限信息
   */
  getPermissionInfo(permission: Permissions): PermissionInfo | null {
    return this.permissionCache.get(permission) || null;
  }

  /**
   * 获取所有权限状态
   */
  getAllPermissionStatus(): Map<Permissions, PermissionInfo> {
    return new Map(this.permissionCache);
  }

  /**
   * 获取已授权的权限列表
   */
  getGrantedPermissions(): Permissions[] {
    const granted: Permissions[] = [];
    const cacheEntries = Array.from(this.permissionCache.entries());
    for (let i = 0; i < cacheEntries.length; i++) {
      const permission = cacheEntries[i][0];
      const info = cacheEntries[i][1];
      if (info.status === PermissionStatus.GRANTED) {
        granted.push(permission);
      }
    }
    return granted;
  }

  /**
   * 获取被拒绝的权限列表
   */
  getDeniedPermissions(): Permissions[] {
    const denied: Permissions[] = [];
    const cacheEntries = Array.from(this.permissionCache.entries());
    for (let i = 0; i < cacheEntries.length; i++) {
      const permission = cacheEntries[i][0];
      const info = cacheEntries[i][1];
      if (info.status === PermissionStatus.DENIED || info.status === PermissionStatus.PERMANENTLY_DENIED) {
        denied.push(permission);
      }
    }
    return denied;
  }

  /**
   * 刷新权限状态缓存
   */
  async refreshPermissionCache(): Promise<void> {
    await this.preloadPermissionStatus();
    hilog.info(DOMAIN_NUMBER, TAG, '权限状态缓存已刷新');
  }

  /**
   * 清除权限缓存
   */
  clearPermissionCache(): void {
    this.permissionCache.clear();
    hilog.info(DOMAIN_NUMBER, TAG, '权限缓存已清除');
  }

  /**
   * 请求位置权限
   *
   * 将 APPROXIMATELY_LOCATION 和 LOCATION 一次性传入 requestPermissionsFromUser，
   * 系统会合并为一个弹窗，用户可一次性选择授权精确或大概位置。
   *
   * @returns true 表示至少获得了大概位置权限
   */
  async requestLocationPermission(): Promise<boolean> {
    if (!this.context || !this.atManager) {
      hilog.error(DOMAIN_NUMBER, TAG, '权限管理器未初始化');
      return false;
    }

    try {
      // 先检查是否已经授权
      const alreadyGranted = await this.checkLocationPermission();
      if (alreadyGranted) {
        return true;
      }

      // 同时请求两个权限，系统将合并为一个弹窗
      const locationPermissions: Permissions[] = [
        'ohos.permission.APPROXIMATELY_LOCATION',
        'ohos.permission.LOCATION'
      ];
      const result = await this.atManager.requestPermissionsFromUser(this.context, locationPermissions);

      const approxGranted = result.authResults[0] === abilityAccessCtrl.GrantStatus.PERMISSION_GRANTED;
      const preciseGranted = result.authResults[1] === abilityAccessCtrl.GrantStatus.PERMISSION_GRANTED;

      // 更新缓存
      const approxInfo = this.permissionCache.get('ohos.permission.APPROXIMATELY_LOCATION');
      if (approxInfo) {
        approxInfo.status = approxGranted ? PermissionStatus.GRANTED : PermissionStatus.DENIED;
        approxInfo.requestCount++;
        approxInfo.lastRequestTime = Date.now();
      }
      const preciseInfo = this.permissionCache.get('ohos.permission.LOCATION');
      if (preciseInfo) {
        preciseInfo.status = preciseGranted ? PermissionStatus.GRANTED : PermissionStatus.DENIED;
        preciseInfo.requestCount++;
        preciseInfo.lastRequestTime = Date.now();
      }

      if (preciseGranted) {
        hilog.info(DOMAIN_NUMBER, TAG, '精确位置权限已授权');
      } else if (approxGranted) {
        hilog.info(DOMAIN_NUMBER, TAG, '仅获得大概位置权限');
      } else {
        hilog.warn(DOMAIN_NUMBER, TAG, '位置权限全部被拒绝');
      }

      return approxGranted || preciseGranted;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `请求位置权限失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 检查媒体权限并请求
   */
  async requestMediaPermissions(): Promise<boolean> {
    try {
      const mediaPermissions: Permissions[] = [
        'ohos.permission.CAMERA',
        'ohos.permission.READ_MEDIA',
        'ohos.permission.WRITE_MEDIA'
      ];

      const result = await this.requestPermissions(mediaPermissions);
      return result.grantedPermissions.length > 0; // 至少有一个媒体权限被授权
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `请求媒体权限失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 打开应用设置页面
   */
  async openAppSettings(): Promise<void> {
    try {
      if (!this.context) {
        throw new Error('Context未初始化');
      }

      // 跳转到应用设置页面
      await this.context.startAbility({
        bundleName: 'com.huawei.hmos.settings',
        abilityName: 'com.huawei.hmos.settings.MainAbility',
        parameters: {
          pushParams: this.context.applicationInfo.name
        }
      });

      hilog.info(DOMAIN_NUMBER, TAG, '已打开应用设置页面');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `打开应用设置页面失败: ${JSON.stringify(error)}`);
      throw new Error('无法打开设置页面');
    }
  }

  /**
   * 显示权限说明对话框
   */
  showPermissionRationale(permission: Permissions): string {
    const config = this.permissionConfig.get(permission);
    return config?.reason || '该功能需要相应权限才能正常使用';
  }

  /**
   * 检查权限是否为必需权限
   */
  isRequiredPermission(permission: Permissions): boolean {
    const config = this.permissionConfig.get(permission);
    return config?.isRequired || false;
  }

  /**
   * 获取权限请求次数
   */
  getPermissionRequestCount(permission: Permissions): number {
    const info = this.permissionCache.get(permission);
    return info?.requestCount || 0;
  }

  /**
   * 检查是否应该显示权限说明
   * 根据HarmonyOS指南，在第二次请求权限时显示说明
   */
  shouldShowPermissionRationale(permission: Permissions): boolean {
    const requestCount = this.getPermissionRequestCount(permission);
    const status = this.permissionCache.get(permission)?.status;
    
    return requestCount > 0 && status === PermissionStatus.DENIED;
  }

  /**
   * 检查位置权限是否已授权
   */
  async checkLocationPermission(): Promise<boolean> {
    try {
      // 先检查精确位置权限
      const preciseGranted = await this.isPermissionGranted('ohos.permission.LOCATION');
      if (preciseGranted) {
        return true;
      }

      // 再检查大概位置权限
      const approximateGranted = await this.isPermissionGranted('ohos.permission.APPROXIMATELY_LOCATION');
      return approximateGranted;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `检查位置权限失败: ${JSON.stringify(error)}`);
      return false;
    }
  }
}

