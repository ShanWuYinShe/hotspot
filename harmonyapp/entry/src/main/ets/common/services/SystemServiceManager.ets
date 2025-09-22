/**
 * 系统服务管理器
 * 统一管理HarmonyOS系统服务的集成
 */

import { notificationManager } from '@kit.NotificationKit';
import { vibrator } from '@kit.SensorServiceKit';
import { camera } from '@kit.CameraKit';
import { photoAccessHelper } from '@kit.MediaLibraryKit';
import { geoLocationManager } from '@kit.LocationKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';
import { PermissionManager } from '../permission/PermissionManager';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'SystemServiceManager';

/**
 * 通知配置接口
 */
export interface NotificationConfig {
  title: string;
  content: string;
  icon?: string;
  sound?: boolean;
  vibration?: boolean;
  autoCancel?: boolean;
  showWhen?: boolean;
}

/**
 * 位置信息接口
 */
export interface LocationInfo {
  latitude: number;
  longitude: number;
  accuracy: number;
  altitude?: number;
  speed?: number;
  bearing?: number;
  timestamp: number;
}

/**
 * 媒体文件信息接口
 */
export interface MediaFileInfo {
  uri: string;
  fileName: string;
  fileSize: number;
  mimeType: string;
  width?: number;
  height?: number;
  duration?: number;
}

export class SystemServiceManager {
  private static instance: SystemServiceManager | null = null;
  private context: common.UIAbilityContext | null = null;
  private permissionManager: PermissionManager;
  private locationCallback: ((location: LocationInfo) => void) | null = null;
  private isLocationListening: boolean = false;

  private constructor() {
    this.permissionManager = PermissionManager.getInstance();
  }

  static getInstance(): SystemServiceManager {
    if (!SystemServiceManager.instance) {
      SystemServiceManager.instance = new SystemServiceManager();
    }
    return SystemServiceManager.instance;
  }

  /**
   * 初始化系统服务管理器
   */
  async initialize(context: common.UIAbilityContext): Promise<void> {
    try {
      this.context = context;
      await this.permissionManager.initialize(context);
      hilog.info(DOMAIN_NUMBER, TAG, '系统服务管理器初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `系统服务管理器初始化失败: ${JSON.stringify(error)}`);
      throw new Error('系统服务管理器初始化失败');
    }
  }

  // ==================== 通知服务 ====================

  /**
   * 发送本地通知
   */
  async sendNotification(config: NotificationConfig): Promise<boolean> {
    try {
      // 检查通知权限
      const hasPermission = await this.permissionManager.isPermissionGranted('ohos.permission.NOTIFICATION_CONTROLLER');
      if (!hasPermission) {
        hilog.warn(DOMAIN_NUMBER, TAG, '缺少通知权限');
        return false;
      }

      const notificationRequest: notificationManager.NotificationRequest = {
        id: Date.now(),
        content: {
          contentType: notificationManager.ContentType.NOTIFICATION_CONTENT_BASIC_TEXT,
          normal: {
            title: config.title,
            text: config.content,
            additionalText: ''
          }
        },
        actionButtons: [],
        wantAgent: undefined,
        extraInfo: undefined,
        color: undefined,
        colorEnabled: undefined,
        isAlertOnce: undefined,
        isStopwatch: undefined,
        isCountDown: undefined,
        isRemoveAllowed: config.autoCancel !== false,
        hashCode: undefined,
        classification: undefined,
        groupName: undefined,
        template: undefined,
        isUnremovable: undefined,
        source: undefined,
        distributedOption: undefined,
        deviceId: undefined,
        notificationFlags: undefined,
        removalWantAgent: undefined,
        badgeIconStyle: undefined,
        showDeliveryTime: config.showWhen !== false
      };

      await notificationManager.publish(notificationRequest);
      hilog.info(DOMAIN_NUMBER, TAG, `通知发送成功: ${config.title}`);
      return true;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `发送通知失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 取消通知
   */
  async cancelNotification(notificationId: number): Promise<boolean> {
    try {
      await notificationManager.cancel(notificationId);
      hilog.info(DOMAIN_NUMBER, TAG, `通知已取消: ${notificationId}`);
      return true;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `取消通知失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 取消所有通知
   */
  async cancelAllNotifications(): Promise<boolean> {
    try {
      await notificationManager.cancelAll();
      hilog.info(DOMAIN_NUMBER, TAG, '所有通知已取消');
      return true;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `取消所有通知失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  // ==================== 震动服务 ====================

  /**
   * 触发震动
   */
  async vibrate(duration: number = 200): Promise<boolean> {
    try {
      // 检查震动权限
      const hasPermission = await this.permissionManager.isPermissionGranted('ohos.permission.VIBRATE');
      if (!hasPermission) {
        hilog.warn(DOMAIN_NUMBER, TAG, '缺少震动权限');
        return false;
      }

      await vibrator.startVibration({
        type: 'time',
        duration: duration
      });

      hilog.info(DOMAIN_NUMBER, TAG, `震动触发成功: ${duration}ms`);
      return true;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `震动触发失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 停止震动
   */
  async stopVibration(): Promise<boolean> {
    try {
      await vibrator.stopVibration();
      hilog.info(DOMAIN_NUMBER, TAG, '震动已停止');
      return true;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `停止震动失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  // ==================== 位置服务 ====================

  /**
   * 获取当前位置
   */
  async getCurrentLocation(): Promise<LocationInfo | null> {
    try {
      // 检查位置权限
      const hasLocationPermission = await this.permissionManager.requestLocationPermission();
      if (!hasLocationPermission) {
        hilog.warn(DOMAIN_NUMBER, TAG, '缺少位置权限');
        return null;
      }

      const requestInfo: geoLocationManager.CurrentLocationRequest = {
        priority: geoLocationManager.LocationRequestPriority.FIRST_FIX,
        scenario: geoLocationManager.LocationRequestScenario.UNSET,
        maxAccuracy: 100,
        timeoutMs: 10000
      };

      const location = await geoLocationManager.getCurrentLocation(requestInfo);
      
      const locationInfo: LocationInfo = {
        latitude: location.latitude,
        longitude: location.longitude,
        accuracy: location.accuracy,
        altitude: location.altitude,
        speed: location.speed,
        bearing: location.direction,
        timestamp: location.timeStamp
      };

      hilog.info(DOMAIN_NUMBER, TAG, `获取位置成功: ${locationInfo.latitude}, ${locationInfo.longitude}`);
      return locationInfo;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取位置失败: ${JSON.stringify(error)}`);
      return null;
    }
  }

  /**
   * 开始位置监听
   */
  async startLocationListening(callback: (location: LocationInfo) => void): Promise<boolean> {
    try {
      if (this.isLocationListening) {
        hilog.warn(DOMAIN_NUMBER, TAG, '位置监听已在进行中');
        return false;
      }

      // 检查位置权限
      const hasLocationPermission = await this.permissionManager.requestLocationPermission();
      if (!hasLocationPermission) {
        hilog.warn(DOMAIN_NUMBER, TAG, '缺少位置权限');
        return false;
      }

      const requestInfo: geoLocationManager.LocationRequest = {
        priority: geoLocationManager.LocationRequestPriority.ACCURACY,
        scenario: geoLocationManager.LocationRequestScenario.UNSET,
        timeInterval: 5000, // 5秒更新一次
        distanceInterval: 10, // 移动10米更新一次
        maxAccuracy: 100
      };

      this.locationCallback = callback;

      await geoLocationManager.on('locationChange', requestInfo, (location: geoLocationManager.Location) => {
        const locationInfo: LocationInfo = {
          latitude: location.latitude,
          longitude: location.longitude,
          accuracy: location.accuracy,
          altitude: location.altitude,
          speed: location.speed,
          bearing: location.direction,
          timestamp: location.timeStamp
        };

        if (this.locationCallback) {
          this.locationCallback(locationInfo);
        }
      });

      this.isLocationListening = true;
      hilog.info(DOMAIN_NUMBER, TAG, '位置监听已开始');
      return true;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `开始位置监听失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 停止位置监听
   */
  async stopLocationListening(): Promise<boolean> {
    try {
      if (!this.isLocationListening) {
        hilog.warn(DOMAIN_NUMBER, TAG, '位置监听未在进行中');
        return false;
      }

      await geoLocationManager.off('locationChange');
      this.isLocationListening = false;
      this.locationCallback = null;
      
      hilog.info(DOMAIN_NUMBER, TAG, '位置监听已停止');
      return true;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `停止位置监听失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  // ==================== 媒体服务 ====================

  /**
   * 拍照
   */
  async takePhoto(): Promise<MediaFileInfo | null> {
    try {
      // 检查相机权限
      const hasCameraPermission = await this.permissionManager.isPermissionGranted('ohos.permission.CAMERA');
      if (!hasCameraPermission) {
        const granted = await this.permissionManager.requestPermission('ohos.permission.CAMERA');
        if (!granted.granted) {
          hilog.warn(DOMAIN_NUMBER, TAG, '缺少相机权限');
          return null;
        }
      }

      // 相机拍照功能需要集成HarmonyOS相机API
      // 需要申请相机权限并调用相机服务
      hilog.warn(DOMAIN_NUMBER, TAG, '相机拍照功能需要集成HarmonyOS相机API');
      throw new Error('相机拍照功能未实现，需要集成HarmonyOS相机API');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `拍照失败: ${JSON.stringify(error)}`);
      return null;
    }
  }

  /**
   * 从相册选择图片
   */
  async selectImageFromGallery(): Promise<MediaFileInfo | null> {
    try {
      // 检查媒体权限
      const hasMediaPermission = await this.permissionManager.requestMediaPermissions();
      if (!hasMediaPermission) {
        hilog.warn(DOMAIN_NUMBER, TAG, '缺少媒体权限');
        return null;
      }

      // 相册选择功能需要集成HarmonyOS媒体库API
      // 需要申请媒体库权限并调用选择器
      hilog.warn(DOMAIN_NUMBER, TAG, '相册选择功能需要集成HarmonyOS媒体库API');
      throw new Error('相册选择功能未实现，需要集成HarmonyOS媒体库API');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `选择图片失败: ${JSON.stringify(error)}`);
      return null;
    }
  }

  /**
   * 保存图片到相册
   */
  async saveImageToGallery(imageUri: string): Promise<boolean> {
    try {
      // 检查媒体权限
      const hasMediaPermission = await this.permissionManager.isPermissionGranted('ohos.permission.WRITE_MEDIA');
      if (!hasMediaPermission) {
        const granted = await this.permissionManager.requestPermission('ohos.permission.WRITE_MEDIA');
        if (!granted.granted) {
          hilog.warn(DOMAIN_NUMBER, TAG, '缺少媒体写入权限');
          return false;
        }
      }

      // 保存图片功能需要集成HarmonyOS媒体库API
      // 需要申请媒体库写入权限
      hilog.warn(DOMAIN_NUMBER, TAG, `保存图片功能需要集成HarmonyOS媒体库API: ${imageUri}`);
      throw new Error('保存图片功能未实现，需要集成HarmonyOS媒体库API');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `保存图片失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  // ==================== 系统信息服务 ====================

  /**
   * 获取设备信息
   */
  getDeviceInfo(): Record<string, string> {
    try {
      // 这里可以获取设备的基本信息
      const deviceInfo = {
        platform: 'HarmonyOS',
        version: '4.0', // 实际应该从系统API获取
        model: 'Unknown', // 实际应该从系统API获取
        brand: 'Unknown', // 实际应该从系统API获取
        language: 'zh-CN' // 实际应该从系统API获取
      };

      hilog.info(DOMAIN_NUMBER, TAG, `设备信息: ${JSON.stringify(deviceInfo)}`);
      return deviceInfo;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取设备信息失败: ${JSON.stringify(error)}`);
      return {};
    }
  }

  /**
   * 获取网络状态
   */
  async getNetworkStatus(): Promise<{ connected: boolean; type: string }> {
    try {
      // 这里应该实现网络状态检查
      // 返回基础设备信息
      const networkStatus = {
        connected: true,
        type: 'wifi'
      };

      hilog.info(DOMAIN_NUMBER, TAG, `网络状态: ${JSON.stringify(networkStatus)}`);
      return networkStatus;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取网络状态失败: ${JSON.stringify(error)}`);
      return { connected: false, type: 'none' };
    }
  }

  // ==================== 工具方法 ====================

  /**
   * 检查系统服务可用性
   */
  async checkServiceAvailability(): Promise<Record<string, boolean>> {
    const availability = {
      notification: false,
      vibration: false,
      location: false,
      camera: false,
      media: false
    };

    try {
      // 检查通知服务
      availability.notification = await this.permissionManager.isPermissionGranted('ohos.permission.NOTIFICATION_CONTROLLER');
      
      // 检查震动服务
      availability.vibration = await this.permissionManager.isPermissionGranted('ohos.permission.VIBRATE');
      
      // 检查位置服务
      availability.location = await this.permissionManager.isPermissionGranted('ohos.permission.LOCATION') ||
                             await this.permissionManager.isPermissionGranted('ohos.permission.APPROXIMATELY_LOCATION');
      
      // 检查相机服务
      availability.camera = await this.permissionManager.isPermissionGranted('ohos.permission.CAMERA');
      
      // 检查媒体服务
      availability.media = await this.permissionManager.isPermissionGranted('ohos.permission.READ_MEDIA');

      hilog.info(DOMAIN_NUMBER, TAG, `服务可用性: ${JSON.stringify(availability)}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `检查服务可用性失败: ${JSON.stringify(error)}`);
    }

    return availability;
  }

  /**
   * 清理资源
   */
  async cleanup(): Promise<void> {
    try {
      // 停止位置监听
      if (this.isLocationListening) {
        await this.stopLocationListening();
      }

      // 取消所有通知
      await this.cancelAllNotifications();

      // 停止震动
      await this.stopVibration();

      hilog.info(DOMAIN_NUMBER, TAG, '系统服务资源清理完成');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清理资源失败: ${JSON.stringify(error)}`);
    }
  }
}

