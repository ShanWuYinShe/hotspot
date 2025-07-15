/**
 * 地理位置相关业务类型定义
 */

// 导入通用位置类型
export type { Location, Address } from './common'

// 位置数据（包含更多元信息）
export interface LocationData {
  latitude: number;
  longitude: number;
  accuracy?: number;
  timestamp?: number;
  locationName?: string;
}

// 地理编码结果
export interface GeocodeResult {
  address: string;
  location: Location;
}

// 反向地理编码结果
export interface ReverseGeocodeResult {
  address: string;
  city?: string;
  district?: string;
  province?: string;
}

// 地理位置权限状态
export enum GeolocationPermissionStatus {
  GRANTED = 'granted',
  DENIED = 'denied',
  PROMPT = 'prompt'
}

// 地图标记类型
export interface MapMarker {
  id: string;
  position: Location;
  title?: string;
  icon?: string;
  content?: string;
}
