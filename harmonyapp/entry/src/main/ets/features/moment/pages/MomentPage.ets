/**
 * 生活瞬间页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 * LazyForEach + RDB 缓存
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import { LikeMomentRequest, MomentSearchItem, SearchMomentsByLocationRequest, MomentImage } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { IntelligentLocationCard } from '../../../components/IntelligentLocationCard';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon } from '../../../components/SystemIcon';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemToast } from '../../../components/SystemToast';
import { MomentCacheService } from '../../../core/database/MomentCacheService';

const DOMAIN_NUMBER: number = 0xFF05;
const TAG: string = 'MomentPage';

@Component
export struct MomentPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  private stateManager = GlobalStateManager.getInstance();
  private momentsDataSource: BasicDataSource<MomentSearchItem> = new BasicDataSource([]);
  private momentCacheService = MomentCacheService.getInstance();

  @State isLoading: boolean = false;
  @State errorMessage: string = '';
  @State hasData: boolean = false;

  async aboutToAppear() {
    await this.loadNearbyMoments();
  }

  aboutToDisappear() {}

  private async loadNearbyMoments(forceRefresh: boolean = false) {
    if (!this.currentLocation && !forceRefresh) {
      await this.loadFromCache();
    }

    if (!this.currentLocation) {
      if (this.momentsDataSource.totalCount() === 0) {
        this.errorMessage = '请先获取位置信息';
      }
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
      const request: SearchMomentsByLocationRequest = {
        reference_point: {
          latitude: this.currentLocation.latitude,
          longitude: this.currentLocation.longitude
        },
        radius: 2000,
        limit: 50
      };
      const response = await ApiClient.searchMomentsByLocation(request);

      if (response.code === 0 && response.data) {
        this.momentsDataSource.setData(response.data.items);
        this.hasData = response.data.items.length > 0;
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

  private async loadFromCache() {
    const cachedItems = await this.momentCacheService.getCachedMoments();
    if (cachedItems.length > 0) {
      this.momentsDataSource.setData(cachedItems);
      this.hasData = true;
      hilog.info(DOMAIN_NUMBER, TAG, `Loaded ${cachedItems.length} moments from cache`);
    }
  }

  private async updateCache(moments: MomentSearchItem[]) {
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

  private async likeMoment(moment: MomentSearchItem, index: number) {
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再点赞', this.getUIContext());
      try {
        const routerInstance = this.getUIContext().getRouter();
        await routerInstance.pushUrl({
          url: `features/auth/pages/${PageNames.LOGIN}`,
          params: { reason: 'action_required', action: '点赞瞬间' }
        });
      } catch (error) {
        // ignore
      }
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

  private previewImage(momentId: string, images: string[], index: number) {
    try {
      const routerInstance = this.getUIContext().getRouter();
      routerInstance.pushUrl({
        url: `features/moment/pages/${PageNames.PHOTO_PREVIEW}`,
        params: { momentId: momentId, images: images, index: index }
      });
    } catch (error) {
      SystemToast.showError('打开图片失败', this.getUIContext());
    }
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
            try {
              const currentUser = this.stateManager.getCurrentUser();
              if (!currentUser) {
                SystemToast.showWarning('请先登录后再发布', this.getUIContext());
                const routerInstance = this.getUIContext().getRouter();
                await routerInstance.pushUrl({
                  url: `features/auth/pages/${PageNames.LOGIN}`,
                  params: { reason: 'action_required', action: '发布瞬间' }
                });
                return;
              }
              const routerInstance = this.getUIContext().getRouter();
              await routerInstance.pushUrl({ url: `features/moment/pages/${PageNames.CREATE_MOMENT}` });
            } catch (error) {
              hilog.error(DOMAIN_NUMBER, TAG, `跳转失败: ${JSON.stringify(error)}`);
            }
          })
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 内容区域
      if (this.isLoading && !this.hasData) {
        SystemLoadingView({ message: '正在加载附近瞬间...' })
      } else if (!this.hasData && this.errorMessage) {
        this.buildEmptyState(AppIcon.WARNING, this.errorMessage, '重试', () => { this.loadNearbyMoments(true); })
      } else if (!this.hasData) {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildEmptyState(AppIcon.CAMERA, '附近暂无瞬间\n快来分享第一个吧！', undefined, undefined)
        }
        .onRefreshing(() => { this.loadNearbyMoments(true); })
      } else {
        Refresh({ refreshing: $$this.isLoading, builder: this.RefreshBuilder() }) {
          this.buildMomentList()
        }
        .onRefreshing(() => { this.loadNearbyMoments(true); })
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
      LazyForEach(this.momentsDataSource, (moment: MomentSearchItem, index: number) => {
        ListItem() {
          this.buildMomentCard(moment, index)
        }
      }, (moment: MomentSearchItem) => moment.moment_id)
    }
    .width('100%')
    .layoutWeight(1)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
    .scrollBar(BarState.Auto)
    .cachedCount(5)
  }

  @Builder
  private buildMomentCard(moment: MomentSearchItem, index: number) {
    Column({ space: DesignConstants.SPACING.MD }) {
      // 用户信息
      Row({ space: DesignConstants.SPACING.MD }) {
        SystemAvatar({
          avatarUrl: moment.user.avatar_url || '',
          name: moment.user.nickname,
          avatarSize: 40
        })

        Column({ space: 2 }) {
          Text(moment.user.nickname)
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
          latitude: moment.location.latitude,
          longitude: moment.location.longitude,
          locationName: moment.location.location_name,
          distance: moment.location.distance,
          altitude: 0
        })
      }

      // 互动区域
      Row({ space: DesignConstants.SPACING.LG }) {
        Row({ space: DesignConstants.SPACING.XS }) {
          LikeButton({
            momentId: moment.moment_id,
            initialLikesCount: moment.likes_count,
            onLike: (): void => { this.likeMoment(moment, index); }
          })
        }

        Row({ space: DesignConstants.SPACING.XS }) {
          SystemIcon({ icon: AppIcon.MESSAGE, iconSize: 20, iconColor: DesignConstants.COLOR.TEXT_SECONDARY_LIGHT })
          Text(moment.comments_count.toString())
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }

        Blank()

        Row({ space: DesignConstants.SPACING.XS }) {
          SystemIcon({ icon: AppIcon.SHARE, iconSize: 20, iconColor: DesignConstants.COLOR.TEXT_SECONDARY_LIGHT })
          Text('分享')
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
  onLike?: () => void;

  async aboutToAppear() {
    this.likesCount = this.initialLikesCount;
  }

  build() {
    Row({ space: DesignConstants.SPACING.XS }) {
      SystemIcon({
        icon: this.isLiked ? AppIcon.HEART_FILL : AppIcon.HEART,
        iconSize: 20,
        iconColor: this.isLiked ? DesignConstants.COLOR.ERROR : DesignConstants.COLOR.TEXT_SECONDARY_LIGHT,
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
        .fontColor(this.isLiked ? DesignConstants.COLOR.ERROR : $r('sys.color.ohos_id_color_text_secondary'))
    }
  }
}
