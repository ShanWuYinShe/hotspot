import { GlobalStateManager, KEY_BROWSE_ANCHOR, KEY_CURRENT_USER, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { AccountInfo, CreateMomentRequest, GeoPoint, MomentItem, PrivacyLevel } from '../../../core/types/api';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToast } from '../../../components/SystemToast';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { LucideIcon } from '../../../components/LucideIcon';

const DOMAIN_NUMBER: number = 0xFF28;
const TAG: string = 'CreateMomentSheet';

/** 此刻氛围预设（对齐前端 CreateSheet TagPill 六枚） */
const MOOD_PRESETS: string[] = ['松弛', '漫游', '咖啡', '展览', '自然', '夜色'];
/** 正文字数上限（对齐前端 CreateSheet 280 字契约） */
const DESCRIPTION_MAX_LENGTH: number = 280;

/**
 * 发动态底部 Sheet（对齐前端 CreateSheet）：拖拽把手 + eyebrow「附近动态」+
 * 标题「记录此刻」+ 右上「创建群组」切换 pill；身份行、正文卡（280 上限 +
 * n/280 计数）、80×80 照片选择、氛围 TagPill 六枚、可见范围图标卡（契约
 * PrivacyLevel 仅 public/private 两档：「附近可见 / 仅自己」）、footer 发布 pill。
 * 发布链路复刻 CreateMomentPage：串行压缩上传拿 file_id → createMoment。
 */
@Component
export struct CreateMomentSheet {
  onClose: () => void = () => {
  };
  /** 切换到创建群组（由宿主决定跳路由页或另一 sheet） */
  onSwitchToGroup: () => void = () => {
  };
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private description: string = '';
  @State private imageUrls: string[] = [];
  @State private selectedMoods: string[] = [];
  @State private privacyLevel: PrivacyLevel = 'public';
  @State private isPublishing: boolean = false;
  @State private publishingMessage: string = '';
  @State private localLocation: LocationInfo | null = null;

  aboutToAppear(): void {
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (refLoc) {
      this.localLocation = {
        latitude: refLoc.latitude,
        longitude: refLoc.longitude,
        locationName: refLoc.locationName
      };
    }
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private canPublish(): boolean {
    return !this.isPublishing && this.description.trim().length > 0 && this.localLocation !== null;
  }

  private toggleMood(mood: string): void {
    const index = this.selectedMoods.indexOf(mood);
    if (index >= 0) {
      this.selectedMoods = this.selectedMoods.filter((item: string) => item !== mood);
    } else {
      this.selectedMoods = this.selectedMoods.concat([mood]);
    }
  }

  private async selectImages(): Promise<void> {
    if (this.imageUrls.length >= 9) {
      SystemToast.show('最多只能选择9张图片', this.getUIContext());
      return;
    }
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selected = await MediaServiceManager.selectImages(context, 9 - this.imageUrls.length);
      if (selected.length > 0) {
        this.imageUrls = this.imageUrls.concat(selected);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `选择图片失败: ${JSON.stringify(error)}`);
      SystemToast.show('选择图片失败，请重试', this.getUIContext());
    }
  }

  private removeImage(index: number): void {
    this.imageUrls = this.imageUrls.filter((_item: string, i: number) => i !== index);
  }

  private async publish(): Promise<void> {
    if (!this.canPublish() || !this.localLocation) {
      if (!this.localLocation) {
        SystemToast.show('位置信息不可用，请先在首页选择浏览位置', this.getUIContext());
      }
      return;
    }
    this.isPublishing = true;
    this.publishingMessage = '正在发布...';
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const localLoc = this.localLocation;
      const imageFileIds: string[] = [];

      // 串行压缩并上传（对齐 CreateMomentPage：压缩失败降级原图）
      for (let i = 0; i < this.imageUrls.length; i++) {
        this.publishingMessage = `正在上传第 ${i + 1}/${this.imageUrls.length} 张图片...`;
        try {
          const compressedPath = await MediaServiceManager.compressImage(context, this.imageUrls[i], 80, 1024 * 1024 * 2);
          const uploadUri = await MediaServiceManager.prepareFileForUpload(context, compressedPath);
          const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'moment_image');
          if (uploadResult && uploadResult.file_id) {
            imageFileIds.push(uploadResult.file_id);
          }
        } catch (processError) {
          hilog.warn(DOMAIN_NUMBER, TAG, `压缩失败，降级原图: ${JSON.stringify(processError)}`);
          const uploadUri = await MediaServiceManager.prepareFileForUpload(context, this.imageUrls[i]);
          const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'moment_image');
          if (uploadResult && uploadResult.file_id) {
            imageFileIds.push(uploadResult.file_id);
          }
        }
      }

      this.publishingMessage = '正在发布...';
      const geoPoint: GeoPoint = { latitude: localLoc.latitude, longitude: localLoc.longitude };
      const request: CreateMomentRequest = {
        description: this.description.trim(),
        location: {
          point: geoPoint,
          location_name: localLoc.locationName || '未知位置'
        },
        tags: this.selectedMoods.length > 0 ? this.selectedMoods : undefined,
        image_file_ids: imageFileIds.length > 0 ? imageFileIds : undefined,
        privacy_level: this.privacyLevel
      };
      const response = await ApiClient.createMoment(request);
      if (response.code === 0) {
        SystemToast.showSuccess('瞬间发布成功', this.getUIContext());
        this.onClose();
      } else {
        SystemToast.showError(response.message || '发布瞬间失败', this.getUIContext());
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `发布失败: ${JSON.stringify(error)}`);
      SystemToast.showError('发布瞬间失败', this.getUIContext());
    } finally {
      this.isPublishing = false;
      this.publishingMessage = '';
    }
  }

  // ==================== 头部 ====================
  @Builder
  private buildHeader() {
    Column({ space: DesignConstants.SPACING.SM }) {
      // 拖拽把手
      Column()
        .width(DesignConstants.SPACING.XL)
        .height(DesignConstants.SPACING.HALF)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getPalette().borderColor)
        .margin({ top: DesignConstants.SPACING.SM })

      Row() {
        Column({ space: DesignConstants.SPACING.THREE }) {
          Text('附近动态')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPalette().accentText)
            .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
          Text('记录此刻')
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().textPrimary)
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        // 创建群组切换 pill
        Text('创建群组')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getPalette().accentText)
          .backgroundColor(this.getPalette().accentMuted)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().accentBackground })
          .borderRadius(DesignConstants.RADIUS.FULL)
          .padding({
            left: DesignConstants.SPACING.MD2,
            right: DesignConstants.SPACING.MD2,
            top: DesignConstants.SPACING.THREE,
            bottom: DesignConstants.SPACING.THREE
          })
          .onClick(() => {
            this.onClose();
            this.onSwitchToGroup();
          })
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.SM
    })
  }

  // ==================== 身份行 ====================
  @Builder
  private buildIdentityRow() {
    Row({ space: DesignConstants.SPACING.SM }) {
      SystemAvatar({
        avatarUrl: this.currentUser?.avatar_url ?? '',
        name: this.currentUser?.nickname ?? '',
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS
      })

      Column({ space: DesignConstants.SPACING.THREE }) {
        Text(this.currentUser?.nickname ?? '匿名用户')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getPalette().textPrimary)
        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
              icon: AppIcon.LOCATION,
              iconSize: DesignConstants.FONT.SIZE.XS2,
              color: this.getPalette().textSecondary
            })
          Text(this.localLocation?.locationName || '当前浏览位置')
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .alignItems(VerticalAlign.Center)
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Text('正在发布')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontColor(this.getPalette().textSecondary)
        .backgroundColor(this.getPalette().surfaceSubtle)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .padding({
          left: DesignConstants.SPACING.SM2,
          right: DesignConstants.SPACING.SM2,
          top: DesignConstants.SPACING.HALF,
          bottom: DesignConstants.SPACING.HALF
        })
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
  }

  // ==================== 正文卡 ====================
  @Builder
  private buildComposerCard() {
    Column() {
      TextArea({ text: this.description, placeholder: '分享此刻的见闻与心情…' })
        .onChange((value: string) => {
          this.description = value.length > DESCRIPTION_MAX_LENGTH ?
            value.substring(0, DESCRIPTION_MAX_LENGTH) : value;
        })
        .maxLength(DESCRIPTION_MAX_LENGTH)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor(this.getPalette().textPrimary)
        .placeholderColor(this.getPalette().textSecondary)
        .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
        .padding(0)
        .constraintSize({ minHeight: 120 })
        .width('100%')

      Row() {
        Blank()
        Text(`${this.description.length}/${DESCRIPTION_MAX_LENGTH}`)
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(this.description.length >= DESCRIPTION_MAX_LENGTH - 20 ?
            this.getPalette().error :
            this.getPalette().textSecondary)
      }
      .width('100%')
      .margin({ top: DesignConstants.SPACING.SM })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .backgroundColor(this.getPalette().inputBackgroundColor)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
  }

  // ==================== 照片行 ====================
  @Builder
  private buildPhotoSection() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Text('照片')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontColor(this.getPalette().accentText)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)

      Scroll() {
        Row({ space: DesignConstants.SPACING.SM }) {
          ForEach(this.imageUrls, (url: string, index: number) => {
            Stack({ alignContent: Alignment.TopEnd }) {
              Image(MediaUrlUtil.normalize(url))
                .width(80)
                .height(80)
                .objectFit(ImageFit.Cover)
                .borderRadius(DesignConstants.RADIUS.SM)
              Row() {
                LucideIcon({
                    icon: AppIcon.CLOSE,
                    iconSize: DesignConstants.FONT.SIZE.XS,
                    color: DesignConstants.COLOR.WHITE
                  })
              }
              .width(DesignConstants.COMPONENT.ICON_SIZE_XS)
              .height(DesignConstants.COMPONENT.ICON_SIZE_XS)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(DesignConstants.COLOR.IMAGE_BADGE_BG)
              .justifyContent(FlexAlign.Center)
              .margin(DesignConstants.SPACING.HALF)
              .onClick(() => this.removeImage(index))
            }
          }, (url: string) => url)

          if (this.imageUrls.length < 9) {
            Column({ space: DesignConstants.SPACING.THREE }) {
              LucideIcon({
                  icon: AppIcon.ADD,
                  iconSize: DesignConstants.FONT.SIZE.LG,
                  color: this.getPalette().textSecondary
                })
              Text('添加')
                .fontSize(DesignConstants.FONT.SIZE.XS2)
                .fontColor(this.getPalette().textSecondary)
            }
            .width(80)
            .height(80)
            .borderRadius(DesignConstants.RADIUS.SM)
            .border({
              width: DesignConstants.BORDER_WIDTH.HAIRLINE,
              color: this.getPalette().borderColor
            })
            .justifyContent(FlexAlign.Center)
            .onClick(() => this.selectImages())
          }
        }
      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)
      .width('100%')
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
  }

  // ==================== 氛围 TagPill ====================
  @Builder
  private buildMoodSection() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row() {
        Text('此刻氛围')
          .fontSize(DesignConstants.FONT.SIZE.XS2)
          .fontColor(this.getPalette().accentText)
          .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
        Blank()
        if (this.selectedMoods.length > 0) {
          Text(`已选 ${this.selectedMoods.length} 项`)
            .fontSize(DesignConstants.FONT.SIZE.XS2)
            .fontColor(this.getPalette().textSecondary)
        }
      }
      .width('100%')

      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(MOOD_PRESETS, (mood: string) => {
          Text(mood)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontWeight(this.selectedMoods.indexOf(mood) >= 0 ?
              DesignConstants.FONT.WEIGHT.SEMIBOLD : DesignConstants.FONT.WEIGHT.REGULAR)
            .fontColor(this.selectedMoods.indexOf(mood) >= 0 ?
              this.getPalette().accentText :
              this.getPalette().textSecondary)
            .backgroundColor(this.selectedMoods.indexOf(mood) >= 0 ?
              this.getPalette().accentMuted :
              DesignConstants.COLOR.TRANSPARENT)
            .border({
              width: DesignConstants.BORDER_WIDTH.HAIRLINE,
              color: this.selectedMoods.indexOf(mood) >= 0 ?
                this.getPalette().accentBackground :
                this.getPalette().borderColor
            })
            .borderRadius(DesignConstants.RADIUS.FULL)
            .padding({
              left: DesignConstants.SPACING.MD,
              right: DesignConstants.SPACING.MD,
              top: DesignConstants.SPACING.SM2,
              bottom: DesignConstants.SPACING.SM2
            })
            .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
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
  private buildPrivacySection() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Text('谁可以看到')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontColor(this.getPalette().accentText)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)

      Row({ space: DesignConstants.SPACING.SM }) {
        this.buildPrivacyOption('public', AppIcon.NEARBY, '附近可见', '5km 内的用户可发现')
        this.buildPrivacyOption('private', AppIcon.LOCK, '仅自己', '仅你可见')
      }
      .width('100%')
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding(DesignConstants.SPACING.SM)
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .backgroundColor(this.getPalette().surfaceSubtle)
  }

  @Builder
  private buildPrivacyOption(level: PrivacyLevel, icon: AppIcon, title: string, hint: string) {
    Column({ space: DesignConstants.SPACING.THREE }) {
      LucideIcon({
          icon: icon,
          iconSize: DesignConstants.FONT.SIZE.LG,
          color: this.privacyLevel === level ? this.getPalette().accentText : this.getPalette().textSecondary
        })
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.privacyLevel === level ?
          this.getPalette().accentText :
          this.getPalette().textPrimary)
      Text(hint)
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontColor(this.getPalette().textSecondary)
    }
    .layoutWeight(1)
    .alignItems(HorizontalAlign.Center)
    .padding({ top: DesignConstants.SPACING.MD2, bottom: DesignConstants.SPACING.MD2 })
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
    .backgroundColor(this.privacyLevel === level ?
      this.getPalette().accentMuted :
      DesignConstants.COLOR.TRANSPARENT)
    .border({
      width: DesignConstants.BORDER_WIDTH.HAIRLINE,
      color: this.privacyLevel === level ?
        this.getPalette().accentBackground :
        this.getPalette().borderColor
    })
    .onClick(() => {
      this.privacyLevel = level;
    })
  }

  // ==================== Footer ====================
  @Builder
  private buildFooter() {
    Row() {
      Text(this.isPublishing ? this.publishingMessage : '发布后附近的用户可以看到这条动态')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(this.getPalette().textSecondary)
        .layoutWeight(1)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })

      Text(this.isPublishing ? '发布中…' : '发布')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.canPublish() ?
          this.getPalette().accentText :
          this.getPalette().textSecondary)
        .backgroundColor(this.canPublish() ?
          this.getPalette().accentBackground :
          DesignConstants.COLOR.TRANSPARENT)
        .border({
          width: DesignConstants.BORDER_WIDTH.HAIRLINE,
          color: this.canPublish() ? this.getPalette().accentBackground : this.getPalette().borderColor
        })
        .borderRadius(DesignConstants.RADIUS.FULL)
        .padding({
          left: DesignConstants.SPACING.LG2,
          right: DesignConstants.SPACING.LG2,
          top: DesignConstants.SPACING.SM2,
          bottom: DesignConstants.SPACING.SM2
        })
        .shadow(this.canPublish() ? this.getPalette().accentShadow : undefined)
        .onClick(() => {
          if (this.canPublish()) {
            this.publish();
          }
        })
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.LG
    })
    .border({
      width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
      color: { top: this.getPalette().borderColor }
    })
  }

  build() {
    Column() {
      this.buildHeader()

      Scroll() {
        Column({ space: DesignConstants.SPACING.LG }) {
          this.buildIdentityRow()
          this.buildComposerCard()
          this.buildPhotoSection()
          this.buildMoodSection()
          this.buildPrivacySection()
        }
        .width('100%')
        .padding({
          left: DesignConstants.SPACING.LG,
          right: DesignConstants.SPACING.LG,
          bottom: DesignConstants.SPACING.LG
        })
      }
      .layoutWeight(1)
      .scrollBar(BarState.Auto)
      .align(Alignment.Top)

      this.buildFooter()
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getPalette().pageBackground)
  }
}
