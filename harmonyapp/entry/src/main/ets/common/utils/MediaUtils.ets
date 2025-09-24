/**
 * 媒体工具函数
 * 简化的媒体选择和处理功能
 */
import photoAccessHelper from '@ohos.file.photoAccessHelper';
import { image } from '@kit.ImageKit';
import { fileIo } from '@kit.CoreFileKit';
import { common } from '@kit.AbilityKit';
import { createLogger } from './LogManager';

const logger = createLogger('MediaUtils');

/**
 * 图片压缩选项接口
 */
interface ImageCompressOptions {
  quality?: number;
  maxWidth?: number;
  maxHeight?: number;
  format?: 'JPEG' | 'PNG' | 'WEBP';
}

/**
 * 默认图片压缩选项实现类
 */
class DefaultImageCompressOptions implements ImageCompressOptions {
  quality: number = 80;
  maxWidth: number = 1920;
  maxHeight: number = 1080;
  format: 'JPEG' | 'PNG' | 'WEBP' = 'JPEG';
}

/**
 * 尺寸接口
 */
interface Size {
  width: number;
  height: number;
}

/**
 * 尺寸实现类
 */
class SizeImpl implements Size {
  width: number;
  height: number;

  constructor(width: number, height: number) {
    this.width = width;
    this.height = height;
  }
}

/**
 * 区域接口
 */
interface Region {
  size: Size;
  x: number;
  y: number;
}

/**
 * 区域实现类
 */
class RegionImpl implements Region {
  size: Size;
  x: number;
  y: number;

  constructor(size: Size, x: number, y: number) {
    this.size = size;
    this.x = x;
    this.y = y;
  }
}

/**
 * 媒体文件信息
 */
export interface MediaFileInfo {
  uri: string;
  name: string;
  size: number;
  mimeType: string;
  width?: number;
  height?: number;
}

/**
 * 媒体文件信息实现类
 */
class MediaFileInfoImpl implements MediaFileInfo {
  uri: string;
  name: string;
  size: number;
  mimeType: string;
  width?: number;
  height?: number;

  constructor(
    uri: string,
    name: string,
    size: number,
    mimeType: string,
    width?: number,
    height?: number
  ) {
    this.uri = uri;
    this.name = name;
    this.size = size;
    this.mimeType = mimeType;
    this.width = width;
    this.height = height;
  }
}

/**
 * 选择图片
 */
export async function selectImages(
  context: common.UIAbilityContext,
  maxCount: number = 9
): Promise<string[]> {
  try {
    logger.info(`开始选择图片，最大数量: ${maxCount}`);

    const photoSelectOptions = new photoAccessHelper.PhotoSelectOptions();
    photoSelectOptions.MIMEType = photoAccessHelper.PhotoViewMIMETypes.IMAGE_TYPE;
    photoSelectOptions.maxSelectNumber = maxCount;

    const photoViewPicker = new photoAccessHelper.PhotoViewPicker();
    const photoSelectResult = await photoViewPicker.select(photoSelectOptions);

    if (!photoSelectResult.photoUris || photoSelectResult.photoUris.length === 0) {
      logger.warn('用户未选择任何图片');
      return [];
    }

    logger.info(`用户选择了 ${photoSelectResult.photoUris.length} 张图片`);
    return photoSelectResult.photoUris;

  } catch (error) {
    logger.error('选择图片失败', error);
    throw new Error(String(error));
  }
}

/**
 * 拍照
 */
export async function capturePhoto(context: common.UIAbilityContext): Promise<string | null> {
  try {
    logger.info('开始拍照');

    const photoSelectOptions = new photoAccessHelper.PhotoSelectOptions();
    photoSelectOptions.MIMEType = photoAccessHelper.PhotoViewMIMETypes.IMAGE_TYPE;
    photoSelectOptions.maxSelectNumber = 1;

    const photoViewPicker = new photoAccessHelper.PhotoViewPicker();
    const photoSelectResult = await photoViewPicker.select(photoSelectOptions);

    if (!photoSelectResult.photoUris || photoSelectResult.photoUris.length === 0) {
      logger.warn('用户取消拍照');
      return null;
    }

    const uri = photoSelectResult.photoUris[0];
    logger.info(`拍照完成: ${uri}`);
    return uri;

  } catch (error) {
    logger.error('拍照失败', error);
    throw new Error(String(error));
  }
}

/**
 * 压缩图片
 */
export async function compressImage(
  sourceUri: string,
  options: ImageCompressOptions = new DefaultImageCompressOptions()
): Promise<string> {
  try {
    const quality = options.quality ?? 80;
    const maxWidth = options.maxWidth ?? 1920;
    const maxHeight = options.maxHeight ?? 1080;
    const format = options.format ?? 'JPEG';

    logger.info(`开始压缩图片: ${sourceUri}`);

    // 创建ImageSource
    const imageSource = image.createImageSource(sourceUri);
    const imageInfo = await imageSource.getImageInfo();

    // 计算压缩后的尺寸
    const compressSize = calculateCompressSize(
      imageInfo.size.width,
      imageInfo.size.height,
      maxWidth,
      maxHeight
    );
    const newWidth = compressSize.width;
    const newHeight = compressSize.height;

    // 创建PixelMap
    const desiredSize = new SizeImpl(newWidth, newHeight);
    const regionSize = new SizeImpl(newWidth, newHeight);
    const desiredRegion = new RegionImpl(regionSize, 0, 0);
    
    const decodingOptions: image.DecodingOptions = {
      desiredSize: desiredSize,
      desiredRegion: desiredRegion,
      desiredPixelFormat: image.PixelMapFormat.RGBA_8888
    };

    const pixelMap = await imageSource.createPixelMap(decodingOptions);

    // 创建ImagePacker进行压缩
    const imagePacker = image.createImagePacker();
    const packOpts: image.PackingOption = {
      format: 'image/jpeg', // 简化为固定格式
      quality: quality
    };

    const compressedData = await imagePacker.packToData(pixelMap, packOpts);

    // 生成临时文件路径
    const timestamp = Date.now();
    const extension = format.toLowerCase();
    const fileName = `compressed_${timestamp}.${extension}`;
    const tempDir = '/data/storage/el2/base/haps/temp'; // 临时目录
    const filePath = `${tempDir}/${fileName}`;

    // 写入文件
    const file = fileIo.openSync(filePath, fileIo.OpenMode.READ_WRITE | fileIo.OpenMode.CREATE);
    fileIo.writeSync(file.fd, compressedData);
    fileIo.closeSync(file.fd);

    // 释放资源
    pixelMap.release();
    imageSource.release();
    imagePacker.release();

    logger.info(`图片压缩完成: ${filePath}`);
    return filePath;

  } catch (error) {
    logger.error('图片压缩失败', error);
    throw new Error(String(error));
  }
}

/**
 * 获取文件信息
 */
export async function getFileInfo(uri: string): Promise<MediaFileInfo> {
  try {
    const file = fileIo.openSync(uri, fileIo.OpenMode.READ_ONLY);
    const stat = fileIo.statSync(file.fd);
    
    const fileName = uri.split('/').pop() || 'unknown';
    
    const fileInfo: MediaFileInfo = new MediaFileInfoImpl(
      uri,
      fileName,
      stat.size,
      getMimeTypeFromUri(uri)
    );

    // 如果是图片，获取图片尺寸
    if (fileInfo.mimeType.startsWith('image/')) {
      try {
        const imageSource = image.createImageSource(uri);
        const imageInfo = await imageSource.getImageInfo();
        fileInfo.width = imageInfo.size.width;
        fileInfo.height = imageInfo.size.height;
        imageSource.release();
      } catch (error) {
        logger.warn(`获取图片尺寸失败: ${uri}`, error);
      }
    }

    fileIo.closeSync(file.fd);
    return fileInfo;

  } catch (error) {
    logger.error(`获取文件信息失败: ${uri}`, error);
    throw new Error(String(error));
  }
}

/**
 * 计算压缩后的尺寸
 */
function calculateCompressSize(
  originalWidth: number,
  originalHeight: number,
  maxWidth: number,
  maxHeight: number
): Size {
  
  if (originalWidth <= maxWidth && originalHeight <= maxHeight) {
    return new SizeImpl(originalWidth, originalHeight);
  }

  const widthRatio = maxWidth / originalWidth;
  const heightRatio = maxHeight / originalHeight;
  const ratio = Math.min(widthRatio, heightRatio);

  return new SizeImpl(
    Math.round(originalWidth * ratio),
    Math.round(originalHeight * ratio)
  );
}

/**
 * 根据URI获取MIME类型
 */
function getMimeTypeFromUri(uri: string): string {
  const extension = uri.split('.').pop()?.toLowerCase();
  
  switch (extension) {
    case 'jpg':
    case 'jpeg':
      return 'image/jpeg';
    case 'png':
      return 'image/png';
    case 'webp':
      return 'image/webp';
    case 'gif':
      return 'image/gif';
    case 'mp4':
      return 'video/mp4';
    case 'mov':
      return 'video/quicktime';
    default:
      return 'application/octet-stream';
  }
}
