import { AppRouter } from '../../../core/router/AppRouter';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemInput } from '../../../components/SystemInput';
import { SystemButton } from '../../../components/SystemButton';
import { SystemDialog } from '../../../components/SystemDialog';
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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @State oldPassword: string = '';
  @State newPassword: string = '';
  @State sessions: SessionItem[] = [];
  @State showRevokeSessionConfirm: boolean = false;
  @State showRevokeAllConfirm: boolean = false;
  @State pendingRevokeSession: SessionItem | null = null;
  @State revokingSessionId: string = '';
  @State revokingAllSessions: boolean = false;

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

  @State private changingPassword: boolean = false;

  private async changePassword(): Promise<void> {
    if (this.isDestroyed || this.changingPassword) return;
    this.changingPassword = true;
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
    } finally {
      this.changingPassword = false;
    }
  }

  private async revokeSession(item: SessionItem): Promise<void> {
    if (this.isDestroyed) return;
    if (this.revokingSessionId.length > 0) return;
    this.revokingSessionId = item.session_id;
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
    } finally {
      this.revokingSessionId = '';
    }
  }

  private async revokeAll(): Promise<void> {
    if (this.isDestroyed) return;
    if (this.revokingAllSessions) return;
    this.revokingAllSessions = true;
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
    } finally {
      this.revokingAllSessions = false;
    }
  }

  build() {
    NavDestination() {
    Stack() {
    PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint}) {
      Column() {
        SystemPageHeader({
          title: '密码与会话安全',
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
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
                    label: this.changingPassword ? '更新中…' : '更新密码',
                    variant: 'primary',
                    buttonSize: 'large',
                    pill: true,
                    fullWidth: true,
                    loading: this.changingPassword,
                    onPress: () => { this.changePassword(); }
                  })
                }
                .width('100%')
                .padding(DesignConstants.SPACING.LG)
              }
            }
            .padding({ top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })

            Column() {
              SettingsSectionCard({ title: '登录设备' }) {
                SessionListSection({
                  sessions: this.sessions,
                  revokeOne: (item: SessionItem) => {
                    this.pendingRevokeSession = item;
                    this.showRevokeSessionConfirm = true;
                  },
                  revokeAll: () => { this.showRevokeAllConfirm = true; }
                })
              }
            }
            .padding({ bottom: DesignConstants.SPACING.LG })
          }
          .width('100%')
          .padding({ bottom: SafeAreaUtils.resolveBottomPadding(0, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG) })
        }
        .scrollBar(BarState.Off)
        .layoutWeight(1)
        .width('100%')
      }
      .width('100%')
      .height('100%')
    }

    // 撤销单个会话二次确认
    SystemDialog({
      visible: this.showRevokeSessionConfirm,
      title: '撤销会话',
      description: '该设备将立即退出登录，需要重新登录才能继续使用',
      confirmText: '撤销',
      onCancel: () => {
        this.showRevokeSessionConfirm = false;
        this.pendingRevokeSession = null;
      },
      onConfirm: () => {
        const target = this.pendingRevokeSession;
        this.showRevokeSessionConfirm = false;
        this.pendingRevokeSession = null;
        if (target) {
          this.revokeSession(target);
        }
      }
    })

    // 全部下线二次确认
    SystemDialog({
      visible: this.showRevokeAllConfirm,
      title: '全部下线',
      description: '除当前设备外，所有已登录设备将立即退出登录',
      confirmText: '全部下线',
      confirmVariant: 'danger',
      onCancel: () => { this.showRevokeAllConfirm = false; },
      onConfirm: () => {
        this.showRevokeAllConfirm = false;
        this.revokeAll();
      }
    })
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