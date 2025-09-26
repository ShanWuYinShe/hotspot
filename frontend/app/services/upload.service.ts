/**
 * 文件上传服务 - 与后端upload_controller.rs保持一致
 * 提供头像上传、图片批量上传等功能
 */

import { UPLOAD_ENDPOINTS, buildUploadUrl } from "~/config/api";
import type { 
  UploadResponse, 
  BatchUploadResponse, 
  UploadConfig 
} from "~/types/api";

export const useUploadService = () => {
  const api = useApi();
  const { handleError } = useErrorHandler();

  // 默认上传配置 - 与后端保持一致
  const DEFAULT_CONFIG: UploadConfig = {
    max_file_size: 10 * 1024 * 1024, // 10MB
    allowed_types: [
      'image/jpeg',
      'image/png', 
      'image/gif',
      'image/webp'
    ],
    max_files_per_batch: 9, // 批量上传最多9张图片
    // 兼容属性
    maxFileSize: 10 * 1024 * 1024,
    allowedTypes: [
      'image/jpeg',
      'image/png', 
      'image/gif',
      'image/webp'
    ],
    maxFiles: 9
  };

  /**
   * 验证文件
   */
  const validateFile = (file: File, config: UploadConfig = DEFAULT_CONFIG): void => {
    // 检查文件大小
    if (file.size > config.maxFileSize) {
      throw new Error(`文件大小超过限制（${Math.round(config.maxFileSize / 1024 / 1024)}MB）`);
    }

    // 检查文件类型
    if (!config.allowedTypes.includes(file.type)) {
      throw new Error(`不支持的文件类型：${file.type}`);
    }
  };

  /**
   * 上传头像
   * @param file 头像文件
   * @returns 上传结果
   */
  const uploadAvatar = async (file: File): Promise<UploadResponse> => {
    try {
      // 验证文件
      validateFile(file);

      // 创建FormData
      const formData = new FormData();
      formData.append('avatar', file);

      // 发送上传请求
      const response = await api.postForm(
        buildUploadUrl(UPLOAD_ENDPOINTS.PROTECTED.AVATAR),
        formData,
        { auth: true }
      );

      return response as UploadResponse;
    } catch (error: any) {
      handleError(error, '头像上传');
      throw error;
    }
  };

  /**
   * 批量上传图片
   * @param files 图片文件数组
   * @param config 上传配置
   * @returns 批量上传结果
   */
  const uploadImages = async (
    files: File[], 
    config: UploadConfig = DEFAULT_CONFIG
  ): Promise<BatchUploadResponse> => {
    try {
      // 检查文件数量
      if (config.maxFiles && files.length > config.maxFiles) {
        throw new Error(`最多只能上传${config.maxFiles}个文件`);
      }

      // 验证所有文件
      for (const file of files) {
        validateFile(file, config);
      }

      // 创建FormData
      const formData = new FormData();
      files.forEach(file => {
        formData.append('images', file);
      });

      // 发送上传请求
      const response = await api.postForm(
        buildUploadUrl(UPLOAD_ENDPOINTS.PROTECTED.IMAGES),
        formData,
        { auth: true }
      );

      return response as BatchUploadResponse;
    } catch (error: any) {
      handleError(error, '图片上传');
      throw error;
    }
  };

  /**
   * 上传单张图片（便捷方法）
   * @param file 图片文件
   * @returns 上传结果
   */
  const uploadImage = async (file: File): Promise<UploadResponse> => {
    const result = await uploadImages([file]);
    
    if (result.upload_errors && result.upload_errors.length > 0) {
      throw new Error(result.upload_errors[0]);
    }
    
    if (!result.uploaded_files || result.uploaded_files.length === 0) {
      throw new Error('上传失败');
    }
    
    return result.uploaded_files[0];
  };

  /**
   * 获取文件预览URL
   * @param file 文件对象
   * @returns 预览URL
   */
  const getPreviewUrl = (file: File): string => {
    return URL.createObjectURL(file);
  };

  /**
   * 释放预览URL
   * @param url 预览URL
   */
  const revokePreviewUrl = (url: string): void => {
    URL.revokeObjectURL(url);
  };

  /**
   * 格式化文件大小
   * @param bytes 字节数
   * @returns 格式化的文件大小
   */
  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return {
    uploadAvatar,
    uploadImages,
    uploadImage,
    validateFile,
    getPreviewUrl,
    revokePreviewUrl,
    formatFileSize,
    DEFAULT_CONFIG
  };
};


