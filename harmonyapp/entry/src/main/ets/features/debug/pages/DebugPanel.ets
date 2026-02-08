/**
 * 调试面板
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemCard } from '../../../components/SystemCard';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { SystemToast } from '../../../components/SystemToast';
import { MyProfileData } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';

@Entry
@Component
struct DebugPanel {
  @State logs: string[] = [];
  @State currentUserId: number | null = null;
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;
  private webSocketService = WebSocketService.getInstance();

  async aboutToAppear() {
    this.currentUserId = await AuthAccountManager.getCurrentUserId();
    this.addLog('DebugPanel initialized');
    this.addLog(`User: ${this.currentUser?.nickname || 'Not logged in'}`);
    this.addLog(`WS Status: ${this.webSocketService.isConnected() ? 'Connected' : 'Disconnected'}`);
  }

  private addLog(log: string) {
    const time = new Date().toLocaleTimeString();
    this.logs.unshift(`[${time}] ${log}`);
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text('调试面板')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        IconButton({
          icon: AppIcon.CLOSE, // Used as Clear
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            this.logs = [];
          }
        })
      }
      .width('100%')
      .height(56)
      .padding({ left: 8, right: 8 })
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

      Scroll() {
        Column({ space: 16 }) {
          // 状态卡片
          SystemCard({
            content: () => {
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
            },
            paddingSize: 16,
            backgroundColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT
          })

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
            })
          }
          .width('100%')
          .height(300)
          .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
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
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }
}

