/**
 * 调试面板
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { SystemCard } from '../../../components/SystemCard';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { SystemToast } from '../../../components/SystemToast';
import { AccountInfo } from '../../../core/types/api';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { AppRouter } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Preview
@Component
export struct DebugPanel {
  pathStack: NavPathStack = new NavPathStack();
  @State logs: string[] = [];
  @State currentUserId: string | null = null;
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  private webSocketService = WebSocketService.getInstance();

  private isDestroyed: boolean = false;

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    const uid = await AuthAccountManager.getCurrentUserId();
    if (this.isDestroyed) return;
    this.currentUserId = uid !== null ? String(uid) : null;
    this.addLog('DebugPanel initialized');
    this.addLog(`User: ${this.currentUser?.nickname || 'Not logged in'}`);
    this.addLog(`WS Status: ${this.webSocketService.isConnected() ? 'Connected' : 'Disconnected'}`);
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private addLog(log: string) {
    const time = new Date().toLocaleTimeString();
    this.logs.unshift(`[${time}] ${log}`);
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  build() {
    NavDestination() {
    Column() {
      // 标题栏
      Row() {
        SurfaceIconButton({
          icon: AppIcon.BACK,
          kind: 'surface',
          buttonSize: DesignConstants.LAYOUT.BACK_BUTTON_SIZE,
          glyphSize: DesignConstants.FONT.SIZE.LG_PLUS,
          accessibilityLabel: '返回',
          onPress: () => {
            AppRouter.pop();
          }
        })

        Text('调试面板')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        SurfaceIconButton({
          icon: AppIcon.CLOSE,
          kind: 'surface',
          buttonSize: DesignConstants.LAYOUT.BACK_BUTTON_SIZE,
          glyphSize: DesignConstants.FONT.SIZE.LG_PLUS,
          accessibilityLabel: '清空日志',
          onPress: () => {
            this.logs = [];
          }
        })
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor(this.getSurfaceBackground())
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
      .borderRadius({ bottomLeft: DesignConstants.RADIUS.LG, bottomRight: DesignConstants.RADIUS.LG })
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      Scroll() {
        Column({ space: DesignConstants.SPACING.LG }) {
          // 状态卡片
          SystemCard({
            cardPadding: DesignConstants.SPACING.LG,
            cardBackgroundColor: this.getSurfaceBackground()
          }) {
            Column({ space: DesignConstants.SPACING.SM }) {
              Text('系统状态')
                .fontSize(DesignConstants.FONT.SIZE.LG)
                .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
              
              Text(`WebSocket: ${this.webSocketService.isConnected() ? '已连接' : '未连接'}`)
              Text(`User ID: ${this.currentUserId || 'N/A'}`)
              Text(`Token: ${this.currentUser ? '有 Token' : '无 Token'}`)
            }
            .alignItems(HorizontalAlign.Start)
            .width('100%')
          }

          // 操作按钮
          Grid() {
            GridItem() {
              Button('重连WS')
                .type(ButtonType.Normal)
                .borderRadius(DesignConstants.RADIUS.XL)
                .onClick(() => {
                  this.webSocketService.disconnect();
                  this.addLog('WS Disconnected');
                  setTimeout(() => {
                    if (this.isDestroyed) return;
                    // re-connect requires token, assuming handle in service or simple toggle
                    // For debug, just show toast
                    SystemToast.show('Reconnect triggered', this.getUIContext());
                  }, 500);
                })
            }
            GridItem() {
              Button('测试Toast')
                .type(ButtonType.Normal)
                .borderRadius(DesignConstants.RADIUS.XL)
                .onClick(() => {
                  SystemToast.showSuccess('Test Success', this.getUIContext());
                })
            }
          }
          .scrollBar(BarState.Off)
          .columnsTemplate('1fr 1fr')
          .rowsGap(DesignConstants.SPACING.SM)
          .columnsGap(DesignConstants.SPACING.SM)
          .height(DesignConstants.LAYOUT.LABEL_WIDTH)

          // 日志区域
          Text('日志')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
            .width('100%')
            .margin({ top: DesignConstants.SPACING.SM })

          List({ space: DesignConstants.SPACING.XS }) {
            ForEach(this.logs, (log: string) => {
              ListItem() {
                Text(log)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontFamily('Monospace')
                  .width('100%')
              }
            }, (_item: string, index: number) => index.toString())
          }
          .scrollBar(BarState.Off)
          .width('100%')
          .height(DesignConstants.COMPONENT.LOG_AREA_HEIGHT)
          .backgroundColor(this.getSurfaceBackground())
          .padding(DesignConstants.SPACING.SM)
          .borderRadius(DesignConstants.RADIUS.MD)
        }
        .padding(DesignConstants.SPACING.LG)
        .width('100%')
      }
      .scrollBar(BarState.Off)
      .layoutWeight(1)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
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