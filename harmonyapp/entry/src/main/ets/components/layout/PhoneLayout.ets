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
 */

import { NearbyPage } from '../../features/nearby/pages/NearbyPage';
import { DiscoverPage } from '../../features/discover/pages/DiscoverPage';
import { MessagePage } from '../../features/message/pages/MessagePage';
import { ProfilePage } from '../../features/profile/pages/ProfilePage';

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
        .tabBar(this.TabBarBuilder('附近', 0))

        TabContent() {
          DiscoverPage()
        }
        .tabBar(this.TabBarBuilder('发现', 1))

        TabContent() {
          MessagePage()
        }
        .tabBar(this.TabBarBuilder('消息', 2))

        TabContent() {
          ProfilePage()
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
}
