/**
 * 发现页面
 * 展示热门话题、推荐用户、热门瞬间
 */

import { KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  UserItem, TrendingTagItem, MomentItem, TrendingBundle, FollowRequest, SearchUsersRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'DiscoverPage';

class SearchParams {
  keyword: string = '';
}

@Component
export struct DiscoverPage {
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State trendingTags: TrendingTagItem[] = [];
  @State recommendedUsers: UserItem[] = [];
  private recommendedUsersDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State hotMoments: MomentItem[] = [];
  private hotMomentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  @State isRefreshing: boolean = false;

  private getHeaderTopPadding(): number {
    return (this.safeAreaTop > 0 ? this.safeAreaTop : DesignConstants.SPACING.LG) + DesignConstants.SPACING.SM;
  }

  private calculateImageHeight(moment: MomentItem): number {
    if (moment.images && moment.images.length > 0) {
      const img = moment.images[0];
      if (img.width && img.height && img.width > 0) {
        const columnWidth = 160;
        const ratio = img.height / img.width;
        const clampedRatio = Math.max(0.6, Math.min(1.8, ratio));
        return columnWidth * clampedRatio;
      }
    }
    return 160;
  }

  private getWaterFlowHeight(): number {
    if (this.hotMoments.length === 0) return 0;
    let leftH = 0;
    let rightH = 0;
    const gap = 8;
    const infoHeight = 64;
    for (let i = 0; i < this.hotMoments.length; i++) {
      const cardH = this.calculateImageHeight(this.hotMoments[i]) + infoHeight;
      if (leftH <= rightH) {
        leftH += cardH + gap;
      } else {
        rightH += cardH + gap;
      }
    }
    return Math.max(leftH, rightH);
  }

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    await this.loadData();
  }

  private hasAnyData(): boolean {
    return this.trendingTags.length > 0 || this.recommendedUsers.length > 0 || this.hotMoments.length > 0;
  }

  private async loadData(refresh: boolean = false): Promise<void> {
    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';
    try {
      // 热门内容不依赖位置，推荐用户需要位置
      const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
      const trendingPromise = ApiClient.getFeedTrending({});

      if (refLoc) {
        const usersRequest: SearchUsersRequest = {
          reference_point: { latitude: refLoc.latitude, longitude: refLoc.longitude },
          radius: DefaultConfig.NEARBY_RADIUS,
          pagination: { limit: 10 }
        };
        const results = await LoadingUtil.withMinimumTime(() =>
          Promise.all([trendingPromise, ApiClient.searchUsers(usersRequest)])
        );
        const usersResponse = results[1];
        if (usersResponse.code === 0) {
          const items = usersResponse.data?.items ?? [];
          this.recommendedUsers = items.filter((u: UserItem) => u.user?.user_id);
          this.recommendedUsersDataSource.setData(this.recommendedUsers);
        }
        const trendingResponse = results[0];
        if (trendingResponse.code === 0) {
          const bundle = trendingResponse.data;
          this.trendingTags = bundle?.trending_tags ?? [];
          this.hotMoments = bundle?.hot_moments ?? [];
          this.hotMomentsDataSource.setData(this.hotMoments);
        }
        if (usersResponse.code !== 0 && trendingResponse.code !== 0 && !this.hasAnyData()) {
          this.errorMessage = trendingResponse.message ?? usersResponse.message ?? '加载发现数据失败';
        }
      } else {
        const trendingResponse = await LoadingUtil.withMinimumTime(() => trendingPromise);
        if (trendingResponse.code === 0) {
          const bundle = trendingResponse.data;
          this.trendingTags = bundle?.trending_tags ?? [];
          this.hotMoments = bundle?.hot_moments ?? [];
          this.hotMomentsDataSource.setData(this.hotMoments);
        } else if (!this.hasAnyData()) {
          this.errorMessage = trendingResponse.message ?? '加载发现数据失败';
        }
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载发现数据失败');
      if (!this.hasAnyData()) {
        this.errorMessage = '加载发现数据失败';
      }
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
    }
  }

  build() {
    Column() {
      // 标题栏 - 使用 SafeArea 避免与状态栏重叠
      Row() {
        Text('发现')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        // 搜索按钮
        Column() {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(22)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
        }
        .width(40)
        .height(40)
        .justifyContent(FlexAlign.Center)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .onClick(() => {
          AppRouter.push(PageNames.SEARCH);
        })
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: this.getHeaderTopPadding(),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor($r('app.color.bg_secondary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 内容区
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载发现内容...' })
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (!this.hasAnyData() && this.errorMessage) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Text() {
            SymbolSpan($r(AppIcon.WARNING))
              .fontSize(48)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
          Text(this.errorMessage)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .height(40)
            .onClick(() => { this.loadData(); })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        Refresh({ refreshing: $$this.isRefreshing }) {
          List({ space: DesignConstants.SPACING.MD }) {
            // 热门话题
            if (this.trendingTags.length > 0) {
              ListItem() {
                this.buildTrendingTagsSection()
              }
            }

            // 推荐用户（标题 + LazyForEach）
            if (this.recommendedUsers.length > 0) {
              ListItem() {
                Row({ space: DesignConstants.SPACING.SM }) {
                  Text() {
                    SymbolSpan($r(AppIcon.PERSON))
                      .fontSize(18)
                      .fontColor([$r('sys.color.ohos_id_color_emphasize')])
                  }
                  Text('推荐用户')
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(FontWeight.Bold)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                }
                .width('100%')
                .padding({ left: 16, top: 16, bottom: 8 })
              }
              LazyForEach(this.recommendedUsersDataSource, (user: UserItem) => {
                ListItem() {
                  this.buildRecommendedUserRow(user)
                }
              }, (user: UserItem) => user.user?.user_id ?? '')
            }

            // 热门瞬间（瀑布流布局）
            if (this.hotMoments.length > 0) {
              ListItem() {
                Column({ space: DesignConstants.SPACING.MD }) {
                  this.buildSectionHeader('热门动态', AppIcon.HEART_FILL)
                  WaterFlow() {
                    LazyForEach(this.hotMomentsDataSource, (moment: MomentItem) => {
                      FlowItem() {
                        this.buildHotMomentCard(moment)
                      }
                    }, (moment: MomentItem) => moment.moment_id)
                  }
                  .height(this.getWaterFlowHeight())
                  .columnsTemplate('1fr 1fr')
                  .columnsGap(8)
                  .rowsGap(8)
                  .cachedCount(10, true)
                  .scrollBar(BarState.Off)
                }
                .width('100%')
                .padding(DesignConstants.SPACING.LG)
                .backgroundColor($r('app.color.bg_secondary'))
                .borderRadius(DesignConstants.RADIUS.LG)
              }
            }

            // 没有任何内容
            if (this.trendingTags.length === 0 && this.recommendedUsers.length === 0 && this.hotMoments.length === 0) {
              ListItem() {
                Column({ space: DesignConstants.SPACING.LG }) {
                  Text() {
                    SymbolSpan($r(AppIcon.RECOMMEND))
                      .fontSize(48)
                      .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
                  }
                  Text('暂无推荐内容')
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                }
                .width('100%')
                .height(300)
                .justifyContent(FlexAlign.Center)
              }
            }
          }
          .width('100%')
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.SM,
            bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
          .scrollBar(BarState.Auto)
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadData(true); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
  }

  // ==================== 热门话题 ====================
  @Builder
  buildTrendingTagsSection() {
    Column({ space: DesignConstants.SPACING.MD }) {
      this.buildSectionHeader('热门话题', AppIcon.CHART)

      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(this.trendingTags, (tag: TrendingTagItem) => {
          Row({ space: 4 }) {
            Text(`#${tag.name}`)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))

            if (tag.trend === 'rising') {
              Text() {
                SymbolSpan($r(AppIcon.ARROW_UP))
                  .fontSize(12)
                  .fontColor([$r('app.color.color_danger')])
              }
            }
          }
          .padding({ left: 12, right: 12, top: 8, bottom: 8 })
          .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .borderRadius(DesignConstants.RADIUS.XL)
          .onClick(() => {
            const searchParams = new SearchParams();
            searchParams.keyword = `#${tag.name}`;
            AppRouter.push(PageNames.SEARCH, searchParams);
          })
        }, (tag: TrendingTagItem) => tag.name)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.LG)
  }

  // ==================== 推荐用户（单行，供 LazyForEach 使用）====================
  @Builder
  buildRecommendedUserRow(user: UserItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        avatarUrl: user.user?.avatar_url ?? '',
        name: user.user?.nickname ?? '',
        avatarSize: 44
      })

      Column({ space: 2 }) {
        Text(user.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))

        Text(user.bio || '可能会感兴趣')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Button('关注')
        .type(ButtonType.Capsule)
        .backgroundColor(Color.Transparent)
        .fontColor($r('sys.color.ohos_id_color_emphasize'))
        .fontSize(13)
        .fontWeight(FontWeight.Medium)
        .height(30)
        .borderWidth(1)
        .borderColor($r('sys.color.ohos_id_color_emphasize'))
        .onClick(() => { this.followUser(user); })
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 10, bottom: 10 })
  }

  // ==================== 热门瞬间卡片（masonry 小红书风格）====================
  @Builder
  buildHotMomentCard(moment: MomentItem) {
    Column({ space: 0 }) {
      // 封面图
      if (moment.images && moment.images.length > 0) {
        Image(moment.images[0].url)
          .width('100%')
          .height(this.calculateImageHeight(moment))
          .objectFit(ImageFit.Cover)
          .borderRadius({ topLeft: 12, topRight: 12 })
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
      } else {
        // 无图纯文本卡片
        Column() {
          Text(moment.description ?? '')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(6)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .lineHeight(22)
        }
        .width('100%')
        .height(120)
        .padding(12)
        .justifyContent(FlexAlign.Center)
        .borderRadius({ topLeft: 12, topRight: 12 })
        .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
      }

      // 信息区（白底下半部分）
      Column({ space: 4 }) {
        if (moment.description && moment.images && moment.images.length > 0) {
          Text(moment.description)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .lineHeight(18)
            .width('100%')
        }

        Row({ space: 4 }) {
          SystemAvatar({
            avatarUrl: moment.user?.avatar_url ?? '',
            name: moment.user?.nickname ?? '',
            avatarSize: 16
          })

          Text(moment.user?.nickname ?? '')
            .fontSize(11)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .layoutWeight(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Row({ space: 2 }) {
            Text() {
              SymbolSpan($r(AppIcon.HEART))
                .fontSize(11)
                .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
            }
            Text(`${moment.likes_count}`)
              .fontSize(11)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)
      }
      .padding(8)
    }
    .width('100%')
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
  }

  // ==================== 热门瞬间列表项（供 LazyForEach 使用）====================
  @Builder
  buildHotMomentRow(moment: MomentItem) {
    Column({ space: DesignConstants.SPACING.SM }) {
      // 用户行
      Row({ space: DesignConstants.SPACING.SM }) {
        SystemAvatar({
          avatarUrl: moment.user?.avatar_url ?? '',
          name: moment.user?.nickname ?? '',
          avatarSize: 32
        })
        Text(moment.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
      }
      .width('100%')

      // 文本
      Text(moment.description)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .maxLines(3)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .lineHeight(22)
        .width('100%')

      // 图片（只显示第一张）
      if (moment.images && moment.images.length > 0) {
        Image(moment.images[0].url)
          .width('100%')
          .constraintSize({ maxHeight: 180 })
          .objectFit(ImageFit.Cover)
          .borderRadius(DesignConstants.RADIUS.MD)
      }

      // 互动数据
      Row({ space: 24 }) {
        this.buildStatChip(AppIcon.HEART, `${moment.likes_count}`)
        this.buildStatChip(AppIcon.MESSAGE, `${moment.comments_count}`)
      }
      .width('100%')
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .backgroundColor($r('app.color.bg_secondary'))

    Divider()
      .strokeWidth(0.5)
      .color($r('sys.color.ohos_id_color_component_normal'))
  }

  // ==================== 通用构建器 ====================
  @Builder
  buildSectionHeader(title: string, icon: AppIcon) {
    Row({ space: DesignConstants.SPACING.SM }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(18)
          .fontColor([$r('sys.color.ohos_id_color_emphasize')])
      }
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
    }
    .width('100%')
  }

  @Builder
  buildStatChip(icon: AppIcon, text: string) {
    Row({ space: 4 }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(14)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
      Text(text)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
  }

  // ==================== 操作 ====================
  private async followUser(user: UserItem): Promise<void> {
    if (!user.user?.user_id) return;
    try {
      const followReq: FollowRequest = { user_id: user.user.user_id };
      const response = await ApiClient.followUser(followReq);
      if (response.code === 0) {
        SystemToast.showSuccess('已关注', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '关注失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('关注失败', this.getUIContext());
    }
  }

}
