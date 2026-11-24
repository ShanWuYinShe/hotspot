import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemInput } from '../../../components/SystemInput';
import { SystemToast } from '../../../components/SystemToast';
import { EmptyState } from '../../../components/EmptyState';
import { LoadingMoreIndicator } from '../../../components/EmptyState';
import { ListEnd } from '../../../components/ListEnd';
import { GroupMemberRow } from './GroupMemberRow';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { GroupMemberItem } from '../../../core/types/api';
import { ApiClient } from '../../../core/api/ApiClient';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

const SEARCH_PAGE_SIZE: number = 50;

/**
 * 群成员列表面板（从 GroupDetailPage.buildMemberListSheet 提取的子域组件）。
 *
 * 状态边界：列表数据/加载态经 @Prop 单向下发（页面增删成员后新数组触发刷新），
 * 数据源对象为页面持有实例的引用传递（LazyForEach 监听随组件注册）；
 * 行内菜单/角色文案等依赖页面权限与服务的派生逻辑经函数 props 回到页面侧。
 *
 * 搜索：走 groups/members/search 契约（昵称关键词 + 分页），面板内自管理搜索状态，
 * 不影响页面持有的成员分页数据；清空关键词恢复页面下发的成员列表。
 */
@Component
export struct GroupMemberListSheet {
  @Prop groupId: string = '';
  @Prop members: GroupMemberItem[] = [];
  @Prop isLoadingMembers: boolean = false;
  @Prop membersErrorMessage: string = '';
  @Prop membersLoadingMore: boolean = false;
  @Prop membersHasMore: boolean = false;
  @Prop safeAreaBottom: number = 0;
  membersDataSource: BasicDataSource<GroupMemberItem> = new BasicDataSource<GroupMemberItem>([]);
  onRetryLoad: () => void = () => {};
  onLoadMore: () => void = () => {};
  roleTextFor: (member: GroupMemberItem) => string = () => '';
  roleColorFor: (member: GroupMemberItem) => ResourceColor = () => '';
  showMenuFor: (member: GroupMemberItem) => boolean = () => false;
  menuItemsFor: (member: GroupMemberItem) => MenuElement[] = () => [];

  @State searchKeyword: string = '';
  @State searchResults: GroupMemberItem[] = [];
  @State searchLoading: boolean = false;
  @State searchError: string = '';
  @State searchLoadingMore: boolean = false;
  @State searchHasMore: boolean = false;
  searchDataSource: BasicDataSource<GroupMemberItem> = new BasicDataSource<GroupMemberItem>([]);
  private searchNextCursor: string = '';
  private searchTimer: number = -1;
  private searchSeq: number = 0;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private onSearchChange(value: string): void {
    this.searchKeyword = value;
    if (this.searchTimer >= 0) {
      clearTimeout(this.searchTimer);
      this.searchTimer = -1;
    }
    const keyword = value.trim();
    if (!keyword) {
      this.searchSeq++;
      this.searchResults = [];
      this.searchDataSource.setData([]);
      this.searchError = '';
      this.searchLoading = false;
      this.searchHasMore = false;
      this.searchNextCursor = '';
      return;
    }
    this.searchTimer = setTimeout(() => {
      this.searchTimer = -1;
      void this.runSearch(keyword);
    }, 350);
  }

  private async runSearch(keyword: string): Promise<void> {
    if (!this.groupId) return;
    const seq = ++this.searchSeq;
    this.searchLoading = true;
    this.searchError = '';
    try {
      const response = await ApiClient.searchGroupMembers({
        group_id: this.groupId,
        keyword: keyword,
        pagination: { limit: SEARCH_PAGE_SIZE }
      });
      if (seq !== this.searchSeq) return;
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.searchResults = items;
        this.searchDataSource.setData(items);
        this.searchNextCursor = response.data?.next_cursor ?? '';
        this.searchHasMore = !!response.data?.next_cursor;
      } else {
        this.searchError = getErrorMessage(response, '搜索成员失败');
        this.searchResults = [];
        this.searchDataSource.setData([]);
      }
    } catch (error) {
      if (seq !== this.searchSeq) return;
      this.searchError = error instanceof Error ? error.message : '搜索成员失败';
      this.searchResults = [];
      this.searchDataSource.setData([]);
    } finally {
      if (seq === this.searchSeq) {
        this.searchLoading = false;
      }
    }
  }

  private async loadMoreSearch(): Promise<void> {
    if (this.searchLoadingMore || !this.searchHasMore || !this.searchKeyword.trim() || !this.groupId) {
      return;
    }
    this.searchLoadingMore = true;
    try {
      const response = await ApiClient.searchGroupMembers({
        group_id: this.groupId,
        keyword: this.searchKeyword.trim(),
        pagination: { limit: SEARCH_PAGE_SIZE, cursor: this.searchNextCursor }
      });
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        if (items.length > 0) {
          this.searchResults = this.searchResults.concat(items);
          this.searchDataSource.appendData(items);
        }
        this.searchNextCursor = response.data?.next_cursor ?? '';
        this.searchHasMore = !!response.data?.next_cursor;
      }
    } catch (_) {
      SystemToast.showError('加载更多成员失败', this.getUIContext());
    } finally {
      this.searchLoadingMore = false;
    }
  }

  @Builder
  private buildMemberRow(member: GroupMemberItem) {
    GroupMemberRow({
      member: member,
      roleText: this.roleTextFor(member),
      roleColor: this.roleColorFor(member),
      // 审计 P3-5：showMenu 判定走轻量布尔方法，不在渲染期构造完整菜单数组
      showMenu: this.showMenuFor(member),
      palette: this.getCurrentThemePalette(),
      surfaceBackground: this.getSurfaceBackground(),
      menuItems: this.menuItemsFor(member)
    })
  }

  @Builder
  private buildSearchList() {
    List() {
      LazyForEach(this.searchDataSource, (member: GroupMemberItem) => {
        ListItem() {
          this.buildMemberRow(member)
        }
      }, (member: GroupMemberItem) => member.user?.user_id ?? member.joined_at ?? '')
      if (this.searchLoadingMore) {
        ListItem() {
          LoadingMoreIndicator()
        }
      } else if (!this.searchHasMore && this.searchResults.length > 0) {
        ListItem() {
          ListEnd()
        }
      }
    }
    .width('100%')
    .layoutWeight(1)
    .scrollBar(BarState.Off)
    .cachedCount(5)
    .onReachEnd(() => { void this.loadMoreSearch(); })
  }

  build() {
    Column() {
      // 标题
      Row() {
        Text('群组成员')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text(this.searchKeyword.trim() ? `${this.searchResults.length} 条匹配` : `${this.members.length} 人`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      // 搜索框（大群快速定位成员；搜索走服务端 members/search，分页自管理）
      if (this.groupId) {
        SystemInput({
          value: this.searchKeyword,
          placeholder: '搜索成员昵称…',
          prefixIcon: AppIcon.SEARCH,
          showPrefixIcon: true,
          showClearButton: true,
          inputHeight: DesignConstants.COMPONENT.SEARCH_INPUT_HEIGHT,
          onChange: (value: string) => { this.onSearchChange(value); }
        })
          .margin({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            bottom: DesignConstants.SPACING.MD
          })
      }

      if (this.searchKeyword.trim()) {
        if (this.searchLoading) {
          SystemLoadingView({ message: '正在搜索成员…' }).layoutWeight(1)
        } else if (this.searchError) {
          EmptyState({
            bare: true,
            icon: AppIcon.WARNING,
            title: this.searchError,
            actionLabel: '重试',
            fillMode: true,
            onAction: () => { void this.runSearch(this.searchKeyword.trim()); }
          })
        } else if (this.searchResults.length === 0) {
          EmptyState({
            icon: AppIcon.SEARCH,
            title: '没有找到成员',
            message: '换个昵称关键词试试，或清空搜索词。',
            bare: true,
            fillMode: true
          })
        } else {
          this.buildSearchList()
        }
      } else if (this.isLoadingMembers && this.members.length === 0) {
        SystemLoadingView({ message: '加载成员列表中…' }).layoutWeight(1)
      } else if (this.members.length === 0 && this.membersErrorMessage) {
        EmptyState({
          bare: true,
          icon: AppIcon.WARNING,
          title: this.membersErrorMessage,
          actionLabel: '重试',
          fillMode: true,
          onAction: () => { this.onRetryLoad(); }
        })
      } else if (this.members.length > 0) {
        List() {
          LazyForEach(this.membersDataSource, (member: GroupMemberItem) => {
            ListItem() {
              this.buildMemberRow(member)
            }
          }, (member: GroupMemberItem) => member.user?.user_id ?? member.joined_at ?? '')
          // 分页状态条：加载中原语 + 到底 ListEnd（原为手写 LoadingProgress + 「加载更多…」，
          // 样式与其它页不一致；四态契约要求「到底」有明确收尾）
          if (this.membersLoadingMore) {
            ListItem() {
              LoadingMoreIndicator()
            }
          } else if (!this.membersHasMore && this.members.length > 0) {
            ListItem() {
              ListEnd()
            }
          }
        }
        .width('100%')
        .layoutWeight(1)
        // 横向留白由外层 PageShellContainer 统一提供，这里不再叠加
        .scrollBar(BarState.Off)
        .cachedCount(5)
        .onReachEnd(() => { this.onLoadMore(); })
      } else {
        EmptyState({
          icon: AppIcon.EMPTY,
          title: '暂无成员信息',
          bare: true,
          fillMode: true
        })
      }
    }
    .padding({ bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0) })
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }
}