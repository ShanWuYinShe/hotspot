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
import { RefreshIndicator } from '../../../components/EmptyState';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter, ProfileRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { KEY_BROWSE_ANCHOR, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { interactionActions } from '../../../core/state/InteractionActions';
import { InteractionStateStore, UserRelationState } from '../../../core/state/InteractionStateStore';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'RecommendationPage';

@Preview
@Component
export struct RecommendationPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State errorMessage: string = '';
  @State trendingTags: TrendingTagItem[] = [];
  @State recommendedUsers: UserItem[] = [];
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'figma_neu_light';

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    await this.loadData();
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

  private hasAnyData(): boolean {
    return this.trendingTags.length > 0 || this.recommendedUsers.length > 0;
  }

  private getHeaderTopPadding(): number {
    return (this.safeAreaTop > 0 ? this.safeAreaTop : DesignConstants.SPACING.LG) + DesignConstants.SPACING.SM;
  }

  private async loadData(forRefresh: boolean = false) {
    if (!forRefresh) {
      this.isLoading = true;
    }
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
            const store = InteractionStateStore.getInstance();
            for (const u of this.recommendedUsers) {
              store.patchUserFromItem(u);
            }
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
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .fontColor([this.getCurrentThemePalette().textPrimary])
          }
        }
        .width(36)
        .height(36)
        .borderRadius(DesignConstants.RADIUS.MD)
        .backgroundColor(this.getSurfaceSubtle())
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
      .backgroundColor(this.getSurfaceBackground())
      .borderRadius({ bottomLeft: DesignConstants.RADIUS.LG, bottomRight: DesignConstants.RADIUS.LG })
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      if (this.isLoading) {
        SystemLoadingView({ message: '加载推荐内容...' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (!this.hasAnyData() && this.errorMessage) {
        Column({ space: DesignConstants.SPACING.LG }) {
          Text() {
            SymbolSpan($r(AppIcon.WARNING))
              .fontSize(48)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
          Text(this.errorMessage)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .textAlign(TextAlign.Center)
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor(this.getCurrentThemePalette().accentBackground)
            .fontColor(this.getCurrentThemePalette().accentText)
            .height(40)
            .shadow(this.getCurrentThemePalette().controlShadow)
            .border({ width: 1, color: this.getSurfaceBorder() })
            .onClick(() => { this.loadData(); })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
        List() {
          // 热门话题
          ListItem() {
            Column({ space: 12 }) {
              this.buildSectionHeader('热门话题', AppIcon.CHART)
              
              Flex({ wrap: FlexWrap.Wrap }) {
                if (this.trendingTags.length > 0) {
                  ForEach(this.trendingTags, (tag: TrendingTagItem) => {
                    Text(`#${tag.name}`)
                      .padding(DesignConstants.SPACING.SM)
                      .backgroundColor(this.getSurfaceSubtle())
                      .borderRadius(DesignConstants.RADIUS.MD)
                      .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                      .onClick(() => {
                        AppRouter.push(PageNames.SEARCH, { keyword: `#${tag.name}` });
                      })
                  }, (tag: TrendingTagItem) => tag.name)
                } else {
                  Text('暂无热门话题')
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .padding(DesignConstants.SPACING.SM)
                }
              }
              .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM })
            }
          }
          .width('100%')
          .borderRadius(DesignConstants.RADIUS.XXL)
          .shadow(this.getCurrentThemePalette().elevatedShadow)
          .backgroundColor(this.getSurfaceBackground())
          .border({ width: 1, color: this.getSurfaceBorder() })
          .margin({ bottom: 12 })
          .padding(DesignConstants.SPACING.LG)

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
                  Row({ space: DesignConstants.SPACING.MD }) {
                    SystemAvatar({
                      avatarUrl: user.user?.avatar_url,
                      name: user.user?.nickname ?? '',
                      avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL
                    })

                    Column({ space: 2 }) {
                      Text(user.user?.nickname ?? '')
                        .fontSize(DesignConstants.FONT.SIZE.LG)
                        .fontWeight(FontWeight.Medium)
                        .fontColor(this.getCurrentThemePalette().textPrimary)
                      Text('可能会感兴趣')
                        .fontSize(DesignConstants.FONT.SIZE.SM)
                        .fontColor(this.getCurrentThemePalette().textSecondary)
                    }
                    .alignItems(HorizontalAlign.Start)
                    .layoutWeight(1)
                  }
                  .layoutWeight(1)
                  .onClick(() => {
                    this.openUserProfile(user);
                  })

                  Button(this.isUserFollowing(user) ? '已关注' : '关注')
                    .type(ButtonType.Capsule)
                    .backgroundColor(this.isUserFollowing(user) ? this.getSurfaceSubtle() : this.getCurrentThemePalette().accentBackground)
                    .fontColor(this.isUserFollowing(user) ? this.getCurrentThemePalette().textPrimary : this.getCurrentThemePalette().accentText)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontWeight(FontWeight.Medium)
                    .height(34)
                    .shadow(this.getCurrentThemePalette().controlShadow)
                    .border({ width: 1, color: this.getSurfaceBorder() })
                    .onClick(async () => {
                      const userId = user.user?.user_id ?? '';
                      if (!userId) return;
                      const isFollowing = this.userStateMap[userId]?.relation?.is_following ?? false;
                      const success = await interactionActions.setFollowUser(userId, !isFollowing, '关注用户');
                      if (success) {
                        SystemToast.showSuccess(isFollowing ? '已取消关注' : '已关注', this.getUIContext());
                      } else {
                        SystemToast.showError('操作失败', this.getUIContext());
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
                .borderRadius(DesignConstants.RADIUS.XXL)
                .shadow(this.getCurrentThemePalette().elevatedShadow)
                .backgroundColor(this.getSurfaceBackground())
                .border({ width: 1, color: this.getSurfaceBorder() })
                .margin({ bottom: 12 })
              }
            }, (user: UserItem) => {
              const uid = user.user?.user_id ?? '';
              return uid + '_f' + (this.userStateMap[uid]?.relation?.is_following ?? false);
            })
          }
        }
        .divider({ strokeWidth: 0 })
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG,
          bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
        .scrollBar(BarState.Auto)
        }
        .onRefreshing(async () => {
          await this.loadData(true);
          this.isRefreshing = false;
        })
      }
    }
        .width('100%')
        .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  private openUserProfile(user: UserItem): void {
    const userId = user.user?.user_id;
    if (!userId) return;
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
  }

  private isUserFollowing(user: UserItem): boolean {
    const userId = user.user?.user_id ?? '';
    const state = this.userStateMap[userId];
    if (state?.relation) {
      return state.relation.is_following;
    }
    return false;
  }

  @Builder
  ListHeader(title: string) {
    Text(title)
      .fontSize(DesignConstants.FONT.SIZE.LG)
      .fontWeight(FontWeight.Bold)
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.SM })
      .fontColor(this.getCurrentThemePalette().textPrimary)
  }

  @Builder
  private buildSectionHeader(title: string, icon: AppIcon) {
    Row({ space: DesignConstants.SPACING.SM }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(18)
          .fontColor([this.getCurrentThemePalette().accentBackground])
      }
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getCurrentThemePalette().textPrimary)
    }
    .width('100%')
  }
}

@Builder
export function RecommendationPageBuilder() {
  RecommendationPage()
}
