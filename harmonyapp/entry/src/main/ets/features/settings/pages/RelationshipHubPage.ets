import { AppRouter } from '../../../core/router/AppRouter';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { ApiClient } from '../../../core/api/ApiClient';
import { BlockedUserItem, FollowerItem, FollowingItem, FriendItem } from '../../../core/types/api';
import { RelationshipListSection } from './SettingsSupport';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { curves } from '@kit.ArkUI';

@Preview
@Component
export struct RelationshipHubPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State friends: FriendItem[] = [];
  @State following: FollowingItem[] = [];
  @State followers: FollowerItem[] = [];
  @State blocked: BlockedUserItem[] = [];
  @State activeRelTab: number = 0;

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
        SystemToast.showError(response.message || '解除失败', this.getUIContext());
      }
    } catch (_) {
      SystemToast.showError('解除失败', this.getUIContext());
    }
  }

  @Builder
  private RelTab(index: number, label: string, count: number) {
    Text(count > 0 ? `${label} (${count})` : label)
      .fontColor(this.activeRelTab === index ? this.getPalette().accentText : this.getPalette().textSecondary)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontWeight(FontWeight.Medium)
      .padding({ left: 14, right: 14, top: 6, bottom: 6 })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.activeRelTab === index ? this.getPalette().accentBackground : this.getPalette().surfaceBackground)
      .border(this.activeRelTab === index
        ? { width: 0 }
        : { width: 1, color: this.getPalette().borderColor })
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_TAB,
          DesignConstants.INTERACTION.SPRING_DAMPING_TAB
        )
      })
      .onClick(() => { this.activeRelTab = index; })
  }

  build() {
    NavDestination() {
    ResponsiveContainer() {
      Column() {
        SystemPageHeader({
          title: '关系',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

        Scroll() {
          Row({ space: DesignConstants.SPACING.SM }) {
            this.RelTab(0, '好友', this.friends.length)
            this.RelTab(1, '关注', this.following.length)
            this.RelTab(2, '粉丝', this.followers.length)
            this.RelTab(3, '黑名单', this.blocked.length)
          }
        }
        .scrollable(ScrollDirection.Horizontal)
        .scrollBar(BarState.Off)
        .width('100%')
        .align(Alignment.Start)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.MD })

        Scroll() {
          Column() {
            if (this.activeRelTab === 0) {
              RelationshipListSection({
                title: '互相关注',
                subtitle: '你们已经成为彼此的好友。',
                emptyText: '暂时还没有好友。',
                friends: this.friends
              })
            } else if (this.activeRelTab === 1) {
              RelationshipListSection({
                title: '我的关注',
                subtitle: '你主动关注的人。',
                emptyText: '还没有关注任何人。',
                following: this.following
              })
            } else if (this.activeRelTab === 2) {
              RelationshipListSection({
                title: '我的粉丝',
                subtitle: '关注你的人。',
                emptyText: '还没有新的粉丝。',
                followers: this.followers
              })
            } else {
              RelationshipListSection({
                title: '已拉黑用户',
                subtitle: '你已屏蔽的用户会在这里出现。',
                emptyText: '当前没有拉黑任何用户。',
                blocked: this.blocked,
                unblockUserAction: (userId: string) => this.unblock(userId)
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
      .backgroundColor(this.getPalette().pageBackground)
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
