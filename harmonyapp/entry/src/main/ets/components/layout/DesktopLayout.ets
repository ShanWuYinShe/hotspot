/**
 * DesktopLayout — 鸿蒙桌面 / 折叠屏 / PC 布局（对齐前端 app-shell 桌面分支）
 *
 * 适用断点：lg / xl / xxl（>= 840vp）
 *
 * 结构（对齐前端桌面）：
 * - 顶栏（52vp）：Logo + 搜索 / 发布(brand 圆) / 设置 / 头像(或登录)，跨全宽
 * - 侧边栏（192vp）：发现组 + 个人中心组 + 底部 AnchorEntry（浏览锚点卡片）
 * - main card：内容区（页面自带 header，本布局不再叠加顶部应用栏）
 *
 * 导航：核心 tab（首页 / 发现 / 消息）切换 currentRoute；其余项 push 或打开通知中心 overlay。
 * sub-page（设置 / 通知 / 我的瞬间 / 我的群组 / 关系 等）push 全屏、自带返回。
 */

import { type BreakpointType } from '../../core/responsive/Breakpoint';
import { DesignConstants } from '../../constants/DesignConstants';
import { DESKTOP_DISCOVER_NAV, DESKTOP_PERSONAL_NAV, type AppNavItem } from '../../constants/AppNavigation';
import { PageNames } from '../../constants/AppConstants';
import { NearbyPage } from '../../features/nearby/pages/NearbyPage';
import { DiscoverPage } from '../../features/discover/pages/DiscoverPage';
import { MessagePage } from '../../features/message/pages/MessagePage';
import { ProfilePage } from '../../features/profile/pages/ProfilePage';
import { LoginPromptView } from '../LoginPromptView';
import { NotificationCenter } from '../NotificationCenter';
import { NotificationBanner } from '../NotificationBanner';
import { PerformanceIndicator } from '../PerformanceIndicator';
import { AnchorEntry } from '../AnchorEntry';
import { SystemBadge } from '../SystemBadge';
import { SystemAvatar } from '../SystemAvatar';
import { SystemButton } from '../SystemButton';
import { SurfaceIconButton } from '../SurfaceIconButton';
import { AppIcon } from '../SystemIcon';
import { NotificationRenderer } from '../../core/services/notification/NotificationRenderer';
import { type SmartNotification, type NotificationAction } from '../../core/services/notification/NotificationDataManager';
import { AppRouter, LoginRouteParam } from '../../core/router/AppRouter';
import { GlobalStateManager, KEY_CURRENT_USER } from '../../core/state/GlobalStateManager';
import { MyProfile } from '../../core/types/api';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';
import { SystemUiController } from '../../core/utils/SystemUiController';
import { SystemToast } from '../SystemToast';
import { type ThemeMode, type ThemeOptionItem } from '../../core/theme/AppThemeTypes';

@Component
export struct DesktopLayout {
  // 来自父组件（Index.ets）的核心状态
  @Link currentRoute: string;
  @Link showNotificationCenter: boolean;
  @Link showPerformanceDetails: boolean;
  @Link showCreateSheet: boolean;

  // 来自父组件的只读状态
  @Prop unreadCount: number = 0;
  @Prop visibleNotifications: SmartNotification[] = [];
  @Prop isLoggedIn: boolean = false;
  @Prop userType: string = '';
  @Prop currentBreakpoint: BreakpointType = 'lg';

  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  @State showUpgradeBanner: boolean = false;
  @State showThemeSheet: boolean = false;

  aboutToAppear() {
    this.showUpgradeBanner = this.isLoggedIn && this.userType === 'temp';
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getCurrentThemeMode(): ThemeMode {
    return AppThemeManager.getThemeMode();
  }

  private async handleThemeChange(mode: ThemeMode): Promise<void> {
    if (mode === this.getCurrentThemeMode()) {
      this.showThemeSheet = false;
      return;
    }
    await AppThemeManager.setThemeMode(mode);
    SystemUiController.syncWithCurrentTheme();
    this.showThemeSheet = false;
    SystemToast.show(`已切换为${AppThemeManager.getThemeLabel(mode)}`, this.getUIContext());
  }

  build() {
    Stack() {
      Column() {
        // 顶栏（52vp，跨全宽）
        this.TopBar()

        // body：侧边栏 + main card
        Row({ space: DesignConstants.SPACING.LG }) {
          Column() {
            this.Sidebar()
          }
          .width(DesignConstants.LAYOUT.DESKTOP_SIDEBAR_WIDTH) // 192
          .height('100%')

          Column() {
            this.MainContent()
          }
          .layoutWeight(1)
          .height('100%')
          .borderRadius(DesignConstants.SHAPE.HERO_CARD) // 24
          .backgroundColor(this.getPalette().surfaceBackground)
          .border({ width: 1, color: this.getPalette().borderColor })
          .clip(true)
        }
        .layoutWeight(1)
        .width('100%')
        .padding(DesignConstants.SPACING.LG) // p-4
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getPalette().pageBackground)

      // 通知中心 overlay
      if (this.showNotificationCenter) {
        Stack() {
          Column()
            .width('100%')
            .height('100%')
            .backgroundColor(this.getPalette().overlayStrong)
            .onClick(() => {
              this.showNotificationCenter = false;
            })
          Column() {
            NotificationCenter({ onClose: () => {
              this.showNotificationCenter = false;
            } })
          }
          .width('100%')
          .height('100%')
          .backgroundColor(this.getPalette().surfaceBackground)
        }
        .width('100%')
        .height('100%')
      }

      // 性能详情 overlay
      if (this.showPerformanceDetails) {
        Column() {
          PerformanceIndicator({ showDetails: true, compact: false })
        }
        .width('100%')
        .height('100%')
        .backgroundColor(this.getPalette().surfaceBackground)
        .onClick(() => {
          this.showPerformanceDetails = false;
        })
      }
    }
    .width('100%')
    .height('100%')
    .bindSheet($$this.showCreateSheet, this.CreateSheetContent(), {
      height: SheetSize.FIT_CONTENT,
      dragBar: true,
      showClose: true,
      backgroundColor: this.getPalette().surfaceBackground
    })
    .bindSheet($$this.showThemeSheet, this.ThemeSheetContent(), {
      height: SheetSize.FIT_CONTENT,
      dragBar: true,
      showClose: true,
      backgroundColor: this.getPalette().surfaceBackground
    })
  }

  // ==================== 顶栏（对齐前端 desktop-header）====================
  @Builder
  TopBar() {
    Row() {
      // Logo
      Row({ space: DesignConstants.SPACING.SM }) {
        Text() {
          SymbolSpan($r(AppIcon.NEARBY))
            .fontSize(22)
            .fontColor([this.getPalette().accentBackground])
        }
        Text('HotSpot')
          .fontSize(DesignConstants.FONT.SIZE.LG_PLUS) // 18
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getPalette().textPrimary)
      }
      .alignItems(VerticalAlign.Center)
      .onClick(() => {
        this.currentRoute = 'nearby';
      })

      Blank().layoutWeight(1)

      // 右侧动作
      Row({ space: DesignConstants.SPACING.XS }) {
        SurfaceIconButton({
          icon: AppIcon.SEARCH,
          kind: 'neutral',
          buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE, // 36
          glyphSize: 17,
          onPress: () => AppRouter.push(PageNames.SEARCH)
        })

        // 发布（brand 圆形按钮）
        SurfaceIconButton({
          icon: AppIcon.ADD,
          kind: 'primary',
          buttonSize: 32,
          glyphSize: 15,
          onPress: () => this.openCreate()
        })

        SurfaceIconButton({
          icon: AppIcon.PALETTE,
          kind: 'neutral',
          buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
          glyphSize: 17,
          onPress: () => { this.showThemeSheet = true; }
        })

        SurfaceIconButton({
          icon: AppIcon.SETTINGS,
          kind: 'neutral',
          buttonSize: DesignConstants.LAYOUT.ICON_BUTTON_SIZE,
          glyphSize: 17,
          onPress: () => AppRouter.push(PageNames.SETTINGS)
        })

        // 头像 / 登录
        if (this.isLoggedIn) {
          Row() {
            SystemAvatar({
              avatarUrl: this.currentUser?.avatar_url ?? '',
              name: this.currentUser?.nickname ?? '',
              avatarSize: 32,
              shape: 'circle'
            })
          }
          .width(36)
          .height(36)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .justifyContent(FlexAlign.Center)
          .onClick(() => {
            this.currentRoute = 'profile';
          })
        } else {
          SystemButton({
            label: '登录',
            variant: 'primary',
            buttonSize: 'small',
            onPress: () => this.openLogin('')
          })
        }
      }
      .alignItems(VerticalAlign.Center)
    }
    .width('100%')
    .height(DesignConstants.LAYOUT.DESKTOP_HEADER_HEIGHT) // 52
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .backgroundColor(this.getPalette().surfaceBackground)
    .border({ width: { bottom: 1 }, color: this.getPalette().borderColor })
  }

  // ==================== 主题家族切换 sheet（顶栏 PALETTE 按钮）====================
  @Builder
  ThemeSheetContent() {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text('外观主题')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getPalette().textPrimary)
        .width('100%')
        .textAlign(TextAlign.Center)

      Text('切换整体观感。当前已对齐前端的森林 / 暖砂 / 靛蓝三族主题。')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.getPalette().textSecondary)
        .width('100%')
        .textAlign(TextAlign.Center)

      Column({ space: DesignConstants.SPACING.SM }) {
        ForEach(AppThemeManager.getThemeOptions(), (option: ThemeOptionItem) => {
          this.ThemeOptionRow(option)
        }, (option: ThemeOptionItem) => option.mode)
      }
      .width('100%')
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XXL
    })
    .backgroundColor(this.getPalette().surfaceBackground)
  }

  @Builder
  private ThemeOptionRow(option: ThemeOptionItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Column({ space: 4 }) {
        Text(option.label)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(FontWeight.Medium)
          .fontColor(this.getPalette().textPrimary)
        Text(option.description)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getPalette().textSecondary)
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      Text(option.mode === this.getCurrentThemeMode() ? '当前' : '切换')
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(option.mode === this.getCurrentThemeMode() ? this.getPalette().accentText : this.getPalette().textSecondary)
        .padding({ left: DesignConstants.SPACING.MD2, right: DesignConstants.SPACING.MD2, top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
        .backgroundColor(option.mode === this.getCurrentThemeMode() ? this.getPalette().accentBackground : this.getPalette().surfaceSubtle)
        .borderRadius(DesignConstants.SHAPE.CHIP)
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(this.getPalette().surfaceSubtle)
    .onClick(() => { this.handleThemeChange(option.mode).catch(() => {}); })
  }

  // ==================== 侧边栏（对齐前端 sidebar：发现组 + 个人组 + 底部 AnchorEntry）====================
  @Builder
  Sidebar() {
    Column() {
      // 发现组
      Column({ space: 2 }) {
        ForEach(DESKTOP_DISCOVER_NAV, (item: AppNavItem) => {
          this.NavItemBuilder(item)
        }, (item: AppNavItem) => item.label)
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)

      // 分隔线
      Divider()
        .width('100%')
        .color(this.getPalette().borderColor)
        .margin({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })

      // 个人中心 section 标签
      Text('个人中心')
        .fontSize(DesignConstants.FONT.SIZE.XS2) // 10
        .fontWeight(FontWeight.Medium)
        .fontColor(this.getPalette().textSecondary)
        .letterSpacing(1.2)
        .margin({ left: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM })

      // 个人组
      Column({ space: 2 }) {
        ForEach(DESKTOP_PERSONAL_NAV, (item: AppNavItem) => {
          this.NavItemBuilder(item)
        }, (item: AppNavItem) => item.label)
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)

      // 管理后台入口（移动端降级概览，对齐前端 admin 侧边栏项；点击进入概览页，复杂操作引导 Web）
      Divider()
        .width('100%')
        .color(this.getPalette().borderColor)
        .margin({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM })

      Row() {
        Text() {
          SymbolSpan($r(AppIcon.SHIELD))
            .fontSize(17)
            .fontColor([this.getPalette().textSecondary])
        }
        Text('管理后台')
          .fontSize(DesignConstants.FONT.SIZE.SM_PLUS) // 13
          .fontColor(this.getPalette().textPrimary)
          .margin({ left: DesignConstants.SPACING.SM })
        Blank().layoutWeight(1)
      }
      .width('100%')
      .height(40)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
      .borderRadius(DesignConstants.SHAPE.INPUT)
      .alignItems(VerticalAlign.Center)
      .onClick(() => {
        if (this.isLoggedIn) {
          AppRouter.push(PageNames.ADMIN_OVERVIEW);
        } else {
          this.openLogin('管理后台');
        }
      })

      Blank().layoutWeight(1)

      // 底部浏览锚点卡片
      Column() {
        AnchorEntry({ variant: 'card' })
      }
      .width('100%')
      .padding({ top: DesignConstants.SPACING.SM })
      .border({ width: { top: 1 }, color: this.getPalette().borderColor })
    }
    .width('100%')
    .height('100%')
    .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.SM })
  }

  @Builder
  NavItemBuilder(item: AppNavItem) {
    Row() {
      Text() {
        SymbolSpan($r(item.icon))
          .fontSize(17)
          .fontColor([this.isItemActive(item) ? this.getPalette().accentBackground : this.getPalette().textSecondary])
      }
      Text(item.label)
        .fontSize(DesignConstants.FONT.SIZE.SM_PLUS) // 13
        .fontColor(this.isItemActive(item) ? this.getPalette().textPrimary : this.getPalette().textSecondary)
        .fontWeight(this.isItemActive(item) ? FontWeight.Medium : FontWeight.Regular)
        .margin({ left: DesignConstants.SPACING.SM })
        .layoutWeight(1)

      if (this.badgeCount(item) > 0) {
        SystemBadge({ count: this.badgeCount(item) })
      }
    }
    .width('100%')
    .height(40)
    .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
    .borderRadius(DesignConstants.SHAPE.INPUT) // 18
    .backgroundColor(this.isItemActive(item) ? this.getPalette().tabBarActiveBackground : DesignConstants.COLOR.TRANSPARENT)
    .alignItems(VerticalAlign.Center)
    .onClick(() => this.handleNavClick(item))
  }

  private isItemActive(item: AppNavItem): boolean {
    return item.isTab && item.tabRoute === this.currentRoute;
  }

  private badgeCount(item: AppNavItem): number {
    if (item.badge === 'messages') {
      return this.unreadCount;
    }
    return 0;
  }

  private handleNavClick(item: AppNavItem): void {
    if (item.openNotificationCenter) {
      this.showNotificationCenter = true;
      return;
    }
    if (item.isTab) {
      this.currentRoute = item.tabRoute;
      return;
    }
    if (item.requiresAuth && !this.isLoggedIn) {
      this.openLogin(item.label);
      return;
    }
    if (item.pageName.length > 0) {
      try {
        AppRouter.push(item.pageName);
      } catch (e) {
        SystemToast.showError(`导航到${item.label}失败`, this.getUIContext());
      }
    }
  }

  // ==================== 主内容区（页面自带 header）====================
  @Builder
  MainContent() {
    Column() {
      if (this.currentRoute === 'nearby') {
        NearbyPage()
      } else if (this.currentRoute === 'discover') {
        DiscoverPage()
      } else if (this.currentRoute === 'messages') {
        if (this.isLoggedIn) {
          Column() {
            if (this.showUpgradeBanner) {
              this.UpgradeBanner()
            }
            MessagePage()
          }
          .width('100%')
          .height('100%')
        } else {
          LoginPromptView({
            title: '消息功能需要登录',
            subtitle: '登录后可与好友实时聊天、加入群组讨论'
          })
        }
      } else if (this.currentRoute === 'profile') {
        ProfilePage()
      } else {
        Column() {
          Text('未知路由')
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontColor(this.getPalette().textSecondary)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  UpgradeBanner() {
    Row() {
      Text('当前为临时用户，注册账号可保留所有数据')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor(this.getPalette().textSecondary)
        .layoutWeight(1)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
      Button('去注册')
        .type(ButtonType.Normal)
        .height(28)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .backgroundColor(this.getPalette().accentBackground)
        .fontColor(this.getPalette().accentText)
        .borderRadius(DesignConstants.RADIUS.SM)
        .padding({ left: 12, right: 12 })
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 8, bottom: 8 })
    .backgroundColor(this.getPalette().accentMuted)
  }

  // ==================== 创建菜单（对齐前端 mobile-create-sheet 三项）====================
  private openCreate(): void {
    if (!this.isLoggedIn) {
      this.openLogin('发布内容');
      return;
    }
    this.showCreateSheet = true;
  }

  private openLogin(action: string): void {
    const params = new LoginRouteParam();
    params.reason = 'action_required';
    params.action = action.length > 0 ? action : '登录';
    AppRouter.push(PageNames.LOGIN, params);
  }

  @Builder
  CreateSheetContent() {
    Column() {
      Text('创建')
        .fontSize(DesignConstants.FONT.SIZE.LG_PLUS) // 18
        .fontWeight(FontWeight.Medium)
        .fontColor(this.getPalette().textPrimary)
        .margin({ top: DesignConstants.SPACING.XL, bottom: DesignConstants.SPACING.LG })

      this.CreateSheetItem(AppIcon.PHOTO, '发布动态', () => {
        this.showCreateSheet = false;
        AppRouter.push(PageNames.CREATE_MOMENT);
      })
      this.CreateSheetItem(AppIcon.GROUP, '创建群组', () => {
        this.showCreateSheet = false;
        AppRouter.push(PageNames.CREATE_GROUP);
      })
      this.CreateSheetItem(AppIcon.LOCATION, '分享位置', () => {
        this.showCreateSheet = false;
        AppRouter.push(PageNames.LOCATION_PICKER);
      })
    }
    .width('100%')
    .padding({ bottom: DesignConstants.SPACING.XXL })
    .alignItems(HorizontalAlign.Center)
  }

  @Builder
  CreateSheetItem(icon: AppIcon, label: string, action: () => void) {
    Row() {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(20)
          .fontColor([this.getPalette().accentBackground])
      }
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.MD) // 14
        .fontColor(this.getPalette().textPrimary)
        .margin({ left: DesignConstants.SPACING.MD })
    }
    .width('90%')
    .height(52)
    .padding({ left: DesignConstants.SPACING.LG })
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .backgroundColor(this.getPalette().surfaceSubtle)
    .alignItems(VerticalAlign.Center)
    .margin({ bottom: DesignConstants.SPACING.SM })
    .onClick(() => action())
  }
}
