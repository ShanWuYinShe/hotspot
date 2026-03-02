/**
 * 发现页面
 * 展示热门话题、推荐用户、热门瞬间
 */

import { KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { LocationInfo, LocationUtil } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam, MomentDetailRouteParam, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames, DefaultConfig } from '../../../constants/AppConstants';
import {
  UserItem, TrendingTagItem, MomentItem, TrendingBundle, FollowRequest, UnfollowRequest,
  SearchUsersRequest, CreateReportRequest, Relationship
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
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { formatCount } from '../../../core/utils/FormatUtil';
import { EmptyState, RefreshIndicator } from '../../../components/EmptyState';
import { hilog } from '@kit.PerformanceAnalysisKit';
import pasteboard from '@ohos.pasteboard';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'DiscoverPage';

class SearchParams {
  keyword: string = '';
}

class LoginRedirectParams {
  reason: string = '';
  action: string = '';
}

@Preview
@Component
export struct DiscoverPage {
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
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
  @State userActionPendingIds: string[] = [];

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
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
    const gap = 12;
    const infoHeight = 118;
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

  private isUserActionPending(userId: string): boolean {
    return this.userActionPendingIds.indexOf(userId) >= 0;
  }

  private isUserFollowing(user: UserItem): boolean {
    return user.relation?.is_following ?? false;
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
      // 顶部搜索 + 标题（与设计图一致）
      Column({ space: DesignConstants.SPACING.MD }) {
        Row({ space: DesignConstants.SPACING.SM }) {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(17)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
          Text('搜索附近的人和群组')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .height(42)
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.XL)
        .border({ width: 1, color: $r('app.color.highlight_border') })
        .onClick(() => {
          AppRouter.push(PageNames.SEARCH);
        })

        Text('发现内容')
          .fontSize(DesignConstants.FONT.SIZE.XXL)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: this.getHeaderTopPadding(),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor(Color.Transparent)
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 内容区
      if (this.isLoading) {
        SystemLoadingView({ message: '正在加载发现内容...' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (!this.hasAnyData() && this.errorMessage) {
        EmptyState({
          icon: AppIcon.WARNING,
          message: this.errorMessage,
          actionLabel: '重试',
          onAction: () => { this.loadData(); }
        })
        .layoutWeight(1)
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          List() {
            // 热门话题
            if (this.trendingTags.length > 0) {
              ListItem() {
                this.buildTrendingTagsSection()
              }
            }

            // 推荐用户
            if (this.recommendedUsers.length > 0) {
              ListItem() {
                Column({ space: DesignConstants.SPACING.MD }) {
                  this.buildSectionHeader(`推荐用户 · ${this.recommendedUsers.length}`, AppIcon.PERSON)

                  Column({ space: DesignConstants.SPACING.MD }) {
                    ForEach(this.recommendedUsers, (user: UserItem, index: number) => {
                      this.buildRecommendedUserCard(user)
                    }, (user: UserItem) => user.user?.user_id ?? '')
                  }
                  .width('100%')
                }
                .width('100%')
                .padding(DesignConstants.SPACING.LG)
                .backgroundColor(Color.Transparent)
              }
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
                      .width('100%')
                      .onClick(() => {
                        this.openMomentDetail(moment);
                      })
                    }, (moment: MomentItem) => moment.moment_id)
                  }
                  .height(this.getWaterFlowHeight())
                  .columnsTemplate('1fr 1fr')
                  .columnsGap(DesignConstants.SPACING.MD)
                  .rowsGap(DesignConstants.SPACING.MD)
                  .cachedCount(10)
                  .scrollBar(BarState.Off)
                  .nestedScroll({
                    scrollForward: NestedScrollMode.PARENT_FIRST,
                    scrollBackward: NestedScrollMode.PARENT_FIRST
                  })
                }
                .width('100%')
                .padding(DesignConstants.SPACING.LG)
                .backgroundColor(Color.Transparent)
              }
            }

            // 没有任何内容
            if (this.trendingTags.length === 0 && this.recommendedUsers.length === 0 && this.hotMoments.length === 0) {
              ListItem() {
                EmptyState({ icon: AppIcon.RECOMMEND, message: '暂无推荐内容' })
                  .height(300)
              }
            }
          }
          .width('100%')
          .padding({
            bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
          })
          .scrollBar(BarState.Auto)
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadData(true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
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
          Row({ space: DesignConstants.SPACING.XS }) {
            Text(`#${tag.name}`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('app.color.brand_primary'))

            if (tag.trend === 'rising') {
              Text() {
                SymbolSpan($r(AppIcon.ARROW_UP))
                  .fontSize(10)
                  .fontColor([$r('app.color.color_danger')])
              }
            }
          }
          .padding({
            left: DesignConstants.SPACING.MD,
            right: DesignConstants.SPACING.MD,
            top: DesignConstants.SPACING.XS,
            bottom: DesignConstants.SPACING.XS
          })
          .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
          .backgroundColor($r('app.color.bg_secondary'))
          .borderRadius(DesignConstants.RADIUS.FULL)
          .shadow(DesignConstants.SHADOW.SMALL)
          .border({ width: 1, color: $r('app.color.highlight_border') })
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
    .backgroundColor(Color.Transparent)
  }

  // ==================== 推荐用户卡片 ====================
  @Builder
  buildRecommendedUserCard(user: UserItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: 6 }) {
        Row({ space: DesignConstants.SPACING.XS }) {
          Text() {
            SymbolSpan($r(AppIcon.RECOMMEND))
              .fontSize(10)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
          Text('推荐用户')
            .fontSize(10)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .padding({ left: 8, right: 8, top: 3, bottom: 3 })
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.XL)

        Blank().layoutWeight(1)

        if (user.location?.distance !== undefined && user.location?.distance !== null) {
          Text(LocationUtil.formatDistance(user.location.distance))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor($r('sys.color.ohos_id_color_emphasize'))
            .padding({ left: 8, right: 8, top: 3, bottom: 3 })
            .backgroundColor($r('app.color.chip_selected_bg'))
            .borderRadius(DesignConstants.RADIUS.XL)
        }
      }
      .width('100%')

      Row({ space: DesignConstants.SPACING.SM }) {
        Row({ space: DesignConstants.SPACING.MD }) {
          SystemAvatar({
            avatarUrl: user.user?.avatar_url ?? '',
            name: user.user?.nickname ?? '',
            avatarSize: 46,
            shape: 'circle'
          })

          Column({ space: DesignConstants.SPACING.XS }) {
            Text(user.user?.nickname ?? '')
              .fontSize(15)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            Row({ space: DesignConstants.SPACING.XS }) {
              if (user.presence_status === 'online') {
                Row()
                  .width(6)
                  .height(6)
                  .borderRadius(DesignConstants.RADIUS.FULL)
                  .backgroundColor($r('app.color.color_success'))
                Text('在线')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('app.color.color_success'))
              } else {
                Row()
                  .width(6)
                  .height(6)
                  .borderRadius(DesignConstants.RADIUS.FULL)
                  .backgroundColor($r('sys.color.ohos_id_color_text_tertiary'))
                Text('离线')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              }
            }
            .alignItems(VerticalAlign.Center)
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
          .layoutWeight(1)
          .onClick(() => {
            this.openUserProfile(user);
          })

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.MORE))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
        }
        .width(32)
        .height(32)
        .justifyContent(FlexAlign.Center)
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.XL)
        .border({ width: 1, color: $r('app.color.highlight_border') })
        .bindMenu([
          {
            value: '私聊',
            action: () => this.startPrivateChat(user)
          },
          {
            value: this.isUserFollowing(user) ? '取消关注' : '关注',
            action: () => this.followUser(user)
          },
          {
            value: '举报用户',
            action: () => this.reportUser(user)
          },
          {
            value: '复制用户ID',
            action: () => this.copyText(user.user?.user_id ?? '', '用户ID已复制')
          }
        ])
      }
      .width('100%')

      Text(user.bio || '可能会感兴趣')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(17)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .maxLines(2)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .width('100%')

      if (user.tags && user.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(user.tags.slice(0, 3), (tag: string) => {
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor($r('app.color.brand_primary'))
              .padding({ left: 7, right: 7, top: 3, bottom: 3 })
              .margin({ right: 4, bottom: 4 })
              .backgroundColor($r('app.color.chip_selected_bg'))
              .borderRadius(DesignConstants.RADIUS.XL)
          }, (tag: string) => tag)
        }
        .width('100%')
      }

      Row({ space: DesignConstants.SPACING.SM }) {
        Button(this.isUserActionPending(user.user?.user_id ?? '') ? '处理中...' :
          (this.isUserFollowing(user) ? '已关注' : '+ 关注'))
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(36)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .enabled(!this.isUserActionPending(user.user?.user_id ?? ''))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(this.isUserFollowing(user) ? $r('app.color.bg_tertiary') : $r('app.color.brand_primary'))
          .fontColor(this.isUserFollowing(user) ? $r('sys.color.ohos_id_color_text_secondary') : Color.White)
          .opacity(this.isUserActionPending(user.user?.user_id ?? '') ? 0.58 : 1)
          .shadow(DesignConstants.SHADOW.SMALL)
          .border({ width: 1, color: $r('app.color.highlight_border') })
          .onClick(() => { this.followUser(user); })

        Button('私聊')
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(36)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(FontWeight.Medium)
          .backgroundColor($r('app.color.bg_tertiary'))
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .opacity(this.isUserActionPending(user.user?.user_id ?? '') ? 0.74 : 1)
          .shadow(DesignConstants.SHADOW.SMALL)
          .border({ width: 1, color: $r('app.color.highlight_border') })
          .onClick(() => { this.startPrivateChat(user); })
      }
      .width('100%')
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.XXL)
    .shadow(DesignConstants.SHADOW.MEDIUM)
    .border({ width: 1, color: $r('app.color.highlight_border') })
  }

  // ==================== 热门瞬间卡片（masonry 小红书风格）====================
  @Builder
  buildHotMomentCard(moment: MomentItem) {
    Column({ space: DesignConstants.SPACING.MD }) {
      Row({ space: 6 }) {
        Row({ space: DesignConstants.SPACING.XS }) {
          Text() {
            SymbolSpan($r(AppIcon.HEART_FILL))
              .fontSize(10)
              .fontColor([$r('sys.color.ohos_id_color_emphasize')])
          }
          Text('热门动态')
            .fontSize(10)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 3, bottom: 3 })
        .backgroundColor($r('app.color.bg_tertiary'))
        .borderRadius(DesignConstants.RADIUS.XL)

        Blank().layoutWeight(1)

        Row({ space: 2 }) {
          Text() {
            SymbolSpan($r(AppIcon.HEART))
              .fontSize(10)
              .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
          }
          Text(formatCount(moment.likes_count))
            .fontSize(10)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
        }
      }
      .width('100%')

      // 封面图
      if (moment.images && moment.images.length > 0) {
        Image(MediaUrlUtil.normalize(moment.images[0].url))
          .width('100%')
          .height(this.calculateImageHeight(moment))
          .objectFit(ImageFit.Cover)
          .borderRadius(DesignConstants.RADIUS.LG)
          .backgroundColor($r('app.color.bg_secondary'))
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
        .padding(DesignConstants.SPACING.MD)
        .justifyContent(FlexAlign.Center)
        .borderRadius(DesignConstants.RADIUS.LG)
        .backgroundColor($r('app.color.bg_secondary'))
      }

      // 信息区（白底下半部分）
      Column({ space: 6 }) {
        if (moment.description && moment.images && moment.images.length > 0) {
          Text(moment.description)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(3)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .lineHeight(18)
            .width('100%')
        }

        Row({ space: DesignConstants.SPACING.XS }) {
          SystemAvatar({
            avatarUrl: moment.user?.avatar_url ?? '',
            name: moment.user?.nickname ?? '',
            avatarSize: 16,
            shape: 'circle'
          })

          Text(moment.user?.nickname ?? '')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .layoutWeight(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .width('100%')
        .alignItems(VerticalAlign.Center)
      }
      .padding(DesignConstants.SPACING.SM)
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.XXL)
    .shadow(DesignConstants.SHADOW.MEDIUM)
    .border({ width: 1, color: $r('app.color.highlight_border') })
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
          avatarSize: 32,
          shape: 'circle'
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
        Image(MediaUrlUtil.normalize(moment.images[0].url))
          .width('100%')
          .constraintSize({ maxHeight: 180 })
          .objectFit(ImageFit.Cover)
          .borderRadius(DesignConstants.RADIUS.LG)
      }

      // 互动数据
      Row({ space: DesignConstants.SPACING.XL }) {
        this.buildStatChip(AppIcon.HEART, formatCount(moment.likes_count))
        this.buildStatChip(AppIcon.MESSAGE, formatCount(moment.comments_count))
      }
      .width('100%')
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.XXL)

    Divider()
      .strokeWidth(0.5)
      .color($r('sys.color.ohos_id_color_list_separator'))
  }

  // ==================== 通用构建器 ====================
  @Builder
  buildSectionHeader(title: string, icon: AppIcon) {
    Row({ space: DesignConstants.SPACING.SM }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor([$r('app.color.brand_primary')])
      }
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
    }
    .padding({ left: 16, right: 16, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    .backgroundColor($r('app.color.bg_inset'))
    .borderRadius(DesignConstants.RADIUS.FULL)
    .shadow(DesignConstants.SHADOW.INSET)
  }

  @Builder
  buildStatChip(icon: AppIcon, text: string) {
    Row({ space: DesignConstants.SPACING.XS }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
      Text(text)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
  }

  // ==================== 操作 ====================
  private async followUser(user: UserItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再关注', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '关注用户';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    if (!user.user?.user_id) return;
    const userId = user.user.user_id;
    const isFollowing = user.relation?.is_following ?? false;
    if (this.isUserActionPending(userId)) return;
    this.setUserActionPending(userId, true);
    try {
      const response = isFollowing
        ? await ApiClient.unfollowUser({ user_id: userId } as UnfollowRequest)
        : await ApiClient.followUser({ user_id: userId } as FollowRequest);
      if (response.code === 0) {
        this.patchRecommendedUserFollowing(userId, !isFollowing);
        SystemToast.showSuccess(isFollowing ? '已取消关注' : '已关注', this.getUIContext());
      } else {
        SystemToast.showError(response.message || (isFollowing ? '取消关注失败' : '关注失败'), this.getUIContext());
      }
    } catch {
      SystemToast.showError(isFollowing ? '取消关注失败' : '关注失败', this.getUIContext());
    } finally {
      this.setUserActionPending(userId, false);
    }
  }

  private ensureUserRelation(user: UserItem): Relationship {
    if (!user.relation) {
      const relation: Relationship = {
        is_following: false,
        is_followed_by: false,
        is_blocked: false
      };
      user.relation = relation;
    }
    return user.relation;
  }

  private patchRecommendedUserFollowing(userId: string, isFollowing: boolean): void {
    const users = this.recommendedUsers;
    for (let i = 0; i < users.length; i++) {
      if ((users[i].user?.user_id ?? '') !== userId) {
        continue;
      }
      const relation = this.ensureUserRelation(users[i]);
      relation.is_following = isFollowing;
      break;
    }
    this.recommendedUsers = users;
    this.recommendedUsersDataSource.setData(users);
  }

  private setUserActionPending(userId: string, pending: boolean): void {
    const next: string[] = [];
    for (let i = 0; i < this.userActionPendingIds.length; i++) {
      const id = this.userActionPendingIds[i];
      if (id !== userId) {
        next.push(id);
      }
    }
    if (pending) {
      next.push(userId);
    }
    this.userActionPendingIds = next;
  }

  private openMomentDetail(moment: MomentItem): void {
    if (!moment.moment_id) return;
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
  }

  private openUserProfile(user: UserItem): void {
    const userId = user.user?.user_id;
    if (!userId) {
      return;
    }
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  private startPrivateChat(user: UserItem): void {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再发起私聊', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '发起私聊';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const userId = user.user?.user_id;
    if (!userId) return;
    const params = new ChatRouteParam();
    params.chatId = userId;
    params.chatType = 'private_chat';
    params.targetId = userId;
    params.title = user.user?.nickname ?? '私聊';
    AppRouter.push(PageNames.CHAT, params);
  }

  private async reportUser(user: UserItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再举报', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '举报用户';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const userId = user.user?.user_id;
    if (!userId) return;
    try {
      const req: CreateReportRequest = {
        target_type: 'user',
        target_id: userId,
        reason: 'other'
      };
      const response = await ApiClient.createReport(req);
      if (response.code === 0) {
        SystemToast.showSuccess('举报已提交', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '举报失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('举报失败', this.getUIContext());
    }
  }

  private async copyText(text: string, successText: string): Promise<void> {
    if (!text) return;
    try {
      const systemPasteboard = pasteboard.getSystemPasteboard();
      const pasteData = pasteboard.createData(pasteboard.MIMETYPE_TEXT_PLAIN, text);
      await systemPasteboard.setData(pasteData);
      SystemToast.show(successText, this.getUIContext());
    } catch {
      SystemToast.showError('复制失败', this.getUIContext());
    }
  }


}
