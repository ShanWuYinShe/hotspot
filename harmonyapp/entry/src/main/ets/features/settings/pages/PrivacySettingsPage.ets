import { AppRouter } from '../../../core/router/AppRouter';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemToast } from '../../../components/SystemToast';
import { ApiClient } from '../../../core/api/ApiClient';
import { UserSettings } from '../../../core/types/api';
import { SettingsSectionCard, ToggleSettingRow, SettingToggleItem } from './SettingsSupport';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

const PRIVACY_ITEMS: SettingToggleItem[] = [
  { key: 'location_visible', title: '位置可见', subtitle: '允许他人看到你的位置标签。' },
  { key: 'presence_visible', title: '在线状态可见', subtitle: '向他人展示你的在线状态。' },
  { key: 'social_lists_visible', title: '社交列表可见', subtitle: '允许他人查看你的好友、关注与粉丝列表。' },
  { key: 'groups_visible', title: '群组公开可见', subtitle: '允许他人查看你加入的公开群组。' },
  { key: 'reacted_moments_visible', title: '点赞瞬间可见', subtitle: '允许他人查看你点赞过的瞬间。' },
  { key: 'stats_visible', title: '数据统计可见', subtitle: '允许他人查看你的个人统计数据。' },
  { key: 'allow_follow', title: '允许被关注', subtitle: '关闭后其他用户将无法关注你。' },
  { key: 'searchable', title: '允许被搜索', subtitle: '控制是否能在全局搜索中被找到。' },
  { key: 'nickname_searchable', title: '昵称可搜索', subtitle: '允许通过昵称搜索到你。' },
  { key: 'location_searchable', title: '位置可搜索', subtitle: '允许在附近和位置相关搜索中出现。' }
];

@Preview
@Component
export struct PrivacySettingsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State settings: UserSettings | null = null;
  @State loading: boolean = false;

  private isDestroyed: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.loadSettings();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private async loadSettings(): Promise<void> {
    if (this.isDestroyed) return;
    this.loading = true;
    try {
      const response = await ApiClient.getSettings();
      if (this.isDestroyed) return;
      if (response.code === 0 && response.data) {
        this.settings = response.data;
      } else {
        SystemToast.showError(response.message || '获取隐私设置失败', this.getUIContext());
      }
    } catch (_) {
      SystemToast.showError('获取隐私设置失败', this.getUIContext());
    } finally {
      this.loading = false;
    }
  }

  private buildUpdatedSettings(current: UserSettings, key: keyof UserSettings, value: boolean): UserSettings {
    return {
      privacy_level: current.privacy_level,
      location_visible: key === 'location_visible' ? value : current.location_visible,
      presence_visible: key === 'presence_visible' ? value : current.presence_visible,
      social_lists_visible: key === 'social_lists_visible' ? value : current.social_lists_visible,
      groups_visible: key === 'groups_visible' ? value : current.groups_visible,
      reacted_moments_visible: key === 'reacted_moments_visible' ? value : current.reacted_moments_visible,
      stats_visible: key === 'stats_visible' ? value : current.stats_visible,
      allow_follow: key === 'allow_follow' ? value : current.allow_follow,
      allow_private_chat: current.allow_private_chat,
      searchable: key === 'searchable' ? value : current.searchable,
      phone_searchable: current.phone_searchable,
      nickname_searchable: key === 'nickname_searchable' ? value : current.nickname_searchable,
      location_searchable: key === 'location_searchable' ? value : current.location_searchable,
      nearby_moment_notify: key === 'nearby_moment_notify' ? value : current.nearby_moment_notify
    };
  }

  private getSettingValue(key: keyof UserSettings): boolean {
    if (!this.settings) {
      return false;
    }
    switch (key) {
      case 'location_visible':
        return this.settings.location_visible;
      case 'presence_visible':
        return this.settings.presence_visible;
      case 'social_lists_visible':
        return this.settings.social_lists_visible;
      case 'groups_visible':
        return this.settings.groups_visible;
      case 'reacted_moments_visible':
        return this.settings.reacted_moments_visible;
      case 'stats_visible':
        return this.settings.stats_visible;
      case 'allow_follow':
        return this.settings.allow_follow;
      case 'searchable':
        return this.settings.searchable;
      case 'nickname_searchable':
        return this.settings.nickname_searchable;
      case 'location_searchable':
        return this.settings.location_searchable;
      default:
        return false;
    }
  }

  private async updateSetting(key: keyof UserSettings, value: boolean): Promise<void> {
    if (this.isDestroyed) return;
    if (!this.settings) {
      return;
    }
    const next = this.buildUpdatedSettings(this.settings, key, value);
    this.settings = next;
    try {
      const response = await ApiClient.updateSettings({ settings: next });
      if (this.isDestroyed) return;
      if (response.code !== 0) {
        throw new Error(response.message || '保存失败');
      }
      SystemToast.showSuccess('设置已更新', this.getUIContext());
    } catch (_) {
      SystemToast.showError('保存失败', this.getUIContext());
      this.loadSettings();
    }
  }

  build() {
    NavDestination() {
    ResponsiveContainer() {
      Column() {
        SystemPageHeader({
          title: '隐私设置',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

        Scroll() {
          Column({ space: 0 }) {
            Column() {
              SettingsSectionCard({ title: '可见性与搜索范围' }) {
                Column() {
                  ForEach(PRIVACY_ITEMS, (item: SettingToggleItem) => {
                    ToggleSettingRow({
                      item,
                      isOn: this.getSettingValue(item.key),
                      onChange: (value: boolean) => this.updateSetting(item.key, value)
                    })
                  }, (item: SettingToggleItem) => String(item.key))
                }
              }
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
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
export function PrivacySettingsPageBuilder() {
  PrivacySettingsPage()
}
