import { AppRouter } from '../../../core/router/AppRouter';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { ApiClient } from '../../../core/api/ApiClient';
import { ChangePasswordRequest, SessionItem } from '../../../core/types/api';
import { SessionListSection, SettingsSectionCard } from './SettingsSupport';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF28;
const TAG: string = 'SafetySettingsPage';

@Preview
@Component
export struct SafetySettingsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State oldPassword: string = '';
  @State newPassword: string = '';
  @State sessions: SessionItem[] = [];

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.loadSessions();
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private async loadSessions(): Promise<void> {
    try {
      const response = await ApiClient.getSessions({});
      if (response.code === 0 && response.data) {
        this.sessions = response.data.items ?? [];
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `加载会话列表失败: ${JSON.stringify(error)}`);
    }
  }

  private async changePassword(): Promise<void> {
    const request: ChangePasswordRequest = {
      old_password: this.oldPassword,
      new_password: this.newPassword
    };
    try {
      const response = await ApiClient.changePassword(request);
      if (response.code === 0) {
        this.oldPassword = '';
        this.newPassword = '';
        SystemToast.showSuccess('密码已更新', this.getUIContext());
      } else {
        SystemToast.showError(getErrorMessage(response, '修改密码失败'), this.getUIContext());
      }
    } catch (_) {
      SystemToast.showError('修改密码失败', this.getUIContext());
    }
  }

  private async revokeSession(item: SessionItem): Promise<void> {
    try {
      const response = await ApiClient.revokeSession({ session_id: item.session_id });
      if (response.code === 0) {
        SystemToast.showSuccess('会话已撤销', this.getUIContext());
        this.loadSessions();
      } else {
        SystemToast.showError(getErrorMessage(response, '撤销失败'), this.getUIContext());
      }
    } catch (_) {
      SystemToast.showError('撤销失败', this.getUIContext());
    }
  }

  private async revokeAll(): Promise<void> {
    try {
      const response = await ApiClient.revokeAllSessions();
      if (response.code === 0) {
        SystemToast.showSuccess('其他会话已全部下线', this.getUIContext());
        this.loadSessions();
      } else {
        SystemToast.showError(getErrorMessage(response, '操作失败'), this.getUIContext());
      }
    } catch (_) {
      SystemToast.showError('操作失败', this.getUIContext());
    }
  }

  build() {
    NavDestination() {
    ResponsiveContainer() {
      Column() {
        SystemPageHeader({
          title: '密码与会话安全',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

        Scroll() {
          Column({ space: 0 }) {
            Column() {
              SettingsSectionCard({ title: '修改密码' }) {
                Column({ space: DesignConstants.SPACING.MD }) {
                  TextInput({ placeholder: '当前密码', text: this.oldPassword })
                    .type(InputType.Password)
                    .height(46)
                    .backgroundColor(this.getPalette().surfaceSubtle)
                    .borderRadius(DesignConstants.SHAPE.INPUT)
                    .border({ width: 1, color: this.getPalette().borderColor })
                    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
                    .onChange((value: string) => { this.oldPassword = value; })
                  TextInput({ placeholder: '新密码（至少 8 位）', text: this.newPassword })
                    .type(InputType.Password)
                    .height(46)
                    .backgroundColor(this.getPalette().surfaceSubtle)
                    .borderRadius(DesignConstants.SHAPE.INPUT)
                    .border({ width: 1, color: this.getPalette().borderColor })
                    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
                    .onChange((value: string) => { this.newPassword = value; })
                  Button('更新密码')
                    .width('100%')
                    .height(48)
                    .type(ButtonType.Normal)
                    .backgroundColor(this.getPalette().accentBackground)
                    .fontColor(this.getPalette().accentText)
                    .borderRadius(DesignConstants.RADIUS.FULL)
                    .shadow(this.getPalette().accentShadow)
                    .onClick(() => this.changePassword())
                }
                .width('100%')
                .padding(DesignConstants.SPACING.LG)
              }
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })

            Column() {
              SettingsSectionCard({ title: '登录设备' }) {
                SessionListSection({
                  sessions: this.sessions,
                  revokeOne: (item: SessionItem) => this.revokeSession(item),
                  revokeAll: () => this.revokeAll()
                })
              }
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
          }
          .width('100%')
          .padding({ bottom: SafeAreaUtils.resolveBottomPadding(0, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG) })
        }
        .layoutWeight(1)
        .width('100%')
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getPalette().pageBackground)
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }
}

@Builder
export function SafetySettingsPageBuilder() {
  SafetySettingsPage()
}
