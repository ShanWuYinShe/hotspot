/**
 * HarmonyOS系统服务管理器
 * 统一管理和调用HarmonyOS各种系统服务
 */

import deviceInfo from '@ohos.deviceInfo';
import batteryInfo from '@ohos.batteryInfo';
import display from '@ohos.display';
import { systemDateTime } from '@kit.BasicServicesKit';
import { networkManager } from '@kit.NetworkKit';
import { vibrator } from '@kit.SensorServiceKit';
import { sensor } from '@kit.SensorServiceKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'SystemServiceManager';

export interface SystemInfo {
  deviceType: string;
  manufacture: string;
  brand: string;
  marketName: string;
  productSeries: string;
  productModel: string;
  softwareModel: string;
  hardwareModel: string;
  osFullName: string;
  displayVersion: string;
  bootloaderVersion: string;
  abiList: string;
  securityPatchTag: string;
  udid: string;
}

export interface BatteryStatus {
  batterySOC: number;        // 电池电量百分比
  chargingStatus: number;    // 充电状态
  healthState: number;       // 电池健康状态
  pluggedType: number;       // 充电器类型
  voltage: number;           // 电池电压
  technology: string;        // 电池技术
  batteryTemperature: number; // 电池温度
}

export interface DisplayInfo {
  id: number;
  width: number;
  height: number;
  densityDPI: number;
  densityPixels: number;
  scaledDensity: number;
  xDPI: number;
  yDPI: number;
}

export interface NetworkStatus {
  isConnected: boolean;
  networkType: string;
  isMetered: boolean;
  isRoaming: boolean;
}

export class SystemServiceManager {
  private static instance: SystemServiceManager | null = null;
  private context: common.UIAbilityContext | null = null;

  private constructor() {}

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
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 系统服务管理器初始化成功');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 系统服务管理器初始化失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 获取设备信息
   */
  async getDeviceInfo(): Promise<SystemInfo> {
    try {
      const info: SystemInfo = {
        deviceType: deviceInfo.deviceType,
        manufacture: deviceInfo.manufacture,
        brand: deviceInfo.brand,
        marketName: deviceInfo.marketName,
        productSeries: deviceInfo.productSeries,
        productModel: deviceInfo.productModel,
        softwareModel: deviceInfo.softwareModel,
        hardwareModel: deviceInfo.hardwareModel,
        osFullName: deviceInfo.osFullName,
        displayVersion: deviceInfo.displayVersion,
        bootloaderVersion: deviceInfo.bootloaderVersion,
        abiList: deviceInfo.abiList,
        securityPatchTag: deviceInfo.securityPatchTag,
        udid: deviceInfo.udid
      };

      hilog.info(DOMAIN_NUMBER, TAG, `设备信息获取成功: ${info.brand} ${info.productModel}`);
      return info;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 获取设备信息失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 获取电池状态
   */
  async getBatteryStatus(): Promise<BatteryStatus> {
    try {
      const status: BatteryStatus = {
        batterySOC: batteryInfo.batterySOC,
        chargingStatus: batteryInfo.chargingStatus,
        healthState: batteryInfo.healthState,
        pluggedType: batteryInfo.pluggedType,
        voltage: batteryInfo.voltage,
        technology: batteryInfo.technology,
        batteryTemperature: batteryInfo.batteryTemperature
      };

      hilog.info(DOMAIN_NUMBER, TAG, `电池状态: ${status.batterySOC}%`);
      return status;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 获取电池状态失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 获取显示信息
   */
  async getDisplayInfo(): Promise<DisplayInfo> {
    try {
      const defaultDisplay = await display.getDefaultDisplaySync();
      
      const info: DisplayInfo = {
        id: defaultDisplay.id,
        width: defaultDisplay.width,
        height: defaultDisplay.height,
        densityDPI: defaultDisplay.densityDPI,
        densityPixels: defaultDisplay.densityPixels,
        scaledDensity: defaultDisplay.scaledDensity,
        xDPI: defaultDisplay.xDPI,
        yDPI: defaultDisplay.yDPI
      };

      hilog.info(DOMAIN_NUMBER, TAG, `显示信息: ${info.width}x${info.height}`);
      return info;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 获取显示信息失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 获取网络状态
   */
  async getNetworkStatus(): Promise<NetworkStatus> {
    try {
      // 这里需要根据实际的HarmonyOS网络API进行实现
      // 由于API可能会变化，这里提供一个基础实现框架
      const status: NetworkStatus = {
        isConnected: true, // 需要通过实际API获取
        networkType: 'wifi', // 需要通过实际API获取
        isMetered: false, // 需要通过实际API获取
        isRoaming: false // 需要通过实际API获取
      };

      hilog.info(DOMAIN_NUMBER, TAG, `网络状态: ${status.networkType}`);
      return status;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 获取网络状态失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 获取系统时间
   */
  async getSystemTime(): Promise<number> {
    try {
      const time = await systemDateTime.getCurrentTime();
      hilog.info(DOMAIN_NUMBER, TAG, `系统时间: ${new Date(time).toISOString()}`);
      return time;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 获取系统时间失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 触发振动
   */
  async vibrate(duration: number = 200): Promise<void> {
    try {
      await vibrator.startVibration({
        type: 'time',
        duration: duration
      });
      
      hilog.info(DOMAIN_NUMBER, TAG, `振动触发成功: ${duration}ms`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 触发振动失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 停止振动
   */
  async stopVibration(): Promise<void> {
    try {
      await vibrator.stopVibration();
      hilog.info(DOMAIN_NUMBER, TAG, '振动已停止');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 停止振动失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 监听传感器数据（示例：加速度传感器）
   */
  async startAccelerometerSensor(callback: (data: sensor.AccelerometerResponse) => void): Promise<void> {
    try {
      await sensor.on(sensor.SensorId.ACCELEROMETER, callback, {
        interval: 100000000 // 100ms
      });
      
      hilog.info(DOMAIN_NUMBER, TAG, '加速度传感器监听已开启');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 开启加速度传感器失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 停止传感器监听
   */
  async stopAccelerometerSensor(): Promise<void> {
    try {
      await sensor.off(sensor.SensorId.ACCELEROMETER);
      hilog.info(DOMAIN_NUMBER, TAG, '加速度传感器监听已停止');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 停止加速度传感器失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取综合系统状态
   */
  async getSystemStatus(): Promise<{
    device: SystemInfo;
    battery: BatteryStatus;
    display: DisplayInfo;
    network: NetworkStatus;
    timestamp: number;
  }> {
    try {
      const [device, battery, display, network, timestamp] = await Promise.all([
        this.getDeviceInfo(),
        this.getBatteryStatus(),
        this.getDisplayInfo(),
        this.getNetworkStatus(),
        this.getSystemTime()
      ]);

      return {
        device,
        battery,
        display,
        network,
        timestamp
      };
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 获取系统状态失败: ${JSON.stringify(error)}`);
      throw error;
    }
  }

  /**
   * 清理资源
   */
  async cleanup(): Promise<void> {
    try {
      await this.stopVibration();
      await this.stopAccelerometerSensor();
      this.context = null;
      
      hilog.info(DOMAIN_NUMBER, TAG, '✅ 系统服务管理器清理完成');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `❌ 系统服务管理器清理失败: ${JSON.stringify(error)}`);
    }
  }
}
