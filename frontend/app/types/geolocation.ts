/**
 * 地理位置相关类型定义
 */

export interface Coordinates {
  latitude: number;
  longitude: number;
  accuracy?: number;
  timestamp?: number;
}
