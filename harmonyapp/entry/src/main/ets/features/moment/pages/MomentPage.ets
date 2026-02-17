/**
 * 生活瞬间页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 * LazyForEach + RDB 缓存
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { LikeMomentRequest, MomentItem, SearchMomentsRequest, MomentImage } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { IntelligentLocationCard } from '../../../components/IntelligentLocationCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon, IconSize } from '../../../components/SystemIcon';
import { SystemShareService } from '../../../core/services/SystemShareService';
import { common } from '@kit.AbilityKit';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemToast } from '../../../components/SystemToast';
import { MomentCacheService } from '../../../core/database/MomentCacheService';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter, LoginRouteParam, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';

const DOMAIN_NUMBER: number = 0xFF05;
const TAG: string = 'MomentPage';

@Component
export struct MomentPage {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;

  private stateManager = GlobalStateManager.getInstance();
  private momentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource([]);
  private momentCacheService = MomentCacheService.getInstance();

  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State hasData: boolean = false;
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State isLoadingMore: boolean = false;

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    await this.loadNearbyMoments();
  }

  aboutToDisappear() {}

  /** 获取搜索参考点：优先浏览锚点，回退到当前定位 */
  private getRefLocation(): LocationInfo | null {
    return this.browseAnchor ?? this.currentLocation;
  }

  private async loadNearbyMoments(forceRefresh: boolean = false) {
    const refLoc = this.getRefLocation();

    if (!refLoc && !forceRefresh) {
      await this.loadFromCache();
    }

    if (!refLoc) {
      return;
    }

    if (!forceRefresh && this.momentsDataSource.totalCount() === 0) {
      await this.loadFromCache();
    }

    if (this.momentsDataSource.totalCount() === 0) {
      this.isLoading = true;
    }
    this.errorMessage = '';

    try {
      const request: SearchMomentsRequest = {
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        radius: DefaultConfig.MOMENT_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.searchMoments(request));

      if (response.code === 0 && response.data) {
        const validItems = response.data.items.filter((m: MomentItem) => m.moment_id && m.user);
        this.momentsDataSource.setData(validItems);
        this.hasData = validItems.length > 0;
        this.hasMore = !!(response.data.next_cursor);
        this.nextCursor = response.data.next_cursor ?? '';
        this.updateCache(response.data.items);
      } else {
        if (this.momentsDataSource.totalCount() === 0) {
          this.errorMessage = response.message || '加载瞬间失败';
          this.hasData = false;
        }
      }
    } catch (error) {
      if (this.momentsDataSource.totalCount() === 0) {
        this.errorMessage = error instanceof Error ? error.message : '加载瞬间失败';
        this.hasData = false;
      }
    } finally {
      this.isLoading = false;
    }
  }

  private async loadMoreMoments(): Promise<void> {
    const refLoc = this.getRefLocation();
    if (this.isLoadingMore || !this.hasMore || !refLoc) return;
    this.isLoadingMore = true;
    try {
      const request: SearchMomentsRequest = {
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        radius: DefaultConfig.MOMENT_RADIUS,
        sort_by: 'distance',
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.nextCursor }
      };
      const response = await ApiClient.searchMoments(request);
      if (response.code === 0 && response.data) {
        const validItems = response.data.items.filter((m: MomentItem) => m.moment_id && m.user);
        this.momentsDataSource.addData(validItems);
        this.hasData = this.momentsDataSource.totalCount() > 0;
        this.hasMore = !!(response.data.next_cursor);
        this.nextCursor = response.data.next_cursor ?? '';
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载更多瞬间失败: ${JSON.stringify(error)}`);
    } finally {
      this.isLoadingMore = false;
    }
  }

  private async loadFromCache() {
    const cachedItems = await this.momentCacheService.getCachedMoments();
    if (cachedItems.length > 0) {
      this.momentsDataSource.setData(cachedItems);
      this.hasData = true;
      hilog.info(DOMAIN_NUMBER, TAG, `Loaded ${cachedItems.length} moments from cache`);
    }
  }

  private async updateCache(moments: MomentItem[]) {
    await this.momentCacheService.cacheMoments(moments);
  }

  private formatTime(timeStr: string): string {
    const time = new Date(timeStr);
    const now = new Date();
    if (time.toDateString() === now.toDateString()) {
      return `${time.getHours().toString().padStart(2, '0')}:${time.getMinutes().toString().padStart(2, '0')}`;
    }
    const yesterday = new Date(now);
    yesterday.setDate(yesterday.getDate() - 1);
    if (time.toDateString() === yesterday.toDateString()) {
      return '昨天';
    }
    if (time.getFullYear() === now.getFullYear()) {
      return `${(time.getMonth() + 1).toString().padStart(2, '0')}-${time.getDate().toString().padStart(2, '0')}`;
    }
    return time.toLocaleDateString();
  }

  private calcImageGridHeight(imageCount: number): number {
    if (imageCount === 0) return 0;
    const rows = Math.ceil(Math.min(imageCount, 9) / 3);
    const gap = 8;
    const itemHeight = 100;
    return rows * itemHeight + (rows - 1) * gap;
  }

  private async likeMoment(moment: MomentItem, index: number) {
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再点赞', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '点赞瞬间';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    try {
      const request: LikeMomentRequest = { moment_id: moment.moment_id };
      const response = await ApiClient.likeMoment(request);
      if (response.code === 0) {
        await this.loadNearbyMoments(true);
      } else {
        hilog.error(DOMAIN_NUMBER, TAG, `点赞失败: ${response.message}`);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `点赞失败: ${JSON.stringify(error)}`);
    }
  }

  private async shareMoment(moment: MomentItem): Promise<void> {
    try {
      const context = getContext(this) as common.UIAbilityContext;
      const content = SystemShareService.buildMomentShareContent(
        moment.description,
        moment.user?.nickname ?? '匿名用户',
        moment.moment_id
      );
      await SystemShareService.getInstance().shareText(context, content);
    } catch (error) {
      // User cancelled or share failed - non-fatal
    }
  }

  private previewImage(momentId: string, images: string[], index: number) {
    const previewParams = new PhotoPreviewRouteParam();
    previewParams.momentId = momentId;
    previewParams.images = images;
    previewParams.index = index;
    AppRouter.push(PageNames.PHOTO_PREVIEW, previewParams);
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        Text('附近瞬间')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        Button('发布')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .height(32)
          .onClick(async () => {
            const currentUser = this.stateManager.getCurrentUser();
            if (!currentUser) {
              SystemToast.showWarning('请先登录后再发布', this.getUIContext());
              const lp = new LoginRouteParam();
              lp.reason = 'action_required';
              lp.action = '发布瞬间';
              AppRouter.push(PageNames.LOGIN, lp);
              return;
            }
            AppRouter.push(PageNames.CREATE_MOMENT);
          })
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor($r('sys.color.ohos_id_color_background'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 内容区域
      if (this.isLoading && !this.hasData) {
        SystemLoadingView({ message: '正在加载附近瞬间...' })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (!this.hasData && this.errorMessage) {
        this.buildEmptyState(AppIcon.WARNING, this.errorMessage, '重试', () => { this.loadNearbyMoments(true); })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (!this.hasData) {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildEmptyState(AppIcon.CAMERA, '附近暂无瞬间\n快来分享第一个吧！', undefined, undefined)
        }
        .onRefreshing(() => { this.loadNearbyMoments(true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildMomentList()
        }
        .onRefreshing(() => { this.loadNearbyMoments(true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private buildEmptyState(icon: AppIcon, message: string, actionLabel: string | undefined, action: (() => void) | undefined) {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(48)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }

      Text(message)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
        .lineHeight(24)

      if (actionLabel && action) {
        Button(actionLabel)
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .height(40)
          .onClick(() => { action(); })
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.XXL)
  }

  @Builder
  RefreshBuilder() {
    Row() {
      LoadingProgress().width(24).height(24)
      Text('正在刷新...')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .margin({ left: DesignConstants.SPACING.SM })
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.SM)
  }

  @Builder
  private buildMomentList() {
    List({ space: DesignConstants.SPACING.MD }) {
      LazyForEach(this.momentsDataSource, (moment: MomentItem, index: number) => {
        ListItem() {
          this.buildMomentCard(moment, index)
        }
      }, (moment: MomentItem) => moment.moment_id)
      // 底部加载更多指示器
      if (this.isLoadingMore) {
        ListItem() {
          Row({ space: DesignConstants.SPACING.SM }) {
            LoadingProgress().width(20).height(20)
            Text('加载更多...')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
          .width('100%')
          .justifyContent(FlexAlign.Center)
          .padding(DesignConstants.SPACING.MD)
        }
      }
    }
    .width('100%')
    .layoutWeight(1)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
    .scrollBar(BarState.Auto)
    .cachedCount(5)
    .onReachEnd(() => { this.loadMoreMoments(); })
  }

  @Builder
  private buildMomentCard(moment: MomentItem, index: number) {
    Column({ space: DesignConstants.SPACING.MD }) {
      // 用户信息
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: moment.user?.avatar_url ?? '',
          name: moment.user?.nickname ?? '',
          avatarSize: 40
        })

        Column({ space: 2 }) {
          Text(moment.user?.nickname ?? '')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))

          Text(this.formatTime(moment.created_at))
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')

      // 瞬间内容
      Text(moment.description)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .lineHeight(24)

      // 图片展示
      if (moment.images && moment.images.length > 0) {
        Grid() {
          ForEach(moment.images.slice(0, 9), (img: MomentImage, imgIndex: number) => {
            GridItem() {
              Image(img.url)
                .width('100%')
                .height('100%')
                .objectFit(ImageFit.Cover)
                .borderRadius(DesignConstants.RADIUS.MD)
                .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                .geometryTransition('moment_img_' + moment.moment_id + '_' + imgIndex)
                .onClick(() => {
                  const imageUrls: string[] = (moment.images || []).map(i => i.url);
                  this.previewImage(moment.moment_id, imageUrls, imgIndex);
                })
            }
            .aspectRatio(1)
          }, (img: MomentImage, idx: number) => img.url + '_' + idx)
        }
        .columnsTemplate('1fr 1fr 1fr')
        .columnsGap(DesignConstants.SPACING.SM)
        .rowsGap(DesignConstants.SPACING.SM)
        .height(this.calcImageGridHeight(moment.images.length))
        .width('100%')
      }

      // 标签
      if (moment.tags && moment.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(moment.tags, (tag: string) => {
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_emphasize'))
              .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
              .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
              .borderRadius(DesignConstants.RADIUS.FULL)
              .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
          }, (tag: string) => tag)
        }
        .width('100%')
      }

      // 位置信息
      if (moment.location) {
        IntelligentLocationCard({
          latitude: moment.location.point.latitude,
          longitude: moment.location.point.longitude,
          locationName: moment.location.location_name,
          distance: moment.location.distance ?? 0,
          altitude: 0
        })
      }

      // 互动区域
      Row({ space: DesignConstants.SPACING.LG }) {
        Row({ space: DesignConstants.SPACING.XS }) {
          LikeButton({
            momentId: moment.moment_id,
            initialLikesCount: moment.likes_count,
            initialIsLiked: moment.my_state?.is_liked ?? false,
            onLike: (): void => { this.likeMoment(moment, index); }
          })
        }

        Row({ space: DesignConstants.SPACING.XS }) {
          SystemIcon({ icon: AppIcon.MESSAGE, iconSize: 20, iconColor: $r('sys.color.ohos_id_color_text_secondary') })
          Text(moment.comments_count.toString())
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }

        Blank()

        Row({ space: DesignConstants.SPACING.XS }) {
          SystemIcon({
            icon: AppIcon.SHARE,
            iconSize: IconSize.SMALL,
            iconColor: $r('sys.color.ohos_id_color_text_secondary'),
            clickable: true,
            onIconClick: () => this.shareMoment(moment)
          })
          Text(moment.shares_count.toString())
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
  }
}

/**
 * 点赞按钮组件
 */
@Component
struct LikeButton {
  @Prop momentId: string;
  @State isLiked: boolean = false;
  @State likesCount: number = 0;
  @Prop initialLikesCount: number = 0;
  @Prop initialIsLiked: boolean = false;
  onLike?: () => void;

  async aboutToAppear() {
    this.likesCount = this.initialLikesCount;
    this.isLiked = this.initialIsLiked;
  }

  build() {
    Row({ space: DesignConstants.SPACING.XS }) {
      SystemIcon({
        icon: this.isLiked ? AppIcon.HEART_FILL : AppIcon.HEART,
        iconSize: 20,
        iconColor: this.isLiked ? $r('app.color.color_danger') : $r('sys.color.ohos_id_color_text_secondary'),
        clickable: true,
        onIconClick: () => {
          this.isLiked = !this.isLiked;
          this.likesCount += this.isLiked ? 1 : -1;
          if (this.onLike) {
            this.onLike();
          }
        }
      })

      Text(this.likesCount.toString())
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor(this.isLiked ? $r('app.color.color_danger') : $r('sys.color.ohos_id_color_text_secondary'))
    }
  }
}
