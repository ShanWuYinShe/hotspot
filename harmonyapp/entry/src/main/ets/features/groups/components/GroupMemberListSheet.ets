import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { EmptyState } from '../../../components/EmptyState';
import { GroupMemberRow } from './GroupMemberRow';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { GroupMemberItem } from '../../../core/types/api';

/**
 * 群成员列表面板（从 GroupDetailPage.buildMemberListSheet 提取的子域组件）。
 *
 * 状态边界：列表数据/加载态经 @Prop 单向下发（页面增删成员后新数组触发刷新），
 * 数据源对象为页面持有实例的引用传递（LazyForEach 监听随组件注册）；
 * 行内菜单/角色文案等依赖页面权限与服务的派生逻辑经函数 props 回到页面侧。
 */
@Component
export struct GroupMemberListSheet {
  @Prop members: GroupMemberItem[] = [];
  @Prop isLoadingMembers: boolean = false;
  @Prop membersErrorMessage: string = '';
  @Prop membersLoadingMore: boolean = false;
  @Prop safeAreaBottom: number = 0;
  membersDataSource: BasicDataSource<GroupMemberItem> = new BasicDataSource<GroupMemberItem>([]);
  onRetryLoad: () => void = () => {};
  onLoadMore: () => void = () => {};
  roleTextFor: (member: GroupMemberItem) => string = () => '';
  roleColorFor: (member: GroupMemberItem) => ResourceColor = () => '';
  showMenuFor: (member: GroupMemberItem) => boolean = () => false;
  menuItemsFor: (member: GroupMemberItem) => MenuElement[] = () => [];

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
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

        Text(`${this.members.length} 人`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      // 成员列表
      if (this.isLoadingMembers && this.members.length === 0) {
        SystemLoadingView({ message: '加载成员列表中…' }).layoutWeight(1)
      } else if (this.members.length === 0 && this.membersErrorMessage) {
        EmptyState({
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
          }, (member: GroupMemberItem) => member.user?.user_id ?? member.joined_at ?? '')
          if (this.membersLoadingMore) {
            ListItem() {
              Row({ space: DesignConstants.SPACING.SM }) {
                LoadingProgress().width(DesignConstants.COMPONENT.LOADING_ICON_SIZE).height(DesignConstants.COMPONENT.LOADING_ICON_SIZE)
                Text('加载更多…')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .width('100%')
              .justifyContent(FlexAlign.Center)
              .padding(DesignConstants.SPACING.MD)
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
