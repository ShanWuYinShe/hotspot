import { KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_IS_LOGGED_IN, KEY_CLIENT_CONFIG } from '../../../core/state/GlobalStateManager';
import { PageShellContainer } from '../../../components/PageShellContainer';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { AppRouter, ChatRouteParam, GroupDetailRouteParam, LoginRouteParam, MomentDetailRouteParam, ProfileRouteParam, SearchRouteParam } from '../../../core/router/AppRouter';
import {
  UserItem, GroupItem, MomentItem, GeoPoint,
  EnterGroupRequest, CreateReportRequest, Relationship,
  SearchSuggestItem, SearchSuggestRequest, ClientConfig,
  SearchSuggestItemTag, SearchSuggestItemUser, SearchSuggestItemGroup, SearchSuggestItemMoment,
  SearchUsersRequest, SearchGroupsRequest, SearchMomentsRequest
} from '../../../core/types/api';
import { interactionActions } from '../../../core/state/InteractionActions';
import { UserRelationState, sumReactionCounts } from '../../../core/state/InteractionStateStore';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { Pressable } from '../../../components/Pressable';
import { SearchSuggestRow, SearchMomentRow } from '../components/SearchRows';
import { EmptyState } from '../../../components/EmptyState';
import { SearchPageHeader } from '../../../components/SearchPageHeader';
import { Skeleton } from '../../../components/states/Skeleton';
import { SkeletonShimmer } from '../../../components/states/SkeletonShimmer';
import { SystemToast } from '../../../components/SystemToast';
import { SystemButton } from '../../../components/SystemButton';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppThemeVisuals, InputBorderResolution } from '../../../core/theme/AppThemeVisuals';
import { AppIcon } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { isDesktopForm } from '../../../core/responsive/Breakpoint';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { ImageUrlUtil } from '../../../core/utils/ImageUrlUtil';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { formatCount } from '../../../core/utils/FormatUtil';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import pasteboard from '@ohos.pasteboard';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { TimerManager } from '../../../core/utils/TimerManager';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { LucideIcon } from '../../../components/LucideIcon';

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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageLink('interaction_user_state') userStateMap: Record<string, UserRelationState> = {};
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  @State currentIndex: number = 0;
  @State keyword: string = '';
  @State isSearchFocused: boolean = false;
  @State isLoading: boolean = false;
  @State private joinPoppedGroupId: string = '';
  @State private followPoppedUserId: string = '';
  @State private enterOpacity: number = 1;
  @State private enterOffset: number = 0;
  private enterPlayed: boolean = false;

  /** 建议词请求进行中（输入后、结果返回前的骨架态；避免出现只有 tab 行、下方全空白的中间态） */
  @State isSuggesting: boolean = false;
  @State userResults: UserItem[] = [];
  private userResultsDataSource: BasicDataSource<UserItem> = new BasicDataSource<UserItem>([]);
  @State groupResults: GroupItem[] = [];
  private groupResultsDataSource: BasicDataSource<GroupItem> = new BasicDataSource<GroupItem>([]);
  @State momentResults: MomentItem[] = [];
  private momentResultsDataSource: BasicDataSource<MomentItem> = new BasicDataSource<MomentItem>([]);
  /** 话题（从动态标签聚合，对齐前端搜索「话题」Tab） */
  @State topicResults: string[] = [];
  /** 无查询发现态（对齐前端 SearchDiscovery）：最近搜索 / 热门搜索 / 推荐话题 */
  @State recentSearches: string[] = [];

  @State discoveryTopics: string[] = [];
  /** R6：搜索结果图已加载 URL（onComplete 渐显） */
  @State loadedImageUrls: string[] = [];
  @State failedImageUrls: string[] = [];

  private readonly RECENT_SEARCHES_KEY: string = 'search_recent';
  @State hasSearched: boolean = false;
  /** 搜索失败内联错误态（P2-4）：结果区展示 EmptyState + 重试，替代纯 toast */
  @State searchError: string = '';
  @State suggestItems: SearchSuggestItem[] = [];
  private suggestTimerId: number = 0;
  @State userActionPendingIds: string[] = [];
  @State groupActionPendingIds: string[] = [];
  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.loadRecentSearches();
    this.loadDiscovery();
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

  private isImageLoaded(url: string): boolean {
    return this.loadedImageUrls.indexOf(url) >= 0;
  }

  private isImageFailed(url: string): boolean {
    return this.failedImageUrls.indexOf(url) >= 0;
  }

  private markImageFailed(url: string): void {
    if (!url || this.isImageFailed(url)) return;
    const next: string[] = this.failedImageUrls.slice();
    next.push(url);
    this.failedImageUrls = next;
  }

  private markImageLoaded(url: string): void {
    if (!url || this.isImageLoaded(url)) return;
    const next: string[] = this.loadedImageUrls.slice();
    next.push(url);
    this.loadedImageUrls = next;
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
      borderWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
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
      this.isSuggesting = false;
      return;
    }
    this.isSuggesting = true;
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
    if (!this.isDestroyed && prefix === this.keyword.trim()) {
      this.isSuggesting = false;
      // 无建议词时自动进入结果态（对齐 web 移动端搜索输入即出结果的节奏）；
      // 否则会出现只有 tab 行、结果区空白的中间态（2026-10-26 真机走查发现）。
      if (this.suggestItems.length === 0 && !this.hasSearched) {
        this.doSearch();
      }
    }
  }

  private async loadRecentSearches(): Promise<void> {
    try {
      const stored = await PreferencesUtil.getObject<string[]>(this.RECENT_SEARCHES_KEY);
      if (!this.isDestroyed && Array.isArray(stored)) {
        this.recentSearches = stored.filter((item: string) => item.trim().length > 0).slice(0, 5);
      }
    } catch (_) {
      // 读取失败保持空记录
    }
  }

  private async persistRecentSearches(): Promise<void> {
    try {
      await PreferencesUtil.setObject(this.RECENT_SEARCHES_KEY, this.recentSearches.slice(0, 5));
    } catch (_) {
      // 持久化失败静默
    }
  }

  private rememberSearch(keyword: string): void {
    const trimmed = keyword.trim();
    if (!trimmed) return;
    this.recentSearches = [trimmed, ...this.recentSearches.filter((item: string) => item !== trimmed)].slice(0, 5);
    this.persistRecentSearches();
  }

  private clearRecentSearches(): void {
    this.recentSearches = [];
    this.persistRecentSearches();
  }

  /** 空查询发现态：对齐前端 SearchDiscovery（空查询不调 suggest，前端以 prefix.length>=1 为阈值） */
  private async loadDiscovery(): Promise<void> {
    // 空 prefix 调 search/suggest 会 10000（spec prefix 1-50），故热门/话题取热门动态
    //（moments/search sort_by=popular，已有热门模式）的 tags 派生；取不到则留空，
    // 渲染层按空隐藏对应卡片（不留“暂无”死区块）
    try {
      const response = await ApiClient.searchMoments({ sort_by: 'popular', pagination: { limit: 20 } });
      if (this.isDestroyed || response.code !== 0 || !response.data) {
        return;
      }
      const tags: string[] = [];
      for (const moment of response.data.items ?? []) {
        for (const tag of moment.tags ?? []) {
          const trimmed = tag.trim();
          if (trimmed.length > 0 && trimmed.length <= 12 && tags.indexOf(trimmed) < 0) {
            tags.push(trimmed);
          }
          if (tags.length >= 10) {
            break;
          }
        }
        if (tags.length >= 10) {
          break;
        }
      }
      if (this.isDestroyed) {
        return;
      }

      this.discoveryTopics = tags;
    } catch (_) {
      // 取不到则不填充，卡片按空隐藏
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
    this.searchError = '';
    this.keyword = trimmedKeyword;
    this.suggestItems = [];
    this.rememberSearch(trimmedKeyword);
    this.userResults = [];
    this.userResultsDataSource.clear();
    this.groupResults = [];
    this.groupResultsDataSource.clear();
    this.momentResults = [];
    this.momentResultsDataSource.clear();
    this.topicResults = [];

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
        // spec moment.md → moments/search 收口：全球 keyword 搜索（无 reference_point）
        const momentsReq: SearchMomentsRequest = {
          keyword: trimmedKeyword,
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
          const topicSet = new Set<string>();
          this.momentResults.forEach((moment: MomentItem) => {
            (moment.tags ?? []).forEach((tag: string) => {
              if (tag.trim().length > 0) {
                topicSet.add(tag.trim());
              }
            });
          });
          this.topicResults = Array.from(topicSet);
        }

        // 三路全部业务失败：结果区转入错误态，避免误显示「未找到相关内容」空态
        if (usersResp.code !== 0 && groupsResp.code !== 0 && momentsResp.code !== 0) {
          this.searchError = getErrorMessage(momentsResp, getErrorMessage(usersResp, '搜索失败，请稍后重试'));
        }
      });
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `搜索失败: ${error}`);
      // 内联错误态承载失败反馈（可重试），不再叠加 toast 双重提示
      this.searchError = '搜索失败，请检查网络后重试';
    } finally {
      this.isLoading = false;
      if (this.hasSearched) {
        this.resetContentEnter();
        this.playContentEnter();
      }
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

  /** 搜索加载骨架：5 行「头像 + 两行文本条」，几何与消息列表 Skeleton 一致 */
  @Builder
  private buildSearchSkeleton() {
    Column() {
      SkeletonShimmer({ active: true }) {
        Column() {
          // N1（对齐 fe SearchResults）：结果分组头占位（eyebrow 条 + 辅助条），避免骨架→内容位移
          Row({ space: DesignConstants.SPACING.SM }) {
            Skeleton({ blockWidth: DesignConstants.SPACING.XXL, blockHeight: DesignConstants.FONT.SIZE.LG, cornerRadius: DesignConstants.RADIUS.SM })
            Skeleton({ blockWidth: DesignConstants.SPACING.XL, blockHeight: DesignConstants.FONT.SIZE.XS, cornerRadius: DesignConstants.RADIUS.SM })
          }
          .width('100%')
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.MD,
            bottom: DesignConstants.SPACING.SM
          })
          ForEach([0, 1, 2, 3, 4], (i: number) => {
            Row({ space: DesignConstants.SPACING.MD }) {
              Skeleton({ blockWidth: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL, blockHeight: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL, cornerRadius: DesignConstants.RADIUS.FULL })
              Column({ space: DesignConstants.SPACING.SM }) {
                Skeleton({ blockWidth: '55%', blockHeight: DesignConstants.FONT.SIZE.MD, cornerRadius: DesignConstants.RADIUS.SM })
                Skeleton({ blockWidth: '35%', blockHeight: DesignConstants.FONT.SIZE.XS, cornerRadius: DesignConstants.RADIUS.SM })
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
            }
            .width('100%')
            .padding({
              left: DesignConstants.SPACING.LG,
              right: DesignConstants.SPACING.LG,
              top: DesignConstants.SPACING.MD,
              bottom: DesignConstants.SPACING.MD
            })
          }, (i: number) => 'search-skeleton-' + i)
        }
        .width('100%')
      }
    }
    .width('100%')
    .layoutWeight(1)
  }

  private playContentEnter(): void {
    if (this.enterPlayed) return;
    this.enterPlayed = true;
    if (MotionUtil.reduced()) {
      this.enterOpacity = 1;
      this.enterOffset = 0;
      return;
    }
    this.enterOpacity = 0;
    this.enterOffset = MotionUtil.enterOffset();
    MotionUtil.contentEnter(this.getUIContext(), () => {
      this.enterOpacity = 1;
      this.enterOffset = 0;
    });
  }

  private resetContentEnter(): void {
    this.enterPlayed = false;
    this.enterOpacity = 1;
    this.enterOffset = 0;
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint }) {
      Column() {
        // Search header —— 统一走 SearchPageHeader（返回 + 内联搜索框）
        SearchPageHeader({
          title: '搜索',
          keyword: this.keyword,
          placeholder: '搜索用户、动态、地点或话题',
          onBack: () => { AppRouter.pop(); },
          onChange: (value: string) => {
            this.keyword = value;
            this.hasSearched = false;
            this.scheduleSuggest();
          },
          onSubmit: () => { this.doSearch(); },
          onFocusChange: (focused: boolean) => { this.isSearchFocused = focused; }
        })

        if (this.isLoading || (this.isSuggesting && !this.hasSearched)) {
          // 搜索/建议词加载态：复用全站 Skeleton 几何（头像块 + 两行文本条），
          // 与消息列表加载态同节奏；避免空白闪烁或退回转圈指示器。
          this.buildSearchSkeleton()
        } else if (this.keyword.trim().length === 0 && !this.hasSearched) {
          this.DiscoveryView()
        } else if (this.keyword.trim().length > 0 && this.suggestItems.length > 0 && !this.hasSearched) {
          List({ space: DesignConstants.SPACING.SM }) {
            ForEach(this.suggestItems, (item: SearchSuggestItem) => {
              ListItem() {
                SearchSuggestRow({
                  itemType: item.type,
                  title: this.getSuggestTitle(item),
                  subtitle: this.getSuggestSubtitle(item),
                  palette: this.getCurrentThemePalette(),
                  surfaceBackground: this.getSurfaceBackground(),
                  surfaceSubtle: this.getSurfaceSubtle(),
                  surfaceBorder: this.getSurfaceBorder(),
                  onPress: () => this.handleSuggestItemTap(item)
                })
              }
            }, (item: SearchSuggestItem, index?: number) => this.getSuggestKey(item, index ?? 0))
          }
          .scrollBar(BarState.Off)
          .divider({ strokeWidth: 0, color: this.getSurfaceBorder() })
          .layoutWeight(1)
          .width('100%')
          .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
        } else {
          Column() {
            if (this.searchError.length > 0) {
              // 搜索失败内联错误态（P2-4）：EmptyState + 重试动作，替代纯 toast
              EmptyState({
                bare: true,
                icon: AppIcon.WARNING,
                title: '搜索失败',
                message: this.searchError,
                actionLabel: '重试',
                onAction: () => { this.doSearch(); },
                fillMode: true
              })
                .width('100%')
                .layoutWeight(1)
            } else {
            TabsPillGroup({
              variant: 'pills',
              tabs: [
                { label: '全部' } as TabItem,
                { label: this.tabLabel('用户', this.userResults.length) } as TabItem,
                { label: this.tabLabel('动态', this.momentResults.length) } as TabItem,
                { label: this.tabLabel('群组', this.groupResults.length) } as TabItem,
                { label: this.tabLabel('话题', this.topicResults.length) } as TabItem
              ],
              activeIndex: this.currentIndex,
              actions: {
                onChange: (index: number) => {
                  this.currentIndex = index;
                }
              }
            })
              .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.MD })

            if (this.currentIndex === 0) {
              this.AllResultList()
            } else if (this.currentIndex === 1) {
              this.UserResultList()
            } else if (this.currentIndex === 2) {
              this.MomentResultList()
            } else if (this.currentIndex === 3) {
              this.TopicResultList()
            } else {
              this.GroupResultList()
            }
            }
          }
          .width('100%')
          .layoutWeight(1)
          .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
        .opacity(this.enterOpacity)
        .translate({ y: this.enterOffset })
        }
      }
      .width('100%')
      .height('100%')
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as SearchRouteParam | undefined;
      if (params && params.initialTab >= 0 && params.initialTab <= 4) {
        this.currentIndex = params.initialTab;
      }
      const initialKeyword = params?.keyword?.trim() ?? '';
      if (initialKeyword.length > 0) {
        this.keyword = initialKeyword;
        this.doSearch();
      }
    })
  }


  private tabLabel(base: string, count: number): string {
    return (this.hasSearched && count > 0) ? `${base} (${count})` : base;
  }

  @Builder
  private buildAllSectionLabel(label: string) {
    Row() {
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .textCase(TextCase.UpperCase)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.SM
    })
  }

  /**
   * 结果分组卡头（对齐前端 SearchResults section 头）：
   * `flex items-center gap-2 border-b border-border-default px-4 py-3`
   * + 图标 size-4 accent + 标题 text-sm(12) font-medium text-text-primary。
   * 前端「全部」结果是「每组一张圆角描边卡」，组内行不再各自成卡。
   */
  @Builder
  buildResultGroupHeader(icon: AppIcon, title: string) {
    Row({ space: DesignConstants.SPACING.SM2 }) {
      LucideIcon({
        icon: icon,
        iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
        paletteRole: 'accent'
      })
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getCurrentThemePalette().textPrimary)
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    // r3：页头无硬线（BRIEF B）
  }

  @Builder
  AllResultList() {
    // 对齐前端：hasResults = 任意一组非空（话题也算）
    if (this.hasSearched && this.userResults.length === 0 && this.momentResults.length === 0 &&
        this.groupResults.length === 0 && this.topicResults.length === 0) {
      EmptyState({
        spotlight: true,
        icon: AppIcon.SEARCH,
        title: `没有找到“${this.keyword.trim()}”`,
        message: '换个关键词再试试。',
        bare: true,
        fillMode: true
      })
    } else {
      List({ space: DesignConstants.SPACING.LG }) {
        if (this.userResults.length > 0) {
          ListItem() {
            Column() {
              this.buildResultGroupHeader(AppIcon.PERSON, '用户')
              ForEach(this.userResults, (item: UserItem, index: number) => {
                this.buildUserItem(item)
                // 仅行间分隔，末行不画（对齐前端 divide-y：末行底部无线，避免卡片内嵌一圈线）
                if (index < this.userResults.length - 1) {
                  Column().height(DesignConstants.SPACING.HALF)
                }
              }, (item: UserItem) => `all-user-${item.user?.user_id ?? ''}`)
            }
            .width('100%')
            // r3 软岛：发现卡去硬框、圆角 20、85% surface 底
            .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
            .borderRadius(DesignConstants.SHAPE.CARD)
            .shadow(this.getCurrentThemePalette().surfaceShadow)
            .clip(true)
          }
        }

        if (this.momentResults.length > 0) {
          ListItem() {
            Column() {
              this.buildResultGroupHeader(AppIcon.PHOTO, '动态')
              ForEach(this.momentResults, (moment: MomentItem, index: number) => {
                SearchMomentRow({
                  moment: moment,
                  palette: this.getCurrentThemePalette(),
                  surfaceBackground: this.getSurfaceBackground(),
                  surfaceBorder: this.getSurfaceBorder(),
                  framed: false,
                  onPress: () => this.openMomentDetail(moment)
                })
                if (index < this.momentResults.length - 1) {
                  Column().height(DesignConstants.SPACING.HALF)
                }
              }, (moment: MomentItem) => `all-moment-${moment.moment_id}`)
            }
            .width('100%')
            // r3 软岛：发现卡去硬框、圆角 20、85% surface 底
            .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
            .borderRadius(DesignConstants.SHAPE.CARD)
            .shadow(this.getCurrentThemePalette().surfaceShadow)
            .clip(true)
          }
        }

        if (this.groupResults.length > 0) {
          ListItem() {
            Column() {
              this.buildResultGroupHeader(AppIcon.GROUP, '群组')
              ForEach(this.groupResults, (group: GroupItem, index: number) => {
                this.buildGroupItem(group)
                if (index < this.groupResults.length - 1) {
                  Column().height(DesignConstants.SPACING.HALF)
                }
              }, (group: GroupItem) => `all-group-${group.group?.group_id ?? ''}`)
            }
            .width('100%')
            // r3 软岛：发现卡去硬框、圆角 20、85% surface 底
            .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
            .borderRadius(DesignConstants.SHAPE.CARD)
            .shadow(this.getCurrentThemePalette().surfaceShadow)
            .clip(true)
          }
        }

        if (this.topicResults.length > 0) {
          ListItem() {
            Column() {
              this.buildResultGroupHeader(AppIcon.SEARCH, '话题')
              ForEach(this.topicResults, (tag: string) => {
                Row({ space: DesignConstants.SPACING.MD }) {
                  Text(`#${tag}`)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                    .layoutWeight(1)
                    .maxLines(1)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                  LucideIcon({
                    icon: AppIcon.ARROW_UP,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
                    color: this.getCurrentThemePalette().textSecondary
                  })
                }
                .width('100%')
                .alignItems(VerticalAlign.Center)
                .padding({
                  left: DesignConstants.SPACING.LG,
                  right: DesignConstants.SPACING.LG,
                  top: DesignConstants.SPACING.LG,
                  bottom: DesignConstants.SPACING.LG
                })
                .onClick(() => { this.applyKeywordAndSearch(tag); })
              }, (tag: string) => `all-topic-${tag}`)
            }
            .width('100%')
            // r3 软岛：发现卡去硬框、圆角 20、85% surface 底
            .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
            .borderRadius(DesignConstants.SHAPE.CARD)
            .shadow(this.getCurrentThemePalette().surfaceShadow)
            .clip(true)
          }
        }
      }
      .divider({ strokeWidth: 0, color: this.getSurfaceBorder() })
      .width('100%').height('100%')
      .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
      .scrollBar(BarState.Off)
    }
  }

  @Builder
  TopicResultList() {
    if (this.hasSearched && this.topicResults.length === 0) {
      EmptyState({
        spotlight: true,
        icon: AppIcon.SEARCH,
        title: '未找到相关话题',
        message: '尝试其他关键词',
        bare: true,
        fillMode: true
      })
    } else {
      List() {
        ListItemGroup() {
        ForEach(this.topicResults, (tag: string) => {
          ListItem() {
            Pressable({ onPress: () => this.applyKeywordAndSearch(tag) }) {
              Row({ space: DesignConstants.SPACING.MD }) {
                Row() {
                  LucideIcon({
                      icon: AppIcon.CHART,
                      iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
                      color: this.getCurrentThemePalette().accentBackground
                    })
                }
                .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
                .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundColor(this.getSurfaceSubtle())
                .justifyContent(FlexAlign.Center)

                Text(`#${tag}`)
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .layoutWeight(1)

                LucideIcon({
                    icon: AppIcon.ARROW_RIGHT,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
                    color: this.getCurrentThemePalette().textSecondary
                  })
              }
              .width('100%')
              .padding(DesignConstants.SPACING.MD)
              // 行不再自带卡片样式：分组外框与行间分隔线由 ListItemGroup 承担
            }
          }
        }, (tag: string) => `topic-${tag}`)
        }
        .divider({
          strokeWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
          color: this.getSurfaceBorder(),
          startMargin: DesignConstants.SPACING.MD,
          endMargin: DesignConstants.SPACING.MD
        })
      }
      .divider({ strokeWidth: 0, color: this.getSurfaceBorder() })
      .width('100%').height('100%')
      .padding({ top: DesignConstants.PAGE_SHELL.PAGE_TOP_GAP, bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
      .scrollBar(BarState.Off)
    }
  }

  @Builder
  private buildDiscoveryCard(title: string, trailing?: () => void) {
    Row() {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .layoutWeight(1)
      if (trailing) {
        Text('清除')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getCurrentThemePalette().accentBackground)
          .padding(DesignConstants.SPACING.MD)
          .onClick(trailing)
      }
    }
    .width('100%')
    // 左右内边距与卡内正文（LG=16）对齐：页面 gutter 只管卡片外留白，管不到卡内标题
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.MD
    })
    // r3：sticky 分区头无硬线（BRIEF B）
  }

  @Builder
  DiscoveryView() {
    Scroll() {
      Column({ space: DesignConstants.SPACING.LG }) {
        // 最近搜索：**无记录时整卡不渲染**（对齐 web SearchDiscovery：空卡片只是占位噪音，
        // 端上实测会占掉约 1/6 屏却只有一行「还没有搜索记录。」——2026-10-21 走查）。
        if (this.recentSearches.length > 0) {
          Column() {
            this.buildDiscoveryCard('最近搜索', () => this.clearRecentSearches())
            ForEach(this.recentSearches, (item: string, index: number) => {
              if (index > 0) {
                Column().height(DesignConstants.SPACING.HALF)
              }
              Pressable({ onPress: () => this.applyKeywordAndSearch(item) }) {
                Row({ space: DesignConstants.SPACING.MD }) {
                  LucideIcon({
                    icon: AppIcon.TIME,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                    color: this.getCurrentThemePalette().textSecondary
                  })
                  Text(item)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                    .maxLines(1)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                    .layoutWeight(1)
                  LucideIcon({
                    icon: AppIcon.ARROW_RIGHT,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                    color: this.getCurrentThemePalette().textSecondary
                  })
                }
                .width('100%')
                .padding({
                  left: DesignConstants.SPACING.LG,
                  right: DesignConstants.SPACING.LG,
                  top: DesignConstants.SPACING.MD,
                  bottom: DesignConstants.SPACING.MD
                })
              }
            }, (item: string) => `recent-${item}`)
          }
          .width('100%')
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
          .shadow(this.getCurrentThemePalette().surfaceShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
        }

        // 热门话题（唯一列表）：此前还有一张「正在关注」编号卡，与这里同源重复，已删除
        // （2026-10-20 端上走查：两卡内容完全相同 #夜色/#新标签）。取不到热门 tags 时不渲染。
        if (this.discoveryTopics.length > 0) {
        Column() {
          this.buildDiscoveryCard('热门话题')
          Flex({ wrap: FlexWrap.Wrap, justifyContent: FlexAlign.Start }) {
            ForEach(this.discoveryTopics, (topic: string) => {
              Text(`#${topic}`)
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                .fontColor(this.getCurrentThemePalette().textPrimary)
                .backgroundColor(this.getSurfaceSubtle())
                .padding({
                  left: DesignConstants.SPACING.MD,
                  right: DesignConstants.SPACING.MD,
                  top: DesignConstants.SPACING.SM,
                  bottom: DesignConstants.SPACING.SM
                })
                .borderRadius(DesignConstants.RADIUS.FULL)
                .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
                .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                .responseRegion({
                  x: -DesignConstants.SPACING.SM2,
                  y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - 39) / 2,
                  width: '120%',
                  height: DesignConstants.LAYOUT.MIN_HIT_TARGET
                })
                .onClick(() => this.applyKeywordAndSearch(topic))
            }, (topic: string) => `topic-chip-${topic}`)
          }
          .width('100%')
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.MD,
            bottom: DesignConstants.SPACING.LG
          })
        }
        .width('100%')
        .backgroundColor(this.getSurfaceBackground())
        .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
        .shadow(this.getCurrentThemePalette().surfaceShadow)
        .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
        }
      }
      .width('100%')
      .padding({
        top: DesignConstants.SPACING.LG,
        bottom: DesignConstants.SPACING.LG
      })
    }
    .width('100%')
    .layoutWeight(1)
    .scrollBar(BarState.Off)
    .align(Alignment.Top)
    .edgeEffect(EdgeEffect.Spring)
  }

  @Builder
  UserResultList() {
    if (this.hasSearched && this.userResults.length === 0) {
      EmptyState({
        spotlight: true,
        icon: AppIcon.SEARCH,
        title: '未找到相关内容',
        message: '尝试其他关键词',
        bare: true,
        fillMode: true
      })
    } else {
      List() {
        ListItemGroup() {
          ForEach(this.userResults, (item: UserItem) => {
            ListItem() {
              this.buildUserItem(item)
            }
          }, (item: UserItem) => {
            const uid = item.user?.user_id ?? '';
            return uid + '_f' + (this.userStateMap[uid]?.relation?.is_following ?? false) + '_p' + (this.userActionPendingIds.indexOf(uid) >= 0);
          })
        }
        .divider({
          strokeWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
          color: this.getSurfaceBorder(),
          startMargin: DesignConstants.SPACING.MD,
          endMargin: DesignConstants.SPACING.MD
        })
      }
      .divider({ strokeWidth: 0, color: this.getSurfaceBorder() })
      .width('100%').height('100%')
      .padding({ top: DesignConstants.PAGE_SHELL.PAGE_TOP_GAP, bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
      .scrollBar(BarState.Off)
    }
  }

  @Builder
  GroupResultList() {
    if (this.hasSearched && this.groupResults.length === 0) {
      EmptyState({
        spotlight: true,
        icon: AppIcon.SEARCH,
        title: '未找到相关内容',
        message: '尝试其他关键词',
        bare: true,
        fillMode: true
      })
    } else {
      List() {
        ListItemGroup() {
          ForEach(this.groupResults, (item: GroupItem) => {
            ListItem() {
              this.buildGroupItem(item)
            }
          }, (item: GroupItem) => {
            const gid = item.group?.group_id ?? '';
            return gid + '_j' + (item.viewer_state != null) + '_p' + (this.groupActionPendingIds.indexOf(gid) >= 0);
          })
        }
        .divider({
          strokeWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
          color: this.getSurfaceBorder(),
          startMargin: DesignConstants.SPACING.MD,
          endMargin: DesignConstants.SPACING.MD
        })
      }
      .width('100%').height('100%')
      .padding({ top: DesignConstants.PAGE_SHELL.PAGE_TOP_GAP, bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
      .scrollBar(BarState.Off)
    }
  }

  @Builder
  MomentResultList() {
    if (this.hasSearched && this.momentResults.length === 0) {
      EmptyState({
        spotlight: true,
        icon: AppIcon.SEARCH,
        title: '未找到相关内容',
        message: '尝试其他关键词',
        bare: true,
        fillMode: true
      })
    } else {
      WaterFlow() {
        LazyForEach(this.momentResultsDataSource, (item: MomentItem) => {
          FlowItem() {
            this.buildMomentSearchCard(item)
          }
          .width('100%')
        }, (item: MomentItem) => item.moment_id)
      }
      .scrollBar(BarState.Off)
      .width('100%').height('100%')
      // 桌面 3 列瀑布流（与 MomentsFeedPage getWallColumns 契约对齐），
      // 平板/手机 2 列紧凑保留
      .columnsTemplate(isDesktopForm(this.currentBreakpoint) ? '1fr 1fr 1fr' : '1fr 1fr')
      .columnsGap(DesignConstants.SPACING.MD)
      .rowsGap(DesignConstants.SPACING.MD)
      .padding({ top: DesignConstants.PAGE_SHELL.PAGE_TOP_GAP, bottom: DesignConstants.PAGE_SHELL.SECTION_GAP })
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
    return group.viewer_state != null;
  }

  /**
   * 搜索结果用户行：对齐 web 移动端 SearchResults 的紧凑列表行——
   * 头像 40 + 昵称(MD) + 副标(@user_id · 首个标签, SM) + 尾部导航箭头，整行点击进资料页。
   * web 同样不在搜索行放「关注/私聊」次级按钮（动作在资料页完成），
   * 此前本端行内两枚整宽按钮 + 简介 + 标签把单行撑到约 180vp，结果列表密度与 web 明显不一致。
   * 平台差异：web 尾部为 ArrowUpRight 外链语义，本端系统符号库无对应项，用全站列表通用的
   * chevron_right 箭头（结构/字号/间距同构，仅图标语义按平台库取近档）。
   */
  @Builder
  buildUserItem(item: UserItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        decorative: true,
        avatarUrl: item.user?.avatar_url ?? '',
        name: item.user?.nickname ?? '',
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
        shape: 'circle'
      })

      Column({ space: DesignConstants.SPACING.HALF }) {
        Text(item.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Text(this.buildUserSubtitle(item))
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      LucideIcon({
        icon: AppIcon.ARROW_RIGHT,
        iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
        color: this.getCurrentThemePalette().textSecondary
      })
    }
    .width('100%')
    // 行内边距对齐 web ListItem px-4 py-3.5（16 / 14），行高约 68vp
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD3,
      bottom: DesignConstants.SPACING.MD3
    })
    .onClick(() => {
      this.openUserProfile(item);
    })
  }

  /** 用户结果副标：对齐 web 的「@user_id · 首个标签」 */
  private buildUserSubtitle(item: UserItem): string {
    const userId = item.user?.user_id ?? '';
    const tag = item.tags && item.tags.length > 0 ? item.tags[0] : '';
    return tag !== '' ? ('@' + userId + ' · ' + tag) : ('@' + userId);
  }

  @Builder
  buildGroupItem(item: GroupItem) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: DesignConstants.SPACING.SM }) {
        Row({ space: DesignConstants.SPACING.SM }) {
          SystemAvatar({
            decorative: true,
            avatarUrl: item.group?.avatar_url ?? '',
            name: item.group?.name ?? '',
            avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
            shape: 'circle'
          })

          Column({ space: DesignConstants.SPACING.HALF }) {
            Text(item.group?.name ?? '')
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getCurrentThemePalette().textPrimary)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
            Text(`${formatCount(item.group?.members_count)} 人`)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
          .alignItems(HorizontalAlign.Start)
          .layoutWeight(1)
        }
        .layoutWeight(1)

        Row() {
          LucideIcon({
              icon: AppIcon.MORE,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
              color: this.getCurrentThemePalette().textSecondary
            })
        }
        .width(DesignConstants.COMPONENT.ICON_SIZE_MD)
        .height(DesignConstants.COMPONENT.ICON_SIZE_MD)
        .borderRadius(DesignConstants.RADIUS.LG)
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
        SystemButton({
          label: this.isGroupJoined(item) ? '进入群聊' : '加入群组',
          variant: this.isGroupJoined(item) ? 'outline' : 'primary',
          buttonSize: 'small',
          loading: this.isGroupActionPending(item.group?.group_id ?? ''),
          onPress: () => { this.joinGroup(item); }
        })
          .layoutWeight(1)
          .scale({ x: MotionUtil.popScale(this.joinPoppedGroupId === (item.group?.group_id ?? '')), y: MotionUtil.popScale(this.joinPoppedGroupId === (item.group?.group_id ?? '')) })

        SystemButton({
          label: '详情',
          variant: 'surface',
          buttonSize: 'small',
          shadowed: true,
          onPress: () => { this.openGroupDetail(item); }
        })
          .layoutWeight(1)
          .opacity(this.isGroupActionPending(item.group?.group_id ?? '') ? DesignConstants.OPACITY.PRESSED : 1)
      }
      .width('100%')
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    // 同 buildUserItem：行级卡片样式移除
  }

  @Builder
  buildMomentSearchCard(item: MomentItem) {
    Column({ space: 0 }) {
      if (item.images && item.images.length > 0) {
        // R8：占位 icon 常驻底层，图片 fade，失败保留占位
        Stack() {
          Column() {
            LucideIcon({
              icon: AppIcon.IMAGES,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
              color: this.getCurrentThemePalette().textTertiary
            })
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.getSurfaceSubtle())
          if (!this.isImageFailed(item.images[0].url)) {
            Image(ImageUrlUtil.optimized(MediaUrlUtil.normalize(item.images[0].url), 320))
              .width('100%')
              .height('100%')
              .objectFit(ImageFit.Cover)
              .opacity(this.isImageLoaded(item.images[0].url) || MotionUtil.reduced() ? 1 : 0)
              .animation({
                duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_SHORT,
                curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
              })
              .onComplete(() => this.markImageLoaded(item.images[0].url))
              .onError(() => this.markImageFailed(item.images[0].url))
          }
        }
        .width('100%')
        .height(this.calculateSearchImageHeight(item))
      } else {
        Column() {
          Text(item.description.substring(0, 60))
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
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
          decorative: true,
          avatarUrl: item.user?.avatar_url ?? '',
          name: item.user?.nickname ?? '',
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL
        })

        Text(item.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .layoutWeight(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row({ space: DesignConstants.SPACING.HALF }) {
          LucideIcon({
              icon: AppIcon.HEART,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              color: this.getCurrentThemePalette().textTertiary
            })
          Text(formatCount(sumReactionCounts(item.reactions)))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getCurrentThemePalette().textTertiary)
        }
      }
      .padding(DesignConstants.SPACING.SM)
      .width('100%')
    }
    // r3 软岛：结果卡去硬框、圆角 20、85% surface 底
    .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .clip(true)
    .shadow(this.getCurrentThemePalette().surfaceShadow)
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
    const wasFollowing: boolean = this.isUserFollowing(userId);
    this.setUserActionPending(userId, true);
    try {
      const success = await interactionActions.setFollowUser(userId, !wasFollowing, '关注用户');
      if (success && !wasFollowing) {
        MotionUtil.statePop(this.getUIContext(),
          () => { this.followPoppedUserId = userId; },
          () => { this.followPoppedUserId = ''; });
      }
      if (success) {
        this.patchUserRelation(userId, !wasFollowing);
        SystemToast.showSuccess(wasFollowing ? '已取消关注' : '已关注', this.getUIContext());
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
    if (group.viewer_state != null) {
      this.enterGroupChat(group);
      return;
    }
    if (this.isGroupActionPending(groupId)) return;
    this.setGroupActionPending(groupId, true);
    try {
      const response = await ApiClient.joinGroup({ group_id: groupId } as EnterGroupRequest);
      if (response.code === 0 && response.data) {
        if (response.data.status === 'joined') {
          SystemToast.showSuccess('加入成功', this.getUIContext());
          await this.doSearch();
        } else {
          SystemToast.show('已提交申请，等待审批', this.getUIContext());
        }
      } else if (response.code === 10400) {
        SystemToast.show(getErrorMessage(response, '加入失败'), this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '加入失败'), this.getUIContext());
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
        SystemToast.showError(getErrorMessage(response, '举报失败'), this.getUIContext());
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
        SystemToast.showError(getErrorMessage(response, '举报失败'), this.getUIContext());
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