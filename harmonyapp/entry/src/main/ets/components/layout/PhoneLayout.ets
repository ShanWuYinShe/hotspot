/**
 * PhoneLayout — 鸿蒙手机 + 平板布局（完整版）
 *
 * 适用断点：xs / sm / md（< 840vp）
 *
 * 完整业务：
 * - 4 Tab（首页 / 发现 / 发布FAB / 消息 / 我的）对齐前端底栏导航
 * - 通知中心 / 通知 banner / 通知铃铛按钮
 * - Create Sheet（bindSheet 模式 + 中间「+」按钮）
 * - 性能详情 overlay
 * - Login Prompt（未登录时包装消息 tab）
 * - Upgrade Banner（临时用户时包装消息 tab）
 *
 * 状态传递：通过 @Link 接收 Index.ets 维护的 6 个 @State
 * 业务回调（initApp / WS / 通知同步等）仍由 Index.ets 的 aboutToAppear 触发
 *
 * 图标：使用系统 SymbolSpan（按 harmonyapp/CLAUDE.md "系统图标" 约束）
 */

import { NearbyPage } from '../../features/nearby/pages/NearbyPage';
import { DiscoverPage } from '../../features/discover/pages/DiscoverPage';
import { MessagePage } from '../../features/message/pages/MessagePage';
import { ProfilePage } from '../../features/profile/pages/ProfilePage';
import { LoginPromptView } from '../LoginPromptView';
import { NotificationCenter } from '../NotificationCenter';
import { NotificationBanner } from '../NotificationBanner';
import { PerformanceIndicator } from '../PerformanceIndicator';
import { NotificationRenderer } from '../../core/services/notification/NotificationRenderer';
import { type SmartNotification, type NotificationAction } from '../../core/services/notification/NotificationDataManager';
import { AppThemeManager } from '../../core/theme/AppThemeManager';
import { DesignConstants } from '../../constants/DesignConstants';
import { curves } from '@kit.ArkUI';
import { SystemBadge } from '../SystemBadge';
import { AppIcon } from '../SystemIcon';
import { AppRouter } from '../../core/router/AppRouter';
import { PageNames } from '../../constants/AppConstants';

interface TabItem {
  label: string;
  symbolId: Resource;
}

const TAB_ITEMS: TabItem[] = [
  { label: '首页', symbolId: $r('sys.symbol.house') },
  { label: '发现', symbolId: $r('sys.symbol.compass') },
  { label: '发布', symbolId: $r('sys.symbol.plus') },
  { label: '消息', symbolId: $r('sys.symbol.message') },
  { label: '我的', symbolId: $r('sys.symbol.person') }
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
  @Prop resolvedThemeId: string = 'default_clean_light';

  // 内部状态
  @State showUpgradeBanner: boolean = false;

  private tabsController: TabsController = new TabsController();

  aboutToAppear() {
    this.showUpgradeBanner = this.isLoggedIn && this.userType === 'temp';
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
            }, (item: SmartNotification) => item.id)
          }
          .width('100%')
          .padding({ top: DesignConstants.SPACING.MD, left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        }

        // 主体：Tabs + 5 TabContent（Nearby / Discover / Create / Message / Profile）
        Tabs({ barPosition: BarPosition.End, controller: this.tabsController, index: this.currentTabIndex }) {
          TabContent() { NearbyPage() }.tabBar(this.TabBarBuilder(0))
          TabContent() { DiscoverPage() }.tabBar(this.TabBarBuilder(1))
          TabContent() { Column() {} }.tabBar(this.CreateTabBuilder())
          TabContent() {
            if (this.isLoggedIn) {
              Column() {
                if (this.showUpgradeBanner) { this.UpgradeBanner() }
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
        .barHeight(this.isTablet ? 0 : 56)
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
        .backgroundBlurStyle(BlurStyle.Thin)
        .backgroundColor($r('app.color.bg_primary'))
        .layoutWeight(1)
        .padding({ bottom: this.safeAreaBottom })
        .constraintSize({ maxWidth: this.isTablet ? 720 : Infinity })

        // 平板浮动药丸底栏（Tabs 系统 bar 已隐藏，用自定义浮动药丸替代，对齐前端 TabletBottomNav）
        if (this.isTablet) {
          this.buildFloatingTabletNav()
        }
      }
      .width('100%')
      .height('100%')

      // 通知中心 overlay（模态遮罩，保留在叠加层）
      if (this.showNotificationCenter) {
        Stack() {
          Column()
            .width('100%').height('100%')
            .backgroundColor('#66000000')
            .onClick(() => { this.showNotificationCenter = false; })
          Column() {
            NotificationCenter({ onClose: () => { this.showNotificationCenter = false; } })
          }
          .width('100%').height('100%')
          .backgroundColor($r('app.color.bg_primary'))
        }
        .width('100%').height('100%')
      }

      // 性能详情 overlay（模态，保留在叠加层）
      if (this.showPerformanceDetails) {
        Column() {
          PerformanceIndicator({ showDetails: true, compact: false })
        }
        .width('100%').height('100%')
        .backgroundColor($r('app.color.bg_primary'))
        .onClick(() => { this.showPerformanceDetails = false; })
      }

    }
    .bindSheet($$this.showCreateSheet, this.CreateSheetContent(), {
      height: SheetSize.FIT_CONTENT,
      dragBar: true,
      showClose: true,
      backgroundColor: $r('app.color.bg_primary')
    })
    .width('100%').height('100%')
    .backgroundColor($r('app.color.bg_primary'))
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  @Builder TabBarBuilder(index: number) {
    Column() {
      Column() {
        Stack({ alignContent: Alignment.TopEnd }) {
          Text() {
            SymbolSpan(TAB_ITEMS[index].symbolId)
              .fontSize(DesignConstants.FONT.SIZE.XXL)
              .fontColor([
                this.currentTabIndex === index
                  ? $r('app.color.brand_primary')
                  : $r('app.color.chip_inactive_text')
              ])
          }
          // 消息 tab（index 3）未读角标，对齐前端 inbox badge
          if (index === 3 && this.unreadCount > 0) {
            SystemBadge({ count: this.unreadCount })
          }
        }
        .width(36)
        .height(36)
        .alignContent(Alignment.TopEnd)
        Text(TAB_ITEMS[index].label)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(
            this.currentTabIndex === index
              ? $r('app.color.brand_primary')
              : $r('app.color.chip_inactive_text')
          )
          .fontWeight(
            this.currentTabIndex === index ? FontWeight.Medium : FontWeight.Regular
          )
          .margin({ top: 2 })
      }
      .width(60)
      .padding({ top: 6, bottom: 6 })
      .borderRadius(DesignConstants.RADIUS.LG)
      .backgroundColor(
        this.currentTabIndex === index
          ? this.getPalette().tabBarActiveBackground
          : DesignConstants.COLOR.TRANSPARENT
      )
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_TAB,
          DesignConstants.INTERACTION.SPRING_DAMPING_TAB
        )
      })
    }
    .height(56)
    .justifyContent(FlexAlign.Center)
    .onClick(() => {
      if (index !== CREATE_TAB_INDEX) {
        this.tabsController.changeIndex(index);
        this.currentTabIndex = index;
      }
    })
  }

  // 平板浮动药丸底栏（对齐前端 TabletBottomNav：max-width 520 + pill 圆角 + 玻璃 blur + safe-area）
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
      .constraintSize({ maxWidth: 520 })
      .justifyContent(FlexAlign.SpaceAround)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getPalette().tabBarBackground)
      .backgroundBlurStyle(BlurStyle.Thin)
      .border({ width: 1, color: this.getPalette().floatingBorderColor })
      .shadow(this.getPalette().elevatedShadow)
    }
    .width('100%')
    .alignItems(HorizontalAlign.Center)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.SM, bottom: Math.max(this.safeAreaBottom, DesignConstants.SPACING.SM) })
  }

  @Builder FloatingTabItem(index: number) {
    Column() {
      Text() {
        SymbolSpan(TAB_ITEMS[index].symbolId)
          .fontSize(DesignConstants.FONT.SIZE.XL_PLUS)
          .fontColor([
            this.currentTabIndex === index
              ? $r('app.color.brand_primary')
              : $r('app.color.chip_inactive_text')
          ])
      }
      Text(TAB_ITEMS[index].label)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(
          this.currentTabIndex === index
            ? $r('app.color.brand_primary')
            : $r('app.color.chip_inactive_text')
        )
        .margin({ top: 2 })
    }
    .padding({ top: 6, bottom: 6 })
    .justifyContent(FlexAlign.Center)
    .onClick(() => {
      this.tabsController.changeIndex(index);
      this.currentTabIndex = index;
    })
  }

  @Builder FloatingFab() {
    Row() {
      Text() {
        SymbolSpan($r('sys.symbol.plus'))
          .fontSize(DesignConstants.FONT.SIZE.XL_PLUS)
          .fontColor([$r('app.color.bg_primary')])
      }
    }
    .width(48)
    .height(48)
    .justifyContent(FlexAlign.Center)
    .backgroundColor($r('app.color.brand_primary'))
    .borderRadius(DesignConstants.RADIUS.FULL)
    .shadow(this.getPalette().accentShadow)
    .onClick(() => {
      this.showCreateSheet = true;
    })
  }

  @Builder CreateTabBuilder() {
    Column() {
      Row() {
        Text() {
          SymbolSpan($r('sys.symbol.plus'))
            .fontSize(DesignConstants.FONT.SIZE.XL_PLUS)
            .fontColor([$r('app.color.bg_primary')])
        }
      }
      .width(52).height(52)
      .justifyContent(FlexAlign.Center)
      .backgroundColor($r('app.color.brand_primary'))
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
    .height(56)
    .justifyContent(FlexAlign.Center)
    .onClick(() => {
      this.showCreateSheet = true;
    })
  }

  @Builder CreateSheetContent() {
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

  @Builder CreateSheetItem(icon: AppIcon, label: string, action: () => void) {
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
    .borderRadius(DesignConstants.SHAPE.INPUT) // 18
    .backgroundColor(this.getPalette().surfaceSubtle)
    .alignItems(VerticalAlign.Center)
    .margin({ bottom: DesignConstants.SPACING.SM })
    .onClick(() => action())
  }

  @Builder UpgradeBanner() {
    Row() {
      Text('当前为临时用户，注册账号可保留所有数据')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor($r('app.color.chip_inactive_text'))
        .layoutWeight(1)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
      Button('去注册')
        .type(ButtonType.Normal)
        .height(28)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .backgroundColor($r('app.color.brand_primary'))
        .fontColor($r('app.color.bg_primary'))
        .borderRadius(DesignConstants.RADIUS.SM)
        .padding({ left: 12, right: 12 })
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 8, bottom: 8 })
    .backgroundColor($r('app.color.chip_selected_bg'))
  }
}
