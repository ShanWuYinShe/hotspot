/**
 * 生活瞬间页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 * LazyForEach + RDB 缓存
 * 使用统一的 MomentFeedCard 组件展示瞬间卡片
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { InteractionStateStore } from '../../../core/state/InteractionStateStore';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import { MomentItem, FetchNearbyMomentsRequest } from '../../../core/types/api';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemPageHeader, PageHeaderAction } from '../../../components/SystemPageHeader';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppIcon } from '../../../components/SystemIcon';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemToast } from '../../../components/SystemToast';
import { MomentCacheService } from '../../../core/database/MomentCacheService';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter, LoginRouteParam, MomentDetailRouteParam, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { MomentFeedCard } from '../../../components/MomentFeedCard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

const DOMAIN_NUMBER: number = 0xFF05;
const TAG: string = 'MomentPage';

@Preview
@Component
export struct MomentPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private stateManager = GlobalStateManager.getInstance();
  private interactionStore = InteractionStateStore.getInstance();
  private momentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource([]);
  private momentCacheService = MomentCacheService.getInstance();
  private isDestroyed: boolean = false;

  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State errorMessage: string = '';
  @State hasData: boolean = false;
  @State hasMore: boolean = false;
  @State nextCursor: string = '';
  @State isLoadingMore: boolean = false;

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    await this.loadNearbyMoments();
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  aboutToDisappear() {
    this.isDestroyed = true;
  }

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

    this.errorMessage = '';

    try {
      // spec moment.md → 附近瞬间走 moments/fetch_nearby（原 feed/nearby 迁移）
      const request: FetchNearbyMomentsRequest = {
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        radius: DefaultConfig.MOMENT_RADIUS,
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      };
      const response = await LoadingUtil.withMinimumTime(() => ApiClient.fetchNearbyMoments(request));
      if (this.isDestroyed) return;

      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validItems = items.filter((m: MomentItem) => m.moment_id && m.user);
        this.momentsDataSource.setData(validItems);
        this.hasData = validItems.length > 0;
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
        validItems.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
        this.updateCache(items);
      } else {
        if (this.momentsDataSource.totalCount() === 0) {
          this.errorMessage = getErrorMessage(response, '加载瞬间失败');
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
      this.isRefreshing = false;
    }
  }

  private async loadMoreMoments(): Promise<void> {
    const refLoc = this.getRefLocation();
    if (this.isLoadingMore || !this.hasMore || !refLoc) return;
    this.isLoadingMore = true;
    try {
      // spec moment.md → 附近瞬间走 moments/fetch_nearby（原 feed/nearby 迁移）
      const request: FetchNearbyMomentsRequest = {
        reference_point: {
          latitude: refLoc.latitude,
          longitude: refLoc.longitude
        },
        radius: DefaultConfig.MOMENT_RADIUS,
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE, cursor: this.nextCursor }
      };
      const response = await ApiClient.fetchNearbyMoments(request);
      if (this.isDestroyed) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        const validItems = items.filter((m: MomentItem) => m.moment_id && m.user);
        this.momentsDataSource.addData(validItems);
        this.hasData = this.momentsDataSource.totalCount() > 0;
        this.hasMore = !!(response.data?.next_cursor);
        this.nextCursor = response.data?.next_cursor ?? '';
        validItems.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
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
      cachedItems.forEach((item: MomentItem) => this.interactionStore.patchMomentFromItem(item));
      hilog.info(DOMAIN_NUMBER, TAG, `Loaded ${cachedItems.length} moments from cache`);
    }
  }

  private async updateCache(moments: MomentItem[]) {
    await this.momentCacheService.cacheMoments(moments);
  }

  private previewImage(momentId: string, images: string[], index: number) {
    const previewParams = new PhotoPreviewRouteParam();
    previewParams.momentId = momentId;
    previewParams.images = images;
    previewParams.index = index;
    AppRouter.push(PageNames.PHOTO_PREVIEW, previewParams);
  }

  build() {
    NavDestination() {
      PageShellContainer({ variant: 'narrow', currentBreakpoint: this.currentBreakpoint }) {
        Column() {
          // 标题栏 —— 统一走 SystemPageHeader（透明背景 + 发布 action）
          SystemPageHeader({
            title: '我的瞬间',
            titleSize: DesignConstants.FONT.SIZE.XXL,
            titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
            backgroundVariant: 'transparent',
            action: {
              label: '发布',
              onClick: async () => {
                const currentUser = this.stateManager.getCurrentUser();
                if (!currentUser) {
                  SystemToast.showWarning('请先登录后再发布', this.getUIContext());
                  const lp = new LoginRouteParam();
                  lp.reason = 'action_required';
                  lp.action = '发布瞬间';
                  lp.redirectUrl = PageNames.CREATE_MOMENT;
                  AppRouter.push(PageNames.LOGIN, lp);
                  return;
                }
                AppRouter.push(PageNames.CREATE_MOMENT);
              }
            } as PageHeaderAction
          })

          // 内容区域
          if (this.isLoading) {
            SystemLoadingView({ message: '正在加载我的瞬间...' })
              .transition(
                TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                  .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
              )
          } else if (!this.hasData && this.errorMessage) {
            EmptyState({
              icon: AppIcon.WARNING, message: this.errorMessage, actionLabel: '重试',
              onAction: () => { this.isLoading = true; this.loadNearbyMoments(true); }
            })
              .transition(
                TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                  .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
              )
          } else if (!this.hasData) {
            Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
              EmptyState({ icon: AppIcon.CAMERA, message: '还没有发布过瞬间\n快来分享第一个吧！' })
            }
            .onRefreshing(() => { this.loadNearbyMoments(true); })
            .transition(
              TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
            )
          } else {
            Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
              this.buildMomentList()
            }
            .layoutWeight(1)
            .onRefreshing(() => { this.loadNearbyMoments(true); })
            .transition(
              TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
                .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
            )
          }
        }
        .width('100%')
        .height('100%')
      }
    }
    .hideTitleBar(true)
  }

  @Builder
  private buildMomentList() {
    List() {
      LazyForEach(this.momentsDataSource, (moment: MomentItem) => {
        ListItem() {
          MomentFeedCard({
            moment: moment,
            imageMode: 'grid',
            showTags: true,
            showLocationCard: true,
            showTime: true,
            showFavorite: false,
            textMaxLines: 0,
            textFontSize: DesignConstants.FONT.SIZE.MD,
            onImagePreview: (momentId: string, images: string[], index: number) => {
              this.previewImage(momentId, images, index);
            }
          })
        }
        .margin({ bottom: DesignConstants.SPACING.MD })
      }, (moment: MomentItem) => moment.moment_id)
      if (this.isLoadingMore) {
        ListItem() {
          LoadingMoreIndicator()
        }
      }
    }
    .divider({ strokeWidth: 0 })
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
    .scrollBar(BarState.Auto)
    .cachedCount(5)
    .onReachEnd(() => { this.loadMoreMoments(); })
  }
}

@Builder
export function MomentPageBuilder() {
  MomentPage()
}