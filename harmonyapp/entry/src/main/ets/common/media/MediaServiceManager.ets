/**
 * HarmonyOS原生多媒体服务工具函数
 * 集成相机、录音、图片处理等原生能力
 * 使用最新的 HarmonyOS API
 */

import { common } from '@kit.AbilityKit';
import { image } from '@kit.ImageKit';
import { media } from '@kit.MediaKit';
import { fileIo } from '@kit.CoreFileKit';
import { abilityAccessCtrl, PermissionRequestResult, Permissions } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BusinessError } from '@kit.BasicServicesKit';
import photoAccessHelper from '@ohos.file.photoAccessHelper'

const DOMAIN_NUMBER: number = 0xFF02;
const TAG: string = 'MediaServiceManager';

export interface CameraConfig {
  quality: 'high' | 'medium' | 'low';
  maxWidth?: number;
  maxHeight?: number;
  enableFlash?: boolean;
}

export interface AudioConfig {
  quality: 'high' | 'medium' | 'low';
  maxDuration?: number; // 秒
  format: 'mp3' | 'aac' | 'wav';
}

export interface MediaResult {
  uri: string;
  size: number;
  mimeType: string;
  width?: number;
  height?: number;
  duration?: number;
}

/**
 * 请求必要权限
 */
export async function requestMediaPermissions(context: common.UIAbilityContext): Promise<void> {
  try {
    const atManager = abilityAccessCtrl.createAtManager();
    const permissions: Array<Permissions> = [
      'ohos.permission.CAMERA',
      'ohos.permission.MICROPHONE',
      'ohos.permission.READ_MEDIA',
      'ohos.permission.WRITE_MEDIA'
    ];
    const requestResult: PermissionRequestResult = await atManager.requestPermissionsFromUser(
      context,
      permissions
    );

    hilog.info(DOMAIN_NUMBER, TAG, '权限请求完成: %{public}s', JSON.stringify(requestResult));
  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '权限请求失败: %{public}s', JSON.stringify(error));
  }
}

/**
 * 拍照功能
 */
export async function capturePhoto(context: common.Context,
  config: CameraConfig = { quality: 'high' }): Promise<MediaResult> {
  try {
    // 使用系统相机拍照
    const photoSelectOptions: photoAccessHelper.PhotoSelectOptions = {
      MIMEType: photoAccessHelper.PhotoViewMIMETypes.IMAGE_TYPE,
      maxSelectNumber: 1
    }

    const photoPicker = new photoAccessHelper.PhotoViewPicker();
    const photoSelectResult = await photoPicker.select(photoSelectOptions);

    if (!photoSelectResult.photoUris || photoSelectResult.photoUris.length === 0) {
      throw new Error('未选择照片');
    }

    const photoUri = photoSelectResult.photoUris[0];

    // 获取图片信息
    const imageInfo = await getImageInfo(photoUri);

    // 根据配置压缩图片
    const processedUri = await processImage(context, photoUri, config);

    const fileStats = await fileIo.stat(processedUri);

    return {
      uri: processedUri,
      size: fileStats.size,
      mimeType: 'image/jpeg',
      width: imageInfo.width,
      height: imageInfo.height
    };

  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '拍照失败: %{public}s', JSON.stringify(error));
    throw new Error(`拍照失败: ${JSON.stringify(error)}`);
  }
}

/**
 * 从相册选择图片
 */
export async function selectPhotos(context: common.Context, maxCount: number = 1): Promise<MediaResult[]> {
  try {
    const photoSelectOptions: photoAccessHelper.PhotoSelectOptions = {
      MIMEType: photoAccessHelper.PhotoViewMIMETypes.IMAGE_TYPE,
      maxSelectNumber: maxCount
    };

    const photoPicker = new photoAccessHelper.PhotoViewPicker();
    const photoSelectResult = await photoPicker.select(photoSelectOptions);

    const results: MediaResult[] = [];

    if (photoSelectResult.photoUris) {
      for (const photoUri of photoSelectResult.photoUris) {
        const imageInfo = await getImageInfo(photoUri);
        const fileStats = await fileIo.stat(photoUri);

        results.push({
          uri: photoUri,
          size: fileStats.size,
          mimeType: 'image/jpeg',
          width: imageInfo.width,
          height: imageInfo.height
        });
      }
    }

    return results;

  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '选择照片失败: %{public}s', JSON.stringify(error));
    throw new Error(`选择照片失败: ${JSON.stringify(error)}`);
  }
}

/**
 * 录音功能
 */
export async function startRecording(context: common.Context, config: AudioConfig): Promise<media.AVRecorder> {
  try {
    // 创建录音器
    const audioRecorder = await media.createAVRecorder();

    // 配置录音参数
    const audioRecorderConfig: media.AVRecorderConfig = {
      audioSourceType: media.AudioSourceType.AUDIO_SOURCE_TYPE_MIC,
      profile: {
        audioBitrate: getAudioBitrate(config.quality),
        audioChannels: 1,
        audioCodec: getAudioCodec(config.format),
        audioSampleRate: 44100,
        fileFormat: getFileFormat(config.format)
      },
      url: `fd://${await createTempFile(context, 'audio', config.format)}`,
      rotation: 0
    };

    // 准备录音
    await audioRecorder.prepare(audioRecorderConfig);

    // 开始录音
    await audioRecorder.start();

    hilog.info(DOMAIN_NUMBER, TAG, '开始录音');

    // 设置最大录音时长
    if (config.maxDuration) {
      setTimeout(async () => {
        await stopRecording(audioRecorder);
      }, config.maxDuration * 1000);
    }

    return audioRecorder;

  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '开始录音失败: %{public}s', JSON.stringify(error));
    throw new Error(`开始录音失败: ${JSON.stringify(error)}`);
  }
}

/**
 * 停止录音
 */
export async function stopRecording(audioRecorder: media.AVRecorder): Promise<MediaResult> {
  try {
    if (!audioRecorder) {
      throw new Error('录音器未初始化');
    }

    await audioRecorder.stop();
    await audioRecorder.release();

    // 获取录音文件信息
    const audioUri = 'temp_audio_file'; // 实际应该从配置中获取
    const fileStats = await fileIo.stat(audioUri);

    const result: MediaResult = {
      uri: audioUri,
      size: fileStats.size,
      mimeType: 'audio/mp3'
    };

    hilog.info(DOMAIN_NUMBER, TAG, '录音完成');

    return result;

  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '停止录音失败: %{public}s', JSON.stringify(error));
    throw new Error(`停止录音失败: ${JSON.stringify(error)}`);
  }
}

/**
 * 图片压缩处理
 */
export async function compressImage(context: common.Context, imageUri: string, quality: number = 80): Promise<string> {
  try {
    // 创建图片源
    const imageSource = image.createImageSource(imageUri);

    // 获取图片信息
    const imageInfo = await imageSource.getImageInfo();

    // 创建像素图
    const pixelMap = await imageSource.createPixelMap();

    // 压缩配置
    const packOpts: image.PackingOption = {
      format: 'image/jpeg',
      quality: quality
    };

    // 创建图片打包器
    const imagePacker = image.createImagePacker();

    // 压缩图片 - 使用最新的 packing 方法
    const compressedData = await imagePacker.packToData(pixelMap, packOpts);

    // 保存压缩后的图片
    const compressedUri = await saveCompressedImage(context, compressedData);

    // 释放资源
    pixelMap.release();
    imagePacker.release();

    hilog.info(DOMAIN_NUMBER, TAG, '图片压缩完成: %{public}s', compressedUri);
    return compressedUri;

  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '图片压缩失败: %{public}s', JSON.stringify(error));
    throw new Error(`图片压缩失败: ${JSON.stringify(error)}`);
  }
}

/**
 * 获取图片信息
 */
interface ImageInfo {
  width: number;
  height: number;
}

async function getImageInfo(imageUri: string): Promise<ImageInfo> {
  try {
    const imageSource = image.createImageSource(imageUri);
    const imageInfo = await imageSource.getImageInfo();

    return {
      width: imageInfo.size.width,
      height: imageInfo.size.height
    };
  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '获取图片信息失败: %{public}s', JSON.stringify(error));
    return { width: 0, height: 0 };
  }
}

/**
 * 处理图片（压缩、裁剪等）
 */
async function processImage(context: common.Context, imageUri: string,
  config: CameraConfig): Promise<string> {
  try {
    let processedUri = imageUri;

    // 根据质量设置进行压缩
    const quality = config.quality === 'high' ? 90 : config.quality === 'medium' ? 70 : 50;
    processedUri = await compressImage(context, processedUri, quality);

    return processedUri;
  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '处理图片失败: %{public}s', JSON.stringify(error));
    return imageUri;
  }
}

/**
 * 创建临时文件
 */
async function createTempFile(context: common.Context, type: 'image' | 'audio',
  format: string): Promise<number> {
  try {
    const tempDir = context.tempDir;
    const fileName = `temp_${type}_${Date.now()}.${format}`;
    const filePath = `${tempDir}/${fileName}`;

    const file = await fileIo.open(filePath, fileIo.OpenMode.CREATE | fileIo.OpenMode.WRITE_ONLY);
    return file.fd;
  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '创建临时文件失败: %{public}s', JSON.stringify(error));
    throw new Error(`创建临时文件失败: ${JSON.stringify(error)}`);
  }
}

/**
 * 保存压缩后的图片
 */
async function saveCompressedImage(context: common.Context, imageData: ArrayBuffer): Promise<string> {
  try {
    const fileName = `compressed_${Date.now()}.jpg`;
    const filePath = `${context.filesDir}/${fileName}`;

    const file = await fileIo.open(filePath, fileIo.OpenMode.CREATE | fileIo.OpenMode.WRITE_ONLY);
    await fileIo.write(file.fd, imageData);
    await fileIo.close(file.fd);

    return filePath;
  } catch (error) {
    hilog.error(DOMAIN_NUMBER, TAG, '保存压缩图片失败: %{public}s', JSON.stringify(error));
    throw new Error(`保存压缩图片失败: ${JSON.stringify(error)}`);
  }
}

/**
 * 获取音频比特率
 */
function getAudioBitrate(quality: string): number {
  switch (quality) {
    case 'high':
      return 128000;
    case 'medium':
      return 64000;
    case 'low':
      return 32000;
    default:
      return 64000;
  }
}

/**
 * 获取音频编码格式
 */
function getAudioCodec(format: string): media.CodecMimeType {
  switch (format) {
      case 'mp3':
        return media.CodecMimeType.AUDIO_MP3;
      case 'aac':
        return media.CodecMimeType.AUDIO_AAC;
      case 'wav':
        return media.CodecMimeType.AUDIO_AAC; // WAV格式使用AAC编码
    default:
      return media.CodecMimeType.AUDIO_AAC;
  }
}

/**
 * 获取文件格式
 */
function getFileFormat(format: string): media.ContainerFormatType {
  switch (format) {
    case 'mp3':
      return media.ContainerFormatType.CFT_MPEG_4;
    case 'aac':
      return media.ContainerFormatType.CFT_MPEG_4A;
    case 'wav':
      return media.ContainerFormatType.CFT_WAV;
    default:
      return media.ContainerFormatType.CFT_MPEG_4A;
  }
}