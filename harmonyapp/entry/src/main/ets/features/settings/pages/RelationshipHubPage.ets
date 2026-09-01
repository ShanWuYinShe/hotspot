import { AppRouter } from '../../../core/router/AppRouter';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { TabsPillGroup } from '../../../components/TabsPillGroup';
import { SystemInput } from '../../../components/SystemInput';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemToast } from '../../../components/SystemToast';
import { ApiClient } from '../../../core/api/ApiClient';
import { BlockedUserItem, RelatedUserItem } from '../../../core/types/api';
import { RelationshipListSection } from './SettingsSupport';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

@Preview
@Component
export struct RelationshipHubPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State friends: RelatedUserItem[] = [];
  @State following: RelatedUserItem[] = [];
  @State followers: RelatedUserItem[] = [];
  @State blocked: BlockedUserItem[] = [];
  @State activeRelTab: number = 0;
  @State searchKeyword: string = '';
  @State onlineFilterIndex: number = 0; // 0=全部 1=在线 2=离线（对齐前端 SectionFilter）

  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.loadAll();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  /** 当前激活 tab 的计数（对齐前端 RelationshipsPage activeCount：好友/关注/粉丝 当前区人数） */
  private getActiveTabCount(): number {
    if (this.activeRelTab === 0) {
      return this.friends.length;
    }
    if (this.activeRelTab === 1) {
      return this.following.length;
    }
    return this.followers.length;
  }

  /** 在线/离线过滤 + 昵称/备注关键词过滤（对齐前端 SectionFilter + SearchInput 客户端过滤） */
  private filterUsers(items: RelatedUserItem[]): RelatedUserItem[] {
    let result: RelatedUserItem[] = items;
    if (this.onlineFilterIndex === 1) {
      result = result.filter((i: RelatedUserItem) => i.user.presence_status === 'online');
    } else if (this.onlineFilterIndex === 2) {
      result = result.filter((i: RelatedUserItem) => i.user.presence_status !== 'online');
    }
    const kw = this.searchKeyword.trim().toLowerCase();
    if (kw.length > 0) {
      result = result.filter((i: RelatedUserItem) => {
        const name = (i.user.user.nickname ?? '').toLowerCase();
        const remark = (i.remark ?? '').toLowerCase();
        return name.includes(kw) || remark.includes(kw);
      });
    }
    return result;
  }

  private async loadAll(): Promise<void> {
    if (this.isDestroyed) return;
    try {
      const results = await Promise.all([
        ApiClient.getFriendsList({ pagination: { limit: 20 } }),
        ApiClient.getFollowingList({ pagination: { limit: 20 } }),
        ApiClient.getFollowersList({ pagination: { limit: 20 } }),
        ApiClient.getBlockedList({ pagination: { limit: 20 } })
      ]);
      if (this.isDestroyed) return;
      this.friends = results[0].data?.items ?? [];
      this.following = results[1].data?.items ?? [];
      this.followers = results[2].data?.items ?? [];
      this.blocked = results[3].data?.items ?? [];
    } catch (_) {
      SystemToast.showError('加载关系数据失败', this.getUIContext());
    }
  }

  private async unblock(userId: string): Promise<void> {
    if (this.isDestroyed) return;
    try {
      const response = await ApiClient.unblockUser({ user_id: userId });
      if (this.isDestroyed) return;
      if (response.code === 0) {
        SystemToast.showSuccess('已解除拉黑', this.getUIContext());
        this.loadAll();
      } else {
        SystemToast.showError(getErrorMessage(response, '解除失败'), this.getUIContext());
      }
    } catch (_) {
      SystemToast.showError('解除失败', this.getUIContext());
    }
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
      Column() {
        // 头部 —— 标题 24pt + 计数 trailing（对齐前端 RelationshipsPage：
        // titleSize 2xl + trailing「N 人」当前区计数）
        SystemPageHeader({
          title: '关系',
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        }) {
          if (this.getActiveTabCount() > 0) {
            Text(`${this.getActiveTabCount()} 人`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getPalette().textSecondary)
          }
        }

        // 关系三 Tab —— below 吸顶（对齐前端 below TabBar；count 角标对齐 TabBar count 契约）
        TabsPillGroup({
          tabs: [
            { label: '好友', count: this.friends.length },
            { label: '关注', count: this.following.length },
            { label: '粉丝', count: this.followers.length }
          ],
          activeIndex: this.activeRelTab,
          actions: {
            onChange: (index: number) => {
              this.activeRelTab = index;
            }
          }
        })
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })

        // 搜索框 + 在线过滤（对齐前端 Section 内嵌：search + filters 位于内容区顶部）
        SystemInput({
          value: this.searchKeyword,
          placeholder: '搜索昵称、备注…',
          prefixIcon: AppIcon.SEARCH,
          showPrefixIcon: true,
          inputHeight: DesignConstants.COMPONENT.INPUT_HEIGHT,
          onChange: (value: string) => {
            this.searchKeyword = value;
          }
        })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })

        TabsPillGroup({
          tabs: [
            { label: '全部' },
            { label: '在线' },
            { label: '离线' }
          ],
          activeIndex: this.onlineFilterIndex,
          actions: {
            onChange: (index: number) => {
              this.onlineFilterIndex = index;
            }
          }
        })
          .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.SM })

        Scroll() {
          Column() {
            if (this.activeRelTab === 0) {
              RelationshipListSection({
                embedded: true,
                emptyText: this.onlineFilterIndex === 0 && this.searchKeyword.length === 0 ? '暂时还没有好友。' : '没有符合条件的好友。',
                friends: this.filterUsers(this.friends)
              })
            } else if (this.activeRelTab === 1) {
              RelationshipListSection({
                embedded: true,
                emptyText: this.onlineFilterIndex === 0 && this.searchKeyword.length === 0 ? '还没有关注任何人。' : '没有符合条件的关注。',
                following: this.filterUsers(this.following)
              })
            } else {
              RelationshipListSection({
                embedded: true,
                emptyText: this.onlineFilterIndex === 0 && this.searchKeyword.length === 0 ? '还没有新的粉丝。' : '没有符合条件的粉丝。',
                followers: this.filterUsers(this.followers)
              })
            }
          }
          .width('100%')
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.XS,
            bottom: SafeAreaUtils.resolveBottomPadding(0, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG)
          })
        }
        .layoutWeight(1)
        .width('100%')
      }
      .width('100%')
      .height('100%')
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }
}

@Builder
export function RelationshipHubPageBuilder() {
  RelationshipHubPage()
}