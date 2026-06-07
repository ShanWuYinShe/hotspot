/**
 * PhoneLayout — 鸿蒙手机 + 平板布局
 *
 * 适用断点：xs / sm / md（< 840vp）
 * - 顶部：全宽应用栏
 * - 主体：底部 Tab 切换 + 4 个 TabContent（Nearby / Discover / Message / Profile）
 * - 与 frontend TabletLayout "mobile-like UX" 风格一致
 *
 * 使用方式（在 Entry / Index 组件的 build() 中）：
 *   if (isMobileForm(this.currentBreakpoint)) {
 *     PhoneLayout()
 *   } else {
 *     DesktopLayout()
 *   }
 *
 * 注意：PhoneLayout 是普通 @Component，不能用 @Entry（Stage 模型只能有一个入口）
 *
 * 图标：使用系统 SymbolSpan（按 harmonyapp/CLAUDE.md "系统图标" 约束）。
 * 具体资源 ID 需在 DevEco Studio 中以代码补全确认。
 */

import { NearbyPage } from '../../features/nearby/pages/NearbyPage';
import { DiscoverPage } from '../../features/discover/pages/DiscoverPage';
import { MessagePage } from '../../features/message/pages/MessagePage';
import { ProfilePage } from '../../features/profile/pages/ProfilePage';

interface TabItem {
  label: string;
  symbolId: Resource;
}

const TAB_ITEMS: TabItem[] = [
  { label: '附近', symbolId: $r('sys.symbol.location_fill') },
  { label: '发现', symbolId: $r('sys.symbol.compass') },
  { label: '消息', symbolId: $r('sys.symbol.message_fill') },
  { label: '我的', symbolId: $r('sys.symbol.person_fill') }
];

@Component
export struct PhoneLayout {
  @State currentTabIndex: number = 0;

  build() {
    Column() {
      // 顶部应用栏
      Row() {
        Text('HotSpot')
          .fontSize($r('app.float.font_size_lg'))
          .fontWeight(FontWeight.Medium)
          .fontColor($r('app.color.brand_primary'))
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('app.color.bg_primary'))

      // 主体：4 个 Tab 接入真实页面
      Tabs({ barPosition: BarPosition.End, index: this.currentTabIndex }) {
        TabContent() {
          NearbyPage()
        }
        .tabBar(this.TabBarBuilder(0))

        TabContent() {
          DiscoverPage()
        }
        .tabBar(this.TabBarBuilder(1))

        TabContent() {
          MessagePage()
        }
        .tabBar(this.TabBarBuilder(2))

        TabContent() {
          ProfilePage()
        }
        .tabBar(this.TabBarBuilder(3))
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
  TabBarBuilder(index: number) {
    Column() {
      SymbolSpan(TAB_ITEMS[index].symbolId)
        .fontSize(20)
        .fontColor([
          this.currentTabIndex === index
            ? $r('app.color.brand_primary')
            : $r('app.color.chip_inactive_text')
        ])
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
    .height(56)
    .justifyContent(FlexAlign.Center)
  }
}
