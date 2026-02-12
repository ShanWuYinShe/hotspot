import { hilog } from '@kit.PerformanceAnalysisKit';
import { LocationInfo, LocationUtil } from '../location/LocationUtil';

const DOMAIN_NUMBER: number = 0xFF25;
const TAG: string = 'CoordinateConverter';

// WGS84 → GCJ02 转换常量
const TRANSFORM_PI: number = Math.PI;
const SEMI_MAJOR_AXIS: number = 6378245.0;
const ECCENTRICITY_SQ: number = 0.00669342162296594323;

export interface CoordinatePoint {
  latitude: number;
  longitude: number;
}

export class CoordinateConverter {
  /**
   * 使用数学算法将 WGS84 坐标转换为 GCJ02（国测局坐标）。
   * 采用业界通用的偏移公式，不依赖任何 Map Kit API。
   * 若当前坐标不在中国境内或转换失败，则返回原始值。
   */
  static convertWgs84ToGcj02(lat: number, lng: number): CoordinatePoint {
    if (!CoordinateConverter.isInsideChina(lat, lng)) {
      return { latitude: lat, longitude: lng };
    }

    try {
      let dLat = CoordinateConverter.transformLat(lng - 105.0, lat - 35.0);
      let dLng = CoordinateConverter.transformLng(lng - 105.0, lat - 35.0);
      const radLat = lat / 180.0 * TRANSFORM_PI;
      let magic = Math.sin(radLat);
      magic = 1 - ECCENTRICITY_SQ * magic * magic;
      const sqrtMagic = Math.sqrt(magic);
      dLat = (dLat * 180.0) / ((SEMI_MAJOR_AXIS * (1 - ECCENTRICITY_SQ)) / (magic * sqrtMagic) * TRANSFORM_PI);
      dLng = (dLng * 180.0) / (SEMI_MAJOR_AXIS / sqrtMagic * Math.cos(radLat) * TRANSFORM_PI);
      return {
        latitude: lat + dLat,
        longitude: lng + dLng
      };
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `convertWgs84ToGcj02 failed: ${JSON.stringify(error)}`);
      return { latitude: lat, longitude: lng };
    }
  }

  /**
   * 获取当前位置并自动转换为 GCJ02 坐标。
   * 通过 geoLocationManager 获取 WGS84 坐标后，使用纯数学算法转换。
   */
  static async getConvertedLocation(): Promise<LocationInfo | null> {
    const location = await LocationUtil.getCurrentLocation();
    if (!location) {
      return null;
    }

    const converted = CoordinateConverter.convertWgs84ToGcj02(
      location.latitude,
      location.longitude
    );

    const result: LocationInfo = {
      latitude: converted.latitude,
      longitude: converted.longitude,
      accuracy: location.accuracy,
      altitude: location.altitude,
      speed: location.speed,
      bearing: location.bearing,
      timestamp: location.timestamp,
      locationName: location.locationName
    };
    return result;
  }

  private static isInsideChina(lat: number, lng: number): boolean {
    return lat >= 0.8293 && lat <= 55.8271 && lng >= 72.004 && lng <= 137.8347;
  }

  private static transformLat(x: number, y: number): number {
    let ret = -100.0 + 2.0 * x + 3.0 * y + 0.2 * y * y + 0.1 * x * y +
      0.2 * Math.sqrt(Math.abs(x));
    ret += (20.0 * Math.sin(6.0 * x * TRANSFORM_PI) +
      20.0 * Math.sin(2.0 * x * TRANSFORM_PI)) * 2.0 / 3.0;
    ret += (20.0 * Math.sin(y * TRANSFORM_PI) +
      40.0 * Math.sin(y / 3.0 * TRANSFORM_PI)) * 2.0 / 3.0;
    ret += (160.0 * Math.sin(y / 12.0 * TRANSFORM_PI) +
      320.0 * Math.sin(y * TRANSFORM_PI / 30.0)) * 2.0 / 3.0;
    return ret;
  }

  private static transformLng(x: number, y: number): number {
    let ret = 300.0 + x + 2.0 * y + 0.1 * x * x + 0.1 * x * y +
      0.1 * Math.sqrt(Math.abs(x));
    ret += (20.0 * Math.sin(6.0 * x * TRANSFORM_PI) +
      20.0 * Math.sin(2.0 * x * TRANSFORM_PI)) * 2.0 / 3.0;
    ret += (20.0 * Math.sin(x * TRANSFORM_PI) +
      40.0 * Math.sin(x / 3.0 * TRANSFORM_PI)) * 2.0 / 3.0;
    ret += (150.0 * Math.sin(x / 12.0 * TRANSFORM_PI) +
      300.0 * Math.sin(x / 30.0 * TRANSFORM_PI)) * 2.0 / 3.0;
    return ret;
  }
}
