/**
 * 搜索页面
 * 提供统一的搜索入口
 */

import { SmartSearchComponent } from '../components/SmartSearchComponent';
import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF14;
const TAG: string = 'SearchPage';

@Entry
@Component
struct SearchPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();

  aboutToAppear() {
    hilog.info(DOMAIN_NUMBER, TAG, '搜索页面初始化');
    
    // 订阅状态变化
    this.stateManager.subscribe('search_page', (state: GlobalState) => {
      this.globalState = state;
    });
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('search_page');
  }

  build() {
    Column() {
      // 顶部导航栏
      Row() {
        Button('←')
          .width(40)
          .height(40)
          .fontSize(20)
          .backgroundColor(Color.Transparent)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .onClick(() => {
            this.getUIContext().getRouter().back();
          })

        Text('搜索')
          .fontSize(20)
          .fontWeight(FontWeight.Bold)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        // 占位符，保持布局平衡
        Row()
          .width(40)
          .height(40)
      }
      .width('100%')
      .height(56)
      .padding({ left: 8, right: 8 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 搜索组件
      SmartSearchComponent()
        .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_background'))
  }
}



