/**
 * DesktopLayout — 鸿蒙桌面 / 折叠屏 / PC 布局
 *
 * 适用断点：lg / xl / xxl（>= 840vp）
 * - 左侧：垂直导航栏（与 frontend 桌面 nav rail 对齐）
 * - 中央：主内容区（与 frontend main column 对齐，CONTAINER_MAX_WIDTH_VP 限宽）
 * - 右侧（可选）：详情面板（xl/xxl 触发）
 *
 * 使用方式（在 Entry / Index 组件的 build() 中）：
 *   if (isDesktopForm(this.currentBreakpoint)) {
 *     DesktopLayout()
 *   } else {
 *     PhoneLayout()
 *   }
 *
 * 注意：DesktopLayout 是普通 @Component，不能用 @Entry（Stage 模型只能有一个入口）
 *
 * 阶段 3 状态：4 个 nav 路由中只接入了 nearby（demo）；discover / messages / profile
 * 占位（用 Text 显示路由名 + "占位"提示）
 *
 * TODO 阶段 4：用 SymbolSpan / SymbolGlyph 替换 NavItemBuilder 内的 Text label
 * （与 frontend lucide-react 图标语义对齐）
 */

import {
  CONTAINER_MAX_WIDTH_VP,
  type BreakpointType
} from '../../core/responsive/Breakpoint';
import { NearbyPage } from '../../features/nearby/pages/NearbyPage';
import { DiscoverPage } from '../../features/discover/pages/DiscoverPage';
import { MessagePage } from '../../features/message/pages/MessagePage';
import { ProfilePage } from '../../features/profile/pages/ProfilePage';

interface NavItem {
  label: string;
  route: string;
  symbolId: Resource;
}

const NAV_ITEMS: NavItem[] = [
  { label: '附近', route: 'nearby', symbolId: $r('sys.symbol.location_fill') },
  { label: '发现', route: 'discover', symbolId: $r('sys.symbol.compass') },
  { label: '消息', route: 'messages', symbolId: $r('sys.symbol.message_fill') },
  { label: '我的', route: 'profile', symbolId: $r('sys.symbol.person_fill') }
];

@Component
export struct DesktopLayout {
  @State currentRoute: string = 'nearby';
  @State currentBreakpoint: BreakpointType = 'lg';

  build() {
    SideBarContainer(SideBarContainerType.Embed) {
      // 左侧导航栏
      this.LeftNavRail()

      // 中央主内容
      this.MainContent()

      // 右侧详情面板（可选；x-breakpoint 时不显示）
    } {
      if (this.shouldShowRightPanel()) {
        this.RightDetailPanel()
      }
    }
    .showSideBar(true)
    .showControlButton(false)
    .sideBarWidth(240)
    .controlButton({ icons: { hidden: '', shown: '' }, pos: { top: 0, left: 0 } })
    .backgroundColor($r('app.color.bg_primary'))
    .width('100%')
    .height('100%')
  }

  @Builder
  LeftNavRail() {
    Column() {
      // Logo / 标题
      Text('HotSpot')
        .fontSize($r('app.float.font_size_xl'))
        .fontWeight(FontWeight.Bold)
        .fontColor($r('app.color.brand_primary'))
        .margin({ top: 24, bottom: 32 })

      // 导航项
      ForEach(NAV_ITEMS, (item: NavItem) => {
        this.NavItemBuilder(item)
      }, (item: NavItem) => item.route)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_secondary'))
    .padding({ left: 16, right: 16 })
  }

  @Builder
  NavItemBuilder(item: NavItem) {
    Row() {
      SymbolSpan(item.symbolId)
        .fontSize(20)
        .fontColor([
          this.currentRoute === item.route
            ? $r('app.color.brand_primary')
            : $r('app.color.chip_inactive_text')
        ])
      Text(item.label)
        .fontSize($r('app.float.font_size_base'))
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
    .borderRadius($r('app.float.radius_md'))
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
      // 顶部应用栏
      Row() {
        Text(this.getCurrentTitle())
          .fontSize($r('app.float.font_size_2xl'))
          .fontWeight(FontWeight.SemiBold)
          .fontColor($r('app.color.brand_primary'))
      }
      .width('100%')
      .height(64)
      .padding({ left: 24, right: 24 })
      .backgroundColor($r('app.color.bg_primary'))

      // 主内容区：4 路由全部接入（nearby / discover / messages / profile）
      if (this.currentRoute === 'nearby') {
        NearbyPage()
      } else if (this.currentRoute === 'discover') {
        DiscoverPage()
      } else if (this.currentRoute === 'messages') {
        MessagePage()
      } else if (this.currentRoute === 'profile') {
        ProfilePage()
      } else {
        Column() {
          Text('未知路由')
            .fontSize($r('app.float.font_size_lg'))
            .fontColor($r('app.color.chip_inactive_text'))
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      }
    }
    .width('100%')
    .height('100%')
    .constraintSize({ maxWidth: `${CONTAINER_MAX_WIDTH_VP[this.currentBreakpoint]}vp` })
  }

  @Builder
  RightDetailPanel() {
    Column() {
      Text('详情面板')
        .fontSize($r('app.float.font_size_lg'))
        .fontWeight(FontWeight.Medium)
        .fontColor($r('app.color.chip_inactive_text'))
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_secondary'))
    .padding(16)
    .justifyContent(FlexAlign.Start)
  }

  shouldShowRightPanel(): boolean {
    return this.currentBreakpoint === 'xl' || this.currentBreakpoint === 'xxl';
  }

  getCurrentTitle(): string {
    const item = NAV_ITEMS.find((it) => it.route === this.currentRoute);
    return item ? item.label : 'HotSpot';
  }
}
