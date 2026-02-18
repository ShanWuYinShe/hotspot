/**
 * 全局搜索页面
 * 重构：使用 HarmonyOS 原生 Tabs 组件
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { AppRouter, GroupDetailRouteParam } from '../../../core/router/AppRouter';
import { UserItem, GroupItem, MomentItem, GeoPoint } from '../../../core/types/api';
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


const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'SearchPage';

@Component
export struct SearchPage {
  pathStack: NavPathStack = new NavPathStack();
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
          .height(40)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .borderRadius(20)
          .padding({ left: 16, right: 16 })
          .margin({ left: 8, right: 16 })
          .onChange((value) => {
            this.keyword = value;
          })
          .onSubmit(() => {
            this.doSearch();
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
        .barHeight(56)
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
    Column() {
      Text(name)
        .fontColor(this.currentIndex === index ? $r('sys.color.ohos_id_color_emphasize') : $r('sys.color.ohos_id_color_text_secondary'))
        .fontSize(16)
        .fontWeight(this.currentIndex === index ? FontWeight.Medium : FontWeight.Normal)
        .lineHeight(22)
        .margin({ top: 17, bottom: 7 })
      Divider()
        .strokeWidth(2)
        .color($r('sys.color.ohos_id_color_emphasize'))
        .opacity(this.currentIndex === index ? 1 : 0)
        .width(this.currentIndex === index ? 20 : 0)
        .animation({ duration: 250, curve: Curve.FastOutSlowIn })
    }.width('100%')
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
      List() {
        LazyForEach(this.userResultsDataSource, (item: UserItem) => {
          ListItem() {
            this.buildUserItem(item)
          }
        }, (item: UserItem) => item.user?.user_id ?? '')
      }
      .width('100%').height('100%')
      .divider({ strokeWidth: 0.5, color: $r('app.color.divider'), startMargin: 76, endMargin: 0 })
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
      List() {
        LazyForEach(this.groupResultsDataSource, (item: GroupItem) => {
          ListItem() {
            this.buildGroupItem(item)
          }
        }, (item: GroupItem) => item.group?.group_id ?? '')
      }
      .width('100%').height('100%')
      .divider({ strokeWidth: 0.5, color: $r('app.color.divider'), startMargin: 76, endMargin: 0 })
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
      .columnsTemplate('repeat(auto-fill, 150vp)')  // 响应式列数
      .columnsGap(8)
      .rowsGap(8)

      .padding(8)
      .cachedCount(6, true)
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

  @Builder
  buildUserItem(item: UserItem) {
    Row({ space: 12 }) {
      SystemAvatar({
        avatarUrl: item.user?.avatar_url ?? '',
        name: item.user?.nickname ?? '',
        avatarSize: 48
      })

      Column({ space: 2 }) {
        Text(item.user?.nickname ?? '')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        if (item.bio) {
          Text(item.bio)
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 12, bottom: 12 })
  }

  @Builder
  buildGroupItem(item: GroupItem) {
    Row({ space: 12 }) {
      SystemAvatar({
        avatarUrl: item.group?.avatar_url ?? '',
        name: item.group?.name ?? '',
        avatarSize: 48,
        shape: 'squircle'
      })

      Column({ space: 2 }) {
        Text(item.group?.name ?? '')
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: 8 }) {
          Text(`${item.group?.members_count ?? 0}人`)
            .fontSize(13)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))

          if (item.description) {
            Text(item.description)
              .fontSize(13)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
              .layoutWeight(1)
          }
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 12, bottom: 12 })
    .onClick(() => {
      const p = new GroupDetailRouteParam();
      p.groupId = item.group?.group_id ?? '';
      AppRouter.push(PageNames.GROUP_DETAIL, p);
    })
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
          .borderRadius({ topLeft: 12, topRight: 12 })
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
        .padding(10)
        .justifyContent(FlexAlign.Center)
        .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
        .borderRadius({ topLeft: 12, topRight: 12 })
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
              .fontSize(10)
              .fontColor([$r('sys.color.ohos_id_color_text_tertiary')])
          }
          Text(`${item.likes_count}`)
            .fontSize(10)
            .fontColor($r('sys.color.ohos_id_color_text_tertiary'))
        }
      }
      .padding(8)
      .width('100%')
    }
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(12)
    .shadow({ radius: 6, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 1 })
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
    .padding(16)
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .margin({ bottom: 12 })
  }
}

@Builder export function SearchPageBuilder() {
  SearchPage();
}
