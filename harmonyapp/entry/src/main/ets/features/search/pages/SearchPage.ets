/**
 * 全局搜索页面
 * 重构：使用 HarmonyOS 原生 Tabs 组件
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_IS_LOGGED_IN } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { AppRouter, ChatRouteParam, GroupDetailRouteParam, MomentDetailRouteParam } from '../../../core/router/AppRouter';
import {
  UserItem, GroupItem, MomentItem, GeoPoint, FollowRequest, UnfollowRequest,
  JoinGroupRequest, CreateReportRequest, Relationship
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import pasteboard from '@ohos.pasteboard';


const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'SearchPage';

class LoginRedirectParams {
  reason: string = '';
  action: string = '';
}

@Preview
@Component
export struct SearchPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  
  @State currentIndex: number = 0;
  @State keyword: string = '';
  @State isLoading: boolean = false;
  @State userResults: UserItem[] = [];
  private userResultsDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State groupResults: GroupItem[] = [];
  private groupResultsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State momentResults: MomentItem[] = [];
  private momentResultsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  @State hasSearched: boolean = false;
  private controller: TabsController = new TabsController();
  @State userActionPendingIds: string[] = [];
  @State groupActionPendingIds: string[] = [];
  @State pressedMomentId: string = '';

  aboutToAppear(): void {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
  }

  private async doSearch() {
    if (!this.keyword.trim()) return;
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (!refLoc) {
      return;
    }

    this.isLoading = true;
    this.hasSearched = true;
    this.userResults = [];
    this.userResultsDataSource.clear();
    this.groupResults = [];
    this.groupResultsDataSource.clear();
    this.momentResults = [];
    this.momentResultsDataSource.clear();

    const referencePoint: GeoPoint = {
      latitude: refLoc.latitude,
      longitude: refLoc.longitude
    };

    try {
      await LoadingUtil.withMinimumTime(async () => {

        const p1 = ApiClient.searchUsers({
          keyword: this.keyword,
          reference_point: referencePoint
        });
        const p2 = ApiClient.searchGroups({
          keyword: this.keyword,
          reference_point: referencePoint
        });
        const p3 = ApiClient.searchMoments({
          keyword: this.keyword,
          reference_point: referencePoint
        });

        const results = await Promise.all([p1, p2, p3]);
        const res1 = results[0];
        const res2 = results[1];
        const res3 = results[2];

        const myId = await AuthAccountManager.getCurrentUserId();
        if (res1.code === 0) {
          const items = res1.data?.items ?? [];
          const validUsers = items.filter((u: UserItem) => u.user?.user_id);
          this.userResults = (myId !== null && typeof myId === 'string')
            ? validUsers.filter((u: UserItem) => u.user.user_id !== myId)
            : validUsers;
          this.userResultsDataSource.setData(this.userResults);
        }
        if (res2.code === 0) {
          const items = res2.data?.items ?? [];
          this.groupResults = items.filter((g: GroupItem) => g.group?.group_id);
          this.groupResultsDataSource.setData(this.groupResults);
        }
        if (res3.code === 0) {
          const items = res3.data?.items ?? [];
          this.momentResults = items.filter((m: MomentItem) => m.moment_id && m.user);
          this.momentResultsDataSource.setData(this.momentResults);
        }
      });
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索失败: ${error}`);
      SystemToast.showError('搜索失败', this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  build() {
    NavDestination() {
    Column() {
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            AppRouter.pop();
          }
        })

        TextInput({ placeholder: '搜索用户、群组、瞬间...', text: this.keyword })
          .layoutWeight(1)
          .height(42)
          .backgroundColor($r('app.color.bg_secondary'))
          .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
          .borderRadius(DesignConstants.RADIUS.XXL)
          .padding({ left: 16, right: 16 })
          .margin({ left: 8, right: 8 })
          .onChange((value) => {
            this.keyword = value;
          })
          .onSubmit(() => {
            this.doSearch();
          })

        IconButton({
          icon: AppIcon.SEARCH,
          iconSize: IconSize.MEDIUM,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: $r('app.color.bg_secondary'),
          onButtonClick: () => {
            this.doSearch();
          }
        })
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor($r('app.color.bg_primary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      if (this.isLoading) {
        SystemLoadingView({ message: '搜索中...' })
          .layoutWeight(1)
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else {
        Tabs({ barPosition: BarPosition.Start, controller: this.controller, index: this.currentIndex }) {
          TabContent() {
            this.UserResultList()
          }.tabBar(this.TabBuilder(0, '用户'))

          TabContent() {
            this.GroupResultList()
          }.tabBar(this.TabBuilder(1, '群组'))

          TabContent() {
            this.MomentResultList()
          }.tabBar(this.TabBuilder(2, '瞬间'))
        }
        .vertical(false)
        .barMode(BarMode.Fixed)
        .barWidth('100%')
        .barHeight(60)
        .animationDuration(300)
        .onChange((index: number) => {
          this.currentIndex = index;
        })
        .width('100%')
        .layoutWeight(1)
        .backgroundColor($r('app.color.bg_primary'))
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
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

  @Builder TabBuilder(index: number, name: string) {
    Row({ space: 4 }) {
      Text(name)
        .fontColor(this.currentIndex === index ? $r('sys.color.ohos_id_color_text_primary') : $r('sys.color.ohos_id_color_text_secondary'))
        .fontSize(16)
        .fontWeight(this.currentIndex === index ? FontWeight.Medium : FontWeight.Normal)
        .lineHeight(20)
    }
    .width('100%')
    .height(42)
    .justifyContent(FlexAlign.Center)
    .alignItems(VerticalAlign.Center)
    .margin({ left: 8, right: 8, top: 9, bottom: 9 })
    .padding({ left: 14, right: 14 })
    .backgroundColor(this.currentIndex === index ? $r('app.color.bg_secondary') : Color.Transparent)
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({
      width: this.currentIndex === index ? 1 : 0,
      color: $r('sys.color.ohos_id_color_list_separator')
    })
  }

  @Builder
  UserResultList() {
    if (this.hasSearched && this.userResults.length === 0) {
      Column({ space: 16 }) {
        Text() {
          SymbolSpan($r(AppIcon.SEARCH))
            .fontSize(48)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }
        Text('未找到相关结果').fontSize(16).fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      List({ space: DesignConstants.SPACING.MD }) {
        ForEach(this.userResults, (item: UserItem) => {
          ListItem() {
            this.buildUserItem(item)
          }
        }, (item: UserItem) => item.user?.user_id ?? '')
      }
      .width('100%').height('100%')
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.SM + 2,
        bottom: DesignConstants.SPACING.MD
      })
      .scrollBar(BarState.Auto)
    }
  }

  @Builder
  GroupResultList() {
    if (this.hasSearched && this.groupResults.length === 0) {
      Column({ space: 16 }) {
        Text() {
          SymbolSpan($r(AppIcon.SEARCH))
            .fontSize(48)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }
        Text('未找到相关结果').fontSize(16).fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      List({ space: DesignConstants.SPACING.MD }) {
        ForEach(this.groupResults, (item: GroupItem) => {
          ListItem() {
            this.buildGroupItem(item)
          }
        }, (item: GroupItem) => item.group?.group_id ?? '')
      }
      .width('100%').height('100%')
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.SM + 2,
        bottom: DesignConstants.SPACING.MD
      })
      .scrollBar(BarState.Auto)
    }
  }

  @Builder
  MomentResultList() {
    if (this.hasSearched && this.momentResults.length === 0) {
      Column({ space: 16 }) {
        Text() {
          SymbolSpan($r(AppIcon.SEARCH))
            .fontSize(48)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }
        Text('未找到相关结果').fontSize(16).fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      // 瀑布流布局展示瞬间结果
      WaterFlow() {
        LazyForEach(this.momentResultsDataSource, (item: MomentItem) => {
          FlowItem() {
            this.buildMomentSearchCard(item)
          }
        }, (item: MomentItem) => item.moment_id)
      }
      .width('100%').height('100%')
      .columnsTemplate('1fr 1fr')
      .columnsGap(DesignConstants.SPACING.MD)
      .rowsGap(DesignConstants.SPACING.MD)
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD,
        top: DesignConstants.SPACING.SM + 2,
        bottom: DesignConstants.SPACING.MD
      })
      .cachedCount(6)
    }
  }

  // 计算搜索卡片图片高度
  private calculateSearchImageHeight(moment: MomentItem): number {
    if (!moment.images || moment.images.length === 0) {
      return 140;
    }
    const image = moment.images[0];
    if (image.width && image.height && image.width > 0) {
      const aspectRatio = image.height / image.width;
      const calculatedHeight = 150 * aspectRatio;
      return Math.max(100, Math.min(calculatedHeight, 240));
    }
    return 160;
  }

  private isUserActionPending(userId: string): boolean {
    return this.userActionPendingIds.indexOf(userId) >= 0;
  }

  private isGroupActionPending(groupId: string): boolean {
    return this.groupActionPendingIds.indexOf(groupId) >= 0;
  }

  private isUserFollowing(user: UserItem): boolean {
    return user.relation?.is_following ?? false;
  }

  private isGroupJoined(group: GroupItem): boolean {
    return group.my_state?.is_joined ?? false;
  }

  @Builder
  buildUserItem(item: UserItem) {
    Column({ space: 8 }) {
      Row({ space: 8 }) {
        Row({ space: 8 }) {
          SystemAvatar({
            avatarUrl: item.user?.avatar_url ?? '',
            name: item.user?.nickname ?? '',
            avatarSize: 40
          })

          Column({ space: 2 }) {
            Text(item.user?.nickname ?? '')
              .fontSize(14)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            if (item.presence_status === 'online') {
              Text('在线')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor($r('app.color.color_success'))
            }
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .layoutWeight(1)

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.MORE))
              .fontSize(16)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
        }
        .width(24)
        .height(24)
        .borderRadius(DesignConstants.RADIUS.XL)
        .backgroundColor($r('app.color.bg_tertiary'))
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)
        .bindMenu([
          {
            value: '私聊',
            action: () => this.startPrivateChat(item)
          },
          {
            value: this.isUserFollowing(item) ? '取消关注' : '关注',
            action: () => this.toggleFollow(item)
          },
          {
            value: '举报用户',
            action: () => this.reportUser(item)
          },
          {
            value: '复制用户ID',
            action: () => this.copyText(item.user?.user_id ?? '', '用户ID已复制')
          }
        ])
      }
      .width('100%')

      if (item.bio) {
        Text(item.bio)
          .fontSize(12)
          .lineHeight(18)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      } else {
        Text('这个人还没有留下简介')
          .fontSize(12)
          .lineHeight(18)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      }

      if (item.tags && item.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(item.tags.slice(0, 3), (tag: string) => {
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .padding({ left: 7, right: 7, top: 3, bottom: 3 })
              .margin({ right: 4, bottom: 4 })
              .backgroundColor($r('app.color.bg_tertiary'))
              .borderRadius(DesignConstants.RADIUS.MD)
          }, (tag: string) => tag)
        }
        .width('100%')
      }

      Row({ space: 8 }) {
        Button(this.isUserActionPending(item.user?.user_id ?? '') ? '处理中...' :
          (this.isUserFollowing(item) ? '已关注' : '关注'))
          .type(ButtonType.Normal)
          .layoutWeight(1)
          .height(30)
          .borderRadius(DesignConstants.RADIUS.XL)
          .enabled(!this.isUserActionPending(item.user?.user_id ?? ''))
          .fontSize(12)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(this.isUserFollowing(item) ? Color.Transparent : $r('sys.color.ohos_id_color_emphasize'))
          .fontColor(this.isUserFollowing(item) ? $r('sys.color.ohos_id_color_emphasize') : Color.White)
          .borderWidth(this.isUserFollowing(item) ? 1 : 0)
          .borderColor($r('sys.color.ohos_id_color_emphasize'))
          .opacity(this.isUserActionPending(item.user?.user_id ?? '') ? 0.58 : 1)
          .onClick(() => { this.toggleFollow(item); })

        Button('私聊')
          .type(ButtonType.Normal)
          .layoutWeight(1)
          .height(30)
          .borderRadius(DesignConstants.RADIUS.XL)
          .fontSize(12)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .borderWidth(1)
          .borderColor($r('sys.color.ohos_id_color_list_separator'))
          .opacity(this.isUserActionPending(item.user?.user_id ?? '') ? 0.74 : 1)
          .onClick(() => { this.startPrivateChat(item); })
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
    .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
  }

  @Builder
  buildGroupItem(item: GroupItem) {
    Column({ space: 8 }) {
      Row({ space: 8 }) {
        Row({ space: 8 }) {
          SystemAvatar({
            avatarUrl: item.group?.avatar_url ?? '',
            name: item.group?.name ?? '',
            avatarSize: 40,
            shape: 'squircle'
          })

          Column({ space: 2 }) {
            Text(item.group?.name ?? '')
              .fontSize(14)
              .fontWeight(FontWeight.Medium)
              .fontColor($r('sys.color.ohos_id_color_text_primary'))
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            Text(`${item.group?.members_count ?? 0} 人`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .layoutWeight(1)

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.MORE))
              .fontSize(16)
              .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
          }
        }
        .width(24)
        .height(24)
        .borderRadius(DesignConstants.RADIUS.XL)
        .backgroundColor($r('app.color.bg_tertiary'))
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)
        .bindMenu([
          {
            value: '查看详情',
            action: () => this.openGroupDetail(item)
          },
          {
            value: this.isGroupJoined(item) ? '进入群聊' : '加入群组',
            action: () => this.joinGroup(item)
          },
          {
            value: '举报群组',
            action: () => this.reportGroup(item)
          },
          {
            value: '复制群组ID',
            action: () => this.copyText(item.group?.group_id ?? '', '群组ID已复制')
          }
        ])
      }
      .width('100%')

      if (item.description) {
        Text(item.description)
          .fontSize(12)
          .lineHeight(18)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      } else {
        Text('这个群还没有群介绍')
          .fontSize(12)
          .lineHeight(18)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      }

      Row({ space: 8 }) {
        Button(this.isGroupActionPending(item.group?.group_id ?? '') ? '处理中...' :
          (this.isGroupJoined(item) ? '进入群聊' : '加入群组'))
          .type(ButtonType.Normal)
          .layoutWeight(1)
          .height(30)
          .borderRadius(DesignConstants.RADIUS.XL)
          .enabled(!this.isGroupActionPending(item.group?.group_id ?? ''))
          .fontSize(12)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(this.isGroupJoined(item) ? Color.Transparent : $r('sys.color.ohos_id_color_emphasize'))
          .fontColor(this.isGroupJoined(item) ? $r('sys.color.ohos_id_color_emphasize') : Color.White)
          .borderWidth(this.isGroupJoined(item) ? 1 : 0)
          .borderColor($r('sys.color.ohos_id_color_emphasize'))
          .opacity(this.isGroupActionPending(item.group?.group_id ?? '') ? 0.58 : 1)
          .onClick(() => { this.joinGroup(item); })

        Button('详情')
          .type(ButtonType.Normal)
          .layoutWeight(1)
          .height(30)
          .borderRadius(DesignConstants.RADIUS.XL)
          .fontSize(12)
          .fontWeight(FontWeight.Medium)
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .borderWidth(1)
          .borderColor($r('sys.color.ohos_id_color_list_separator'))
          .opacity(this.isGroupActionPending(item.group?.group_id ?? '') ? 0.74 : 1)
          .onClick(() => { this.openGroupDetail(item); })
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
    .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
  }

  @Builder
  buildMomentSearchCard(item: MomentItem) {
    Column({ space: 0 }) {
      // 图片展示
      if (item.images && item.images.length > 0) {
        Image(item.images[0].url)
          .width('100%')
          .height(this.calculateSearchImageHeight(item))
          .objectFit(ImageFit.Cover)
          .borderRadius({ topLeft: DesignConstants.RADIUS.XXL, topRight: DesignConstants.RADIUS.XXL })
      } else {
        // 无图片展示文字摘要
        Column() {
          Text(item.description.substring(0, 60))
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(4)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .width('100%')
        .height(120)
        .padding(DesignConstants.SPACING.MD)
        .justifyContent(FlexAlign.Center)
        .backgroundColor($r('app.color.bg_secondary'))
        .borderRadius({ topLeft: DesignConstants.RADIUS.XXL, topRight: DesignConstants.RADIUS.XXL })
      }

      // 用户信息
      Row({ space: 6 }) {
        SystemAvatar({
          avatarUrl: item.user?.avatar_url ?? '',
          name: item.user?.nickname ?? '',
          avatarSize: 18
        })

        Text(item.user?.nickname ?? '')
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .layoutWeight(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        // 点赞数
        Row({ space: 2 }) {
          Text() {
            SymbolSpan($r(AppIcon.HEART))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
          }
          Text(`${item.likes_count}`)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
        }
      }
      .padding(8)
      .width('100%')
    }
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
    .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
    .scale(this.pressedMomentId === item.moment_id ? { x: 0.985, y: 0.985 } : { x: 1, y: 1 })
    .animation({ duration: 100, curve: Curve.EaseOut })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedMomentId = item.moment_id;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedMomentId = '';
      }
    })
    .onClick(() => {
      this.openMomentDetail(item);
    })
  }

  // 保留原有的列表项构建器用于其他场景
  @Builder
  buildMomentItem(item: MomentItem) {
    Column({ space: 8 }) {
      Row() {
        SystemAvatar({
          avatarUrl: item.user?.avatar_url ?? '',
          name: item.user?.nickname ?? '',
          avatarSize: 32
        })
        .margin({ right: 8 })

        Text(item.user?.nickname ?? '')
          .fontSize(14)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
      }

      Text(item.description)
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .maxLines(3)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
    .shadow({ radius: 8, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 2 })
    .margin({ bottom: DesignConstants.SPACING.MD })
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

  private setGroupActionPending(groupId: string, pending: boolean): void {
    const next: string[] = [];
    for (let i = 0; i < this.groupActionPendingIds.length; i++) {
      const id = this.groupActionPendingIds[i];
      if (id !== groupId) {
        next.push(id);
      }
    }
    if (pending) {
      next.push(groupId);
    }
    this.groupActionPendingIds = next;
  }

  private patchUserRelation(userId: string, isFollowing: boolean): void {
    const users = this.userResults;
    for (let i = 0; i < users.length; i++) {
      const currentId = users[i].user?.user_id ?? '';
      if (currentId !== userId) {
        continue;
      }
      if (!users[i].relation) {
        const relation: Relationship = {
          is_following: false,
          is_followed_by: false,
          is_blocked: false
        };
        users[i].relation = relation;
      }
      const relation: Relationship = users[i].relation as Relationship;
      relation.is_following = isFollowing;
      break;
    }
    this.userResults = users;
    this.userResultsDataSource.setData(users);
  }

  private async toggleFollow(user: UserItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再关注', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '关注用户';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const userId = user.user?.user_id;
    if (!userId) return;
    const isFollowing = user.relation?.is_following ?? false;
    if (this.isUserActionPending(userId)) return;
    this.setUserActionPending(userId, true);
    try {
      const response = isFollowing
        ? await ApiClient.unfollowUser({ user_id: userId } as UnfollowRequest)
        : await ApiClient.followUser({ user_id: userId } as FollowRequest);
      if (response.code === 0) {
        this.patchUserRelation(userId, !isFollowing);
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

  private async joinGroup(group: GroupItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '加入群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const groupId = group.group?.group_id;
    if (!groupId) return;
    if (group.my_state?.is_joined) {
      this.enterGroupChat(group);
      return;
    }
    if (this.isGroupActionPending(groupId)) return;
    this.setGroupActionPending(groupId, true);
    try {
      const response = await ApiClient.joinGroup({ group_id: groupId } as JoinGroupRequest);
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          SystemToast.showSuccess('加入成功', this.getUIContext());
          await this.doSearch();
        } else {
          SystemToast.show('已提交申请，等待审批', this.getUIContext());
        }
      } else if (response.code === 1305) {
        SystemToast.show('您已是该群成员', this.getUIContext());
      } else {
        SystemToast.showError(response.message || '加入失败', this.getUIContext());
      }
    } catch {
      SystemToast.showError('加入失败', this.getUIContext());
    } finally {
      this.setGroupActionPending(groupId, false);
    }
  }

  private openGroupDetail(group: GroupItem): void {
    const p = new GroupDetailRouteParam();
    p.groupId = group.group?.group_id ?? '';
    AppRouter.push(PageNames.GROUP_DETAIL, p);
  }

  private openMomentDetail(moment: MomentItem): void {
    if (!moment.moment_id) {
      return;
    }
    const params = new MomentDetailRouteParam();
    params.momentId = moment.moment_id;
    AppRouter.push(PageNames.MOMENT_DETAIL, params);
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

  private enterGroupChat(group: GroupItem): void {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再进入群聊', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '进入群聊';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const groupId = group.group?.group_id;
    if (!groupId) return;
    const params = new ChatRouteParam();
    params.chatId = groupId;
    params.chatType = 'group_chat';
    params.title = group.group?.name ?? '群聊';
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

  private async reportGroup(group: GroupItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再举报', this.getUIContext());
      const loginParams = new LoginRedirectParams();
      loginParams.reason = 'action_required';
      loginParams.action = '举报群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }
    const groupId = group.group?.group_id;
    if (!groupId) return;
    try {
      const req: CreateReportRequest = {
        target_type: 'group',
        target_id: groupId,
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

@Builder export function SearchPageBuilder() {
  SearchPage();
}
