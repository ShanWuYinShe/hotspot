/**
 * 统一的格式化工具函数
 * 用于替换项目中重复的格式化逻辑
 */

/**
 * 格式化时间为相对时间显示
 * @param dateString ISO 8601 时间字符串
 * @returns 格式化后的相对时间字符串
 */
export const formatTime = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);
  
  if (diffInSeconds < 60) return '刚刚';
  if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}分钟前`;
  if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}小时前`;
  if (diffInSeconds < 2592000) return `${Math.floor(diffInSeconds / 86400)}天前`;
  if (diffInSeconds < 31536000) return `${Math.floor(diffInSeconds / 2592000)}个月前`;
  return `${Math.floor(diffInSeconds / 31536000)}年前`;
};

/**
 * 格式化距离为可读的距离字符串
 * @param distance 距离（米）
 * @returns 格式化后的距离字符串
 */
export const formatDistance = (distance: number): string => {
  if (distance < 1000) {
    return `${Math.round(distance)}m`;
  } else {
    return `${(distance / 1000).toFixed(1)}km`;
  }
};

/**
 * 格式化日期为可读的日期字符串
 * @param dateString ISO 8601 时间字符串
 * @param format 格式类型：'short' | 'long' | 'time'
 * @returns 格式化后的日期字符串
 */
export const formatDate = (dateString: string, format: 'short' | 'long' | 'time' = 'short'): string => {
  const date = new Date(dateString);
  
  switch (format) {
    case 'long':
      return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
    case 'time':
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit'
      });
    case 'short':
    default:
      return date.toLocaleDateString('zh-CN', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      });
  }
};

/**
 * 格式化数字为可读的数量字符串（如点赞数、成员数等）
 * @param count 数量
 * @returns 格式化后的数量字符串
 */
export const formatCount = (count: number): string => {
  if (count < 1000) {
    return count.toString();
  } else if (count < 10000) {
    return `${(count / 1000).toFixed(1)}k`;
  } else if (count < 100000000) {
    return `${Math.floor(count / 10000)}万`;
  } else {
    return `${(count / 100000000).toFixed(1)}亿`;
  }
};

/**
 * 格式化文件大小
 * @param bytes 字节数
 * @returns 格式化后的文件大小字符串
 */
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
};
