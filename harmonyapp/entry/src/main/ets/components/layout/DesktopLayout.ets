/**
 * DesktopLayout — 鸿蒙桌面 / 折叠屏 / PC 布局
 *
 * 适用断点：lg / xl / xxl（>= 840vp）
 * - 左侧：垂直导航栏（与 frontend 桌面 nav rail 对齐）
 * - 中央：主内容区（与 frontend main column 对齐，CONTAINER_MAX_WIDTH_VP 限宽）
 * - 右侧（可选）：详情面板
 *
 * 阶段 2 状态：占位实现
 * 阶段 3：从 features/main/pages/Index.ets 抽取业务内容接入
 *
 * 使用方式：
 *   if (isDesktopForm(currentBreakpoint)) {
 *     DesktopLayout()
 *   } else {
 *     PhoneLayout()
 *   }
 */

import {
  GRID_BREAKPOINTS_VALUE,
  CONTAINER_MAX_WIDTH_VP,
  type BreakpointType
} from '../../core/responsive/Breakpoint';

interface NavItem {
  label: string;
  route: string;
}

const NAV_ITEMS: NavItem[] = [
  { label: '附近', route: 'nearby' },
  { label: '发现', route: 'discover' },
  { label: '消息', route: 'messages' },
  { label: '我的', route: 'profile' }
];

@Entry
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

      // 主内容区
      Scroll() {
        Column() {
          Text('DesktopLayout 占位')
            .fontSize($r('app.float.font_size_2xl'))
            .fontWeight(FontWeight.Medium)
            .fontColor($r('app.color.brand_primary'))
            .margin({ bottom: 12 })
          Text('阶段 3 从 features/main/pages/Index.ets 抽取业务内容接入此布局')
            .fontSize($r('app.float.font_size_sm'))
            .fontColor($r('app.color.chip_inactive_text'))
            .textAlign(TextAlign.Center)
            .padding({ left: 24, right: 24 })
        }
        .width('100%')
        .justifyContent(FlexAlign.Center)
        .padding({ top: 48, bottom: 48 })
      }
      .layoutWeight(1)
      .backgroundColor($r('app.color.bg_primary'))
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
