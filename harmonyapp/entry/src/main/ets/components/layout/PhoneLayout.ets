/**
 * PhoneLayout — 鸿蒙手机 + 平板布局（完整版）
 *
 * 适用断点：xs / sm / md（< 840vp）
 *
 * 完整业务：
 * - 4 Tab（Nearby / Discover / Create / Message / Profile）
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

interface TabItem {
  label: string;
  symbolId: Resource;
}

const TAB_ITEMS: TabItem[] = [
  { label: '附近', symbolId: $r('sys.symbol.position') },
  { label: '发现', symbolId: $r('sys.symbol.compass') },
  { label: '发布', symbolId: $r('sys.symbol.plus') },
  { label: '消息', symbolId: $r('sys.symbol.message_fill') },
  { label: '我的', symbolId: $r('sys.symbol.person_fill') }
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
  @Prop resolvedThemeId: string = 'figma_neu_light';

  // 内部状态
  @State showUpgradeBanner: boolean = false;

  private tabsController: TabsController = new TabsController();

  aboutToAppear() {
    this.showUpgradeBanner = this.isLoggedIn && this.userType === 'temp';
  }

  build() {
    Stack({ alignContent: Alignment.Top }) {
      // 主体：Tabs + 5 TabContent（Nearby / Discover / Create / Message / Profile）
      Tabs({ barPosition: BarPosition.End, controller: this.tabsController, index: this.currentTabIndex }) {
        TabContent() { NearbyPage() }.tabBar(this.TabBarBuilder(0))
        TabContent() { Stack() { DiscoverPage(); this.DiscoverNotificationButton() } }.tabBar(this.TabBarBuilder(1))
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
      .barHeight(56)
      .scrollable(false)
      .barMode(BarMode.Fixed)
      .divider({ strokeWidth: 0, color: Color.Transparent, startMargin: 0, endMargin: 0 })
      .onChange((index: number) => {
        if (index === CREATE_TAB_INDEX) {
          this.showCreateSheet = true;
          return;
        }
        this.currentTabIndex = index;
      })
      .barBackgroundColor(Color.Transparent)
      .backgroundColor($r('app.color.bg_primary'))
      .width('100%')
      .height('100%')
      .padding({ bottom: this.safeAreaBottom })
      .constraintSize({ maxWidth: this.isTablet ? 720 : Infinity })

      // 通知 banner overlay（顶部）
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
        .padding({ top: 12, left: 16, right: 16 })
      }

      // 通知中心 overlay
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

      // 性能详情 overlay
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
        Text() {
          SymbolSpan(TAB_ITEMS[index].symbolId)
            .fontSize(24)
            .fontColor([
              this.currentTabIndex === index
                ? $r('app.color.brand_primary')
                : $r('app.color.chip_inactive_text')
            ])
        }
        Text(TAB_ITEMS[index].label)
          .fontSize($r('app.float.font_size_xs'))
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
      .borderRadius(16)
      .backgroundColor(
        this.currentTabIndex === index
          ? this.getPalette().tabBarActiveBackground
          : Color.Transparent
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

  @Builder CreateTabBuilder() {
    Column() {
      Row() {
        Text() {
          SymbolSpan($r('sys.symbol.plus'))
            .fontSize(20)
            .fontColor([
              this.currentTabIndex === CREATE_TAB_INDEX
                ? $r('app.color.bg_primary')
                : $r('app.color.brand_primary')
            ])
        }
      }
      .width(40).height(40)
      .justifyContent(FlexAlign.Center)
      .backgroundColor(
        this.currentTabIndex === CREATE_TAB_INDEX
          ? $r('app.color.brand_primary')
          : $r('app.color.bg_secondary')
      )
      .borderRadius($r('app.float.radius_full'))
      .border({ width: 1, color: $r('app.color.brand_primary') })
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
      this.currentTabIndex = CREATE_TAB_INDEX;
      this.showCreateSheet = true;
    })
  }

  @Builder DiscoverNotificationButton() {
    Row() {
      Blank().layoutWeight(1)
      Stack({ alignContent: Alignment.TopEnd }) {
        Button() {
          Text() {
            SymbolSpan(
              this.unreadCount > 0
                ? $r('sys.symbol.bell_fill')
                : $r('sys.symbol.bell')
            )
              .fontSize(18)
              .fontColor([$r('app.color.brand_primary')])
          }
      }
        .type(ButtonType.Normal)
        .width(40).height(40)
        .backgroundColor($r('app.color.bg_secondary'))
        .borderRadius($r('app.float.radius_full'))
        .border({ width: 1, color: $r('app.color.divider') })
        .scale(this.isNotifBtnPressed ? { x: 0.9, y: 0.9 } : { x: 1, y: 1 })
        .onClick(() => { this.showNotificationCenter = true; })

        if (this.unreadCount > 0) {
          Text(this.unreadCount > 99 ? '99+' : `${this.unreadCount}`)
            .fontSize(10)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('app.color.bg_primary'))
            .padding({ left: 6, right: 6, top: 2, bottom: 2 })
            .backgroundColor($r('app.color.brand_primary'))
            .borderRadius($r('app.float.radius_full'))
            .translate({ x: 8, y: -6 })
        }
      }
      .margin({ top: this.safeAreaTop > 0 ? this.safeAreaTop + 12 : 24, right: 16 })
    }
    .width('100%')
    .position({ x: 0, y: 0 })
  }

  @Builder CreateSheetContent() {
    Column() {
      Text('发布动态')
        .fontSize($r('app.float.font_size_lg'))
        .fontWeight(FontWeight.Medium)
        .fontColor($r('app.color.brand_primary'))
        .margin({ top: 24, bottom: 16 })
      Button('发布文字动态')
        .type(ButtonType.Normal)
        .width('90%').height(48)
        .backgroundColor($r('app.color.brand_primary'))
        .fontColor($r('app.color.bg_primary'))
        .borderRadius($r('app.float.radius_md'))
        .margin({ bottom: 12 })
        .onClick(() => { this.showCreateSheet = false; })
      Button('发布图片动态')
        .type(ButtonType.Normal)
        .width('90%').height(48)
        .backgroundColor($r('app.color.bg_secondary'))
        .fontColor($r('app.color.brand_primary'))
        .borderRadius($r('app.float.radius_md'))
        .onClick(() => { this.showCreateSheet = false; })
    }
    .width('100%')
    .padding({ bottom: 24 })
  }

  @Builder UpgradeBanner() {
    Row() {
      Text('当前为临时用户，注册账号可保留所有数据')
        .fontSize($r('app.float.font_size_sm'))
        .fontColor($r('app.color.chip_inactive_text'))
        .layoutWeight(1)
        .maxLines(1)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
      Button('去注册')
        .type(ButtonType.Normal)
        .height(28)
        .fontSize($r('app.float.font_size_sm'))
        .backgroundColor($r('app.color.brand_primary'))
        .fontColor($r('app.color.bg_primary'))
        .borderRadius($r('app.float.radius_sm'))
        .padding({ left: 12, right: 12 })
    }
    .width('100%')
    .padding({ left: 16, right: 16, top: 8, bottom: 8 })
    .backgroundColor($r('app.color.chip_selected_bg'))
  }
}
