/**
 * 调试面板
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemCard } from '../../../components/SystemCard';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { SystemToast } from '../../../components/SystemToast';
import { MyProfile } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { AppRouter } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';

@Component
export struct DebugPanel {
  pathStack: NavPathStack = new NavPathStack();
  @State logs: string[] = [];
  @State currentUserId: string | null = null;
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  private webSocketService = WebSocketService.getInstance();

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    const uid = await AuthAccountManager.getCurrentUserId();
    this.currentUserId = uid !== null ? String(uid) : null;
    this.addLog('DebugPanel initialized');
    this.addLog(`User: ${this.currentUser?.nickname || 'Not logged in'}`);
    this.addLog(`WS Status: ${this.webSocketService.isConnected() ? 'Connected' : 'Disconnected'}`);
  }

  private addLog(log: string) {
    const time = new Date().toLocaleTimeString();
    this.logs.unshift(`[${time}] ${log}`);
  }

  build() {
    NavDestination() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            AppRouter.pop();
          }
        })

        Text('调试面板')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        IconButton({
          icon: AppIcon.CLOSE, // Used as Clear
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            this.logs = [];
          }
        })
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .backgroundColor($r('app.color.bg_secondary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      Scroll() {
        Column({ space: 16 }) {
          // 状态卡片
          SystemCard({
            cardPadding: 16,
            cardBackgroundColor: $r('sys.color.ohos_id_color_background')
          }) {
            Column({ space: 8 }) {
              Text('系统状态')
                .fontSize(16)
                .fontWeight(FontWeight.Bold)
              
              Text(`WebSocket: ${this.webSocketService.isConnected() ? '✅ 已连接' : '❌ 未连接'}`)
              Text(`User ID: ${this.currentUserId || 'N/A'}`)
              Text(`Token: ${this.currentUser ? 'Has Token' : 'No Token'}`)
            }
            .alignItems(HorizontalAlign.Start)
            .width('100%')
          }

          // 操作按钮
          Grid() {
            GridItem() {
              Button('重连WS')
                .onClick(() => {
                  this.webSocketService.disconnect();
                  this.addLog('WS Disconnected');
                  setTimeout(() => {
                    // re-connect requires token, assuming handle in service or simple toggle
                    // For debug, just show toast
                    SystemToast.show('Reconnect triggered', this.getUIContext());
                  }, 500);
                })
            }
            GridItem() {
              Button('测试Toast')
                .onClick(() => {
                  SystemToast.showSuccess('Test Success', this.getUIContext());
                })
            }
          }
          .columnsTemplate('1fr 1fr')
          .rowsGap(8)
          .columnsGap(8)
          .height(60)

          // 日志区域
          Text('日志')
            .fontSize(16)
            .fontWeight(FontWeight.Bold)
            .width('100%')
            .margin({ top: 8 })

          List({ space: 4 }) {
            ForEach(this.logs, (log: string) => {
              ListItem() {
                Text(log)
                  .fontSize(12)
                  .fontFamily('Monospace')
                  .width('100%')
              }
            }, (_item: string, index: number) => index.toString())
          }
          .width('100%')
          .height(300)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .padding(8)
          .borderRadius(8)
        }
        .padding(16)
        .width('100%')
      }
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }
}

@Builder
export function DebugPanelBuilder() {
  DebugPanel()
}
