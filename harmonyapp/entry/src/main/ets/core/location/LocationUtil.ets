/**
 * 位置工具类
 * 提供位置获取、地址解析（正/反向地理编码）、距离显示格式化等功能
 */

import { geoLocationManager } from '@kit.LocationKit';
import { abilityAccessCtrl } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { common } from '@kit.AbilityKit';

const DOMAIN_NUMBER: number = 0xFF16;
const TAG: string = 'LocationUtil';

/**
 * 位置信息接口
 */
export interface LocationInfo {
  latitude: number;
  longitude: number;
  accuracy?: number;
  altitude?: number;
  speed?: number;
  bearing?: number;
  timestamp?: number;
  locationName?: string;
}

/**
 * 地址信息接口
 */
export interface AddressInfo {
  country?: string;
  province?: string;
  city?: string;
  district?: string;
  street?: string;
  streetNumber?: string;
  fullAddress?: string;
}

/**
 * 位置监听回调接口
 */
export type LocationCallback = (location: LocationInfo) => void;

/**
 * 位置工具类
 */
export class LocationUtil {
  private static isListening: boolean = false;
  private static locationCallback: LocationCallback | null = null;

  /**
   * 请求位置权限
   */
  static async requestLocationPermission(context: common.UIAbilityContext): Promise<boolean> {
    try {
      const atManager = abilityAccessCtrl.createAtManager();
      
      // 优先请求精确位置权限
      let result = await atManager.checkAccessToken(
        context.applicationInfo.accessTokenId, 
        'ohos.permission.LOCATION'
      );
      
      if (result === abilityAccessCtrl.GrantStatus.PERMISSION_GRANTED) {
        hilog.info(DOMAIN_NUMBER, TAG, '精确位置权限已授权');
        return true;
      }

      // 请求精确位置权限
      const requestResult = await atManager.requestPermissionsFromUser(
        context, 
        ['ohos.permission.LOCATION']
      );
      
      if (requestResult.authResults[0] === abilityAccessCtrl.GrantStatus.PERMISSION_GRANTED) {
        hilog.info(DOMAIN_NUMBER, TAG, '精确位置权限授权成功');
        return true;
      }

      // 如果精确位置被拒绝，尝试请求大概位置权限
      result = await atManager.checkAccessToken(
        context.applicationInfo.accessTokenId, 
        'ohos.permission.APPROXIMATELY_LOCATION'
      );
      
      if (result === abilityAccessCtrl.GrantStatus.PERMISSION_GRANTED) {
        hilog.info(DOMAIN_NUMBER, TAG, '大概位置权限已授权');
        return true;
      }

      const approximateResult = await atManager.requestPermissionsFromUser(
        context, 
        ['ohos.permission.APPROXIMATELY_LOCATION']
      );
      
      const granted = approximateResult.authResults[0] === abilityAccessCtrl.GrantStatus.PERMISSION_GRANTED;
      hilog.info(DOMAIN_NUMBER, TAG, `大概位置权限请求结果: ${granted}`);
      return granted;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `请求位置权限失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 获取当前位置
   * 使用 SingleLocationRequest（API 12+）替代废弃的 CurrentLocationRequest，
   * 优先精度模式，与地图蓝点使用相同的高精度定位策略。
   */
  static async getCurrentLocation(): Promise<LocationInfo | null> {
    try {
      hilog.info(DOMAIN_NUMBER, TAG, '开始获取当前位置');

      // SingleLocationRequest（API 12+）：优先精度，与 Map Kit 蓝点定位一致
      const requestInfo: geoLocationManager.SingleLocationRequest = {
        locatingTimeoutMs: 10000,
        locatingPriority: geoLocationManager.LocatingPriority.PRIORITY_ACCURACY
      };

      // 获取位置
      const location = await geoLocationManager.getCurrentLocation(requestInfo);
      
      const locationInfo: LocationInfo = {
        latitude: location.latitude,
        longitude: location.longitude,
        accuracy: location.accuracy,
        altitude: location.altitude,
        speed: location.speed,
        bearing: location.direction,
        timestamp: location.timeStamp,
        locationName: '当前位置'
      };

      // 尝试获取地址信息
      try {
        const addressInfo = await LocationUtil.reverseGeocode(location.latitude, location.longitude);
        if (addressInfo && addressInfo.fullAddress) {
          locationInfo.locationName = addressInfo.fullAddress;
        }
      } catch (error) {
        hilog.warn(DOMAIN_NUMBER, TAG, `获取地址信息失败: ${JSON.stringify(error)}`);
        locationInfo.locationName = '未知位置';
      }

      hilog.info(DOMAIN_NUMBER, TAG, `位置获取成功: ${locationInfo.latitude}, ${locationInfo.longitude}`);
      return locationInfo;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取当前位置失败: ${JSON.stringify(error)}`);
      return null;
    }
  }

  /**
   * 开始位置监听
   */
  static async startLocationListening(callback: LocationCallback): Promise<boolean> {
    try {
      if (LocationUtil.isListening) {
        hilog.warn(DOMAIN_NUMBER, TAG, '位置监听已在进行中');
        return false;
      }

      hilog.info(DOMAIN_NUMBER, TAG, '开始位置监听');

      // ContinuousLocationRequest（API 12+）：替代废弃的 LocationRequest
      // 使用 NAVIGATION 场景以获得最高定位精度（GNSS + WiFi + 基站多源融合），
      // 确保持续定位精度与 MapKit 蓝点一致。
      const requestInfo: geoLocationManager.ContinuousLocationRequest = {
        interval: 5, // 5 秒更新一次（单位：秒）
        locationScenario: geoLocationManager.UserActivityScenario.NAVIGATION
      };

      LocationUtil.locationCallback = callback;

      geoLocationManager.on('locationChange', requestInfo, (location: geoLocationManager.Location) => {
        const locationInfo: LocationInfo = {
          latitude: location.latitude,
          longitude: location.longitude,
          accuracy: location.accuracy,
          altitude: location.altitude,
          speed: location.speed,
          bearing: location.direction,
          timestamp: location.timeStamp,
          locationName: '实时位置'
        };

        // 异步获取地址信息
        LocationUtil.reverseGeocode(location.latitude, location.longitude)
          .then(addressInfo => {
            if (addressInfo && addressInfo.fullAddress) {
              locationInfo.locationName = addressInfo.fullAddress;
            }
          })
          .catch((error: Error) => {
            hilog.warn(DOMAIN_NUMBER, TAG, `获取地址信息失败: ${JSON.stringify(error)}`);
          })
          .finally(() => {
            if (LocationUtil.locationCallback) {
              LocationUtil.locationCallback(locationInfo);
            }
          });
      });

      LocationUtil.isListening = true;
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
  static async stopLocationListening(): Promise<boolean> {
    try {
      if (!LocationUtil.isListening) {
        hilog.warn(DOMAIN_NUMBER, TAG, '位置监听未在进行中');
        return false;
      }

      geoLocationManager.off('locationChange');
      LocationUtil.isListening = false;
      LocationUtil.locationCallback = null;
      
      hilog.info(DOMAIN_NUMBER, TAG, '位置监听已停止');
      return true;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `停止位置监听失败: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 反向地理编码（坐标转地址）
   */
  static async reverseGeocode(latitude: number, longitude: number): Promise<AddressInfo | null> {
    try {
      hilog.debug(DOMAIN_NUMBER, TAG, `反向地理编码: ${latitude}, ${longitude}`);

      const reverseRequest: geoLocationManager.ReverseGeoCodeRequest = {
        latitude: latitude,
        longitude: longitude,
        maxItems: 1
      };

      const geoAddresses = await geoLocationManager.getAddressesFromLocation(reverseRequest);

      if (!geoAddresses || geoAddresses.length === 0) {
        hilog.warn(DOMAIN_NUMBER, TAG, '未找到地址信息');
        return null;
      }

      const geoAddress = geoAddresses[0];
      const addressInfo: AddressInfo = {
        country: geoAddress.countryName,
        province: geoAddress.administrativeArea,
        city: geoAddress.locality,
        district: geoAddress.subLocality,
        street: geoAddress.roadName,
        streetNumber: geoAddress.subRoadName,
        fullAddress: geoAddress.placeName
      };

      hilog.debug(DOMAIN_NUMBER, TAG, `地址解析成功: ${addressInfo.fullAddress}`);
      return addressInfo;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `反向地理编码失败: ${JSON.stringify(error)}`);
      return null;
    }
  }

  /**
   * 正向地理编码（地址转坐标）
   */
  static async geocode(address: string): Promise<LocationInfo | null> {
    try {
      hilog.debug(DOMAIN_NUMBER, TAG, `正向地理编码: ${address}`);

      const geocodeRequest: geoLocationManager.GeoCodeRequest = {
        description: address,
        maxItems: 1
      };

      const geoAddresses = await geoLocationManager.getAddressesFromLocationName(geocodeRequest);
      
      if (!geoAddresses || geoAddresses.length === 0) {
        hilog.warn(DOMAIN_NUMBER, TAG, '未找到位置信息');
        return null;
      }

      const geoAddress = geoAddresses[0];
      const locationInfo: LocationInfo = {
        latitude: geoAddress.latitude || 0,
        longitude: geoAddress.longitude || 0,
        locationName: geoAddress.placeName
      };

      hilog.debug(DOMAIN_NUMBER, TAG, `位置解析成功: ${locationInfo.latitude}, ${locationInfo.longitude}`);
      return locationInfo;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `正向地理编码失败: ${JSON.stringify(error)}`);
      return null;
    }
  }

  /**
   * 正向地理编码（地址转坐标）- 多结果版本
   * 返回多个匹配位置，供用户选择
   */
  static async geocodeMultiple(address: string, maxItems: number = 10): Promise<LocationInfo[]> {
    try {
      hilog.debug(DOMAIN_NUMBER, TAG, `正向地理编码（多结果）: ${address}`);

      const geocodeRequest: geoLocationManager.GeoCodeRequest = {
        description: address,
        maxItems: maxItems
      };

      const geoAddresses = await geoLocationManager.getAddressesFromLocationName(geocodeRequest);

      if (!geoAddresses || geoAddresses.length === 0) {
        hilog.warn(DOMAIN_NUMBER, TAG, '未找到匹配位置');
        return [];
      }

      const results: LocationInfo[] = [];
      for (const addr of geoAddresses) {
        if (addr.latitude !== undefined && addr.longitude !== undefined &&
          LocationUtil.isValidCoordinate(addr.latitude, addr.longitude)) {
          results.push({
            latitude: addr.latitude,
            longitude: addr.longitude,
            locationName: addr.placeName || `${addr.locality || ''}${addr.subLocality || ''}${addr.roadName || ''}`
          });
        }
      }

      hilog.debug(DOMAIN_NUMBER, TAG, `地理编码找到 ${results.length} 个结果`);
      return results;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `正向地理编码（多结果）失败: ${JSON.stringify(error)}`);
      return [];
    }
  }

  /**
   * 格式化距离显示
   *
   * 注意：这里仅做“显示格式化”，距离值必须来自后端返回的 RelativeLocation.distance（米），
   * 客户端严禁基于经纬度自行计算距离（例如 Haversine/地球半径等）。
   */
  static formatDistance(distance: number): string {
    if (distance < 1000) {
      return `${distance}m`;
    } else if (distance < 10000) {
      return `${(distance / 1000).toFixed(1)}km`;
    } else {
      return `${Math.round(distance / 1000)}km`;
    }
  }

  /**
   * 获取位置精度描述
   */
  static getAccuracyDescription(accuracy: number): string {
    if (accuracy <= 5) {
      return '高精度';
    } else if (accuracy <= 20) {
      return '中等精度';
    } else if (accuracy <= 100) {
      return '低精度';
    } else {
      return '精度较差';
    }
  }

  /**
   * 验证坐标有效性
   */
  static isValidCoordinate(latitude: number, longitude: number): boolean {
    return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180;
  }

  /**
   * 生成位置摘要
   */
  static generateLocationSummary(location: LocationInfo): string {
    const parts: string[] = [];
    
    if (location.locationName) {
      parts.push(location.locationName);
    }
    
    parts.push(`${location.latitude.toFixed(6)}, ${location.longitude.toFixed(6)}`);
    
    if (location.accuracy) {
      parts.push(`精度: ${location.accuracy}m`);
    }
    
    return parts.join(' | ');
  }

  /**
   * 清理位置服务资源
   */
  static async cleanup(): Promise<void> {
    try {
      if (LocationUtil.isListening) {
        await LocationUtil.stopLocationListening();
      }
      hilog.info(DOMAIN_NUMBER, TAG, '位置服务资源清理完成');
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `清理位置服务资源失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 根据经纬度获取位置名称
   */
  static async getLocationName(latitude: number, longitude: number): Promise<string | null> {
    try {
      const reverseGeoCodeRequest: geoLocationManager.ReverseGeoCodeRequest = {
        latitude: latitude,
        longitude: longitude,
        maxItems: 1
      };

      const addressInfo = await geoLocationManager.getAddressesFromLocation(reverseGeoCodeRequest);
      
      if (addressInfo && addressInfo.length > 0) {
        const address = addressInfo[0];
        // 构建位置名称
        let locationName = '';
        
        if (address.subAdministrativeArea) {
          locationName += address.subAdministrativeArea;
        }
        if (address.locality && address.locality !== address.subAdministrativeArea) {
          locationName += (locationName ? ', ' : '') + address.locality;
        }
        if (address.subLocality) {
          locationName += (locationName ? ', ' : '') + address.subLocality;
        }
        if (address.roadName) {
          locationName += (locationName ? ', ' : '') + address.roadName;
        }
        
        return locationName || '未知位置';
      }
      
      return null;
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `获取位置名称失败: ${JSON.stringify(error)}`);
      return null;
    }
  }
}