import { AppRouter } from '../../../core/router/AppRouter';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { ApiClient } from '../../../core/api/ApiClient';
import { BlockedUserItem, FollowerItem, FollowingItem, FriendItem } from '../../../core/types/api';
import { RelationshipListSection, SettingsSectionCard } from './SettingsSupport';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

@Preview
@Component
export struct RelationshipHubPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State friends: FriendItem[] = [];
  @State following: FollowingItem[] = [];
  @State followers: FollowerItem[] = [];
  @State blocked: BlockedUserItem[] = [];

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.loadAll();
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private async loadAll(): Promise<void> {
    try {
      const results = await Promise.all([
        ApiClient.getFriendsList({ pagination: { limit: 20 } }),
        ApiClient.getFollowingList({ pagination: { limit: 20 } }),
        ApiClient.getFollowersList({ pagination: { limit: 20 } }),
        ApiClient.getBlockedList({ pagination: { limit: 20 } })
      ]);
      this.friends = results[0].data?.items ?? [];
      this.following = results[1].data?.items ?? [];
      this.followers = results[2].data?.items ?? [];
      this.blocked = results[3].data?.items ?? [];
    } catch (_) {
      SystemToast.showError('加载关系数据失败', this.getUIContext());
    }
  }

  private async unblock(userId: string): Promise<void> {
    try {
      const response = await ApiClient.unblockUser({ user_id: userId });
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

  build() {
    NavDestination() {
      Column() {
        SystemPageHeader({
          title: '关系与黑名单',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

        Scroll() {
          Column({ space: 0 }) {
            Column() {
              SettingsSectionCard({ title: '好友' }) {
                RelationshipListSection({
                  title: '互相关注',
                  subtitle: '你们已经成为彼此的好友。',
                  emptyText: '暂时还没有好友。',
                  friends: this.friends
                })
              }
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })

            Column() {
              SettingsSectionCard({ title: '关注与粉丝' }) {
                Column({ space: DesignConstants.SPACING.LG }) {
                  RelationshipListSection({
                    title: '我的关注',
                    subtitle: '你主动关注的人。',
                    emptyText: '还没有关注任何人。',
                    following: this.following
                  })
                  RelationshipListSection({
                    title: '我的粉丝',
                    subtitle: '关注你的人。',
                    emptyText: '还没有新的粉丝。',
                    followers: this.followers
                  })
                }
                .width('100%')
              }
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })

            Column() {
              SettingsSectionCard({ title: '黑名单' }) {
                RelationshipListSection({
                  title: '已拉黑用户',
                  subtitle: '你已屏蔽的用户会在这里出现。',
                  emptyText: '当前没有拉黑任何用户。',
                  blocked: this.blocked,
                  unblockUserAction: (userId: string) => this.unblock(userId)
                })
              }
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
          }
          .width('100%')
          .padding({ bottom: SafeAreaUtils.resolveBottomPadding(0, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG) })
        }
        .layoutWeight(1)
        .width('100%')
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getPalette().pageBackground)
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
