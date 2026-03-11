/**
 * 国内坐标系转换工具（WGS-84 <-> GCJ-02）。
 *
 * 约定：
 * - 后端 / 类型定义中的 GeoPoint 默认使用 WGS-84。
 * - 地图 SDK（高德 AMap）使用 GCJ-02。
 * - 在前端，只在与地图交互的边界做坐标系转换：
 *   - 传入 AMap 之前：WGS-84 -> GCJ-02。
 *   - 从 AMap 读取坐标用于业务 / 回传后端时：GCJ-02 -> WGS-84。
 */

export interface GeoPointLike {
  latitude: number;
  longitude: number;
}

const PI = Math.PI;
const A = 6378245.0; // 地球半长轴
const EE = 0.00669342162296594323; // 第一偏心率平方

// 是否在中国范围之外（简单包围盒），在国外无需 GCJ 偏移
function outOfChina(lat: number, lng: number): boolean {
  return lng < 72.004 || lng > 137.8347 || lat < 0.8293 || lat > 55.8271;
}

function transformLat(x: number, y: number): number {
  let ret = -100.0 + 2.0 * x + 3.0 * y + 0.2 * y * y + 0.1 * x * y + 0.2 * Math.sqrt(Math.abs(x));
  ret += (20.0 * Math.sin(6.0 * x * PI) + 20.0 * Math.sin(2.0 * x * PI)) * 2.0 / 3.0;
  ret += (20.0 * Math.sin(y * PI) + 40.0 * Math.sin(y / 3.0 * PI)) * 2.0 / 3.0;
  ret += (160.0 * Math.sin(y / 12.0 * PI) + 320.0 * Math.sin(y * PI / 30.0)) * 2.0 / 3.0;
  return ret;
}

function transformLng(x: number, y: number): number {
  let ret = 300.0 + x + 2.0 * y + 0.1 * x * x + 0.1 * x * y + 0.1 * Math.sqrt(Math.abs(x));
  ret += (20.0 * Math.sin(6.0 * x * PI) + 20.0 * Math.sin(2.0 * x * PI)) * 2.0 / 3.0;
  ret += (20.0 * Math.sin(x * PI) + 40.0 * Math.sin(x / 3.0 * PI)) * 2.0 / 3.0;
  ret += (150.0 * Math.sin(x / 12.0 * PI) + 300.0 * Math.sin(x / 30.0 * PI)) * 2.0 / 3.0;
  return ret;
}

/**
 * WGS-84 -> GCJ-02
 */
export function wgs84ToGcj02(point: GeoPointLike): GeoPointLike {
  const { latitude: lat, longitude: lng } = point;
  if (outOfChina(lat, lng)) {
    return { latitude: lat, longitude: lng };
  }
  let dLat = transformLat(lng - 105.0, lat - 35.0);
  let dLng = transformLng(lng - 105.0, lat - 35.0);
  const radLat = (lat / 180.0) * PI;
  let magic = Math.sin(radLat);
  magic = 1 - EE * magic * magic;
  const sqrtMagic = Math.sqrt(magic);
  dLat = (dLat * 180.0) / (((A * (1 - EE)) / (magic * sqrtMagic)) * PI);
  dLng = (dLng * 180.0) / ((A / sqrtMagic * Math.cos(radLat)) * PI);
  const mgLat = lat + dLat;
  const mgLng = lng + dLng;
  return { latitude: mgLat, longitude: mgLng };
}

/**
 * GCJ-02 -> WGS-84（近似反变换，误差在数米量级）
 */
export function gcj02ToWgs84(point: GeoPointLike): GeoPointLike {
  const { latitude: lat, longitude: lng } = point;
  if (outOfChina(lat, lng)) {
    return { latitude: lat, longitude: lng };
  }
  const converted = wgs84ToGcj02(point);
  // 反推：wgs = gcj * 2 - wgs2gcj(wgs)，常用近似算法，精度足够
  const wgsLat = lat * 2 - converted.latitude;
  const wgsLng = lng * 2 - converted.longitude;
  return { latitude: wgsLat, longitude: wgsLng };
}

/**
 * 小工具：将 WGS-84 GeoPoint 转为 AMap 需要的 [lng, lat]（GCJ-02）。
 */
export function toGcjLngLat(point: GeoPointLike): [number, number] {
  const gcj = wgs84ToGcj02(point);
  return [gcj.longitude, gcj.latitude];
}
