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
 * 导航：核心 tab（首页 / 附近瞬间 / 消息）切换 currentRoute；其余项 push 或打开通知中心 overlay。
 * sub-page（设置 / 通知 / 我的瞬间 / 我的群组 / 关系 等）push 全屏、自带返回。
 */

import { type BreakpointType } from '../../core/responsive/Breakpoint';
import { DesignConstants } from '../../constants/DesignConstants';
import { DESKTOP_PRIMARY_NAV, DESKTOP_PERSONAL_NAV, type AppNavItem } from '../../constants/AppNavigation';
import { PageNames } from '../../constants/AppConstants';
import { HomePage } from '../../features/home/pages/HomePage';
import { MomentsFeedPage } from '../../features/moments/pages/MomentsFeedPage';
import { MessagePage } from '../../features/messages/pages/MessagePage';
import { ProfilePage } from '../../features/me/pages/ProfilePage';
import { LoginPromptView } from '../LoginPromptView';
import { NotificationCenter } from '../NotificationCenter';
import { NotificationBanner } from '../NotificationBanner';
import { PerformanceIndicator } from '../PerformanceIndicator';
import { curves } from '@kit.ArkUI';
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
import { AccountInfo } from '../../core/types/api';
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

  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  @State showUpgradeBanner: boolean = false;
  @State showThemeSheet: boolean = false;
  @State logoPressed: boolean = false;
  // 当前按压中的侧栏行（nav label / '__admin__'），驱动整行 tap-97 按压缩放
  @State pressedNavLabel: string = '';
  // hover 中的侧栏导航行（桌面断点淡填充反馈）
  @State hoveredNavLabel: string = '';
  // 其余可按压入口：主题行 / 创建菜单行 / 顶栏头像
  @State pressedThemeMode: string = '';
  @State pressedCreateItem: string = '';
  @State avatarPressed: boolean = false;

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
          .width(DesignConstants.LAYOUT.DESKTOP_SIDEBAR_WIDTH) // 224
          .height('100%')

          Column() {
            this.MainContent()
          }
          .layoutWeight(1)
          .height('100%')
          .borderRadius(DesignConstants.SHAPE.HERO_CARD) // 24
          .backgroundColor(this.getPalette().surfaceBackground)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
          .clip(true)
        }
        .layoutWeight(1)
        .width('100%')
        .padding(DesignConstants.SPACING.LG) // p-4
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getPalette().pageBackground)

      // 通知中心 overlay（遮罩淡入淡出，避免 if 硬切闪现）
      if (this.showNotificationCenter) {
        Stack() {
          Column()
            .width('100%')
            .height('100%')
            .backgroundColor(this.getPalette().overlayStrong)
            .transition(TransitionEffect.OPACITY
              .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
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
          .transition(TransitionEffect.OPACITY
            .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
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
        .transition(TransitionEffect.OPACITY
          .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
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
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontColor([this.getPalette().accentBackground])
        }
        Text('HotSpot')
          .fontSize(DesignConstants.FONT.SIZE.LG_PLUS) // 18
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getPalette().textPrimary)
      }
      .alignItems(VerticalAlign.Center)
      .scale(this.logoPressed
        ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW, y: DesignConstants.INTERACTION.SCALE_PRESSED_INPUT_ROW }
        : { x: 1, y: 1 })
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
          DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
        )
      })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.logoPressed = true;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.logoPressed = false;
        }
      })
      .onClick(() => {
        this.currentRoute = 'home';
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
              avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.XS,
              shape: 'circle'
            })
          }
          .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .justifyContent(FlexAlign.Center)
          // 按压反馈（对齐顶栏圆钮语义）
          .scale(this.avatarPressed ?
            { x: DesignConstants.INTERACTION.SCALE_PRESSED_ICON, y: DesignConstants.INTERACTION.SCALE_PRESSED_ICON } :
            { x: 1, y: 1 })
          .animation({ curve: curves.springMotion(
            DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
            DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
          ) })
          .onTouch((event: TouchEvent) => {
            if (event.type === TouchType.Down) {
              this.avatarPressed = true;
            } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
              this.avatarPressed = false;
            }
          })
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
    .border({ width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getPalette().borderColor })
  }

  // ==================== 主题家族切换 sheet（顶栏 PALETTE 按钮）====================
  @Builder
  ThemeSheetContent() {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text('外观主题')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(this.getPalette().textPrimary)
        .width('100%')
        .textAlign(TextAlign.Center)

      Text('切换整体观感。当前已对齐前端的森林 / 暖砂 / 星际蓝三族主题。')
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
      Column({ space: DesignConstants.SPACING.XS }) {
        Text(option.label)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
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
    // 按压反馈（对齐侧栏导航行 tap-97 语义）
    .scale(this.pressedThemeMode === option.mode ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedThemeMode = option.mode;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedThemeMode = '';
      }
    })
    .onClick(() => { this.handleThemeChange(option.mode).catch(() => {}); })
  }

  // ==================== 侧边栏（对齐前端 sidebar：发现组 + 个人组 + 底部 AnchorEntry）====================
  @Builder
  Sidebar() {
    Column() {
      // 发现组
      Column({ space: DesignConstants.SPACING.HALF }) {
        ForEach(DESKTOP_PRIMARY_NAV, (item: AppNavItem) => {
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
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
        .fontColor(this.getPalette().textSecondary)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
        .margin({ left: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.SM })

      // 个人组
      Column({ space: DesignConstants.SPACING.HALF }) {
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
            .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
            .fontColor([this.getPalette().textSecondary])
        }
        Text('管理后台')
          .fontSize(DesignConstants.FONT.SIZE.SM) // 12
          .fontColor(this.getPalette().textPrimary)
          .margin({ left: DesignConstants.SPACING.SM })
        Blank().layoutWeight(1)
      }
      .width('100%')
      .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
      .borderRadius(DesignConstants.SHAPE.INPUT)
      .alignItems(VerticalAlign.Center)
      // 按压反馈与 NavItemBuilder 同款（tap-97 + spring）
      .scale(this.pressedNavLabel === '__admin__' ?
        { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
        { x: 1, y: 1 })
      .animation({ curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      ) })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.pressedNavLabel = '__admin__';
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.pressedNavLabel = '';
        }
      })
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
      .border({ width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getPalette().borderColor })
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
          .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
          .fontColor([this.isItemActive(item) ? this.getPalette().accentBackground : this.getPalette().textSecondary])
      }
      Text(item.label)
        .fontSize(DesignConstants.FONT.SIZE.SM) // 12
        .fontColor(this.isItemActive(item) ? this.getPalette().textPrimary : this.getPalette().textSecondary)
        .fontWeight(this.isItemActive(item) ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.REGULAR)
        .margin({ left: DesignConstants.SPACING.SM })
        .layoutWeight(1)

      if (this.badgeCount(item) > 0) {
        SystemBadge({ count: this.badgeCount(item) })
      }
    }
    .width('100%')
    .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD)
    .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
    .borderRadius(DesignConstants.SHAPE.INPUT) // 16
    // 激活/hover 底色：激活项常驻淡底，非激活项仅 hover 时淡填充（对齐前端侧栏导航 hover 反馈）
    .backgroundColor(this.isItemActive(item) || this.hoveredNavLabel === item.label
      ? this.getPalette().tabBarActiveBackground
      : DesignConstants.COLOR.TRANSPARENT)
    // 激活底色/文字色切换过渡（对齐前端侧栏导航 transition-colors，避免硬跳）
    .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_SMOOTH })
    // 按压反馈（对齐前端 tap-97）；滑出/抬起经 onTouch 统一复位
    .scale(this.pressedNavLabel === item.label ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .alignItems(VerticalAlign.Center)
    .onHover((isHover: boolean) => {
      this.hoveredNavLabel = isHover ? item.label : '';
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedNavLabel = item.label;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedNavLabel = '';
      }
    })
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
      if (this.currentRoute === 'home') {
        HomePage()
      } else if (this.currentRoute === 'moments') {
        MomentsFeedPage()
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
        .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .backgroundColor(this.getPalette().accentBackground)
        .fontColor(this.getPalette().accentText)
        .borderRadius(DesignConstants.RADIUS.SM)
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
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
        .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
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
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontColor([this.getPalette().accentBackground])
      }
      Text(label)
        .fontSize(DesignConstants.FONT.SIZE.MD) // 14
        .fontColor(this.getPalette().textPrimary)
        .margin({ left: DesignConstants.SPACING.MD })
    }
    .width('90%')
    .height(DesignConstants.LAYOUT.FAB_SIZE)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .borderRadius(DesignConstants.SHAPE.INPUT)
    .backgroundColor(this.getPalette().surfaceSubtle)
    .alignItems(VerticalAlign.Center)
    .justifyContent(FlexAlign.Center)
    .margin({ bottom: DesignConstants.SPACING.SM })
    // 按压反馈（对齐侧栏导航行 tap-97 语义）
    .scale(this.pressedCreateItem === label ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedCreateItem = label;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedCreateItem = '';
      }
    })
    .onClick(() => action())
  }
}
