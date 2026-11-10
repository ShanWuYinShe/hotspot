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
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { AppIcon } from '../../../components/SystemIcon';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { ApiResponse, CreateMomentRequest, MomentItem, GeoPoint, PrivacyLevel } from '../../../core/types/api';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { AppRouter, LocationPickerRouteParam, EditMomentRouteParam } from '../../../core/router/AppRouter';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LucideIcon } from '../../../components/LucideIcon';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';

const DOMAIN_NUMBER: number = 0xFF06;
const TAG: string = 'CreateMomentPage';
/** 此刻氛围预设（对齐 web CreateSheet MOOD_TAGS 六枚） */
const MOOD_PRESETS: string[] = ['松弛', '漫游', '咖啡', '展览', '自然', '夜色'];
/** 正文字数上限（对齐 web CreateSheet 280 字契约） */
const DESCRIPTION_MAX_LENGTH: number = 280;

@Preview
@Component
export struct CreateMomentPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_LOCATION_PICK_RESULT) pickResult: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  
  private stateManager = GlobalStateManager.getInstance();
  @State description: string = '';
  @State tags: string[] = [];
  @State imageUrls: string[] = [];
  @State isPublishing: boolean = false;
  @State localLocation: LocationInfo | null = null;
  @State publishingMessage: string = ''; // 显示发布进度提示
  @State isEditMode: boolean = false;
  @State editMomentId: string = '';
  @State existingImageUrls: string[] = [];
  @State privacyLevel: string = 'public';
  /** 可见范围 UI 两档：nearby → public / private → private（spec PrivacyLevel 仅两档） */
  @State audience: string = 'nearby';

  private isDestroyed: boolean = false;

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

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private async requestPermissions(): Promise<void> {
    if (this.isDestroyed) return;
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
      if (this.isDestroyed) return;
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

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  /** 氛围标签单选/多选（对齐 web toggleMood：已选则取消，未选则追加） */
  private toggleMood(mood: string): void {
    const index = this.tags.indexOf(mood);
    if (index >= 0) {
      this.tags = this.tags.filter((item: string) => item !== mood);
    } else {
      this.tags = this.tags.concat([mood]);
    }
  }

  private resolveAudienceHint(): string {
    return this.audience === 'nearby' ? '进入附近瞬间' : '仅自己可见';
  }

  private canPublish(): boolean {
    return !this.isPublishing && this.description.trim().length > 0 && this.localLocation !== null;
  }

  private async selectImages(): Promise<void> {
    if (this.isDestroyed) return;
    try {
      const maxCount = 9 - this.imageUrls.length;
      if (maxCount <= 0) {
        SystemToast.show('最多只能选择9张图片', this.getUIContext());
        return;
      }
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedImages = await MediaServiceManager.selectImages(context, maxCount);
      if (this.isDestroyed) return;
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
    if (this.isDestroyed) return;
    try {
      if (this.imageUrls.length >= DefaultConfig.MAX_MOMENT_IMAGES) {
        SystemToast.show('最多只能选择9张图片', this.getUIContext());
        return;
      }
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const result = await MediaServiceManager.takePhoto(context);
      if (this.isDestroyed) return;
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
    if (this.isDestroyed) return;
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
    this.publishingMessage = '正在处理图片…';
    
    try {
      if (this.isDestroyed) return;
      await LoadingUtil.withMinimumTime(async () => {
        const imageFileIds: string[] = []; 
        const context = this.getUIContext().getHostContext() as common.UIAbilityContext;

        // 串行压缩并上传图片
        for (let i = 0; i < this.imageUrls.length; i++) {
          const imageUrl = this.imageUrls[i];
          this.publishingMessage = `正在上传第 ${i + 1}/${this.imageUrls.length} 张图片…`;
          
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
        
        this.publishingMessage = '正在发布…';

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
            // 编辑模式不修改图片（undefined = 保持不变）：MediaAsset 契约不含
            // file_id，客户端无法把已有图片并入替换列表，上传新图会触发服务端
            // 全量替换、静默丢弃旧图——图片调整暂不支持，UI 已在编辑态隐藏入口。
            image_file_ids: undefined,
            privacy_level: (this.audience === 'nearby' ? 'public' : 'private') as PrivacyLevel
          });
        } else {
          const request: CreateMomentRequest = {
            description: this.description.trim(),
            location: {
              point: geoPoint,
              location_name: localLoc.locationName || "未知位置"
            },
            tags: this.tags.length > 0 ? this.tags : undefined,
            image_file_ids: imageFileIds.length > 0 ? imageFileIds : undefined,
            // 创建分支补传可见范围（原仅编辑分支生效，属遗漏；对齐前端 CreateSheet 隐私选择契约）
            privacy_level: (this.audience === 'nearby' ? 'public' : 'private') as PrivacyLevel
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
    if (this.isDestroyed) return;
    try {
      const location = await LocationUtil.getCurrentLocation();
      if (this.isDestroyed) return;
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
    if (this.isDestroyed) return;
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
      PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint }) {
        Stack() {
          Column() {
            SystemPageHeader({
              title: this.isEditMode ? '编辑瞬间' : '发布瞬间',
              titleSize: DesignConstants.FONT.SIZE.LG_PLUS,
              titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD,
              showBackButton: true,
              onBack: () => { AppRouter.pop(); },
              backgroundVariant: 'transparent'
            })

            Scroll() {
              Column({ space: DesignConstants.SPACING.LG }) {
                this.buildIdentityRow()
                this.buildComposerCard()
                this.buildPhotoSection()
                this.buildMoodSection()
                this.buildAudienceSection()
              }
              .width('100%')
              .padding({
                top: DesignConstants.SPACING.MD,
                bottom: DesignConstants.SPACING.XL
              })
            }
            .layoutWeight(1)
            .scrollBar(BarState.Off)
            .align(Alignment.Top)

            this.buildFooter()
          }
          .width('100%')
          .height('100%')

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
        this.audience = this.privacyLevel === 'private' ? 'private' : 'nearby';
        this.localLocation = {
          latitude: editParam.latitude,
          longitude: editParam.longitude,
          locationName: editParam.location_name
        };
      }
    })
  }

  private getCurrentUserName(): string {
    return this.stateManager.getCurrentUser()?.nickname ?? '匿名用户';
  }

  private getCurrentUserAvatar(): string {
    return this.stateManager.getCurrentUser()?.avatar_url ?? '';
  }

  // ==================== 身份行（头像 + 昵称 + 当前浏览位置） ====================
  @Builder
  private buildIdentityRow() {
    Row({ space: DesignConstants.SPACING.SM }) {
      SystemAvatar({
        avatarUrl: this.getCurrentUserAvatar(),
        name: this.getCurrentUserName(),
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS
      })

      Column({ space: DesignConstants.SPACING.THREE }) {
        Text(this.getCurrentUserName())
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
            icon: AppIcon.LOCATION,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.getCurrentThemePalette().accentBackground
          })
          Text(this.localLocation?.locationName || '当前浏览位置')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(VerticalAlign.Center)
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      SystemButton({
        label: '选点',
        variant: 'ghost',
        buttonSize: 'small',
        onPress: () => { this.openLocationPicker(); }
      })
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
  }

  // ==================== 正文输入区（对齐 web CreateSheet：正文 + 壳内工具行） ====================
  @Builder
  private buildComposerCard() {
    Column() {
      TextArea({
        text: this.description,
        placeholder: '写下此刻发生的事，或分享一个想被附近的人看见的瞬间…'
      })
        .onChange((value: string) => {
          this.description = value.length > DESCRIPTION_MAX_LENGTH ?
            value.substring(0, DESCRIPTION_MAX_LENGTH) : value;
        })
        .maxLength(DESCRIPTION_MAX_LENGTH)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .placeholderColor(this.getCurrentThemePalette().textSecondary)
        .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
        .background(DesignConstants.COLOR.TRANSPARENT)
        .border({ width: 0 })
        .borderRadius(0)
        .padding(0)
        .constraintSize({ minHeight: DesignConstants.COMPONENT.PLACEHOLDER_AREA_HEIGHT })
        .width('100%')

      Row() {
        Row({ space: DesignConstants.SPACING.MD }) {
          LucideIcon({
            icon: AppIcon.EMOJI,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            color: this.getCurrentThemePalette().textSecondary
          })
          Text('#')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        Blank()
        Text(`${this.description.length}/${DESCRIPTION_MAX_LENGTH}`)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.description.length >= DESCRIPTION_MAX_LENGTH - 20 ?
            this.getCurrentThemePalette().error : this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .margin({ top: DesignConstants.SPACING.SM })
      .padding({ top: DesignConstants.SPACING.SM })
      .border({
        width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
        color: { top: this.getSurfaceBorder() }
      })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD3)
    // ArkUI 陷阱：.border() 写在 .borderRadius() 之后会把圆角重置为 0，故边框在前
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .backgroundColor(this.getCurrentThemePalette().inputBackgroundColor)
    .clip(true)
  }

  // ==================== 照片行 ====================
  @Builder
  private buildPhotoSection() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Text('照片')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)

      if (this.isEditMode) {
        if (this.existingImageUrls.length > 0) {
          Scroll() {
            Row({ space: DesignConstants.SPACING.SM }) {
              ForEach(this.existingImageUrls, (url: string, index: number) => {
                Image(MediaUrlUtil.normalize(url))
                  .width(DesignConstants.COMPONENT.IMAGE_THUMBNAIL_SIZE)
                  .height(DesignConstants.COMPONENT.IMAGE_THUMBNAIL_SIZE)
                  .objectFit(ImageFit.Cover)
                  .borderRadius(DesignConstants.RADIUS.SM)
              }, (url: string, index: number) => `existing_${index}_${url}`)
            }
          }
          .scrollable(ScrollDirection.Horizontal)
          .scrollBar(BarState.Off)
          .align(Alignment.Start)
          .width('100%')
        }
        Text('编辑暂不支持修改图片；如需调整，请删除该瞬间后重新发布')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getCurrentThemePalette().textSecondary)
      } else {
        Scroll() {
          Row({ space: DesignConstants.SPACING.SM }) {
            ForEach(this.imageUrls, (url: string, index: number) => {
              Stack({ alignContent: Alignment.TopEnd }) {
                Image(MediaUrlUtil.normalize(url))
                  .width(DesignConstants.COMPONENT.IMAGE_THUMBNAIL_SIZE)
                  .height(DesignConstants.COMPONENT.IMAGE_THUMBNAIL_SIZE)
                  .objectFit(ImageFit.Cover)
                  .borderRadius(DesignConstants.RADIUS.SM)
                Row() {
                  LucideIcon({
                    icon: AppIcon.CLOSE,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                    color: DesignConstants.COLOR.WHITE
                  })
                }
                .width(DesignConstants.COMPONENT.ICON_SIZE_XS)
                .height(DesignConstants.COMPONENT.ICON_SIZE_XS)
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundColor(DesignConstants.COLOR.IMAGE_BADGE_BG)
                .justifyContent(FlexAlign.Center)
                .margin(DesignConstants.SPACING.HALF)
                .accessibilityText('移除图片')
                .onClick(() => this.removeImage(index))
              }
            }, (url: string, index: number) => `${index}_${url}`)

            if (this.imageUrls.length < DefaultConfig.MAX_MOMENT_IMAGES) {
              Column({ space: DesignConstants.SPACING.THREE }) {
                LucideIcon({
                  icon: AppIcon.ADD,
                  iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
                  color: this.getCurrentThemePalette().textSecondary
                })
                Text('添加')
                  .fontSize(DesignConstants.FONT.SIZE.XS2)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .width(DesignConstants.COMPONENT.IMAGE_THUMBNAIL_SIZE)
              .height(DesignConstants.COMPONENT.IMAGE_THUMBNAIL_SIZE)
              .border({
                width: DesignConstants.BORDER_WIDTH.HAIRLINE,
                color: this.getSurfaceBorder()
              })
              .borderRadius(DesignConstants.RADIUS.SM)
              .justifyContent(FlexAlign.Center)
              .accessibilityText('添加照片')
              .onClick(() => this.selectImages())
            }
          }
        }
        .scrollable(ScrollDirection.Horizontal)
        .scrollBar(BarState.Off)
        .width('100%')
        .align(Alignment.Start)
      }
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
  }

  // ==================== 氛围 chips ====================
  @Builder
  private buildMoodSection() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row() {
        Text('此刻氛围')
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
        Blank()
        if (this.tags.length > 0) {
          Text(`已选 ${this.tags.length} 项`)
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
      }
      .width('100%')

      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(MOOD_PRESETS, (mood: string) => {
          Text(mood)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontWeight(this.tags.includes(mood) ?
              DesignConstants.FONT.WEIGHT.SEMIBOLD : DesignConstants.FONT.WEIGHT.REGULAR)
            .fontColor(this.tags.includes(mood) ?
              this.getCurrentThemePalette().textAccentOnSoft : this.getCurrentThemePalette().textSecondary)
            .backgroundColor(this.tags.includes(mood) ?
              this.getCurrentThemePalette().accentMuted : DesignConstants.COLOR.TRANSPARENT)
            .border({
              width: DesignConstants.BORDER_WIDTH.HAIRLINE,
              color: this.tags.includes(mood) ?
                this.getCurrentThemePalette().accentBackground : this.getSurfaceBorder()
            })
            .borderRadius(DesignConstants.RADIUS.FULL)
            .padding({
              left: DesignConstants.SPACING.MD,
              right: DesignConstants.SPACING.MD,
              top: DesignConstants.SPACING.SM2,
              bottom: DesignConstants.SPACING.SM2
            })
            .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
            .accessibilityText(this.tags.includes(mood) ? `氛围 ${mood}，已选中` : `氛围 ${mood}`)
            .onClick(() => this.toggleMood(mood))
        }, (mood: string) => mood)
      }
      .width('100%')
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
  }

  // ==================== 可见范围（契约仅 public/private 两档） ====================
  @Builder
  private buildAudienceSection() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row() {
        Text('谁可以看到')
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
        Blank()
        Text(this.resolveAudienceHint())
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')

      TabsPillGroup({
        variant: 'segmented',
        tabs: [
          { label: '附近可见', icon: AppIcon.NEARBY },
          { label: '仅自己', icon: AppIcon.LOCK }
        ] as TabItem[],
        activeIndex: this.audience === 'nearby' ? 0 : 1,
        actions: {
          onChange: (index: number) => {
            this.audience = index === 0 ? 'nearby' : 'private';
          }
        }
      })
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding(DesignConstants.SPACING.SM)
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
  }

  // ==================== 吸底发布条 ====================
  @Builder
  private buildFooter() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Text(this.isPublishing ? this.publishingMessage :
        (this.tags.length > 0 ?
          `将以 ${this.tags.join(' · ')} 的氛围出现` :
          '选择氛围，让附近的人更容易找到你'))
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .layoutWeight(1)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })

      SystemButton({
        label: this.isEditMode ? '保存修改' : '发布瞬间',
        variant: 'primary',
        buttonSize: 'default',
        loading: this.isPublishing,
        disabled: !this.canPublish(),
        onPress: () => { this.publishMoment(); }
      })
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD,
      bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.MD, 0)
    })
    .border({
      width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
      color: { top: this.getSurfaceBorder() }
    })
  }
}

@Builder
export function CreateMomentPageBuilder() {
  CreateMomentPage()
}