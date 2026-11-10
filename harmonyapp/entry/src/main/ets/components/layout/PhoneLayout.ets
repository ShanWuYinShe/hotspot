/**
 * PhoneLayout — 鸿蒙手机 + 平板布局（完整版）
 *
 * 适用断点：xs / sm / md（< 840vp）
 *
 * 完整业务：
 * - 4 Tab（首页 / 附近 / 发布FAB / 消息 / 我的）对齐前端底栏导航
 * - 通知中心 / 通知 banner / 通知铃铛按钮
 * - Create Sheet（bindSheet 模式 + 中间「+」按钮）
 * - 性能详情 overlay
 * - Login Prompt（未登录时包装消息 tab）
 * - Upgrade Banner（临时用户时包装消息 tab）
 *
 * 状态传递：通过 @Link 接收 Index.ets 维护的 6 个 @State
 * 业务回调（initApp / WS / 通知同步等）仍由 Index.ets 的 aboutToAppear 触发
 *
 * 图标：lucide 线描 SVG（LucideIcon，对齐前端 @lucide/svelte；见 harmonyapp/AGENTS.md）
 */

import { HomePage } from '../../features/home/pages/HomePage';
import { MessagePage } from '../../features/messages/pages/MessagePage';
import { ProfilePage } from '../../features/me/pages/ProfilePage';
// 附近一级 Tab 落点：NearbyPage 枢纽页（对齐 web /nearby）。
// 2026-10-19 信息架构调整（两端已同步）：web 将 /nearby 从 307 跳转改为枢纽页、
// 首页改为「关注/发现」双轴；本端同步——Tab 落点 → NearbyPage 枢纽；
// HomePage → 「关注/发现」双轴（附近用户/群组区块移出首页，统计卡迁到枢纽页）。
// 【仍未对齐】① MomentsFeedPage 仍带「热门/周边/关注」三 Tab（web 侧已收敛为纯地理流）；
// ② 枢纽表现形态：web 用分区平铺（人/瞬间/群组三段同屏），本端用 TabsPillGroup 段位。
import { NearbyPage } from '../../features/nearby/pages/NearbyPage';
import { LoginPromptView } from '../LoginPromptView';
import { type BreakpointType, isWideForm } from '../../core/responsive/Breakpoint';
import { NotificationCenter } from '../NotificationCenter';
import { NotificationBanner } from '../NotificationBanner';
import { PerformanceIndicator } from '../PerformanceIndicator';
import { NotificationRenderer } from '../../core/services/notification/NotificationRenderer';
import { type SmartNotification, type NotificationAction } from '../../core/services/notification/NotificationDataManager';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../core/theme/AppThemeVisuals';
import { DesignConstants } from '../../constants/DesignConstants';
import { curves } from '@kit.ArkUI';
import { SystemBadge } from '../SystemBadge';
import { SystemButton } from '../SystemButton';
import { AppIcon } from '../SystemIcon';
import { LucideIcon } from '../LucideIcon';
import { AppRouter } from '../../core/router/AppRouter';
import { PageNames, StorageKeys } from '../../constants/AppConstants';
import { CreateMomentSheet } from '../../features/moments/components/CreateMomentSheet';
import { AmbientBackdrop } from './AmbientBackdrop';
import { DEFAULT_RESOLVED_THEME_ID } from '../../core/theme/AppThemeTypes';

interface TabItem {
  label: string;
  icon: AppIcon;
}

const TAB_ITEMS: TabItem[] = [
  { label: '首页', icon: AppIcon.HOME },
  { label: '附近', icon: AppIcon.COMPASS },
  { label: '发布', icon: AppIcon.ADD },
  { label: '消息', icon: AppIcon.MESSAGE },
  { label: '我的', icon: AppIcon.PERSON }
];

const CREATE_TAB_INDEX: number = 2;

@Component
export struct PhoneLayout {
  // 来自父组件（Index.ets）的 6 个核心状态
  @Link currentTabIndex: number;
  @Link showCreateSheet: boolean;
  @Link showNotificationCenter: boolean;
  @Link showPerformanceDetails: boolean;
  @Link isCreateTabPressed: boolean;

  // 来自父组件的只读状态
  @Prop unreadCount: number = 0;
  @Prop visibleNotifications: SmartNotification[] = [];
  @Prop isLoggedIn: boolean = false;
  @Prop safeAreaBottom: number = 0;
  @Prop safeAreaTop: number = 0;
  @Prop isTablet: boolean = false;
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  /** 实时断点（AppStorage 写入），用于在 PhoneLayout 内决策宽屏通知侧栏 */
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageProp('currentWidthVp') currentWidthVp: number = 0;

  // 内部状态
  /** 当前按下的底部导航 Tab 索引（-1 表示无），用于按下缩放反馈，对齐前端 hover-btn:active */
  @State pressedTabIndex: number = -1;
  @State private tabletFabPressed: boolean = false;
  /**
   * 编辑资料态隐藏底栏：编辑资料是「我的」页内的 isEditing 状态（不是 push 页），
   * 底栏在编辑期间一直可见 —— 用户切 tab 会丢失草稿（2026-10-21 端上走查）。
   * 由 ProfilePage 在进入/退出编辑时通过 AppStorage 写入。
   */
  @StorageLink('hideMainTabBar') hideMainTabBar: boolean = false;
  /** 路由页切 tab 一次性信号（设置页「个人资料」等）：≥0 为目标 tab，消费后复位 -1 */
  @StorageLink(StorageKeys.SWITCH_TAB_REQUEST) @Watch('onSwitchTabRequest') switchTabRequest: number = -1;

  private tabsController: TabsController = new TabsController();

  /** 消费切 tab 请求：立即复位，避免同值不再触发 @Watch */
  private onSwitchTabRequest(): void {
    const target = this.switchTabRequest;
    if (target < 0) {
      return;
    }
    AppStorage.setOrCreate<number>(StorageKeys.SWITCH_TAB_REQUEST, -1);
    if (target === CREATE_TAB_INDEX) {
      // 「+」不是真实标签，路由页也不应从底栏触达发布器
      return;
    }
    this.tabsController.changeIndex(target);
    this.currentTabIndex = target;
  }

  build() {
    Stack() {
      Column() {
        // 通知 banner（流式排布，把下方内容往下推，不再浮盖在页面之上）
        if (this.visibleNotifications.length > 0) {
          Column() {
            ForEach(this.visibleNotifications, (notification: SmartNotification) => {
              NotificationBanner({
                notification: notification,
                onClose: () => NotificationRenderer.getInstance().hideNotification(notification.id, 'manual'),
                onBannerClick: () => NotificationRenderer.getInstance().handleClick(notification),
                onAction: (action: NotificationAction) => NotificationRenderer.getInstance().handleAction(action, notification)
              })
                .transition(TransitionEffect.OPACITY
                  .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
            }, (item: SmartNotification) => item.id)
          }
          .width('100%')
          .padding({ top: DesignConstants.SPACING.MD, left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        }

        // 主体：Tabs + 5 TabContent（Nearby / Discover / Create / Message / Profile）
        Tabs({ barPosition: BarPosition.End, controller: this.tabsController, index: this.currentTabIndex }) {
          TabContent() { HomePage() }.tabBar(this.TabBarBuilder(0))
          // 附近一级 Tab 落点：枢纽页（人/群组/瞬间预览 + 查看全部），对齐 web /nearby。
          // 2026-10-19 IA 调整：此前直挂 NearbyUsersListPage，导致一级 Tab 的语义比落点窄
          // （点「附近」只看到人）。三个完整列表页仍是叶子页，从枢纽「查看全部」进入。
          TabContent() { NearbyPage() }.tabBar(this.TabBarBuilder(1))
          TabContent() { Column() {} }.tabBar(this.CreateTabBuilder())
          TabContent() {
            if (this.isLoggedIn) {
              Column() {
                MessagePage()
              }.width('100%').height('100%')
            } else {
              // Tab 内嵌场景无 PageShell 托底，必须显式绘制页面底色：
              // 透明背景会让窗口默认底色（浅色）透出，深色主题下整屏发灰。
              // 底板与氛围层分开：底板铺画布色，氛围画布压在其上、提示内容之下，
              // 与已登录页（PageShellContainer 内挂 AmbientBackdrop）保持同一视觉语言。
              Stack({ alignContent: Alignment.Center }) {
                Column()
                  .width('100%')
                  .height('100%')
                  .backgroundColor(this.getPalette().pageBackground)

                AmbientBackdrop()

                Column() {
                  // 内嵌场景不注入 surfaceColor：底色由本分支底层 Column 的 pageBackground 提供，
                  // LoginPromptView 保持透明才能让中间的 AmbientBackdrop 透出
                  // （此前传 pageBackground 会让其根 Stack 不透明底把氛围层盖住）。
                  LoginPromptView({
                    title: '登录后查看消息',
                    subtitle: '登录后即可查看和发起会话',
                    highlights: [
                      { icon: AppIcon.MESSAGE, text: '私聊与群聊会话' },
                      { icon: AppIcon.GROUP, text: '在线好友与群组' },
                      { icon: AppIcon.SYNC, text: '消息实时同步' }
                    ]
                  })
                }
                .width('100%')
                .height('100%')
              }
              .width('100%')
              .height('100%')
            }
          }.tabBar(this.TabBarBuilder(3))
          TabContent() { ProfilePage() }.tabBar(this.TabBarBuilder(4))
        }
        // 手机/平板统一隐藏系统 Tabs bar，改用悬浮 Dock（对齐 web 移动端悬浮胶囊导航）
        .barHeight(0)
        .scrollable(false)
        .barMode(BarMode.Fixed)
        .divider({ strokeWidth: 0, color: DesignConstants.COLOR.TRANSPARENT, startMargin: 0, endMargin: 0 })
        .onChange((index: number) => {
          if (index === CREATE_TAB_INDEX) {
            this.showCreateSheet = true;
            return;
          }
          this.currentTabIndex = index;
        })
        // 系统 bar 已隐藏，玻璃/描边/投影由自定义浮动 Dock 承载；Tabs 自身不再铺底
        .layoutWeight(1)
        // 不在 Tabs 容器再加 maxWidth 约束：各页面通过 PageShellContainer / ResponsiveContainer
        // 按断点自管宽度。原先 isTablet ? 720 会把 ≥960vp 开启的 master-detail 双栏
        // （HomePage / MomentsFeedPage 右栏 420/220）挤在 720vp 内、左侧被压到 300/500vp，
        // 与 useMasterDetail 的设计意图冲突。

        // 平板浮动药丸底栏（Tabs 系统 bar 已隐藏，用自定义浮动药丸替代，对齐前端 TabletBottomNav）
        if (this.isTablet) {
          this.buildFloatingTabletNav()
        } else if (!this.hideMainTabBar) {
          this.buildFloatingMobileNav()
        }
      }
      .width('100%')
      .height('100%')

      // 通知中心 overlay：
      // - 窄屏（< 960vp）：全屏模态遮罩（与原行为一致，遮罩外点击关闭）
      // - 宽屏（≥ 960vp / 平板横屏 / 折叠屏展开）：右侧滑出侧栏卡片（约 420vp），
      //   整页不被通知中心全屏接管（与「桌面侧边栏常驻 + 通知/详情内嵌」心智一致）
      if (this.showNotificationCenter) {
        if (isWideForm(this.currentBreakpoint, this.currentWidthVp)) {
          Stack({ alignContent: Alignment.Center }) {
            Column()
              .width('100%').height('100%')
              .backgroundColor(this.getPalette().overlayStrong)
              .transition(TransitionEffect.OPACITY
                .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
              .onClick(() => { this.showNotificationCenter = false; })
            Row() {
              Blank().layoutWeight(1)
              Column() {
                NotificationCenter({ onClose: () => { this.showNotificationCenter = false; } })
              }
              .width(DesignConstants.LAYOUT.HOME_COLUMN_WIDTH)
              .height('100%')
              .backgroundColor(this.getPalette().pageBackground)
              .borderRadius({ topLeft: DesignConstants.SHAPE.HERO_CARD, bottomLeft: DesignConstants.SHAPE.HERO_CARD })
              .shadow(this.getPalette().elevatedShadow)
              .transition(TransitionEffect.OPACITY
                .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
            }
            .width('100%').height('100%')
          }
          .width('100%').height('100%')
        } else {
          Stack() {
            Column()
              .width('100%').height('100%')
              .backgroundColor(this.getPalette().overlayStrong)
              .transition(TransitionEffect.OPACITY
                .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
              .onClick(() => { this.showNotificationCenter = false; })
            Column() {
              NotificationCenter({ onClose: () => { this.showNotificationCenter = false; } })
            }
            .width('100%').height('100%')
            // 全屏通知面板避让状态栏：否则页头（标题/返回钮）画进系统时钟区域
            .padding({ top: this.safeAreaTop })
            .backgroundColor(this.getPalette().pageBackground)
            .transition(TransitionEffect.OPACITY
              .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
          }
          .width('100%').height('100%')
        }
      }

      // 性能详情 overlay（模态，保留在叠加层）
      if (this.showPerformanceDetails) {
        Column() {
          PerformanceIndicator({ showDetails: true, compact: false })
        }
        .width('100%').height('100%')
        .backgroundColor(this.getPalette().pageBackground)
        .transition(TransitionEffect.OPACITY
          .animation({ duration: DesignConstants.ANIMATION.DURATION_FAST, curve: DesignConstants.ANIMATION.CURVE_EASE_OUT }))
        .onClick(() => { this.showPerformanceDetails = false; })
      }

    }
    .bindSheet($$this.showCreateSheet, this.CreateSheetContent(), {
      // 92% 高完整表单 Sheet（对齐前端 CreateSheet Dialog side=bottom height:92%）
      height: '92%',
      dragBar: false,
      showClose: false,
      backgroundColor: this.getPalette().pageBackground,
      // 键盘避让：输入框随键盘上移（沿用 LoginPromptView 先例）
      keyboardAvoidMode: SheetKeyboardAvoidMode.TRANSLATE_AND_SCROLL
    })
    .width('100%').height('100%')
    .backgroundColor(this.getPalette().pageBackground)
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  @Builder TabBarBuilder(index: number) {
    Column() {
      Column() {
        // Stack 不设固定尺寸 → 跟随图标 Text 尺寸，图标居中撑满；
        // 未读 badge 由 TopEnd 钉到图标右上角，不再把图标挤偏。
        Stack({ alignContent: Alignment.TopEnd }) {
          LucideIcon({
            icon: TAB_ITEMS[index].icon,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
            paletteRole: this.currentTabIndex === index ? 'tabActive' : 'tabInactive'
          })
          // 消息 tab（index 3）未读角标，对齐前端 inbox badge
          if (index === 3 && this.unreadCount > 0) {
            SystemBadge({ count: this.unreadCount })
          }
        }
        // 对齐 web NavTab：图标选中微放大 1.1、文字选中 semibold，不再用整块选中底色
        .scale(this.currentTabIndex === index ? { x: 1.1, y: 1.1 } : { x: 1, y: 1 })
        .animation({
          duration: DesignConstants.ANIMATION.DURATION_FAST,
          curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
        })

        Text(TAB_ITEMS[index].label)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(
            this.currentTabIndex === index
              ? this.getPalette().accentBackground
              : this.getPalette().tabBarInactive
          )
          .fontWeight(
            this.currentTabIndex === index ? DesignConstants.FONT.WEIGHT.SEMIBOLD : DesignConstants.FONT.WEIGHT.MEDIUM
          )
          .margin({ top: DesignConstants.SPACING.XS })

        // web 底部 NavTab 的选中指示微胶囊（占位常驻，仅切透明度/缩放，防止文字上下跳动）
        Column()
          .width(DesignConstants.SPACING.MD3)
          .height(DesignConstants.SPACING.THREE)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().accentBackground)
          .shadow({
            radius: DesignConstants.SHADOW.SMALL.radius,
            color: this.getPalette().accentBackground
          })
          .opacity(this.currentTabIndex === index ? 1 : 0)
          .scale(this.currentTabIndex === index ? { x: 1, y: 1 } : { x: 0.5, y: 0.5 })
          .animation({
            duration: DesignConstants.ANIMATION.DURATION_FAST,
            curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
          })
      }
      .width(DesignConstants.LAYOUT.LABEL_WIDTH)
      .padding({ top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
      // 按压差异：选中 tab 按压不缩放（accent 指示条已表达选中），未选中按压缩放预览
      .scale(
        this.pressedTabIndex === index && this.currentTabIndex !== index
          ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_PILL, y: DesignConstants.INTERACTION.SCALE_PRESSED_PILL }
          : { x: 1, y: 1 }
      )
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_TAB,
          DesignConstants.INTERACTION.SPRING_DAMPING_TAB
        )
      })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.pressedTabIndex = index;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.pressedTabIndex = -1;
        }
      })
    }
    .height(DesignConstants.LAYOUT.MOBILE_BOTTOM_NAV_HEIGHT)
    .justifyContent(FlexAlign.Center)
    .onClick(() => {
      if (index !== CREATE_TAB_INDEX) {
        this.tabsController.changeIndex(index);
        this.currentTabIndex = index;
      }
    })
  }

  // 手机悬浮 Dock（对齐前端 MobilePortrait nav：max-w-md + rounded-[28px] + p-1.5 + 玻璃 + 投影）
  @Builder buildFloatingMobileNav() {
    Column() {
      Row() {
        this.MobileDockItem(0)
        this.MobileDockItem(1)
        this.FloatingFab()
        this.MobileDockItem(3)
        this.MobileDockItem(4)
      }
      .width('100%')
      .height(60)
      .constraintSize({ maxWidth: DesignConstants.LAYOUT.MOBILE_DOCK_MAX_WIDTH })
      .justifyContent(FlexAlign.SpaceAround)
      .alignItems(VerticalAlign.Center)
      .padding({ left: DesignConstants.SPACING.SM2, right: DesignConstants.SPACING.SM2 })
      .borderRadius(DesignConstants.SHAPE.SHEET_TOP_RADIUS)
      .backgroundColor(AppThemeVisuals.withAlpha(this.getPalette().tabBarBackground, 0.95))
      .backgroundBlurStyle(BlurStyle.Thick, {
        colorMode: this.getPalette().isDarkTheme ? ThemeColorMode.DARK : ThemeColorMode.LIGHT
      })
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor })
      .shadow(this.getPalette().elevatedShadow)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Center)
    .padding({
      left: DesignConstants.SPACING.MD3,
      right: DesignConstants.SPACING.MD3,
      top: DesignConstants.SPACING.XS,
      bottom: Math.max(this.safeAreaBottom, DesignConstants.SPACING.MD)
    })
  }

  @Builder MobileDockItem(index: number) {
    Column() {
      Stack({ alignContent: Alignment.TopEnd }) {
        LucideIcon({
          icon: TAB_ITEMS[index].icon,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
          paletteRole: this.currentTabIndex === index ? 'tabActive' : 'tabInactive'
        })
        if (index === 3 && this.unreadCount > 0) {
          SystemBadge({ count: this.unreadCount })
        }
      }
      .scale(this.currentTabIndex === index ? { x: 1.1, y: 1.1 } : { x: 1, y: 1 })
      .animation({
        duration: DesignConstants.ANIMATION.DURATION_FAST,
        curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
      })

      Text(TAB_ITEMS[index].label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(
          this.currentTabIndex === index
            ? this.getPalette().accentBackground
            : this.getPalette().tabBarInactive
        )
        .fontWeight(
          this.currentTabIndex === index ? DesignConstants.FONT.WEIGHT.SEMIBOLD : DesignConstants.FONT.WEIGHT.MEDIUM
        )
        .margin({ top: DesignConstants.SPACING.HALF })

      Column()
        .width(DesignConstants.SPACING.MD3)
        .height(DesignConstants.SPACING.THREE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getPalette().accentBackground)
        .opacity(this.currentTabIndex === index ? 1 : 0)
        .scale(this.currentTabIndex === index ? { x: 1, y: 1 } : { x: 0.5, y: 0.5 })
        .animation({
          duration: DesignConstants.ANIMATION.DURATION_FAST,
          curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
        })
    }
    .layoutWeight(1)
    .height(48)
    .justifyContent(FlexAlign.Center)
    .accessibilityText(TAB_ITEMS[index].label)
    .onClick(() => {
      this.tabsController.changeIndex(index);
      this.currentTabIndex = index;
    })
  }

  // 平板浮动药丸底栏（对齐前端 TabletLayout 宽平板浮动 Dock：max-width 680 + pill 圆角 + 玻璃 blur + safe-area）
  @Builder buildFloatingTabletNav() {
    Column() {
      Row() {
        this.FloatingTabItem(0)
        this.FloatingTabItem(1)
        this.FloatingFab()
        this.FloatingTabItem(3)
        this.FloatingTabItem(4)
      }
      .width('100%')
      .constraintSize({ maxWidth: DesignConstants.LAYOUT.TABLET_NAV_MAX_WIDTH })
      .justifyContent(FlexAlign.SpaceAround)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(AppThemeVisuals.withAlpha(this.getPalette().tabBarBackground, 0.94))
      // 平板浮动药丸：Medium 档毛玻璃（对齐前端 tablet dock --cst-blur-lg）
      .backgroundBlurStyle(BlurStyle.Thick, {
        colorMode: this.getPalette().isDarkTheme ? ThemeColorMode.DARK : ThemeColorMode.LIGHT
      })
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().floatingBorderColor })
      .shadow(this.getPalette().elevatedShadow)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Center)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: Math.max(this.safeAreaBottom, DesignConstants.SPACING.SM) })
  }

  @Builder FloatingTabItem(index: number) {
    Column() {
      Stack({ alignContent: Alignment.TopEnd }) {
        LucideIcon({
          icon: TAB_ITEMS[index].icon,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
          paletteRole: this.currentTabIndex === index ? 'tabActive' : 'tabInactive'
        })
        // 消息 tab 未读角标，与手机端 TabBarBuilder 一致
        if (index === 3 && this.unreadCount > 0) {
          SystemBadge({ count: this.unreadCount })
        }
      }
      Text(TAB_ITEMS[index].label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(
          this.currentTabIndex === index
            ? this.getPalette().accentBackground
            : this.getPalette().tabBarInactive
        )
        .fontWeight(
          this.currentTabIndex === index ? DesignConstants.FONT.WEIGHT.MEDIUM : DesignConstants.FONT.WEIGHT.REGULAR
        )
        .margin({ top: DesignConstants.SPACING.HALF })
    }
    .width(DesignConstants.LAYOUT.LABEL_WIDTH)
    .padding({ top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
    .borderRadius(DesignConstants.RADIUS.LG)
    .backgroundColor(
      this.currentTabIndex === index
        ? this.getPalette().tabBarActiveBackground
        : DesignConstants.COLOR.TRANSPARENT
    )
    // 按压差异：选中 tab 按压不缩放（accent 底已表达选中），未选中按压缩放预览
    .scale(
      this.pressedTabIndex === index && this.currentTabIndex !== index
        ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_PILL, y: DesignConstants.INTERACTION.SCALE_PRESSED_PILL }
        : { x: 1, y: 1 }
    )
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_TAB,
        DesignConstants.INTERACTION.SPRING_DAMPING_TAB
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedTabIndex = index;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedTabIndex = -1;
      }
    })
    .onClick(() => {
      this.tabsController.changeIndex(index);
      this.currentTabIndex = index;
    })
  }

  @Builder FloatingFab() {
    Row() {
      LucideIcon({
        icon: AppIcon.ADD,
        iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
        paletteRole: 'accentText'
      })
    }
    .width(DesignConstants.LAYOUT.FAB_SIZE)
    .height(DesignConstants.LAYOUT.FAB_SIZE)
    .justifyContent(FlexAlign.Center)
    .backgroundColor(this.getPalette().accentBackground)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .shadow(this.getPalette().elevatedShadow)
    .accessibilityText('发布动态')
    // 按压反馈（对齐手机端 CreateTabBuilder FAB 语义）
    .scale(this.tabletFabPressed ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON, y: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON } :
      { x: 1, y: 1 })
    .animation({
      curve: curves.springMotion(
        DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
        DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
      )
    })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.tabletFabPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.tabletFabPressed = false;
      }
    })
    .onClick(() => {
      this.showCreateSheet = true;
    })
  }

  @Builder CreateTabBuilder() {
    Column() {
      Row() {
        LucideIcon({
          icon: AppIcon.ADD,
          iconSize: DesignConstants.COMPONENT.ICON_SIZE_SM,
          paletteRole: 'accentText'
        })
      }
      .width(DesignConstants.LAYOUT.FAB_SIZE).height(DesignConstants.LAYOUT.FAB_SIZE)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(this.getPalette().accentBackground)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .scale(
        this.isCreateTabPressed
          ? { x: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON, y: DesignConstants.INTERACTION.SCALE_PRESSED_BUTTON }
          : { x: 1, y: 1 }
      )
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
          DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
        )
      })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          this.isCreateTabPressed = true;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.isCreateTabPressed = false;
        }
      })
    }
    .height(DesignConstants.LAYOUT.MOBILE_BOTTOM_NAV_HEIGHT)
    .justifyContent(FlexAlign.Center)
    .accessibilityText('发布动态')
    .onClick(() => {
      this.showCreateSheet = true;
    })
  }

  // 发动态完整表单 Sheet（对齐前端 CreateSheet；「创建群组」切路由页，分享位置入口在地图页）
  @Builder CreateSheetContent() {
    CreateMomentSheet({
      onClose: () => { this.showCreateSheet = false; },
      onSwitchToGroup: () => { AppRouter.push(PageNames.CREATE_GROUP); }
    })
  }
}
