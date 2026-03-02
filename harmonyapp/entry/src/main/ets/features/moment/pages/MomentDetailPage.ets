/**
 * 动态详情页
 * 支持查看详情、点赞、收藏、分享
 */

import { ApiClient } from '../../../core/api/ApiClient';
import { GlobalStateManager } from '../../../core/state/GlobalStateManager';
import { AppRouter, LoginRouteParam, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import {
  FavoriteMomentRequest,
  LikeMomentRequest,
  MomentImage,
  MomentItem,
  MomentMyState,
  UnfavoriteMomentRequest,
  UnlikeMomentRequest
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

  private stateManager = GlobalStateManager.getInstance();
  private momentId: string = '';

  @State moment: MomentItem | null = null;
  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State likePending: boolean = false;
  @State favoritePending: boolean = false;

  aboutToAppear(): void {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,
      preferDarkContent: true
    });
  }

  private ensureLoggedIn(actionName: string): boolean {
    if (this.stateManager.getCurrentUser()) {
      return true;
    }
    SystemToast.showWarning(`请先登录后再${actionName}`, this.getUIContext());
    const loginParams = new LoginRouteParam();
    loginParams.reason = 'action_required';
    loginParams.action = actionName;
    AppRouter.push(PageNames.LOGIN, loginParams);
    return false;
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

  private cloneMomentWithState(
    current: MomentItem,
    nextMyState: MomentMyState,
    nextLikesCount: number
  ): MomentItem {
    return {
      moment_id: current.moment_id,
      description: current.description,
      images: current.images,
      tags: current.tags,
      mentions: current.mentions,
      created_at: current.created_at,
      likes_count: nextLikesCount < 0 ? 0 : nextLikesCount,
      comments_count: current.comments_count,
      shares_count: current.shares_count,
      user: current.user,
      location: current.location,
      privacy_level: current.privacy_level,
      my_state: nextMyState,
      recommendation: current.recommendation
    };
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
    if (!this.ensureLoggedIn('点赞瞬间')) return;
    if (this.likePending) return;

    this.likePending = true;
    const current = this.moment;
    const currentLiked = current.my_state?.is_liked ?? false;
    const currentFavorited = current.my_state?.is_favorited ?? false;

    try {
      const request: LikeMomentRequest | UnlikeMomentRequest = { moment_id: current.moment_id };
      const response = currentLiked
        ? await ApiClient.unlikeMoment(request as UnlikeMomentRequest)
        : await ApiClient.likeMoment(request as LikeMomentRequest);
      if (response.code !== 0) {
        SystemToast.showError(response.message || '操作失败', this.getUIContext());
        return;
      }

      const nextLiked = !currentLiked;
      const nextLikesCount = current.likes_count + (nextLiked ? 1 : -1);
      this.moment = this.cloneMomentWithState(
        current,
        { is_liked: nextLiked, is_favorited: currentFavorited },
        nextLikesCount
      );
    } catch {
      SystemToast.showError('点赞操作失败', this.getUIContext());
    } finally {
      this.likePending = false;
    }
  }

  private async toggleFavorite(): Promise<void> {
    if (!this.moment) return;
    if (!this.ensureLoggedIn('收藏瞬间')) return;
    if (this.favoritePending) return;

    this.favoritePending = true;
    const current = this.moment;
    const currentLiked = current.my_state?.is_liked ?? false;
    const currentFavorited = current.my_state?.is_favorited ?? false;

    try {
      const request: FavoriteMomentRequest | UnfavoriteMomentRequest = { moment_id: current.moment_id };
      const response = currentFavorited
        ? await ApiClient.unfavoriteMoment(request as UnfavoriteMomentRequest)
        : await ApiClient.favoriteMoment(request as FavoriteMomentRequest);
      if (response.code !== 0) {
        SystemToast.showError(response.message || '操作失败', this.getUIContext());
        return;
      }

      const nextFavorited = !currentFavorited;
      this.moment = this.cloneMomentWithState(
        current,
        { is_liked: currentLiked, is_favorited: nextFavorited },
        current.likes_count
      );
    } catch {
      SystemToast.showError('收藏操作失败', this.getUIContext());
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
    return moment.my_state?.is_liked ?? false;
  }

  private isMomentFavorited(moment: MomentItem): boolean {
    return moment.my_state?.is_favorited ?? false;
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
    .backgroundColor($r('app.color.bg_primary'))
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
                .backgroundColor($r('app.color.bg_secondary'))
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
              .backgroundColor($r('app.color.bg_tertiary'))
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
        Text(`点赞 ${formatCount(moment.likes_count)}`)
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
      .type(ButtonType.Normal)
      .layoutWeight(1)
      .height(40)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .shadow(DesignConstants.SHADOW.SMALL)
      .border({ width: 1, color: $r('app.color.highlight_border') })
      .enabled(!this.likePending)
      .opacity(this.likePending ? 0.58 : 1)
      .backgroundColor($r('app.color.bg_tertiary'))
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
      .type(ButtonType.Normal)
      .layoutWeight(1)
      .height(40)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .shadow(DesignConstants.SHADOW.SMALL)
      .border({ width: 1, color: $r('app.color.highlight_border') })
      .enabled(!this.favoritePending)
      .opacity(this.favoritePending ? 0.58 : 1)
      .backgroundColor($r('app.color.bg_tertiary'))
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
      .type(ButtonType.Normal)
      .layoutWeight(1)
      .height(40)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .shadow(DesignConstants.SHADOW.SMALL)
      .border({ width: 1, color: $r('app.color.highlight_border') })
      .backgroundColor($r('app.color.bg_tertiary'))
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
    .backgroundColor($r('app.color.bg_secondary'))
    .border({ width: { top: 1 }, color: $r('app.color.divider') })
    .borderRadius({ topLeft: DesignConstants.RADIUS.LG, topRight: DesignConstants.RADIUS.LG })
  }
}

@Builder
export function MomentDetailPageBuilder() {
  MomentDetailPage()
}
