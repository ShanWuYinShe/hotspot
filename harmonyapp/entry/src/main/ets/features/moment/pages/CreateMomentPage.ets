/**
 * 创建生活瞬间页面
 * 重构：使用 HarmonyOS 原生 Grid 组件展示图片
 * 新增：上传前图片压缩功能
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { PermissionManager } from '../../../core/permission/PermissionManager';
import { Permissions } from '@kit.AbilityKit';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader, PageHeaderAction } from '../../../components/SystemPageHeader';
import { IconButton, AppIcon, IconSize } from '../../../components/SystemIcon';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemCard } from '../../../components/SystemCard';
import { ApiResponse, CreateMomentRequest, MomentItem, GeoPoint, PrivacyLevel } from '../../../core/types/api';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { AppRouter, LocationPickerRouteParam, EditMomentRouteParam } from '../../../core/router/AppRouter';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF06;
const TAG: string = 'CreateMomentPage';

@Preview
@Component
export struct CreateMomentPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_LOCATION_PICK_RESULT) pickResult: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  
  private stateManager = GlobalStateManager.getInstance();
  @State description: string = '';
  @State tags: string[] = [];
  @State currentTag: string = '';
  @State imageUrls: string[] = [];
  @State isPublishing: boolean = false;
  @State localLocation: LocationInfo | null = null;
  @State publishingMessage: string = ''; // 显示发布进度提示
  @State isEditMode: boolean = false;
  @State editMomentId: string = '';
  @State existingImageUrls: string[] = [];
  @State privacyLevel: string = 'public';

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    await this.requestPermissions();
    // 优先消费“地图选点”结果
    if (this.pickResult) {
      this.localLocation = {
        latitude: this.pickResult.latitude,
        longitude: this.pickResult.longitude,
        locationName: this.pickResult.locationName || '未知位置'
      };
      AppStorage.setOrCreate<LocationInfo | null>(KEY_LOCATION_PICK_RESULT, null);
      return;
    }

    this.initDefaultLocation();
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
      SystemToast.showError('权限请求失败，请重试', this.getUIContext());
    }
  }

  /**
   * 初始化默认位置：优先浏览锚点 → 当前定位。
   * 浏览锚点是用户正在浏览的区域中心，更符合用户发布瞬间的意图。
   */
  private initDefaultLocation(): void {
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (refLoc) {
      this.localLocation = {
        latitude: refLoc.latitude,
        longitude: refLoc.longitude,
        locationName: refLoc.locationName || '未知位置'
      };
    }
  }

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getOverlayBorderColor(): string {
    return this.getCurrentThemePalette().isDarkTheme ? '#40FFFFFF' : '#26000000';
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private getSoftCardShadow(): ShadowOptions {
    return AppThemeVisuals.getSurfaceShadow(this.resolvedThemeId);
  }

  private getPrimaryButtonShadow(): ShadowOptions {
    return AppThemeVisuals.getAccentShadow(this.resolvedThemeId);
  }

  private addTag(): void {
    const tag = this.currentTag.trim();
    if (!tag) return;
    if (this.tags.length >= DefaultConfig.MAX_MOMENT_TAGS) {
      SystemToast.show(`最多只能添加${DefaultConfig.MAX_MOMENT_TAGS}个标签`, this.getUIContext());
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
      if (this.imageUrls.length >= DefaultConfig.MAX_MOMENT_IMAGES) {
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
      SystemToast.show('位置信息不可用，请选择位置', this.getUIContext());
      return;
    }

    const localLoc = this.localLocation;
    this.isPublishing = true;
    this.publishingMessage = '正在处理图片...';
    
    try {
      await LoadingUtil.withMinimumTime(async () => {
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
          latitude: localLoc.latitude,
          longitude: localLoc.longitude
        };

        let response: ApiResponse<MomentItem>;
        if (this.isEditMode) {
          response = await ApiClient.updateMoment({
            moment_id: this.editMomentId,
            description: this.description.trim(),
            location: {
              point: geoPoint,
              location_name: localLoc.locationName || "未知位置"
            },
            tags: this.tags.length > 0 ? this.tags : undefined,
            image_file_ids: imageFileIds.length > 0 ? imageFileIds : undefined,
            privacy_level: this.privacyLevel as PrivacyLevel
          });
        } else {
          const request: CreateMomentRequest = {
            description: this.description.trim(),
            location: {
              point: geoPoint,
              location_name: localLoc.locationName || "未知位置"
            },
            tags: this.tags.length > 0 ? this.tags : undefined,
            image_file_ids: imageFileIds.length > 0 ? imageFileIds : undefined
          };
          response = await ApiClient.createMoment(request);
        }

        if (response.code === 0) {
          SystemToast.showSuccess(this.isEditMode ? '瞬间已更新' : '瞬间发布成功', this.getUIContext());
          AppRouter.pop();
        } else {
          SystemToast.showError(response.message || (this.isEditMode ? '更新瞬间失败' : '发布瞬间失败'), this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch (error) {
      SystemToast.showError(this.isEditMode ? '更新瞬间失败' : '发布瞬间失败', this.getUIContext());
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
          locationName: location.locationName || '未知位置'
        };
        SystemToast.show('位置已更新', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('获取位置失败', this.getUIContext());
    }
  }

  private async openLocationPicker(): Promise<void> {
    try {
      const param = new LocationPickerRouteParam();
      param.mode = 'location_pick';
      AppRouter.push(PageNames.LOCATION_PICKER, param);
    } catch (error) {
      SystemToast.showError('跳转失败', this.getUIContext());
    }
  }

  build() {
    NavDestination() {
    ResponsiveContainer({ fill: true }) {
    Stack() {
      Column() {
      // 标题栏 —— 统一走 SystemPageHeader（居中标题 + 返回 + 保存/发布 action）
      SystemPageHeader({
        title: this.isEditMode ? '编辑瞬间' : '发布瞬间',
        showBackButton: true,
        onBack: () => { AppRouter.pop(); },
        titleAlign: TextAlign.Center,
        titleWeight: FontWeight.Medium,
        backgroundVariant: 'transparent',
        action: {
          label: this.isEditMode ? '保存' : '发布',
          onClick: () => this.publishMoment(),
          disabled: this.isPublishing
        } as PageHeaderAction
      })

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
                .backgroundColor(this.getSurfaceSubtle())
                .borderRadius(DesignConstants.RADIUS.XL)
                .padding(DesignConstants.SPACING.SM)
            }
          }
        }
        .padding({ top: DesignConstants.SPACING.MD, left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
        .margin({ bottom: 14 })

        ListItem() {
          SystemCard({ cardPadding: DesignConstants.SPACING.LG }) {
            Column() {
              Row() {
                Text('添加图片')
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .layoutWeight(1)

                Text(`${this.imageUrls.length}/9`)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .width('100%')
              .margin({ bottom: DesignConstants.SPACING.MD })

              // 使用 Grid 组件显示图片
              if (this.isEditMode && this.existingImageUrls.length > 0) {
                Text('已有图片')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .margin({ top: 8 })

                Grid() {
                  ForEach(this.existingImageUrls, (url: string, index: number) => {
                    GridItem() {
                      Image(MediaUrlUtil.normalize(url))
                        .width('100%')
                        .height(110)
                        .objectFit(ImageFit.Cover)
                        .borderRadius(DesignConstants.RADIUS.MD)
                    }
                  }, (url: string, index: number) => `existing_${index}_${url}`)
                }
                .columnsTemplate('1fr 1fr 1fr')
                .rowsGap(8)
                .columnsGap(8)
                .margin({ bottom: DesignConstants.SPACING.MD })
              }

              Grid() {
                // 已选图片
                ForEach(this.imageUrls, (imageUrl: string, index: number) => {
                  GridItem() {
                    this.buildImageItem(imageUrl, index)
                  }
                }, (imageUrl: string, index: number) => `${index}_${imageUrl}`)

                // 添加按钮
                if (this.imageUrls.length < 9) {
                  GridItem() {
                    Button() {
                      Column() {
                        Text() {
                          SymbolSpan($r(AppIcon.ADD))
                            .fontSize(DesignConstants.FONT.SIZE.XXL)
                            .fontColor([this.getCurrentThemePalette().textSecondary])
                        }
                        Text('相册')
                          .fontSize(DesignConstants.FONT.SIZE.XS)
                          .fontColor(this.getCurrentThemePalette().textSecondary)
                      }
                    }
                    .type(ButtonType.Normal)
                    .backgroundColor(this.getSurfaceSubtle())
                    .width('100%')
                    .height('100%')
                    .borderRadius(DesignConstants.RADIUS.XL)
                    .border({ width: 1, color: this.getSurfaceBorder() })
                    .onClick(() => this.selectImages())
                  }
                }

                // 拍照按钮
                if (this.imageUrls.length < 9) {
                  GridItem() {
                    Button() {
                      Column() {
                        Text() {
                          SymbolSpan($r(AppIcon.CAMERA))
                            .fontSize(DesignConstants.FONT.SIZE.XXL)
                            .fontColor([this.getCurrentThemePalette().textSecondary])
                        }
                        Text('拍照')
                          .fontSize(DesignConstants.FONT.SIZE.XS)
                          .fontColor(this.getCurrentThemePalette().textSecondary)
                      }
                    }
                    .type(ButtonType.Normal)
                    .backgroundColor(this.getSurfaceSubtle())
                    .width('100%')
                    .height('100%')
                    .borderRadius(DesignConstants.RADIUS.XL)
                    .border({ width: 1, color: this.getSurfaceBorder() })
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
        .margin({ bottom: 14 })

        ListItem() {
          SystemCard({ cardPadding: DesignConstants.SPACING.LG }) {
            Column() {
              Text('添加标签')
                .fontSize(DesignConstants.FONT.SIZE.LG)
                .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                .fontColor(this.getCurrentThemePalette().textPrimary)
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
                  .backgroundColor(this.getSurfaceSubtle())
                  .borderRadius(DesignConstants.RADIUS.XL)
                  .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

                Button('添加')
                  .type(ButtonType.Capsule)
                  .backgroundColor(this.getCurrentThemePalette().accentBackground)
                  .fontColor(this.getCurrentThemePalette().accentText)
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .border({ width: 1, color: this.getSurfaceBorder() })
                  .shadow(this.getPrimaryButtonShadow())
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
        .margin({ bottom: 14 })

        ListItem() {
          SystemCard({ cardPadding: DesignConstants.SPACING.LG }) {
            Column() {
              Row() {
                Text('位置信息')
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .layoutWeight(1)

                Row({ space: DesignConstants.SPACING.SM }) {
                  Button('选点')
                    .type(ButtonType.Capsule)
                    .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
                    .fontColor(this.getCurrentThemePalette().accentBackground)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .onClick(() => this.openLocationPicker())

                  Button('刷新')
                    .type(ButtonType.Capsule)
                    .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
                    .fontColor(this.getCurrentThemePalette().accentBackground)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .onClick(() => this.refreshLocation())
                }
              }
              .width('100%')
              .margin({ bottom: DesignConstants.SPACING.MD })

              if (this.localLocation) {
                Row() {
                  Text() {
                    SymbolSpan($r(AppIcon.LOCATION))
                      .fontSize(DesignConstants.FONT.SIZE.XL)
                      .fontColor([this.getCurrentThemePalette().accentBackground])
                  }

                  Text(this.localLocation.locationName)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                    .layoutWeight(1)
                    .maxLines(2)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                    .margin({ left: DesignConstants.SPACING.MD })
                }
                .width('100%')
                .padding(DesignConstants.SPACING.MD)
                .backgroundColor(this.getSurfaceSubtle())
                .borderRadius(DesignConstants.RADIUS.MD)
                .border({ width: 1, color: this.getSurfaceBorder() })
              } else {
                Text('正在获取位置信息...')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .width('100%')
                  .textAlign(TextAlign.Center)
                  .padding(DesignConstants.SPACING.LG)
              }
            }
          }
        }
        .padding({ top: DesignConstants.SPACING.MD, left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD,
          bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.XL, 0) })
      }
      .width('100%')
      .layoutWeight(1)
    }
      .width('100%')
      .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)

      if (this.isPublishing) {
        SystemLoadingView({ message: this.publishingMessage })
      }
    }
    .width('100%')
    .height('100%')
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param: EditMomentRouteParam | undefined = context.pathInfo?.param as EditMomentRouteParam | undefined;
      // 检查编辑参数
      const editParam = param;
      if (editParam && editParam.moment_id) {
        this.isEditMode = true;
        this.editMomentId = editParam.moment_id;
        this.description = editParam.description;
        this.tags = [...editParam.tags];
        this.existingImageUrls = [...editParam.existingImageUrls];
        this.privacyLevel = editParam.privacy_level ?? 'public';
        this.localLocation = {
          latitude: editParam.latitude,
          longitude: editParam.longitude,
          locationName: editParam.location_name
        };
      }
    })
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
        .fontColor(this.getCurrentThemePalette().accentBackground)

      IconButton({
        icon: AppIcon.CLOSE,
        iconSize: IconSize.SMALL,
        bgColor: DesignConstants.COLOR.TRANSPARENT,
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
    .backgroundColor(this.getSurfaceSubtle())
    .borderRadius(DesignConstants.RADIUS.LG)
    .border({ width: 1, color: this.getSurfaceBorder() })
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
        .backgroundColor(this.getSurfaceBackground())

      Button() {
        Text() {
          SymbolSpan($r(AppIcon.CANCEL))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor([this.getCurrentThemePalette().accentText])
        }
      }
      .type(ButtonType.Normal)
      .backgroundColor(this.getCurrentThemePalette().overlayStrong)
      .width(28)
      .height(28)
      .borderRadius(DesignConstants.RADIUS.XXL)
      .border({ width: 1, color: this.getOverlayBorderColor() })
      .margin({ top: 2, right: 2 })
      .onClick(() => this.removeImage(index))
    }
    .width('100%')
    .height('100%')
  }
}

@Builder
export function CreateMomentPageBuilder() {
  CreateMomentPage()
}
