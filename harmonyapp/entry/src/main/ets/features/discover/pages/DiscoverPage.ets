/**
 * 发现页面
 * 展示热门话题、推荐用户、热门瞬间
 */

import { KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import {
  UserItem, TagItem, MomentItem, TrendingBundle, FollowRequest, SearchUsersRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF11;
const TAG: string = 'DiscoverPage';

class SearchParams {
  keyword: string = '';
}

@Component
export struct DiscoverPage {
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;

  @State isLoading: boolean = false;
  @State trendingTags: TagItem[] = [];
  @State recommendedUsers: UserItem[] = [];
  private recommendedUsersDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State hotMoments: MomentItem[] = [];
  private hotMomentsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  @State isRefreshing: boolean = false;

  async aboutToAppear() {
    await this.loadData();
  }

  private async loadData(): Promise<void> {
    this.isLoading = true;
    try {
      // 并行加载：热门内容 + 推荐用户
      const usersRequest: SearchUsersRequest = {
        reference_point: this.currentLocation
          ? { latitude: this.currentLocation.latitude, longitude: this.currentLocation.longitude }
          : { latitude: 0, longitude: 0 },
        pagination: { limit: 10 }
      };
      const results = await LoadingUtil.withMinimumTime(() =>
        Promise.all([ApiClient.getFeedTrending({}), ApiClient.searchUsers(usersRequest)])
      );
      const trendingResponse = results[0];
      const usersResponse = results[1];

      if (trendingResponse.code === 0 && trendingResponse.data) {
        const bundle: TrendingBundle = trendingResponse.data;
        this.trendingTags = bundle.trending_tags || [];
        this.hotMoments = bundle.hot_moments || [];
        this.hotMomentsDataSource.setData(this.hotMoments);
      }

      if (usersResponse.code === 0 && usersResponse.data) {
        this.recommendedUsers = usersResponse.data.items.filter((u: UserItem) => u.user?.user_id);
        this.recommendedUsersDataSource.setData(this.recommendedUsers);
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载发现数据失败');
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
    }
  }

  build() {
    Column() {
      // 标题栏
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
      .height(56)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 内容区
      if (this.isLoading && this.trendingTags.length === 0) {
        SystemLoadingView({ message: '正在加载发现内容...' })
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
                Column({ space: DesignConstants.SPACING.MD }) {
                  this.buildSectionHeader('推荐用户', AppIcon.PERSON)
                }
                .width('100%')
                .padding(DesignConstants.SPACING.LG)
                .backgroundColor($r('sys.color.ohos_id_color_background'))
                .borderRadius(DesignConstants.RADIUS.LG)
              }
              LazyForEach(this.recommendedUsersDataSource, (user: UserItem) => {
                ListItem() {
                  this.buildRecommendedUserRow(user)
                }
              }, (user: UserItem) => user.user?.user_id ?? '')
            }

            // 热门瞬间（标题 + LazyForEach）
            if (this.hotMoments.length > 0) {
              ListItem() {
                Column({ space: DesignConstants.SPACING.MD }) {
                  this.buildSectionHeader('热门动态', AppIcon.HEART_FILL)
                }
                .width('100%')
                .padding(DesignConstants.SPACING.LG)
                .backgroundColor($r('sys.color.ohos_id_color_background'))
                .borderRadius(DesignConstants.RADIUS.LG)
              }
              LazyForEach(this.hotMomentsDataSource, (moment: MomentItem) => {
                ListItem() {
                  this.buildHotMomentRow(moment)
                }
              }, (moment: MomentItem) => moment.moment_id)
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
          .layoutWeight(1)
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.LG })
          .scrollBar(BarState.Auto)
        }
        .onRefreshing(() => { this.loadData(); })
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  // ==================== 热门话题 ====================
  @Builder
  buildTrendingTagsSection() {
    Column({ space: DesignConstants.SPACING.MD }) {
      this.buildSectionHeader('热门话题', AppIcon.CHART)

      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(this.trendingTags, (tag: TagItem) => {
          Row({ space: 4 }) {
            Text(`#${tag.name}`)
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))

            if (tag.trend === 'rising') {
              Text() {
                SymbolSpan($r(AppIcon.ARROW_UP))
                  .fontSize(12)
                  .fontColor([DesignConstants.COLOR.ERROR])
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
        }, (tag: TagItem) => tag.name)
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
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
        .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
        .fontColor(Color.White)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .height(30)
        .onClick(() => { this.followUser(user); })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    .backgroundColor($r('sys.color.ohos_id_color_background'))
  }

  // ==================== 热门瞬间（单条，供 LazyForEach 使用）====================
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
    .backgroundColor($r('sys.color.ohos_id_color_background'))

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
