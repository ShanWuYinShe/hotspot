/**
 * 动态详情页
 * 支持查看详情、点赞、收藏、分享
 */

import { ApiClient } from '../../../core/api/ApiClient';
import { GlobalStateManager } from '../../../core/state/GlobalStateManager';
import { interactionActions } from '../../../core/state/InteractionActions';
import { InteractionStateStore, MomentSocialState } from '../../../core/state/InteractionStateStore';
import { AppRouter, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import {
  MomentImage,
  MomentItem,
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { EmptyState } from '../../../components/EmptyState';
import { AppIcon, IconButton, IconSize } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SystemShareService } from '../../../core/services/SystemShareService';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { formatCount } from '../../../core/utils/FormatUtil';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF1A;
const TAG: string = 'MomentDetailPage';

interface MomentDetailParams {
  momentId?: string;
  moment_id?: string;
}

@Preview
@Component
export struct MomentDetailPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink('interaction_moment_state') momentStateMap: Record<string, MomentSocialState> = {};
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private stateManager = GlobalStateManager.getInstance();
  private momentId: string = '';

  @State moment: MomentItem | null = null;
  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State likePending: boolean = false;
  @State favoritePending: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
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

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private formatTime(timeStr: string): string {
    const time = new Date(timeStr);
    const now = new Date();
    if (time.toDateString() === now.toDateString()) {
      return `${time.getHours().toString().padStart(2, '0')}:${time.getMinutes().toString().padStart(2, '0')}`;
    }
    return time.toLocaleDateString();
  }


  private calcImageGridHeight(imageCount: number): number {
    if (imageCount === 0) return 0;
    const rows = Math.ceil(Math.min(imageCount, 9) / 3);
    const gap = 8;
    const itemHeight = 110;
    return rows * itemHeight + (rows - 1) * gap;
  }

  private async loadMomentDetail(): Promise<void> {
    if (!this.momentId) {
      this.errorMessage = '缺少动态ID';
      this.isLoading = false;
      return;
    }

    this.isLoading = true;
    this.errorMessage = '';
    try {
      const response = await ApiClient.getMomentDetail({ moment_id: this.momentId });
      if (response.code === 0 && response.data) {
        this.moment = response.data;
        InteractionStateStore.getInstance().patchMomentFromItem(response.data);
      } else {
        this.errorMessage = response.message || '加载动态详情失败';
      }
    } catch (error) {
      this.errorMessage = '加载动态详情失败';
      hilog.error(DOMAIN_NUMBER, TAG, `loadMomentDetail failed: ${JSON.stringify(error)}`);
    } finally {
      this.isLoading = false;
    }
  }

  private async toggleLike(): Promise<void> {
    if (!this.moment) return;
    if (this.likePending) return;
    this.likePending = true;
    try {
      const success = await interactionActions.toggleMomentLike(this.moment.moment_id);
      if (!success) {
        SystemToast.showError('点赞操作失败', this.getUIContext());
      }
    } finally {
      this.likePending = false;
    }
  }

  private async toggleFavorite(): Promise<void> {
    if (!this.moment) return;
    if (this.favoritePending) return;
    this.favoritePending = true;
    try {
      const success = await interactionActions.toggleMomentFavorite(this.moment.moment_id);
      if (!success) {
        SystemToast.showError('收藏操作失败', this.getUIContext());
      }
    } finally {
      this.favoritePending = false;
    }
  }

  private async shareMoment(): Promise<void> {
    if (!this.moment) return;
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const content = SystemShareService.buildMomentShareContent(
        this.moment.description,
        this.moment.user?.nickname ?? '匿名用户',
        this.moment.moment_id
      );
      await SystemShareService.getInstance().shareText(context, content);
    } catch {
      // 用户取消分享或系统分享失败，不阻断主流程
    }
  }

  private previewImage(index: number): void {
    if (!this.moment || !this.moment.images || this.moment.images.length === 0) return;
    const previewParams = new PhotoPreviewRouteParam();
    previewParams.momentId = this.moment.moment_id;
    previewParams.images = this.moment.images.map((image: MomentImage) => MediaUrlUtil.normalize(image.url));
    previewParams.index = index;
    AppRouter.push(PageNames.PHOTO_PREVIEW, previewParams);
  }

  private isMomentLiked(moment: MomentItem): boolean {
    const state = this.momentStateMap[moment.moment_id];
    if (state) {
      return state.my_state?.is_liked ?? false;
    }
    return moment.my_state?.is_liked ?? false;
  }

  private isMomentFavorited(moment: MomentItem): boolean {
    const state = this.momentStateMap[moment.moment_id];
    if (state) {
      return state.my_state?.is_favorited ?? false;
    }
    return moment.my_state?.is_favorited ?? false;
  }

  private getMomentLikesCount(moment: MomentItem): number {
    const state = this.momentStateMap[moment.moment_id];
    return state ? state.likes_count : moment.likes_count;
  }

  build() {
    NavDestination() {
      Column() {
        this.buildTopBar()
        if (this.isLoading) {
          SystemLoadingView({ message: '正在加载动态详情...' })
        } else if (!this.moment) {
          EmptyState({
            icon: AppIcon.WARNING,
            message: this.errorMessage || '动态不存在或已被删除',
            actionLabel: '重试',
            onAction: () => { this.loadMomentDetail(); }
          })
        } else {
          Scroll() {
            this.buildMomentContent(this.moment as MomentItem)
          }
          .layoutWeight(1)
          .width('100%')
          .align(Alignment.TopStart)

          this.buildActionBar(this.moment as MomentItem)
        }
      }
      .width('100%')
      .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as MomentDetailParams | undefined;
      this.momentId = params?.momentId ?? params?.moment_id ?? '';
      this.loadMomentDetail();
    })
  }

  @Builder
  private buildTopBar() {
    Row({ space: DesignConstants.SPACING.MD }) {
      IconButton({
        icon: AppIcon.BACK,
        iconSize: IconSize.MEDIUM,
        bgColor: Color.Transparent,
        onButtonClick: () => {
          AppRouter.pop();
        }
      })

      Text('动态详情')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .layoutWeight(1)
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
      bottom: DesignConstants.SPACING.SM
    })
    .backgroundColor(Color.Transparent)
    .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])
  }

  @Builder
  private buildMomentContent(moment: MomentItem) {
    Column({ space: DesignConstants.SPACING.LG }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: moment.user?.avatar_url ?? '',
          name: moment.user?.nickname ?? '未知用户',
          avatarSize: 44,
          userInfo: moment.user
        })

        Column({ space: DesignConstants.SPACING.XS }) {
          Text(moment.user?.nickname ?? '未知用户')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))

          Row({ space: DesignConstants.SPACING.SM }) {
            Text(this.formatTime(moment.created_at))
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            if (moment.location?.distance !== undefined) {
              Text(LocationUtil.formatDistance(moment.location.distance))
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            }
          }
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)
      }
      .width('100%')

      Text(moment.description)
        .width('100%')
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .lineHeight(26)

      if (moment.images && moment.images.length > 0) {
        Grid() {
          ForEach(moment.images.slice(0, 9), (img: MomentImage, imgIndex: number) => {
            GridItem() {
              Image(MediaUrlUtil.normalize(img.url))
                .width('100%')
                .height('100%')
                .objectFit(ImageFit.Cover)
                .borderRadius(DesignConstants.RADIUS.MD)
                .backgroundColor(this.getSurfaceBackground())
                .onClick(() => {
                  this.previewImage(imgIndex);
                })
            }
            .aspectRatio(1)
          }, (img: MomentImage, index: number) => `${img.url}_${index}`)
        }
        .columnsTemplate('1fr 1fr 1fr')
        .columnsGap(DesignConstants.SPACING.SM)
        .rowsGap(DesignConstants.SPACING.SM)
        .height(this.calcImageGridHeight(moment.images.length))
        .width('100%')
      }

      if (moment.tags && moment.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(moment.tags, (tag: string, index: number) => {
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_emphasize'))
              .backgroundColor(this.getSurfaceSubtle())
              .padding({
                left: DesignConstants.SPACING.MD,
                right: DesignConstants.SPACING.MD,
                top: DesignConstants.SPACING.XS,
                bottom: DesignConstants.SPACING.XS
              })
              .borderRadius(DesignConstants.RADIUS.MD)
              .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
          }, (tag: string, index: number) => `${tag}_${index}`)
        }
        .width('100%')
      }

      if (moment.location?.location_name) {
        Row({ space: DesignConstants.SPACING.XS }) {
          Text() {
            SymbolSpan($r(AppIcon.LOCATION))
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
          Text(moment.location.location_name)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .onClick(() => {
          if (moment.location?.point) {
            const locationInfo: LocationInfo = {
              latitude: moment.location.point.latitude,
              longitude: moment.location.point.longitude,
              locationName: moment.location.location_name ?? ''
            };
            AppStorage.setOrCreate<LocationInfo | null>(KEY_BROWSE_ANCHOR, locationInfo);
            SystemToast.showSuccess('已设为浏览锚点', this.getUIContext());
          }
        })
      }

      Row({ space: DesignConstants.SPACING.LG }) {
        Text(`点赞 ${formatCount(this.getMomentLikesCount(moment))}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        Text(`评论 ${formatCount(moment.comments_count)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        Text(`分享 ${formatCount(moment.shares_count)}`)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .alignItems(HorizontalAlign.Start)
  }

  @Builder
  private buildActionBar(moment: MomentItem) {
    Row({ space: DesignConstants.SPACING.LG }) {
      Button() {
        Row({ space: 6 }) {
          Text() {
            SymbolSpan($r(this.isMomentLiked(moment) ? AppIcon.HEART_FILL : AppIcon.HEART))
              .fontSize(18)
              .fontColor([this.isMomentLiked(moment) ? $r('app.color.color_danger') : $r('sys.color.ohos_id_color_text_secondary')])
          }
          Text(this.likePending ? '处理中...' : (this.isMomentLiked(moment) ? '已点赞' : '点赞'))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
        }
      }
      .type(ButtonType.Capsule)
      .layoutWeight(1)
      .height(40)
      .shadow(DesignConstants.SHADOW.SMALL)
      .border({ width: 1, color: this.getSurfaceBorder() })
      .enabled(!this.likePending)
      .opacity(this.likePending ? 0.58 : 1)
      .backgroundColor(this.getSurfaceSubtle())
      .onClick(() => {
        this.toggleLike();
      })

      Button() {
        Row({ space: 6 }) {
          Text() {
            SymbolSpan($r(this.isMomentFavorited(moment) ? AppIcon.STAR_FILL : AppIcon.STAR))
              .fontSize(18)
              .fontColor([this.isMomentFavorited(moment) ? $r('sys.color.ohos_id_color_warning') : $r('sys.color.ohos_id_color_text_secondary')])
          }
          Text(this.favoritePending ? '处理中...' : (this.isMomentFavorited(moment) ? '已收藏' : '收藏'))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
        }
      }
      .type(ButtonType.Capsule)
      .layoutWeight(1)
      .height(40)
      .shadow(DesignConstants.SHADOW.SMALL)
      .border({ width: 1, color: this.getSurfaceBorder() })
      .enabled(!this.favoritePending)
      .opacity(this.favoritePending ? 0.58 : 1)
      .backgroundColor(this.getSurfaceSubtle())
      .onClick(() => {
        this.toggleFavorite();
      })

      Button() {
        Row({ space: 6 }) {
          Text() {
            SymbolSpan($r(AppIcon.SHARE))
              .fontSize(18)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
          Text('分享')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
        }
      }
      .type(ButtonType.Capsule)
      .layoutWeight(1)
      .height(40)
      .shadow(DesignConstants.SHADOW.SMALL)
      .border({ width: 1, color: this.getSurfaceBorder() })
      .backgroundColor(this.getSurfaceSubtle())
      .onClick(() => {
        this.shareMoment();
      })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD + this.safeAreaBottom
    })
    .backgroundColor(this.getSurfaceBackground())
    .border({ width: { top: 1 }, color: $r('app.color.divider') })
    .borderRadius({ topLeft: DesignConstants.RADIUS.LG, topRight: DesignConstants.RADIUS.LG })
  }
}

@Builder
export function MomentDetailPageBuilder() {
  MomentDetailPage()
}
