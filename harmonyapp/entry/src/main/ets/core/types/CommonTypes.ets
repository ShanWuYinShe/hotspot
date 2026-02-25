/**
 * 客户端本地扩展类型
 * 
 * 这些类型仅用于 HarmonyOS 客户端组件和逻辑，不是 API 文档定义的类型
 * API 文档定义的类型请查看 api.ets
 * 用户类型工具函数请查看 ../utils/UserTypeUtil.ets
 */

// 重新导出用户类型工具函数，方便使用
export { 
  UserTypeUtil, 
  UserType, 
  SYSTEM_USER_IDS, 
  SYSTEM_USER_ID_MAX 
} from '../utils/UserTypeUtil';

/**
 * 便捷函数：判断是否为临时用户
 * @param userType 用户类型
 */
export function isTempUser(userType: string): boolean {
  return userType === 'temp';
}

/**
 * 便捷函数：判断是否为普通用户
 * @param userType 用户类型
 */
export function isNormalUser(userType: string): boolean {
  return userType === 'normal';
}

/**
 * 便捷函数：判断是否为系统用户
 * @param userType 用户类型
 */
export function isSystemUser(userType: string): boolean {
  return userType === 'system';
}

/**
 * 便捷函数：判断是否为管理员用户
 * @param userType 用户类型
 */
export function isAdminUser(userType: string): boolean {
  return userType === 'admin';
}

/**
 * 便捷函数：获取用户类型显示名称
 * @param userType 用户类型
 */
export function getUserTypeDisplayName(userType: string): string {
  switch (userType) {
    case 'temp':
      return '临时用户';
    case 'normal':
      return '普通用户';
    case 'admin':
      return '管理员';
    case 'system':
      return '系统用户';
    default:
      return '未知类型';
  }
}

/**
 * 便捷函数：检查用户是否具有指定权限
 * @param userType 用户类型
 * @param permission 权限名称
 */
export function hasUserPermission(userType: string, permission: string): boolean {
  const UserTypeUtilModule = require('../utils/UserTypeUtil');
  const UserTypeUtil = UserTypeUtilModule.UserTypeUtil;
  return UserTypeUtil.hasPermission(userType, permission);
}

// ============================================================================
// 地理位置类型（客户端扩展）
// ============================================================================

// 地理坐标
export interface Coordinates {
  latitude: number;         // 纬度，范围：[-90, 90]
  longitude: number;        // 经度，范围：[-180, 180]
}

// 错误响应详情
export interface ErrorDetail {
  field?: string;          // 错误字段
  message: string;         // 错误消息
  code?: string;          // 错误代码
}

// 文件上传响应
export interface FileUploadResponse {
  file_id: string;        // 文件ID
  file_url: string;       // 文件访问URL
  file_name: string;      // 文件名
  file_size: number;      // 文件大小（字节）
  mime_type: string;      // MIME类型
  upload_time: string;    // 上传时间（ISO 8601）
}

// 系统配置
export interface SystemConfig {
  max_file_size: number;      // 最大文件大小（字节）
  max_image_count: number;    // 最大图片数量
  supported_formats: string[]; // 支持的文件格式
  api_version: string;        // API版本
}

// 应用状态
export interface AppState {
  is_online: boolean;         // 是否在线
  network_type: string;       // 网络类型
  location_enabled: boolean;  // 位置服务是否启用
  notification_enabled: boolean; // 通知是否启用
}

// 设备信息
export interface DeviceInfo {
  device_id: string;         // 设备ID
  device_type: string;       // 设备类型
  os_version: string;        // 操作系统版本
  app_version: string;       // 应用版本
  screen_width: number;      // 屏幕宽度
  screen_height: number;     // 屏幕高度
  screen_density: number;    // 屏幕密度
}