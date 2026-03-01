/**
 * 推荐页面
 * 展示推荐内容，使用原生 List 和 Grid 布局
 */

import { ApiClient } from '../../../core/api/ApiClient';
import { UserItem, TrendingTagItem } from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { AppIcon } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'RecommendationPage';

@Preview
@Component
export struct RecommendationPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @State isLoading: boolean = true;
  @State errorMessage: string = '';
  @State trendingTags: TrendingTagItem[] = [];
  @State recommendedUsers: UserItem[] = [];
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    await this.loadData();
  }

  private hasAnyData(): boolean {
    return this.trendingTags.length > 0 || this.recommendedUsers.length > 0;
  }

  private getHeaderTopPadding(): number {
    return (this.safeAreaTop > 0 ? this.safeAreaTop : DesignConstants.SPACING.LG) + DesignConstants.SPACING.SM;
  }

  private async loadData() {
    this.isLoading = true;
    this.errorMessage = '';
    try {
      const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
      const trendingPromise = ApiClient.getFeedTrending({});
      const usersPromise = refLoc
        ? ApiClient.getFeedUsers({ reference_point: { latitude: refLoc.latitude, longitude: refLoc.longitude }, pagination: { limit: 5 } })
        : null;

      await LoadingUtil.withMinimumTime(async () => {
        const trendingResponse = await trendingPromise;
        if (trendingResponse.code === 0) {
          this.trendingTags = trendingResponse.data?.trending_tags ?? [];
        }

        if (usersPromise) {
          const usersResponse = await usersPromise;
          if (usersResponse.code === 0) {
            this.recommendedUsers = usersResponse.data?.items ?? [];
          } else if (!this.hasAnyData()) {
            this.errorMessage = usersResponse.message ?? '加载推荐数据失败';
          }
        }

        if (trendingResponse.code !== 0 && !this.hasAnyData()) {
          this.errorMessage = trendingResponse.message ?? '加载推荐数据失败';
        }
      });
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载推荐数据失败: ${JSON.stringify(error)}`);
      if (!this.hasAnyData()) {
        this.errorMessage = '加载推荐数据失败';
      }
    } finally {
      this.isLoading = false;
    }
  }

  build() {
    NavDestination() {
    Column() {
      // 标题栏
      Row() {
        Text('推荐')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(20)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
        }
        .width(36)
        .height(36)
        .borderRadius(DesignConstants.RADIUS.XL)
        .backgroundColor($r('app.color.bg_secondary'))
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)
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
      .backgroundColor($r('app.color.bg_primary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      if (this.isLoading) {
        SystemLoadingView({ message: '加载推荐内容...' })
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
            .type(ButtonType.Normal)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .height(40)
            .borderRadius(DesignConstants.RADIUS.XL)
            .onClick(() => { this.loadData(); })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        List({ space: 16 }) {
          // 热门话题
          ListItem() {
            Column({ space: 12 }) {
              this.buildSectionHeader('热门话题', AppIcon.CHART)
              
              Flex({ wrap: FlexWrap.Wrap }) {
                if (this.trendingTags.length > 0) {
                  ForEach(this.trendingTags, (tag: TrendingTagItem) => {
                    Text(`#${tag.name}`)
                      .padding(DesignConstants.SPACING.SM)
                      .backgroundColor($r('app.color.bg_tertiary'))
                      .borderRadius(DesignConstants.RADIUS.XL)
                      .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
                      .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                  }, (tag: TrendingTagItem) => tag.name)
                } else {
                  Text('暂无热门话题')
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .padding(DesignConstants.SPACING.SM)
                }
              }
              .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
            }
          }
          .width('100%')
          .backgroundColor($r('app.color.bg_secondary'))
          .padding(DesignConstants.SPACING.LG)
          .borderRadius(DesignConstants.RADIUS.XXL)
          .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
          .shadow(DesignConstants.SHADOW.SMALL)


          // 推荐用户
          if (this.recommendedUsers.length > 0) {
            ListItem() {
              this.buildSectionHeader('推荐用户', AppIcon.PERSON)
                .padding({
                  left: DesignConstants.SPACING.MD,
                  right: DesignConstants.SPACING.MD,
                  top: DesignConstants.SPACING.MD,
                  bottom: DesignConstants.SPACING.XS
                })
            }
            ForEach(this.recommendedUsers, (user: UserItem) => {
              ListItem() {
                Row({ space: DesignConstants.SPACING.MD }) {
                  SystemAvatar({
                    avatarUrl: user.user?.avatar_url,
                    name: user.user?.nickname ?? '',
                    avatarSize: 44
                  })

                  Column({ space: 2 }) {
                    Text(user.user?.nickname ?? '')
                      .fontSize(DesignConstants.FONT.SIZE.LG)
                      .fontWeight(FontWeight.Medium)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    Text('可能会感兴趣')
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  }
                  .alignItems(HorizontalAlign.Start)
                  .layoutWeight(1)

                  Button('关注')
                    .type(ButtonType.Normal)
                    .backgroundColor($r('app.color.bg_tertiary'))
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontWeight(FontWeight.Medium)
                    .height(34)
                    .borderRadius(DesignConstants.RADIUS.XXL)
                    .borderWidth(1)
                    .borderColor($r('sys.color.ohos_id_color_emphasize'))
                    .onClick(async () => {
                      try {
                        await ApiClient.followUser({ user_id: user.user?.user_id ?? '' });
                        SystemToast.showSuccess('已关注', this.getUIContext());
                      } catch (error) {
                        SystemToast.showError('关注失败', this.getUIContext());
                      }
                    })
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
                .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
                .shadow(DesignConstants.SHADOW.SMALL)
              }
            }, (user: UserItem) => user.user?.user_id ?? '')
          }
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG,
          bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
        .scrollBar(BarState.Auto)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  ListHeader(title: string) {
    Text(title)
      .fontSize(DesignConstants.FONT.SIZE.LG)
      .fontWeight(FontWeight.Bold)
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.SM })
      .fontColor($r('sys.color.ohos_id_color_text_primary'))
  }

  @Builder
  private buildSectionHeader(title: string, icon: AppIcon) {
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
}

@Builder
export function RecommendationPageBuilder() {
  RecommendationPage()
}
