/**
 * DesktopLayout — 鸿蒙桌面 / 折叠屏 / PC 布局（完整版）
 *
 * 适用断点：lg / xl / xxl（>= 840vp）
 *
 * 完整业务：
 * - 左侧 nav rail（4 路由：首页 / 发现 / 消息 / 我的），宽度 192vp 对齐前端侧边栏
 * - 中央主内容（按 currentRoute 切换）
 * - 右侧详情面板（xl/xxl 触发），宽度 280vp
 * - 通知中心 / 通知 banner overlay
 * - 性能详情 overlay
 * - Login Prompt 包装（未登录时）
 * - Upgrade Banner（临时用户时）
 *
 * 状态传递：通过 @Link 接收 Index.ets 维护的 4 个 @State
 * 业务回调仍由 Index.ets 的 aboutToAppear 触发
 *
 * 注意：DesktopLayout 是普通 @Component，不能用 @Entry
 * 图标：使用系统 SymbolSpan
 */

import {
  CONTAINER_MAX_WIDTH_VP,
  type BreakpointType
} from '../../core/responsive/Breakpoint';
import { DesignConstants } from '../../constants/DesignConstants';
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

interface NavItem {
  label: string;
  route: string;
  symbolId: Resource;
}

const NAV_ITEMS: NavItem[] = [
  { label: '首页', route: 'nearby', symbolId: $r('sys.symbol.house_fill') },
  { label: '发现', route: 'discover', symbolId: $r('sys.symbol.compass') },
  { label: '消息', route: 'messages', symbolId: $r('sys.symbol.message_fill') },
  { label: '我的', route: 'profile', symbolId: $r('sys.symbol.person_fill') }
];

@Component
export struct DesktopLayout {
  // 来自父组件的核心状态
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

  // 内部状态
  @State showUpgradeBanner: boolean = false;

  aboutToAppear() {
    this.showUpgradeBanner = this.isLoggedIn && this.userType === 'temp';
  }

  build() {
    Stack() {
      Column() {
        // 通知 banner（流式排布，把下方内容往下推，不再浮盖）
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

        Row() {
          Column() {
            this.LeftNavRail()
          }
            .width(192)
            .height('100%')

          Column() {
            this.MainContent()
          }
            .layoutWeight(1)
            .height('100%')

          if (this.shouldShowRightPanel()) {
            Column() {
              this.RightDetailPanel()
            }
              .width(280)
              .height('100%')
          }
        }
        .layoutWeight(1)
        .width('100%')
        .backgroundColor($r('app.color.bg_primary'))
      }
      .width('100%')
      .height('100%')

      // 通知中心 overlay（模态遮罩，保留叠加）
      if (this.showNotificationCenter) {
        Stack() {
          Column()
            .width('100%')
            .height('100%')
            .backgroundColor('#66000000')
            .onClick(() => { this.showNotificationCenter = false; })

          Column() {
            NotificationCenter({
              onClose: () => { this.showNotificationCenter = false; }
            })
          }
          .width('100%')
          .height('100%')
          .backgroundColor($r('app.color.bg_primary'))
        }
        .width('100%')
        .height('100%')
      }

      // 性能详情 overlay（模态，保留叠加）
      if (this.showPerformanceDetails) {
        Column() {
          PerformanceIndicator({ showDetails: true, compact: false })
        }
        .width('100%')
        .height('100%')
        .backgroundColor($r('app.color.bg_primary'))
        .onClick(() => { this.showPerformanceDetails = false; })
      }
    }
    .width('100%')
    .height('100%')
  }

  @Builder
  LeftNavRail() {
    Column() {
      Text('HotSpot')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('app.color.brand_primary'))
        .margin({ top: 24, bottom: 32 })

      ForEach(NAV_ITEMS, (item: NavItem) => {
        this.NavItemBuilder(item)
      }, (item: NavItem) => item.route)

      // 通知铃铛入口
      Blank().layoutWeight(1)
      Row() {
        Text() {
          SymbolSpan(
            this.unreadCount > 0
              ? $r('sys.symbol.bell_fill')
              : $r('sys.symbol.bell')
          )
            .fontSize(20)
            .fontColor([$r('app.color.brand_primary')])
        }
        Text('通知')
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor($r('app.color.brand_primary'))
          .margin({ left: 12 })
      }
      .width('100%')
      .height(44)
      .padding({ left: 12, right: 12 })
      .borderRadius(DesignConstants.RADIUS.MD)
      .backgroundColor($r('app.color.chip_selected_bg'))
      .margin({ bottom: 24 })
      .onClick(() => {
        this.showNotificationCenter = true;
      })
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_secondary'))
    .padding({ left: 16, right: 16, top: 16 })
  }

  @Builder
  NavItemBuilder(item: NavItem) {
    Row() {
      Text() {
        SymbolSpan(item.symbolId)
          .fontSize(20)
          .fontColor([
            this.currentRoute === item.route
              ? $r('app.color.brand_primary')
              : $r('app.color.chip_inactive_text')
          ])
      }
      Text(item.label)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor(
          this.currentRoute === item.route
            ? $r('app.color.brand_primary')
            : $r('app.color.chip_inactive_text')
        )
        .fontWeight(
          this.currentRoute === item.route ? FontWeight.Medium : FontWeight.Regular
        )
        .margin({ left: 12 })
    }
    .width('100%')
    .height(44)
    .padding({ left: 12, right: 12 })
    .borderRadius(DesignConstants.RADIUS.MD)
    .backgroundColor(
      this.currentRoute === item.route ? $r('app.color.chip_selected_bg') : Color.Transparent
    )
    .onClick(() => {
      this.currentRoute = item.route;
    })
  }

  @Builder
  MainContent() {
    Column() {
      // 顶部应用栏（含标题 + 通知铃铛 badge）
      Row() {
        Text(this.getCurrentTitle())
          .fontSize(DesignConstants.FONT.SIZE.XXL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('app.color.brand_primary'))

        Blank().layoutWeight(1)

        if (this.unreadCount > 0) {
          Text(`${this.unreadCount} 未读`)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('app.color.chip_inactive_text'))
        }
      }
      .width('100%')
      .height(64)
      .padding({ left: 24, right: 24 })
      .backgroundColor($r('app.color.bg_primary'))

      // 主内容区
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
            .fontColor($r('app.color.chip_inactive_text'))
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      }
    }
    .width('100%')
    .height('100%')
    .constraintSize({ maxWidth: this.getContainerMaxWidth() })
  }

  private getContainerMaxWidth(): string {
    switch (this.currentBreakpoint) {
      case 'md':
        return `${CONTAINER_MAX_WIDTH_VP.md}vp`;
      case 'lg':
        return `${CONTAINER_MAX_WIDTH_VP.lg}vp`;
      case 'xl':
        return `${CONTAINER_MAX_WIDTH_VP.xl}vp`;
      case 'xxl':
        return `${CONTAINER_MAX_WIDTH_VP.xxl}vp`;
      case 'xs':
      case 'sm':
      default:
        return '100%';
    }
  }

  @Builder
  RightDetailPanel() {
    Column() {
      Text('详情面板')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('app.color.chip_inactive_text'))
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_secondary'))
    .padding(16)
    .justifyContent(FlexAlign.Start)
  }

  @Builder
  UpgradeBanner() {
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

  shouldShowRightPanel(): boolean {
    return this.currentBreakpoint === 'xl' || this.currentBreakpoint === 'xxl';
  }

  getCurrentTitle(): string {
    const item = NAV_ITEMS.find((it) => it.route === this.currentRoute);
    return item ? item.label : 'HotSpot';
  }
}
