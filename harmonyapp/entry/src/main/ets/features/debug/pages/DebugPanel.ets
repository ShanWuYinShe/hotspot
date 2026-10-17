/**
 * 调试面板
 */

import { DesignConstants, textLineHeight } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { SystemCard } from '../../../components/SystemCard';
import { SystemButton } from '../../../components/SystemButton';
import pasteboard from '@ohos.pasteboard';
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

  /** User ID 一键复制（Gemini 审查 r33：长 ID 无法直接提取，实机排查效率低） */
  private async copyUserId(): Promise<void> {
    if (!this.currentUserId) {
      return;
    }
    const systemPasteboard = pasteboard.getSystemPasteboard();
    const pasteData = pasteboard.createData(pasteboard.MIMETYPE_TEXT_PLAIN, this.currentUserId);
    await systemPasteboard.setData(pasteData);
    SystemToast.show('User ID 已复制', this.getUIContext());
  }

  /** 模块标题：eyebrow 小标，对齐 settings SettingsGroupLabel 语言（r33：两模块标题样式不一） */
  @Builder
  private SectionLabel(label: string) {
    Text(label)
      .fontSize(DesignConstants.FONT.SIZE.XS2)
      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
      .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
      .fontColor(this.getCurrentThemePalette().textSecondary)
      .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
      .width('100%')
  }

  build() {
    NavDestination() {
    Column() {
      // 标题栏：透明沉浸式，去掉原白底 + 底部圆角描边 + 语义错乱的「×」（图标是关闭、
      // 行为却是清空日志），清空日志入口移至「日志」标题旁（Gemini 审查 r33）。
      // DebugPanel 属已登记无 SystemPageHeader 例外，但顶栏语言与其余二级页对齐。
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
          .fontSize(DesignConstants.FONT.SIZE.XXL)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XXL))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        // 右侧等宽占位：保证标题严格居中（对齐 SystemPageHeader 的居中机制）
        Row()
          .width(DesignConstants.LAYOUT.BACK_BUTTON_SIZE)
          .height(DesignConstants.LAYOUT.BACK_BUTTON_SIZE)
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: DesignConstants.SPACING.SM
      })
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      Scroll() {
        Column({ space: DesignConstants.SPACING.LG }) {
          // 模块标题统一移至卡片外左上方（r33：「系统状态」原在卡内、「日志」在卡外，样式不一）
          this.SectionLabel('系统状态')

          // 状态卡片
          SystemCard({
            cardPadding: DesignConstants.SPACING.LG,
            cardBackgroundColor: this.getSurfaceBackground()
          }) {
            Column({ space: DesignConstants.SPACING.SM }) {
              // 键值两端对齐;未连接用警示色强化状态感知(Gemini 审查 r33)
              Row() {
                Text('WebSocket').fontColor(this.getCurrentThemePalette().textSecondary)
                Blank()
                Text(this.webSocketService.isConnected() ? '已连接' : '未连接')
                  .fontColor(this.webSocketService.isConnected()
                    ? this.getCurrentThemePalette().textPrimary
                    : this.getCurrentThemePalette().error)
              }
              .width('100%')
              Row({ space: DesignConstants.SPACING.SM }) {
                Text('User ID').fontColor(this.getCurrentThemePalette().textSecondary)
                Blank()
                Text(this.currentUserId || 'N/A').maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                SurfaceIconButton({
                  icon: AppIcon.COPY,
                  kind: 'neutral',
                  buttonSize: 28,
                  glyphSize: 14,
                  accessibilityLabel: '复制 User ID',
                  onPress: (): void => {
                    this.copyUserId();
                  }
                })
              }
              .width('100%')
              Row() {
                Text('Token').fontColor(this.getCurrentThemePalette().textSecondary)
                Blank()
                Text(this.currentUser ? '有 Token' : '无 Token')
                  .fontColor(this.getCurrentThemePalette().textPrimary)
              }
              .width('100%')
            }
            .alignItems(HorizontalAlign.Start)
            .width('100%')
          }

          // 操作按钮：双列等宽（r33）；SystemButton 原语区分主次——重连为主动作，
          // 测试 Toast 降级为浅色次要（原两颗同权默认蓝按钮）
          Row({ space: DesignConstants.SPACING.SM }) {
            SystemButton({
              label: '重连WS',
              variant: 'primary',
              onPress: (): void => {
                this.webSocketService.disconnect();
                this.addLog('WS Disconnected');
                setTimeout(() => {
                  if (this.isDestroyed) return;
                  // re-connect requires token, assuming handle in service or simple toggle
                  // For debug, just show toast
                  SystemToast.show('Reconnect triggered', this.getUIContext());
                }, 500);
              }
            }).layoutWeight(1)
            SystemButton({
              label: '测试Toast',
              variant: 'surface',
              onPress: (): void => {
                SystemToast.showSuccess('Test Success', this.getUIContext());
              }
            }).layoutWeight(1)
          }
          .width('100%')

          // 日志区域：标题行带清空入口（自原顶栏「×」迁来，语义对齐）
          Row({ space: DesignConstants.SPACING.SM }) {
            Text('日志')
              .fontSize(DesignConstants.FONT.SIZE.XS2)
              .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS2))
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .letterSpacing(DesignConstants.FONT.LETTER_SPACING.LABEL)
            Blank()
            SurfaceIconButton({
              icon: AppIcon.CLEAR,
              kind: 'neutral',
              buttonSize: 28,
              glyphSize: 14,
              accessibilityLabel: '清空日志',
              onPress: (): void => {
                this.logs = [];
              }
            })
          }
          .width('100%')
          .margin({ top: DesignConstants.SPACING.SM })

          List({ space: DesignConstants.SPACING.XS }) {
            ForEach(this.logs, (log: string) => {
              ListItem() {
                Text(log)
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                  .fontFamily('Monospace')
                  .width('100%')
              }
            }, (_item: string, index: number) => index.toString())
          }
          .scrollBar(BarState.Off)
          .width('100%')
          .height(DesignConstants.COMPONENT.LOG_AREA_HEIGHT)
          .backgroundColor(this.getSurfaceBackground())
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getCurrentThemePalette().borderColor })
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