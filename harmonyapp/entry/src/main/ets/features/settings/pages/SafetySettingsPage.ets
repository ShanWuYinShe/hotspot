import { AppRouter } from '../../../core/router/AppRouter';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemInput } from '../../../components/SystemInput';
import { SystemButton } from '../../../components/SystemButton';
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
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State oldPassword: string = '';
  @State newPassword: string = '';
  @State sessions: SessionItem[] = [];

  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.loadSessions();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private async loadSessions(): Promise<void> {
    if (this.isDestroyed) return;
    try {
      const response = await ApiClient.getSessions({});
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.sessions = response.data.items ?? [];
      }
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `加载会话列表失败: ${JSON.stringify(error)}`);
    }
  }

  private async changePassword(): Promise<void> {
    if (this.isDestroyed) return;
    const request: ChangePasswordRequest = {
      old_password: this.oldPassword,
      new_password: this.newPassword
    };
    try {
      const response = await ApiClient.changePassword(request);
      if (this.isDestroyed) return;
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
    if (this.isDestroyed) return;
    try {
      const response = await ApiClient.revokeSession({ session_id: item.session_id });
      if (this.isDestroyed) return;
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
    if (this.isDestroyed) return;
    try {
      const response = await ApiClient.revokeAllSessions();
      if (this.isDestroyed) return;
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
    PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
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
                  SystemInput({
                    value: this.oldPassword,
                    placeholder: '当前密码',
                    passwordMode: true,
                    inputHeight: DesignConstants.COMPONENT.PASSWORD_INPUT_HEIGHT,
                    onChange: (value: string) => { this.oldPassword = value; }
                  })
                  SystemInput({
                    value: this.newPassword,
                    placeholder: '新密码（至少 8 位）',
                    passwordMode: true,
                    inputHeight: DesignConstants.COMPONENT.PASSWORD_INPUT_HEIGHT,
                    onChange: (value: string) => { this.newPassword = value; }
                  })
                  SystemButton({
                    label: '更新密码',
                    variant: 'primary',
                    buttonSize: 'large',
                    pill: true,
                    fullWidth: true,
                    onPress: () => { this.changePassword(); }
                  })
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