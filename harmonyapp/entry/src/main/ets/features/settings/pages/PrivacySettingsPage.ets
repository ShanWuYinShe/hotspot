import { AppRouter } from '../../../core/router/AppRouter';
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
  { key: 'liked_moments_visible', title: '点赞瞬间可见', subtitle: '允许他人查看你点赞过的瞬间。' },
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
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @State settings: UserSettings | null = null;
  @State loading: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    this.loadSettings();
  }

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private async loadSettings(): Promise<void> {
    this.loading = true;
    try {
      const response = await ApiClient.getSettings();
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

  private async updateSetting(key: keyof UserSettings, value: boolean): Promise<void> {
    if (!this.settings) {
      return;
    }
    const next = { ...this.settings } as UserSettings;
    (next as Record<string, boolean>)[String(key)] = value;
    this.settings = next;
    try {
      const partial: Record<string, boolean> = {};
      partial[String(key)] = value;
      const response = await ApiClient.updateSettings({ settings: partial as Partial<UserSettings> });
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
      Column() {
        SystemPageHeader({
          title: '隐私设置',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); }
        })

        Scroll() {
          Column({ space: 0 }) {
            SettingsSectionCard({ title: '可见性与搜索范围' }) {
              Column() {
                ForEach(PRIVACY_ITEMS, (item: SettingToggleItem) => {
                  ToggleSettingRow({
                    item,
                    isOn: this.settings ? Boolean(this.settings[item.key]) : false,
                    onChange: (value: boolean) => this.updateSetting(item.key, value)
                  })
                }, (item: SettingToggleItem) => String(item.key))
              }
            }
            .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.LG })
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
