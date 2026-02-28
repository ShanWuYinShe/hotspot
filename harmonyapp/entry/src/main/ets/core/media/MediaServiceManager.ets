/**
 * HarmonyOS原生多媒体服务工具管理器
 * 充分利用 HarmonyOS 原生 Kit 能力：
 * - ImageKit: 图片压缩、处理
 * - MediaKit: 音频录制
 * - CameraKit: 拍照 (通过 cameraPicker)
 * - CoreFileKit: 文件操作
 * - AbilityKit: 权限管理
 */

import { common, abilityAccessCtrl, Permissions, PermissionRequestResult } from '@kit.AbilityKit';
import { image } from '@kit.ImageKit';
import { media } from '@kit.MediaKit';
import { fileIo } from '@kit.CoreFileKit';
import photoAccessHelper from '@ohos.file.photoAccessHelper';
import cameraPicker from '@ohos.multimedia.cameraPicker'; // 使用原生相机选择器
import { BusinessError } from '@kit.BasicServicesKit';
import { hilog } from '@kit.PerformanceAnalysisKit';

const TAG = 'MediaServiceManager';
const DOMAIN = 0xFF02;

// 录音文件路径映射 (解决 AVRecorder 无法直接获取路径的问题)
const recorderPathMap = new WeakMap<media.AVRecorder, string>();

/**
 * 录音配置接口
 */
export interface AudioConfig {
  sampleRate?: number;
  channels?: number;
  bitrate?: number;
  format?: media.ContainerFormatType;
  encoding?: media.CodecMimeType;
}

/**
 * 媒体文件结果
 */
export interface MediaResult {
  uri: string;
  size: number;
  mimeType: string;
  width?: number; // 仅图片/视频
  height?: number; // 仅图片/视频
  duration?: number; // 仅音频/视频
}

/**
 * 图片尺寸接口
 */
interface ImageDimensions {
  width: number;
  height: number;
}

export class MediaServiceManager {
  private static normalizeUriToFilePath(context: common.Context, uri: string): string {
    if (uri.startsWith('internal://cache/')) {
      const cacheRelativePath = uri.substring('internal://cache/'.length);
      return `${context.cacheDir}/${cacheRelativePath}`;
    }
    if (uri.startsWith('file://')) {
      return uri.substring('file://'.length);
    }
    return uri;
  }

  /**
   * 请求媒体相关权限
   * @param context UIAbilityContext
   */
  public static async requestPermissions(context: common.UIAbilityContext): Promise<boolean> {
    const permissions: Permissions[] = [
      'ohos.permission.CAMERA',
      'ohos.permission.MICROPHONE',
      'ohos.permission.READ_MEDIA',
      'ohos.permission.WRITE_MEDIA'
    ];

    try {
      const atManager = abilityAccessCtrl.createAtManager();
      const result: PermissionRequestResult = await atManager.requestPermissionsFromUser(context, permissions);
      
      const isGranted = result.authResults.every(auth => auth === 0);
      hilog.info(DOMAIN, TAG, `权限请求结果: ${isGranted}`);
      return isGranted;
    } catch (error) {
      hilog.error(DOMAIN, TAG, `权限请求异常: ${JSON.stringify(error)}`);
      return false;
    }
  }

  /**
   * 从相册选择图片
   * 使用 PhotoViewPicker
   */
  public static async selectImages(context: common.Context, maxCount: number = 9): Promise<string[]> {
    try {
      const picker = new photoAccessHelper.PhotoViewPicker();
      const result = await picker.select({
        MIMEType: photoAccessHelper.PhotoViewMIMETypes.IMAGE_TYPE,
        maxSelectNumber: maxCount
      });
      hilog.info(DOMAIN, TAG, `选择了 ${result.photoUris.length} 张图片`);
      return result.photoUris;
    } catch (error) {
      hilog.error(DOMAIN, TAG, `选择图片失败: ${JSON.stringify(error)}`);
      throw (error instanceof Error) ? error : new Error(String(error));
    }
  }

  /**
   * 调用系统相机拍照
   * 使用 cameraPicker (API 11+)
   */
  public static async takePhoto(context: common.Context): Promise<MediaResult | null> {
    try {
      const pickerProfile: cameraPicker.PickerProfile = {
        cameraPosition: 0
      };
      
      const pickerResult = await cameraPicker.pick(context, 
        [cameraPicker.PickerMediaType.PHOTO], 
        pickerProfile
      );

      if (pickerResult.resultUri) {
        // 获取文件信息
        const file = fileIo.openSync(pickerResult.resultUri);
        const stat = fileIo.statSync(file.fd);
        fileIo.closeSync(file);

        // 获取图片尺寸
        const imageInfo = await MediaServiceManager.getImageInfo(pickerResult.resultUri);

        return {
          uri: pickerResult.resultUri,
          size: stat.size,
          mimeType: 'image/jpeg',
          width: imageInfo.width,
          height: imageInfo.height
        };
      }
      return null;
    } catch (error) {
      hilog.error(DOMAIN, TAG, `拍照失败: ${JSON.stringify(error)}`);
      // 降级方案：如果 cameraPicker 不可用，尝试 PhotoViewPicker
      try {
        const uris = await MediaServiceManager.selectImages(context, 1);
        if (uris.length > 0) {
           const imageInfo = await MediaServiceManager.getImageInfo(uris[0]);
           
           // 获取文件大小
           let size = 0;
           try {
             const file = fileIo.openSync(uris[0], fileIo.OpenMode.READ_ONLY);
             const stat = fileIo.statSync(file.fd);
             size = stat.size;
             fileIo.closeSync(file);
           } catch (e) {
             hilog.warn(DOMAIN, TAG, `获取文件大小失败: ${JSON.stringify(e)}`);
           }

           return {
             uri: uris[0],
             size: size,
             mimeType: 'image/jpeg',
             width: imageInfo.width,
             height: imageInfo.height
           }
        }
        return null;
      } catch (e) {
        throw (error instanceof Error) ? error : new Error(String(error));
      }
    }
  }

  /**
   * 开始录音
   * 使用 AVRecorder
   */
  public static async startRecording(context: common.Context, config?: AudioConfig): Promise<media.AVRecorder> {
    try {
      const avRecorder = await media.createAVRecorder();
      
      // 1. 创建录音文件
      const tempDir = context.tempDir;
      const fileName = `audiorecord_${Date.now()}.m4a`;
      const filePath = `${tempDir}/${fileName}`;
      const file = fileIo.openSync(filePath, fileIo.OpenMode.CREATE | fileIo.OpenMode.READ_WRITE);
      
      // 保存路径映射，供 stop 使用
      recorderPathMap.set(avRecorder, filePath);

      // 2. 配置参数
      const avConfig: media.AVRecorderConfig = {
        audioSourceType: media.AudioSourceType.AUDIO_SOURCE_TYPE_MIC,
        profile: {
          audioBitrate: config?.bitrate ?? 48000,
          audioChannels: config?.channels ?? 1,
          audioCodec: config?.encoding ?? media.CodecMimeType.AUDIO_AAC,
          audioSampleRate: config?.sampleRate ?? 48000,
          fileFormat: config?.format ?? media.ContainerFormatType.CFT_MPEG_4A,
        },
        url: `fd://${file.fd}`
      };

      // 3. 准备并启动
      await avRecorder.prepare(avConfig);
      await avRecorder.start();
      
      hilog.info(DOMAIN, TAG, `开始录音: ${filePath}`);
      return avRecorder;
    } catch (error) {
      hilog.error(DOMAIN, TAG, `启动录音失败: ${JSON.stringify(error)}`);
      throw (error instanceof Error) ? error : new Error(String(error));
    }
  }

  /**
   * 停止录音
   */
  public static async stopRecording(recorder: media.AVRecorder): Promise<MediaResult> {
    try {
      await recorder.stop();
      await recorder.release();

      const filePath = recorderPathMap.get(recorder);
      if (!filePath) {
        throw new Error('无法找到录音文件路径');
      }
      
      // 清理映射
      recorderPathMap.delete(recorder);

      // 获取文件详情
      const stat = fileIo.statSync(filePath);
      
      // 尝试获取时长
      const duration = await MediaServiceManager.getAudioDuration(filePath);
      
      return {
        uri: `file://${filePath}`, // 转换为 file:// 协议 URI
        size: stat.size,
        mimeType: 'audio/mp4a-latm', // m4a
        duration: duration 
      };
    } catch (error) {
      hilog.error(DOMAIN, TAG, `停止录音失败: ${JSON.stringify(error)}`);
      throw (error instanceof Error) ? error : new Error(String(error));
    }
  }

  /**
   * 获取音频文件时长
   * 使用 AVMetadataExtractor
   */
  private static async getAudioDuration(filePath: string): Promise<number> {
    try {
      const avMetadataExtractor = await media.createAVMetadataExtractor();
      const file = fileIo.openSync(filePath, fileIo.OpenMode.READ_ONLY);
      
      avMetadataExtractor.fdSrc = { fd: file.fd };
      const metadata = await avMetadataExtractor.fetchMetadata();
      
      let duration = 0;
      if (metadata.duration) {
        duration = parseInt(metadata.duration);
      }
      
      await avMetadataExtractor.release();
      fileIo.closeSync(file);
      
      return duration; // 单位通常是毫秒
    } catch (error) {
      hilog.error(DOMAIN, TAG, `获取音频时长失败: ${JSON.stringify(error)}`);
      return 0;
    }
  }

  /**
   * 压缩图片
   * 使用 ImageKit 的 ImagePacker
   * @param uri 图片URI
   * @param quality 目标质量 0-100
   * @param maxSize 目标大小限制 (字节)
   */
  public static async compressImage(context: common.Context, uri: string, quality: number = 80, maxSize?: number): Promise<string> {
    try {
      // 1. 创建 ImageSource
      const sourcePath = MediaServiceManager.normalizeUriToFilePath(context, uri);
      const file = fileIo.openSync(sourcePath, fileIo.OpenMode.READ_ONLY);
      const imageSource = image.createImageSource(file.fd);
      
      // 2. 创建 ImagePacker
      const imagePacker = image.createImagePacker();
      
      // 3. 压缩配置
      const packingOption: image.PackingOption = {
        format: 'image/jpeg',
        quality: quality
      };

      // 4. 执行压缩
      const data = await imagePacker.packToData(imageSource, packingOption);
      hilog.info(DOMAIN, TAG, `压缩后大小: ${data.byteLength}`);

      // 5. 如果有大小限制且超出，递归降低质量 (简单实现)
      if (maxSize && data.byteLength > maxSize && quality > 10) {
        fileIo.closeSync(file);
        return MediaServiceManager.compressImage(context, uri, quality - 10, maxSize);
      }

      // 6. 保存压缩后的文件
      const tempDir = context.tempDir;
      const fileName = `compressed_${Date.now()}.jpg`;
      const outPath = `${tempDir}/${fileName}`;
      const outFile = fileIo.openSync(outPath, fileIo.OpenMode.CREATE | fileIo.OpenMode.READ_WRITE);
      
      fileIo.writeSync(outFile.fd, data);
      fileIo.closeSync(outFile);
      
      // 释放资源
      fileIo.closeSync(file);
      imageSource.release();
      imagePacker.release();

      return `file://${outPath}`;
    } catch (error) {
      hilog.error(DOMAIN, TAG, `压缩图片失败: ${JSON.stringify(error)}`);
      return uri; // 失败则返回原图
    }
  }

  /**
   * 准备文件用于上传
   * 将文件复制到 cache 目录，并返回可直接读取的本地路径
   */
  public static async prepareFileForUpload(context: common.Context, uri: string): Promise<string> {
    let srcFile: fileIo.File | null = null;
    let destFile: fileIo.File | null = null;
    try {
      const sourcePath = MediaServiceManager.normalizeUriToFilePath(context, uri);
      const cacheDir = context.cacheDir;
      const fileName = `upload_${Date.now()}_${Math.floor(Math.random() * 1000)}`;
      let extension = 'tmp';
      const lowerSourcePath = sourcePath.toLowerCase();
      
      // 简单推断扩展名
      if (lowerSourcePath.endsWith('.jpg') || lowerSourcePath.endsWith('.jpeg')) extension = 'jpg';
      else if (lowerSourcePath.endsWith('.png')) extension = 'png';
      else if (lowerSourcePath.endsWith('.gif')) extension = 'gif';
      else if (lowerSourcePath.endsWith('.webp')) extension = 'webp';
      else if (lowerSourcePath.endsWith('.mp4')) extension = 'mp4';
      else if (lowerSourcePath.endsWith('.mov')) extension = 'mov';
      else if (lowerSourcePath.endsWith('.m4a')) extension = 'm4a';
      else if (lowerSourcePath.endsWith('.aac')) extension = 'aac';
      else if (lowerSourcePath.endsWith('.ogg')) extension = 'ogg';
      
      const internalFileName = `${fileName}.${extension}`;
      const internalFilePath = `${cacheDir}/${internalFileName}`;
      
      srcFile = fileIo.openSync(sourcePath, fileIo.OpenMode.READ_ONLY);
      destFile = fileIo.openSync(internalFilePath, fileIo.OpenMode.READ_WRITE | fileIo.OpenMode.CREATE | fileIo.OpenMode.TRUNC);
      
      fileIo.copyFileSync(srcFile.fd, destFile.fd);

      hilog.info(DOMAIN, TAG, `文件已准备好上传: ${internalFilePath}`);
      
      return internalFilePath;
    } catch (error) {
      hilog.error(DOMAIN, TAG, `准备上传文件失败: ${JSON.stringify(error)}`);
      throw (error instanceof Error) ? error : new Error(String(error));
    } finally {
      if (srcFile) {
        try {
          fileIo.closeSync(srcFile);
        } catch (_) {
        }
      }
      if (destFile) {
        try {
          fileIo.closeSync(destFile);
        } catch (_) {
        }
      }
    }
  }


  /**
   * 获取图片信息 (宽、高)
   */
  private static async getImageInfo(uri: string): Promise<ImageDimensions> {
    try {
      // file:// 走 openSync，datashare 等 URI 直接走 ImageSource(uri)
      if (uri.startsWith('file://')) {
         const readablePath = uri.substring('file://'.length);
         const file = fileIo.openSync(readablePath, fileIo.OpenMode.READ_ONLY);
         const imageSource = image.createImageSource(file.fd);
         const info = await imageSource.getImageInfo();
         imageSource.release();
         fileIo.closeSync(file);
         return { width: info.size.width, height: info.size.height };
      } else {
         const imageSource = image.createImageSource(uri);
         const info = await imageSource.getImageInfo();
         imageSource.release();
         return { width: info.size.width, height: info.size.height };
      }
    } catch (error) {
      return { width: 0, height: 0 };
    }
  }
}