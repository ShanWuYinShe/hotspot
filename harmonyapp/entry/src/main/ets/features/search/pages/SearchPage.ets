import { KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_IS_LOGGED_IN, KEY_CLIENT_CONFIG } from '../../../core/state/GlobalStateManager';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { AppRouter, ChatRouteParam, GroupDetailRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam, SearchRouteParam } from '../../../core/router/AppRouter';
import {
  UserItem, GroupItem, MomentItem, GeoPoint,
  JoinGroupRequest, CreateReportRequest, Relationship,
  SearchSuggestItem, SearchSuggestRequest, ClientConfig,
  SearchSuggestItemTag, SearchSuggestItemUser, SearchSuggestItemGroup, SearchSuggestItemMoment,
  SearchUsersRequest, SearchGroupsRequest, SearchMomentsRequest
} from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { UserRelationState } from '../../../core/state/InteractionStateStore';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SearchPageHeader } from '../../../components/SearchPageHeader';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppThemeVisuals, InputBorderResolution } from '../../../core/theme/AppThemeVisuals';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { formatCount } from '../../../core/utils/FormatUtil';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { TimerManager } from '../../../core/utils/TimerManager';
import { curves } from '@kit.ArkUI';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'SearchPage';


@Preview
@Component
export struct SearchPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_CLIENT_CONFIG) clientConfig: ClientConfig | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  @State currentIndex: number = 0;
  @State keyword: string = '';
  @State isSearchFocused: boolean = false;
  @State isLoading: boolean = false;
  @State userResults: UserItem[] = [];
  private userResultsDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State groupResults: GroupItem[] = [];
  private groupResultsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State momentResults: MomentItem[] = [];
  private momentResultsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  @State hasSearched: boolean = false;
  @State suggestItems: SearchSuggestItem[] = [];
  private suggestTimerId: number = 0;
  private controller: TabsController = new TabsController();
  @State userActionPendingIds: string[] = [];
  @State groupActionPendingIds: string[] = [];
  @State isBackPressed: boolean = false;
  @State isRowPressed: boolean = false;
  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
    KeyboardDismissManager.getInstance().dismiss();
    if (this.suggestTimerId) {
      TimerManager.clearTimeout(this.suggestTimerId);
      this.suggestTimerId = 0;
    }
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

  private fieldBorder(focused: boolean, hasError: boolean): InputBorderResolution {
    return AppThemeVisuals.resolveInputBorder({
      backgroundColor: this.getSurfaceBackground(),
      borderColor: this.getSurfaceBorder(),
      borderWidth: 1,
      borderRadius: DesignConstants.SHAPE.INPUT,
      shadow: { radius: 0, color: DesignConstants.COLOR.TRANSPARENT, offsetX: 0, offsetY: 0 },
      height: 38,
      horizontalPadding: DesignConstants.SPACING.LG
    }, focused, hasError, this.resolvedThemeId);
  }

  private getReferencePoint(): GeoPoint | null {
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (refLoc) {
      return {
        latitude: refLoc.latitude,
        longitude: refLoc.longitude
      };
    }
    const fallback = this.clientConfig?.defaults?.default_location?.point;
    if (fallback) {
      return {
        latitude: fallback.latitude,
        longitude: fallback.longitude
      };
    }
    return null;
  }

  private scheduleSuggest(): void {
    if (this.suggestTimerId) {
      TimerManager.clearTimeout(this.suggestTimerId);
      this.suggestTimerId = 0;
    }
    const trimmed = this.keyword.trim();
    if (!trimmed) {
      this.suggestItems = [];
      return;
    }
    this.suggestTimerId = TimerManager.setTimeout(() => {
      this.fetchSuggest(trimmed);
    }, 220, 'search-suggest');
  }

  private async fetchSuggest(prefix: string): Promise<void> {
    if (!prefix) {
      this.suggestItems = [];
      return;
    }
    try {
      // 契约：reference_point 可选，无定位时服务端按回退链解析，照常请求
      const request: SearchSuggestRequest = {
        prefix,
        reference_point: this.getReferencePoint() ?? undefined,
        radius: this.clientConfig?.limits?.search_radius_max
      };
      const response = await ApiClient.searchSuggest(request);
      if (this.isDestroyed) return;
      if (prefix !== this.keyword.trim() || this.hasSearched) {
        return;
      }
      if (response.code === 0 && response.data) {
        this.suggestItems = response.data.items ?? [];
      } else {
        this.suggestItems = [];
      }
    } catch (_) {
      if (prefix === this.keyword.trim() && !this.hasSearched) {
        this.suggestItems = [];
      }
    }
  }

  private async doSearch(): Promise<void> {
    if (this.suggestTimerId) {
      TimerManager.clearTimeout(this.suggestTimerId);
      this.suggestTimerId = 0;
    }
    const trimmedKeyword = this.keyword.trim();
    if (!trimmedKeyword) {
      return;
    }

    this.isLoading = true;
    this.hasSearched = true;
    this.keyword = trimmedKeyword;
    this.suggestItems = [];
    this.userResults = [];
    this.userResultsDataSource.clear();
    this.groupResults = [];
    this.groupResultsDataSource.clear();
    this.momentResults = [];
    this.momentResultsDataSource.clear();

    try {
      await LoadingUtil.withMinimumTime(async () => {
        // 契约：reference_point 可选，无定位时服务端按回退链解析
        const refPoint = this.getReferencePoint() ?? undefined;
        const usersReq: SearchUsersRequest = {
          keyword: trimmedKeyword,
          reference_point: refPoint,
          pagination: { limit: 10 }
        };
        const groupsReq: SearchGroupsRequest = {
          keyword: trimmedKeyword,
          reference_point: refPoint,
          pagination: { limit: 10 }
        };
        const momentsReq: SearchMomentsRequest = {
          keyword: trimmedKeyword,
          reference_point: refPoint,
          pagination: { limit: 10 }
        };

        const usersResp = await ApiClient.searchUsers(usersReq);
        const groupsResp = await ApiClient.searchGroups(groupsReq);
        const momentsResp = await ApiClient.searchMoments(momentsReq);

        if (this.isDestroyed) return;

        const myId = await AuthAccountManager.getCurrentUserId();

        if (usersResp.code === 0 && usersResp.data) {
          const validUsers = usersResp.data.items.filter((u: UserItem) => u.user?.user_id);
          this.userResults = (myId !== null && typeof myId === 'string')
            ? validUsers.filter((u: UserItem) => u.user.user_id !== myId)
            : validUsers;
          this.userResultsDataSource.setData(this.userResults);
        }

        if (groupsResp.code === 0 && groupsResp.data) {
          this.groupResults = groupsResp.data.items.filter((g: GroupItem) => g.group?.group_id);
          this.groupResultsDataSource.setData(this.groupResults);
        }

        if (momentsResp.code === 0 && momentsResp.data) {
          this.momentResults = momentsResp.data.items.filter((m: MomentItem) => m.moment_id && m.user);
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

  private applyKeywordAndSearch(keyword: string): void {
    this.keyword = keyword;
    this.doSearch();
  }

  private handleSuggestItemTap(item: SearchSuggestItem): void {
    switch (item.type) {
      case 'tag': {
        const tagItem = item as SearchSuggestItemTag;
        this.applyKeywordAndSearch(`#${tagItem.payload.name}`);
        break;
      }
      case 'user': {
        const userItem = item as SearchSuggestItemUser;
        const params = new ProfileRouteParam();
        params.userId = userItem.payload.user.user_id;
        AppRouter.push(PageNames.PROFILE, params);
        break;
      }
      case 'group': {
        const groupItem = item as SearchSuggestItemGroup;
        const params = new GroupDetailRouteParam();
        params.groupId = groupItem.payload.group.group_id;
        AppRouter.push(PageNames.GROUP_DETAIL, params);
        break;
      }
      case 'moment': {
        const momentItem = item as SearchSuggestItemMoment;
        const params = new MomentDetailRouteParam();
        params.momentId = momentItem.payload.moment_id;
        AppRouter.push(PageNames.MOMENT_DETAIL, params);
        break;
      }
    }
  }

  private getSuggestKey(item: SearchSuggestItem, index: number): string {
    switch (item.type) {
      case 'tag':
        return `tag-${(item as SearchSuggestItemTag).payload.name}-${index}`;
      case 'user':
        return `user-${(item as SearchSuggestItemUser).payload.user.user_id}-${index}`;
      case 'group':
        return `group-${(item as SearchSuggestItemGroup).payload.group.group_id}-${index}`;
      case 'moment':
        return `moment-${(item as SearchSuggestItemMoment).payload.moment_id}-${index}`;
      default:
        return `suggest-${index}`;
    }
  }

  private getSuggestTitle(item: SearchSuggestItem): string {
    switch (item.type) {
      case 'tag':
        return `#${(item as SearchSuggestItemTag).payload.name}`;
      case 'user':
        return (item as SearchSuggestItemUser).payload.user.nickname;
      case 'group':
        return (item as SearchSuggestItemGroup).payload.group.name;
      case 'moment':
        return (item as SearchSuggestItemMoment).payload.description_preview;
      default:
        return '';
    }
  }

  private getSuggestSubtitle(item: SearchSuggestItem): string {
    switch (item.type) {
      case 'tag':
        return `${(item as SearchSuggestItemTag).payload.usage_count} 次使用`;
      case 'user':
        return (item as SearchSuggestItemUser).payload.bio ?? '打开用户主页';
      case 'group':
        return `${(item as SearchSuggestItemGroup).payload.group.members_count} 人`;
      case 'moment':
        return (item as SearchSuggestItemMoment).payload.author.nickname ?? '打开瞬间详情';
      default:
        return '';
    }
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint }) {
      Column() {
        // Search header —— 统一走 SearchPageHeader（返回 + 内联搜索框）
        SearchPageHeader({
          keyword: this.keyword,
          placeholder: '搜索用户、地点、话题…',
          onBack: () => { AppRouter.pop(); },
          onChange: (value: string) => {
            this.keyword = value;
            this.hasSearched = false;
            this.scheduleSuggest();
          },
          onSubmit: () => { this.doSearch(); },
          onFocusChange: (focused: boolean) => { this.isSearchFocused = focused; }
        })

        if (this.isLoading) {
          SystemLoadingView({ message: '搜索中...' })
            .layoutWeight(1)
        } else if (this.keyword.trim().length > 0 && this.suggestItems.length > 0 && !this.hasSearched) {
          List({ space: DesignConstants.SPACING.SM }) {
            ForEach(this.suggestItems, (item: SearchSuggestItem) => {
              ListItem() {
                this.buildSuggestRow(item)
              }
            }, (item: SearchSuggestItem, index?: number) => this.getSuggestKey(item, index ?? 0))
          }
          .divider({ strokeWidth: 0, color: this.getSurfaceBorder() })
          .layoutWeight(1)
          .width('100%')
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
        } else {
          Tabs({ barPosition: BarPosition.Start, controller: this.controller, index: this.currentIndex }) {
            TabContent() {
              this.UserResultList()
            }.tabBar(this.TabBuilder(0, this.tabLabel('用户', this.userResults.length)))

            TabContent() {
              this.MomentResultList()
            }.tabBar(this.TabBuilder(1, this.tabLabel('动态', this.momentResults.length)))

            TabContent() {
              this.GroupResultList()
            }.tabBar(this.TabBuilder(2, this.tabLabel('群组', this.groupResults.length)))
          }
          .vertical(false)
          .barMode(BarMode.Scrollable)
          .barWidth('100%')
          .barHeight(44)
          .animationDuration(220)
          .onChange((index: number) => {
            this.currentIndex = index;
          })
          .width('100%')
          .layoutWeight(1)
          .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
          .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.MD, bottom: 0 })
        }
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as SearchRouteParam | undefined;
      if (params && params.initialTab >= 0 && params.initialTab <= 2) {
        this.currentIndex = params.initialTab;
      }
      const initialKeyword = params?.keyword?.trim() ?? '';
      if (initialKeyword.length > 0) {
        this.keyword = initialKeyword;
        this.doSearch();
      }
    })
  }

  @Builder
  private buildSuggestRow(item: SearchSuggestItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Text() {
          SymbolSpan($r(
            item.type === 'tag' ? AppIcon.CHART : item.type === 'user' ? AppIcon.PERSON : item.type === 'group' ? AppIcon.GROUP : AppIcon.CAMERA
          ))
            .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
            .fontColor([this.getCurrentThemePalette().accentBackground])
        }
      }
      .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getSurfaceSubtle())
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: DesignConstants.SPACING.XS }) {
        Text(this.getSuggestTitle(item))
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        Text(this.getSuggestSubtitle(item))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      Text() {
        SymbolSpan($r(AppIcon.ARROW_RIGHT))
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor([this.getCurrentThemePalette().textSecondary])
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor(this.getSurfaceBackground())
    .borderRadius(DesignConstants.SHAPE.CARD)
    .shadow(this.getCurrentThemePalette().surfaceShadow)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .onClick(() => this.handleSuggestItemTap(item))
  }

  private tabLabel(base: string, count: number): string {
    return (this.hasSearched && count > 0) ? `${base} (${count})` : base;
  }

  @Builder TabBuilder(index: number, name: string) {
    Text(name)
      .fontColor(this.currentIndex === index ? this.getCurrentThemePalette().accentText : this.getCurrentThemePalette().textSecondary)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
      .padding({ left: DesignConstants.SPACING.MD3, right: DesignConstants.SPACING.MD3, top: DesignConstants.SPACING.SM2, bottom: DesignConstants.SPACING.SM2 })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.currentIndex === index ? this.getCurrentThemePalette().accentBackground : this.getSurfaceBackground())
      .shadow(this.currentIndex === index
        ? this.getCurrentThemePalette().accentShadow
        : this.getCurrentThemePalette().surfaceShadow)
      .border(this.currentIndex === index
        ? { width: 0 }
        : { width: 1, color: this.getSurfaceBorder() })
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_TAB,
          DesignConstants.INTERACTION.SPRING_DAMPING_TAB
        )
      })
      .onClick(() => {
        if (this.currentIndex !== index) {
          this.currentIndex = index;
          this.controller.changeIndex(index);
        }
      })
  }

  @Builder
  UserResultList() {
    if (this.hasSearched && this.userResults.length === 0) {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text() {
          SymbolSpan($r(AppIcon.SEARCH))
            .fontSize(DesignConstants.FONT.SIZE.XXXL_PLUS)
            .fontColor([this.getCurrentThemePalette().textSecondary])
            .opacity(DesignConstants.OPACITY.FADED)
        }
        Text('未找到相关内容').fontSize(DesignConstants.FONT.SIZE.MD_PLUS).fontColor(this.getCurrentThemePalette().textPrimary)
        Text('尝试其他关键词').fontSize(DesignConstants.FONT.SIZE.SM_PLUS).fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      List({ space: DesignConstants.SPACING.MD2 }) {
        ForEach(this.userResults, (item: UserItem) => {
          ListItem() {
            this.buildUserItem(item)
          }
        }, (item: UserItem) => {
          const uid = item.user?.user_id ?? '';
          return uid + '_f' + (this.userStateMap[uid]?.relation?.is_following ?? false) + '_p' + (this.userActionPendingIds.indexOf(uid) >= 0);
        })
      }
      .divider({ strokeWidth: 0, color: this.getSurfaceBorder() })
      .width('100%').height('100%')
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.MD2, bottom: DesignConstants.SPACING.MD })
      .scrollBar(BarState.Auto)
    }
  }

  @Builder
  GroupResultList() {
    if (this.hasSearched && this.groupResults.length === 0) {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text() {
          SymbolSpan($r(AppIcon.SEARCH))
            .fontSize(DesignConstants.FONT.SIZE.XXXL_PLUS)
            .fontColor([this.getCurrentThemePalette().textSecondary])
            .opacity(DesignConstants.OPACITY.FADED)
        }
        Text('未找到相关内容').fontSize(DesignConstants.FONT.SIZE.MD_PLUS).fontColor(this.getCurrentThemePalette().textPrimary)
        Text('尝试其他关键词').fontSize(DesignConstants.FONT.SIZE.SM_PLUS).fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      List() {
        ForEach(this.groupResults, (item: GroupItem) => {
          ListItem() {
            this.buildGroupItem(item)
          }
        }, (item: GroupItem) => {
          const gid = item.group?.group_id ?? '';
          return gid + '_j' + (item.my_state?.is_joined ?? false) + '_p' + (this.groupActionPendingIds.indexOf(gid) >= 0);
        })
      }
      .divider({ strokeWidth: 0.5, color: this.getSurfaceBorder() })
      .width('100%').height('100%')
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.MD2, bottom: DesignConstants.SPACING.MD })
      .scrollBar(BarState.Auto)
    }
  }

  @Builder
  MomentResultList() {
    if (this.hasSearched && this.momentResults.length === 0) {
      Column({ space: DesignConstants.SPACING.LG }) {
        Text() {
          SymbolSpan($r(AppIcon.SEARCH))
            .fontSize(DesignConstants.FONT.SIZE.XXXL_PLUS)
            .fontColor([this.getCurrentThemePalette().textSecondary])
            .opacity(DesignConstants.OPACITY.FADED)
        }
        Text('未找到相关内容').fontSize(DesignConstants.FONT.SIZE.MD_PLUS).fontColor(this.getCurrentThemePalette().textPrimary)
        Text('尝试其他关键词').fontSize(DesignConstants.FONT.SIZE.SM_PLUS).fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%').height('100%').justifyContent(FlexAlign.Center)
    } else {
      WaterFlow() {
        LazyForEach(this.momentResultsDataSource, (item: MomentItem) => {
          FlowItem() {
            this.buildMomentSearchCard(item)
          }
          .width('100%')
        }, (item: MomentItem) => item.moment_id)
      }
      .width('100%').height('100%')
      .columnsTemplate('1fr 1fr')
      .columnsGap(DesignConstants.SPACING.MD)
      .rowsGap(DesignConstants.SPACING.MD)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.MD2, bottom: DesignConstants.SPACING.MD })
      .cachedCount(6)
    }
  }

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

  private isGroupJoined(group: GroupItem): boolean {
    return group.my_state?.is_joined ?? false;
  }

  @Builder
  buildUserItem(item: UserItem) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Row({ space: DesignConstants.SPACING.SM }) {
          SystemAvatar({
            avatarUrl: item.user?.avatar_url ?? '',
            name: item.user?.nickname ?? '',
            avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
            shape: 'circle'
          })

          Column({ space: DesignConstants.SPACING.HALF }) {
            Text(item.user?.nickname ?? '')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getCurrentThemePalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            if (item.presence_status === 'online') {
              Text('在线')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getCurrentThemePalette().accentBackground)
            }
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .layoutWeight(1)
        .onClick(() => {
          this.openUserProfile(item);
        })

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.MORE))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
        }
        .width(DesignConstants.COMPONENT.ICON_SIZE_MD)
        .height(DesignConstants.COMPONENT.ICON_SIZE_MD)
        .borderRadius(DesignConstants.RADIUS.XXL)
        .backgroundColor(this.getSurfaceSubtle())
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)
        .bindMenu([
          { value: '私聊', action: () => this.startPrivateChat(item) },
          { value: this.isUserFollowing(item.user?.user_id ?? '') ? '取消关注' : '关注', action: () => this.toggleFollow(item) },
          { value: '举报用户', action: () => this.reportUser(item) },
          { value: '复制用户ID', action: () => this.copyText(item.user?.user_id ?? '', '用户ID已复制') }
        ])
      }
      .width('100%')

      Text(item.bio ? item.bio : '这个人还没有留下简介')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(DesignConstants.FONT.LINE_HEIGHT.XS)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .maxLines(2)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .width('100%')

      if (item.tags && item.tags.length > 0) {
        Flex({ wrap: FlexWrap.Wrap }) {
          ForEach(item.tags.slice(0, 3), (tag: string) => {
            Text(`#${tag}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.THREE, bottom: DesignConstants.SPACING.THREE })
              .margin({ right: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
              .backgroundColor(this.getSurfaceSubtle())
              .borderRadius(DesignConstants.RADIUS.XL)
          }, (tag: string) => tag)
        }
        .width('100%')
      }

      Row({ space: DesignConstants.SPACING.SM }) {
        Button(this.isUserActionPending(item.user?.user_id ?? '') ? '处理中...' : (this.isUserFollowing(item.user?.user_id ?? '') ? '已关注' : '关注'))
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .enabled(!this.isUserActionPending(item.user?.user_id ?? ''))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .backgroundColor(this.isUserFollowing(item.user?.user_id ?? '') ? DesignConstants.COLOR.TRANSPARENT : this.getCurrentThemePalette().accentBackground)
          .fontColor(this.isUserFollowing(item.user?.user_id ?? '') ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().accentText)
          .borderWidth(this.isUserFollowing(item.user?.user_id ?? '') ? 1 : 0)
          .borderColor(this.getCurrentThemePalette().accentBackground)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .opacity(this.isUserActionPending(item.user?.user_id ?? '') ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => { this.toggleFollow(item); })

        Button('私聊')
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .backgroundColor(this.getSurfaceSubtle())
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .opacity(this.isUserActionPending(item.user?.user_id ?? '') ? DesignConstants.OPACITY.PRESSED : 1)
          .onClick(() => { this.startPrivateChat(item); })
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor(this.getSurfaceBackground())
    .borderRadius(DesignConstants.SHAPE.CARD)
    .shadow(this.getCurrentThemePalette().surfaceShadow)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
  }

  @Builder
  buildGroupItem(item: GroupItem) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Row({ space: DesignConstants.SPACING.SM }) {
          SystemAvatar({
            avatarUrl: item.group?.avatar_url ?? '',
            name: item.group?.name ?? '',
            avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
            shape: 'circle'
          })

          Column({ space: DesignConstants.SPACING.HALF }) {
            Text(item.group?.name ?? '')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getCurrentThemePalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
            Text(`${formatCount(item.group?.members_count)} 人`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .layoutWeight(1)

        Row() {
          Text() {
            SymbolSpan($r(AppIcon.MORE))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([this.getCurrentThemePalette().textSecondary])
          }
        }
        .width(DesignConstants.COMPONENT.ICON_SIZE_MD)
        .height(DesignConstants.COMPONENT.ICON_SIZE_MD)
        .borderRadius(DesignConstants.RADIUS.XXL)
        .backgroundColor(this.getSurfaceSubtle())
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)
        .bindMenu([
          { value: '查看详情', action: () => this.openGroupDetail(item) },
          { value: this.isGroupJoined(item) ? '进入群聊' : '加入群组', action: () => this.joinGroup(item) },
          { value: '举报群组', action: () => this.reportGroup(item) },
          { value: '复制群组ID', action: () => this.copyText(item.group?.group_id ?? '', '群组ID已复制') }
        ])
      }
      .width('100%')

      Text(item.description ? item.description : '这个群还没有群介绍')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(DesignConstants.FONT.LINE_HEIGHT.XS)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .maxLines(2)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
        .width('100%')

      Row({ space: DesignConstants.SPACING.SM }) {
        Button(this.isGroupActionPending(item.group?.group_id ?? '') ? '处理中...' : (this.isGroupJoined(item) ? '进入群聊' : '加入群组'))
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .enabled(!this.isGroupActionPending(item.group?.group_id ?? ''))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .backgroundColor(this.isGroupJoined(item) ? DesignConstants.COLOR.TRANSPARENT : this.getCurrentThemePalette().accentBackground)
          .fontColor(this.isGroupJoined(item) ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().accentText)
          .borderWidth(this.isGroupJoined(item) ? 1 : 0)
          .borderColor(this.getCurrentThemePalette().accentBackground)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .opacity(this.isGroupActionPending(item.group?.group_id ?? '') ? DesignConstants.OPACITY.LOADING : 1)
          .onClick(() => { this.joinGroup(item); })

        Button('详情')
          .type(ButtonType.Capsule)
          .layoutWeight(1)
          .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .backgroundColor(this.getSurfaceSubtle())
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
          .opacity(this.isGroupActionPending(item.group?.group_id ?? '') ? DesignConstants.OPACITY.PRESSED : 1)
          .onClick(() => { this.openGroupDetail(item); })
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor(this.getSurfaceBackground())
    .borderRadius(DesignConstants.SHAPE.CARD)
    .shadow(this.getCurrentThemePalette().surfaceShadow)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
  }

  @Builder
  buildMomentSearchCard(item: MomentItem) {
    Column({ space: 0 }) {
      if (item.images && item.images.length > 0) {
        Image(MediaUrlUtil.normalize(item.images[0].url))
          .width('100%')
          .height(this.calculateSearchImageHeight(item))
          .objectFit(ImageFit.Cover)
          .backgroundColor(this.getSurfaceSubtle())
      } else {
        Column() {
          Text(item.description.substring(0, 60))
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(4)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .width('100%')
        .height(DesignConstants.COMPONENT.PLACEHOLDER_AREA_HEIGHT)
        .padding(DesignConstants.SPACING.MD)
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.getSurfaceBackground())
      }

      Row({ space: DesignConstants.SPACING.SM2 }) {
        SystemAvatar({
          avatarUrl: item.user?.avatar_url ?? '',
          name: item.user?.nickname ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL
        })

        Text(item.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .layoutWeight(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.HALF }) {
          Text() {
            SymbolSpan($r(AppIcon.HEART))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor([this.getCurrentThemePalette().textTertiary])
          }
          Text(formatCount(item.likes_count))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textTertiary)
        }
      }
      .padding(DesignConstants.SPACING.SM)
      .width('100%')
    }
    .backgroundColor(this.getSurfaceBackground())
    .borderRadius(DesignConstants.RADIUS.XXL)
    .clip(true)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .onClick(() => {
      this.openMomentDetail(item);
    })
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
    const userId = user.user?.user_id;
    if (!userId) return;
    if (this.isUserActionPending(userId)) return;
    this.setUserActionPending(userId, true);
    try {
      const isFollowing = this.isUserFollowing(userId);
      const success = await interactionActions.setFollowUser(userId, !isFollowing, '关注用户');
      if (success) {
        this.patchUserRelation(userId, !isFollowing);
        SystemToast.showSuccess(isFollowing ? '已取消关注' : '已关注', this.getUIContext());
      } else {
        SystemToast.showError('操作失败', this.getUIContext());
      }
    } finally {
      this.setUserActionPending(userId, false);
    }
  }

  private isUserFollowing(userId: string): boolean {
    const state = this.userStateMap[userId];
    if (state?.relation) {
      return state.relation.is_following;
    }
    return false;
  }

  private async joinGroup(group: GroupItem): Promise<void> {
    if (!this.isLoggedIn) {
      SystemToast.showWarning('请先登录后再加入群组', this.getUIContext());
      const loginParams = new LoginRouteParam();
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
      } else if (response.code === 10400) {
        SystemToast.show(response.message || '加入失败', this.getUIContext());
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

  private openUserProfile(user: UserItem): void {
    const userId = user.user?.user_id;
    if (!userId) {
      return;
    }
    const params = new ProfileRouteParam();
    params.userId = userId;
    AppRouter.push(PageNames.PROFILE, params);
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
      const loginParams = new LoginRouteParam();
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
      const loginParams = new LoginRouteParam();
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
      const loginParams = new LoginRouteParam();
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
      const loginParams = new LoginRouteParam();
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
