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
import { AppRouter, LoginRouteParam } from '../../core/router/AppRouter';
import { PageNames } from '../../constants/AppConstants';
import { CreateMomentSheet } from '../../features/moments/components/CreateMomentSheet';

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
  @Link isNotifBtnPressed: boolean;

  // 来自父组件的只读状态
  @Prop unreadCount: number = 0;
  @Prop visibleNotifications: SmartNotification[] = [];
  @Prop isLoggedIn: boolean = false;
  @Prop userType: string = '';
  @Prop safeAreaBottom: number = 0;
  @Prop safeAreaTop: number = 0;
  @Prop isTablet: boolean = false;
  @Prop resolvedThemeId: string = 'vibrant_light';
  /** 实时断点（AppStorage 写入），用于在 PhoneLayout 内决策宽屏通知侧栏 */
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  // 内部状态
  /** 当前按下的底部导航 Tab 索引（-1 表示无），用于按下缩放反馈，对齐前端 hover-btn:active */
  @State pressedTabIndex: number = -1;
  @State private tabletFabPressed: boolean = false;

  private tabsController: TabsController = new TabsController();

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
          TabContent() { NearbyPage() }.tabBar(this.TabBarBuilder(1))
          TabContent() { Column() {} }.tabBar(this.CreateTabBuilder())
          TabContent() {
            if (this.isLoggedIn) {
              Column() {
                MessagePage()
              }.width('100%').height('100%')
            } else {
              LoginPromptView({
                title: '消息功能需要登录',
                subtitle: '登录后可与好友实时聊天、加入群组讨论'
              })
            }
          }.tabBar(this.TabBarBuilder(3))
          TabContent() { ProfilePage() }.tabBar(this.TabBarBuilder(4))
        }
        .barHeight(this.isTablet ? 0 : DesignConstants.LAYOUT.MOBILE_BOTTOM_NAV_HEIGHT)
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
        .barBackgroundColor(DesignConstants.COLOR.TRANSPARENT)
        // 底部导航毛玻璃：Medium 档（对齐前端 --cst-blur-lg 底部导航档，见 AGENTS 映射表）
        // 背景改半透明（对齐前端 nav-glass 90%），否则实色会盖住 backgroundBlurStyle 使模糊失效
        .backgroundBlurStyle(BlurStyle.Thick)
        .backgroundColor(AppThemeVisuals.withAlpha(this.getPalette().tabBarBackground, 0.9))
        .layoutWeight(1)
        .padding({ bottom: this.safeAreaBottom })
        // 不在 Tabs 容器再加 maxWidth 约束：各页面通过 PageShellContainer / ResponsiveContainer
        // 按断点自管宽度。原先 isTablet ? 720 会把 ≥960vp 开启的 master-detail 双栏
        // （HomePage / MomentsFeedPage 右栏 420/220）挤在 720vp 内、左侧被压到 300/500vp，
        // 与 useMasterDetail 的设计意图冲突。

        // 平板浮动药丸底栏（Tabs 系统 bar 已隐藏，用自定义浮动药丸替代，对齐前端 TabletBottomNav）
        if (this.isTablet) {
          this.buildFloatingTabletNav()
        }
      }
      .width('100%')
      .height('100%')

      // 通知中心 overlay：
      // - 窄屏（< 960vp）：全屏模态遮罩（与原行为一致，遮罩外点击关闭）
      // - 宽屏（≥ 960vp / 平板横屏 / 折叠屏展开）：右侧滑出侧栏卡片（约 420vp），
      //   整页不被通知中心全屏接管（与「桌面侧边栏常驻 + 通知/详情内嵌」心智一致）
      if (this.showNotificationCenter) {
        if (isWideForm(this.currentBreakpoint)) {
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
            iconSize: DesignConstants.FONT.SIZE.XXL,
            paletteRole: this.currentTabIndex === index ? 'tabActive' : 'tabInactive'
          })
          // 消息 tab（index 3）未读角标，对齐前端 inbox badge
          if (index === 3 && this.unreadCount > 0) {
            SystemBadge({ count: this.unreadCount })
          }
        }
        Text(TAB_ITEMS[index].label)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(
            this.currentTabIndex === index
              ? this.getPalette().accentBackground
              : this.getPalette().tabBarInactive
          )
          .fontWeight(
            this.currentTabIndex === index ? DesignConstants.FONT.WEIGHT.SEMIBOLD : DesignConstants.FONT.WEIGHT.REGULAR
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
      .backgroundColor(AppThemeVisuals.withAlpha(this.getPalette().tabBarBackground, 0.9))
      // 平板浮动药丸：Medium 档毛玻璃（对齐前端 tablet dock --cst-blur-lg）
      .backgroundBlurStyle(BlurStyle.Thick)
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
          iconSize: DesignConstants.FONT.SIZE.XL,
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
        iconSize: DesignConstants.FONT.SIZE.XL,
        paletteRole: 'accentText'
      })
    }
    .width(DesignConstants.LAYOUT.FAB_SIZE)
    .height(DesignConstants.LAYOUT.FAB_SIZE)
    .justifyContent(FlexAlign.Center)
    .backgroundColor(this.getPalette().accentBackground)
    .borderRadius(DesignConstants.RADIUS.FULL)
    .shadow(this.getPalette().accentShadow)
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
          iconSize: DesignConstants.FONT.SIZE.XL,
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
