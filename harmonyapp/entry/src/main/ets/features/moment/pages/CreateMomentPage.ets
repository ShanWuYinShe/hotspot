/**
 * 创建生活瞬间页面
 * 重构：使用 HarmonyOS 原生 Grid 组件展示图片
 * 新增：上传前图片压缩功能
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { PermissionManager } from '../../../core/permission/PermissionManager';
import { Permissions } from '@kit.AbilityKit';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { SystemToast } from '../../../components/SystemToast';
import { IconButton, AppIcon, IconSize } from '../../../components/SystemIcon';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemCard } from '../../../components/SystemCard';
import { ApiResponse, CreateMomentRequest, MomentItem, GeoPoint } from '../../../core/types/api';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { PageNames } from '../../../constants/AppConstants';

const DOMAIN_NUMBER: number = 0xFF06;
const TAG: string = 'CreateMomentPage';

@Entry
@Component
struct CreateMomentPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_LOCATION_PICK_RESULT) pickResult: LocationInfo | null = null;
  
  private stateManager = GlobalStateManager.getInstance();
  @State description: string = '';
  @State tags: string[] = [];
  @State currentTag: string = '';
  @State imageUrls: string[] = [];
  @State isPublishing: boolean = false;
  @State localLocation: LocationInfo | null = null;
  @State publishingMessage: string = ''; // 显示发布进度提示

  async aboutToAppear() {
    await this.requestPermissions();
    // 优先消费“地图选点”结果
    if (this.pickResult) {
      this.localLocation = {
        latitude: this.pickResult.latitude,
        longitude: this.pickResult.longitude,
        locationName: this.pickResult.locationName || '南天门'
      };
      AppStorage.setOrCreate<LocationInfo | null>(KEY_LOCATION_PICK_RESULT, null);
      return;
    }

    await this.getCurrentLocation();
  }

  private async requestPermissions(): Promise<void> {
    try {
      const permissionManager = PermissionManager.getInstance();
      const permissions: Permissions[] = [
        'ohos.permission.READ_MEDIA',
        'ohos.permission.WRITE_MEDIA',
        'ohos.permission.CAMERA',
        'ohos.permission.LOCATION',
        'ohos.permission.APPROXIMATELY_LOCATION'
      ];
      
      const granted = await permissionManager.requestPermissions(permissions);
      if (!granted) {
        SystemToast.showWarning('部分权限未授予，可能影响功能使用', this.getUIContext());
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `请求权限失败: ${JSON.stringify(error)}`);
    }
  }

  private async getCurrentLocation(): Promise<void> {
    try {
      if (this.currentLocation) {
        this.localLocation = {
          latitude: this.currentLocation.latitude,
          longitude: this.currentLocation.longitude,
          locationName: this.currentLocation.locationName || '南天门'
        };
      } else {
        const location = await LocationUtil.getCurrentLocation();
        if (location) {
          this.localLocation = {
            latitude: location.latitude,
            longitude: location.longitude,
            locationName: location.locationName || '南天门'
          };
        }
      }
    } catch (error) {
      SystemToast.showError('获取位置信息失败', this.getUIContext());
    }
  }

  private addTag(): void {
    const tag = this.currentTag.trim();
    if (!tag) return;
    if (this.tags.length >= 10) {
      SystemToast.show('最多只能添加10个标签', this.getUIContext());
      return;
    }
    if (this.tags.includes(tag)) {
      SystemToast.show('标签已存在', this.getUIContext());
      return;
    }
    this.tags.push(tag);
    this.currentTag = '';
  }

  private removeTag(index: number) {
    this.tags.splice(index, 1);
  }

  private async selectImages(): Promise<void> {
    try {
      const maxCount = 9 - this.imageUrls.length;
      if (maxCount <= 0) {
        SystemToast.show('最多只能选择9张图片', this.getUIContext());
        return;
      }
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedImages = await MediaServiceManager.selectImages(context, maxCount);
      for (const imageUri of selectedImages) {
        this.imageUrls.push(imageUri);
      }
      if (selectedImages.length > 0) {
        SystemToast.show(`已选择${selectedImages.length}张图片`, this.getUIContext());
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `选择图片失败: ${JSON.stringify(error)}`);
      SystemToast.show('选择图片失败，请重试', this.getUIContext());
    }
  }

  private async capturePhoto(): Promise<void> {
    try {
      if (this.imageUrls.length >= 9) {
        SystemToast.show('最多只能选择9张图片', this.getUIContext());
        return;
      }
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const result = await MediaServiceManager.takePhoto(context);
      if (result && result.uri) {
        this.imageUrls.push(result.uri);
        SystemToast.show('拍照成功', this.getUIContext());
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `拍照失败: ${JSON.stringify(error)}`);
      SystemToast.show('拍照失败，请重试', this.getUIContext());
    }
  }

  private removeImage(index: number) {
    this.imageUrls.splice(index, 1);
  }

  private async publishMoment(): Promise<void> {
    if (!this.description.trim()) {
      SystemToast.show('请输入瞬间描述', this.getUIContext());
      return;
    }
    if (!this.localLocation) {
      SystemToast.show('请先获取位置信息', this.getUIContext());
      return;
    }

    this.isPublishing = true;
    this.publishingMessage = '正在处理图片...';
    
    try {
      const imageFileIds: string[] = []; 
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;

      // 串行压缩并上传图片
      for (let i = 0; i < this.imageUrls.length; i++) {
        const imageUrl = this.imageUrls[i];
        this.publishingMessage = `正在上传第 ${i + 1}/${this.imageUrls.length} 张图片...`;
        
        try {
          // 1. 压缩图片
          // 压缩到最大 1920x1080，质量 80%
          const compressedPath = await MediaServiceManager.compressImage(context, imageUrl, 80, 1024 * 1024 * 2); // 限制 2MB
          
          hilog.info(DOMAIN_NUMBER, TAG, `图片压缩成功: ${compressedPath}`);
          
          // 准备上传文件 (复制到 cache 并获取 internal URI)
          const uploadUri = await MediaServiceManager.prepareFileForUpload(context, compressedPath);

          // 2. 上传压缩后的图片
          const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'moment_image');
          if (uploadResult && uploadResult.file_id) {
            imageFileIds.push(uploadResult.file_id);
          }
          
        } catch (processError) {
          hilog.error(DOMAIN_NUMBER, TAG, `图片处理/上传失败: ${JSON.stringify(processError)}`);
          // 尝试直接上传原图（作为降级策略）
          try {
             hilog.warn(DOMAIN_NUMBER, TAG, `压缩失败，尝试上传原图: ${imageUrl}`);
             // 准备原图
             const uploadUri = await MediaServiceManager.prepareFileForUpload(context, imageUrl);
             const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'moment_image');
             if (uploadResult && uploadResult.file_id) {
               imageFileIds.push(uploadResult.file_id);
             }
          } catch (uploadError) {
             hilog.error(DOMAIN_NUMBER, TAG, `原图上传也失败: ${JSON.stringify(uploadError)}`);
             SystemToast.showError(`第 ${i + 1} 张图片上传失败`, this.getUIContext());
             this.isPublishing = false;
             return;
          }
        }
      }
      
      this.publishingMessage = '正在发布...';
      
      const geoPoint: GeoPoint = {
        latitude: this.localLocation.latitude,
        longitude: this.localLocation.longitude
      };
      const request: CreateMomentRequest = {
        description: this.description.trim(),
        location: {
          point: geoPoint,
          location_name: this.localLocation.locationName || "南天门"
        },
        tags: this.tags.length > 0 ? this.tags : undefined,
        image_file_ids: imageFileIds.length > 0 ? imageFileIds : undefined
      };

      const response: ApiResponse<MomentItem> = await ApiClient.createMoment(request);

      if (response.code === 0) {
        SystemToast.showSuccess('瞬间发布成功', this.getUIContext());
        const routerInstance = this.getUIContext().getRouter();
        routerInstance.back();
      } else {
        SystemToast.showError(response.message || '发布瞬间失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('发布瞬间失败', this.getUIContext());
    } finally {
      this.isPublishing = false;
    }
  }

  private async refreshLocation(): Promise<void> {
    try {
      const location = await LocationUtil.getCurrentLocation();
      if (location) {
        this.localLocation = {
          latitude: location.latitude,
          longitude: location.longitude,
          locationName: location.locationName || '南天门'
        };
        SystemToast.show('位置已更新', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('获取位置失败', this.getUIContext());
    }
  }

  private async openLocationPicker(): Promise<void> {
    try {
      const routerInstance = this.getUIContext().getRouter();
      await routerInstance.pushUrl({
        url: `features/map/pages/${PageNames.LOCATION_PICKER}`
      });
    } catch (error) {
      SystemToast.showError('跳转失败', this.getUIContext());
    }
  }

  build() {
    Stack() {
      Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text('发布瞬间')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Button('发布')
          .type(ButtonType.Normal)
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_emphasize'))
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .enabled(!this.isPublishing)
          .opacity(this.isPublishing ? 0.5 : 1)
          .onClick(() => this.publishMoment())
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      List() {
        ListItem() {
          SystemCard({ cardPadding: DesignConstants.SPACING.LG }) {
            Column() {
              TextArea({ placeholder: '写下你想分享的内容...', text: this.description })
                .onChange((value: string) => {
                  this.description = value;
                })
                .maxLength(500)
                .height(120)
                .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                .borderRadius(DesignConstants.RADIUS.MD)
            }
          }
        }
        .padding({ top: DesignConstants.SPACING.MD, left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })

        ListItem() {
          SystemCard({ cardPadding: DesignConstants.SPACING.LG }) {
            Column() {
              Row() {
                Text('添加图片')
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .layoutWeight(1)

                Text(`${this.imageUrls.length}/9`)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              }
              .width('100%')
              .margin({ bottom: DesignConstants.SPACING.MD })

              // 使用 Grid 组件显示图片
              Grid() {
                // 已选图片
                ForEach(this.imageUrls, (imageUrl: string, index: number) => {
                  GridItem() {
                    this.buildImageItem(imageUrl, index)
                  }
                })

                // 添加按钮
                if (this.imageUrls.length < 9) {
                  GridItem() {
                    Button() {
                      Column() {
                        Image($r('sys.media.ohos_ic_public_add'))
                          .width(24)
                          .height(24)
                          .fillColor($r('sys.color.ohos_id_color_text_secondary'))
                        Text('相册')
                          .fontSize(10)
                          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      }
                    }
                    .type(ButtonType.Normal)
                    .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                    .width('100%')
                    .height('100%')
                    .borderRadius(DesignConstants.RADIUS.MD)
                    .onClick(() => this.selectImages())
                  }
                }

                // 拍照按钮
                if (this.imageUrls.length < 9) {
                  GridItem() {
                    Button() {
                      Column() {
                        Image($r('sys.media.ohos_ic_public_camera'))
                          .width(24)
                          .height(24)
                          .fillColor($r('sys.color.ohos_id_color_text_secondary'))
                        Text('拍照')
                          .fontSize(10)
                          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      }
                    }
                    .type(ButtonType.Normal)
                    .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                    .width('100%')
                    .height('100%')
                    .borderRadius(DesignConstants.RADIUS.MD)
                    .onClick(() => this.capturePhoto())
                  }
                }
              }
              .columnsTemplate('1fr 1fr 1fr')
              .rowsGap(8)
              .columnsGap(8)
              .height(this.calcGridHeight()) // 动态高度
              .width('100%')
            }
          }
        }
        .padding({ top: DesignConstants.SPACING.MD, left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })

        ListItem() {
          SystemCard({ cardPadding: DesignConstants.SPACING.LG }) {
            Column() {
              Text('添加标签')
                .fontSize(DesignConstants.FONT.SIZE.LG)
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor($r('sys.color.ohos_id_color_text_primary'))
                .width('100%')
                .margin({ bottom: DesignConstants.SPACING.MD })

              Row() {
                TextInput({ placeholder: '输入标签名称', text: this.currentTag })
                  .onChange((value: string) => {
                    this.currentTag = value;
                  })
                  .layoutWeight(1)
                  .maxLength(20)
                  .height(DesignConstants.COMPONENT.INPUT_HEIGHT)
                  .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                  .borderRadius(DesignConstants.RADIUS.MD)

                Button('添加')
                  .type(ButtonType.Capsule)
                  .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                  .fontColor(Color.White)
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .onClick(() => this.addTag())
                  .margin({ left: DesignConstants.SPACING.SM })
              }
              .width('100%')
              .margin({ bottom: DesignConstants.SPACING.MD })

              if (this.tags.length > 0) {
                Flex({ wrap: FlexWrap.Wrap }) {
                  ForEach(this.tags, (tag: string, index: number) => {
                    this.buildTagItem(tag, index)
                  })
                }
                .width('100%')
              }
            }
          }
        }
        .padding({ top: DesignConstants.SPACING.MD, left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })

        ListItem() {
          SystemCard({ cardPadding: DesignConstants.SPACING.LG }) {
            Column() {
              Row() {
                Text('位置信息')
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .layoutWeight(1)

                Row({ space: DesignConstants.SPACING.SM }) {
                  Button('选点')
                    .type(ButtonType.Normal)
                    .backgroundColor(Color.Transparent)
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .onClick(() => this.openLocationPicker())

                  Button('刷新')
                    .type(ButtonType.Normal)
                    .backgroundColor(Color.Transparent)
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .onClick(() => this.refreshLocation())
                }
              }
              .width('100%')
              .margin({ bottom: DesignConstants.SPACING.MD })

              if (this.localLocation) {
                Row() {
                  Image($r('sys.media.ohos_ic_public_location'))
                    .width(20)
                    .height(20)
                    .fillColor($r('sys.color.ohos_id_color_emphasize'))

                  Column() {
                    Text(this.localLocation.locationName)
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                      .width('100%')

                    Text(`${this.localLocation.latitude.toFixed(6)}, ${this.localLocation.longitude.toFixed(6)}`)
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      .width('100%')
                      .margin({ top: 2 })
                  }
                  .alignItems(HorizontalAlign.Start)
                  .layoutWeight(1)
                  .margin({ left: DesignConstants.SPACING.MD })
                }
                .width('100%')
                .padding(DesignConstants.SPACING.MD)
                .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                .borderRadius(DesignConstants.RADIUS.MD)
              } else {
                Text('正在获取位置信息...')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .width('100%')
                  .textAlign(TextAlign.Center)
                  .padding(DesignConstants.SPACING.LG)
              }
            }
          }
        }
        .padding({ top: DesignConstants.SPACING.MD, left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, bottom: 80 })
      }
      .width('100%')
      .layoutWeight(1)
    }
      .width('100%')
      .height('100%')
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      if (this.isPublishing) {
        SystemLoadingView({ message: this.publishingMessage })
      }
    }
    .width('100%')
    .height('100%')
  }

  // 简单计算Grid高度
  private calcGridHeight(): number {
    const totalItems = this.imageUrls.length + (this.imageUrls.length < 9 ? 1 : 0) + (this.imageUrls.length < 9 ? 1 : 0); // 图片 + 加号 + 拍照
    // 实际上可能显示2个按钮或者1个或者0个，逻辑稍微复杂，这里简化计算：
    // 显示项目数：imageUrls.length + (imageUrls.length < 9 ? 2 : 0) ? 不对，是两个按钮都显示
    // 修正：如果 < 9，显示两个按钮。
    const displayCount = this.imageUrls.length + (this.imageUrls.length < 9 ? 2 : 0);
    const rows = Math.ceil(displayCount / 3);
    const itemHeight = 80; // 假设宽度大概80-100，高度固定80
    const gap = 8;
    return rows * itemHeight + (rows - 1) * gap;
  }

  @Builder
  buildTagItem(tag: string, index: number) {
    Row() {
      Text(`#${tag}`)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_emphasize'))

      IconButton({
        icon: AppIcon.CLOSE,
        iconSize: IconSize.SMALL,
        bgColor: Color.Transparent,
        paddingSize: 2,
        onButtonClick: (): void => this.removeTag(index)
      })
      .margin({ left: 4 })
    }
    .padding({
      left: DesignConstants.SPACING.SM,
      right: DesignConstants.SPACING.XS,
      top: DesignConstants.SPACING.XS,
      bottom: DesignConstants.SPACING.XS
    })
    .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
  }

  @Builder
  buildImageItem(imageUrl: string, index: number) {
    Stack({ alignContent: Alignment.TopEnd }) {
      Image(imageUrl)
        .width('100%')
        .height('100%')
        .borderRadius(DesignConstants.RADIUS.MD)
        .objectFit(ImageFit.Cover)
        .backgroundColor($r('sys.color.ohos_id_color_component_normal'))

      Button() {
        Image($r('sys.media.ohos_ic_public_cancel'))
          .width(12)
          .height(12)
          .fillColor(Color.White)
      }
      .type(ButtonType.Circle)
      .backgroundColor('rgba(0,0,0,0.6)')
      .width(20)
      .height(20)
      .margin({top: 4, right: 4})
      .onClick(() => this.removeImage(index))
    }
    .width('100%')
    .height('100%')
  }
}
