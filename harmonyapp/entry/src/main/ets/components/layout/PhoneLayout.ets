/**
 * PhoneLayout — 鸿蒙手机 + 平板布局
 *
 * 适用断点：xs / sm / md（< 840vp）
 * - 顶部：全宽应用栏（与 frontend StatusBar 对齐）
 * - 主体：底部 Tab 切换 + 4 个 TabContent（附近 / 发现 / 消息 / 我的）
 * - 与 frontend TabletLayout "mobile-like UX" 风格一致
 *
 * 阶段 2 状态：占位实现（4 个 TabContent 各显示一句占位文字）
 * 阶段 3：将 features/main/pages/Index.ets 中手机端逻辑迁移至此
 *
 * 使用方式：
 *   if (isMobileForm(currentBreakpoint)) {
 *     PhoneLayout()
 *   } else {
 *     DesktopLayout()
 *   }
 */

import { GRID_BREAKPOINTS_VALUE } from '../../core/responsive/Breakpoint';

@Entry
@Component
export struct PhoneLayout {
  @State currentTabIndex: number = 0;

  build() {
    Column() {
      // 顶部状态栏占位（与 frontend StatusBar 对齐；阶段 3 接入）
      Row() {
        Text('HotSpot')
          .fontSize($r('app.float.font_size_lg'))
          .fontWeight(FontWeight.Medium)
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('app.color.bg_primary'))

      // 主体：4 个 Tab
      Tabs({ barPosition: BarPosition.End, index: this.currentTabIndex }) {
        TabContent() {
          this.PlaceholderContent('附近', 'PhoneLayout 占位 — 阶段 3 从 Index.ets 接入 NearbyPage')
        }
        .tabBar(this.TabBarBuilder('附近', 0))

        TabContent() {
          this.PlaceholderContent('发现', 'PhoneLayout 占位 — 阶段 3 接入 DiscoverPage')
        }
        .tabBar(this.TabBarBuilder('发现', 1))

        TabContent() {
          this.PlaceholderContent('消息', 'PhoneLayout 占位 — 阶段 3 接入 MessagePage')
        }
        .tabBar(this.TabBarBuilder('消息', 2))

        TabContent() {
          this.PlaceholderContent('我的', 'PhoneLayout 占位 — 阶段 3 接入 ProfilePage')
        }
        .tabBar(this.TabBarBuilder('我的', 3))
      }
      .barMode(BarMode.Fixed)
      .barBackgroundColor($r('app.color.tab_bar_bg'))
      .onChange((index: number) => {
        this.currentTabIndex = index;
      })
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
  }

  @Builder
  TabBarBuilder(label: string, _index: number) {
    Column() {
      Text(label)
        .fontSize($r('app.float.font_size_sm'))
        .fontColor(
          this.currentTabIndex === _index
            ? $r('app.color.brand_primary')
            : $r('app.color.chip_inactive_text')
        )
        .fontWeight(
          this.currentTabIndex === _index ? FontWeight.Medium : FontWeight.Regular
        )
    }
    .height(56)
    .justifyContent(FlexAlign.Center)
  }

  @Builder
  PlaceholderContent(title: string, hint: string) {
    Column() {
      Text(title)
        .fontSize($r('app.float.font_size_3xl'))
        .fontWeight(FontWeight.SemiBold)
        .fontColor($r('app.color.brand_primary'))
        .margin({ bottom: 12 })
      Text(hint)
        .fontSize($r('app.float.font_size_sm'))
        .fontColor($r('app.color.chip_inactive_text'))
        .textAlign(TextAlign.Center)
        .padding({ left: 32, right: 32 })
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
  }
}
