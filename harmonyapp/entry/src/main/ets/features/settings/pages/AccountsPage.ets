/**
 * 账号管理（多账号切换）
 *
 * - 展示本地保存的账号列表
 * - 支持切换当前账号（切换后重连 WS，并清空本地消息缓存，避免串数据）
 * - 支持删除账号
 */

import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { ResponsiveContainer } from '../../../components/ResponsiveContainer';
import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { MessageCacheService } from '../../../core/database/MessageCacheService';
import { MomentCacheService } from '../../../core/database/MomentCacheService';
import { MyProfile, AuthData } from '../../../core/types/api';
import { ApiClient } from '../../../core/api/ApiClient';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemButton } from '../../../components/SystemButton';
import { EmptyState } from '../../../components/EmptyState';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

@Preview
@Component
export struct AccountsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  @State accounts: AuthData[] = [];
  @State currentUserId: string | null = null;
  @State isLoading: boolean = true;
  @State errorMessage: string = '';

  private stateManager = GlobalStateManager.getInstance();
  private ws = WebSocketService.getInstance();
  private cache = MessageCacheService.getInstance();

  private isDestroyed: boolean = false;

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    await this.load();
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private async load() {
    if (this.isDestroyed) return;
    this.isLoading = true;
    this.errorMessage = '';
    try {
      await LoadingUtil.withMinimumTime(async () => {
        if (this.isDestroyed) return;
        this.accounts = await AuthAccountManager.listAccounts();
        const uid = await AuthAccountManager.getCurrentUserId();
        this.currentUserId = uid !== null ? String(uid) : null;
      });
    } catch {
      if (this.accounts.length === 0) {
        this.errorMessage = '加载账号列表失败';
      }
    } finally {
      this.isLoading = false;
    }
  }

  private getAccountDisplayName(auth: AuthData): string {
    // AuthData 只有 user_id/token 等字段，用 user_id 作为显示名
    return auth.user_id;
  }

  private toProfile(auth: AuthData): MyProfile {
    const nickname = auth.user_id;
    return {
      user_id: auth.user_id,
      nickname: nickname,
      avatar_url: '',
      user_type: 'temp',
      bio: '',
      tags: [],
      email: '',
      phone: '',
      location: {
        point: { latitude: 0, longitude: 0 },
        location_name: ''
      },
      presence_status: 'online',
      last_online_at: '',
      created_at: new Date().toISOString(),
      stats: {
        moments_count: 0,
        friends_count: 0,
        followers_count: 0,
        following_count: 0,
        groups_count: 0
      }
    };
  }

  private async switchTo(auth: AuthData) {
    if (this.isDestroyed) return;
    try {
      // 1) 清空缓存（避免串数据）
      await this.cache.clearAll();
      await MomentCacheService.getInstance().clearAll();

      // 2) 切换 token
      const next = await AuthAccountManager.switchTo(auth.user_id);
      if (!next) {
        SystemToast.showError('账号不存在', this.getUIContext());
        return;
      }

      // 3) 用真实 profile 更新 UI 层
      try {
        const profileRes = await ApiClient.getMyProfile();
        if (profileRes.code === 0 && profileRes.data) {
          this.stateManager.updateUser(profileRes.data);
        } else {
          this.stateManager.updateUser(this.toProfile(next));
        }
      } catch {
        this.stateManager.updateUser(this.toProfile(next));
      }

      // 4) 重连 WS
      this.ws.disconnect();
      try {
        await this.ws.connect(next.access_token);
      } catch {
        // ignore
      }

      SystemToast.showSuccess(`已切换到：${next.user_id}`, this.getUIContext());
      await this.load();
    } catch {
      SystemToast.showError('切换失败', this.getUIContext());
    }
  }

  private async remove(auth: AuthData) {
    if (this.isDestroyed) return;
    try {
      await AuthAccountManager.removeAccount(auth.user_id);
      if (this.currentUserId !== null && this.currentUserId === auth.user_id) {
        // 删除当前账号 => 锁定
        await this.cache.clearAll();
        await MomentCacheService.getInstance().clearAll();
        this.ws.disconnect();
        this.stateManager.updateUser(null);
      }
      SystemToast.showSuccess('已删除账号', this.getUIContext());
      await this.load();
    } catch {
      SystemToast.showError('删除失败', this.getUIContext());
    }
  }

  build() {
    NavDestination() {
  ResponsiveContainer() {
    Column() {
      // 标题栏 —— 统一走 SystemPageHeader（居中标题 + 返回）
      SystemPageHeader({
        title: '账号管理',
        showBackButton: true,
        onBack: () => { AppRouter.pop(); },
        titleAlign: TextAlign.Center,
        titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
        backgroundVariant: 'transparent'
      })

      if (this.isLoading) {
        SystemLoadingView({ message: '加载账号列表...' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
          )
      } else if (this.accounts.length === 0 && this.errorMessage) {
        EmptyState({
          icon: AppIcon.WARNING,
          title: this.errorMessage,
          actionLabel: '重试',
          bare: true,
          fillMode: true,
          onAction: () => { this.load(); }
        })
      } else if (this.accounts.length === 0) {
        EmptyState({
          icon: AppIcon.EMPTY,
          title: '暂无已保存账号',
          actionLabel: '去登录/创建访客',
          bare: true,
          fillMode: true,
          onAction: () => {
            AppRouter.push(PageNames.LOGIN);
          }
        })
      } else {
        List() {
          ForEach(this.accounts, (acc: AuthData) => {
            ListItem() {
              Row({ space: DesignConstants.SPACING.MD }) {
                Row() {
                  Text() {
                    SymbolSpan($r(AppIcon.PERSON))
                      .fontSize(DesignConstants.FONT.SIZE.LG_PLUS)
                      .fontColor([this.getCurrentThemePalette().accentBackground])
                  }
                }
                .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
                .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
                .borderRadius(DesignConstants.RADIUS.XL)
                .backgroundColor(this.getSurfaceSubtle())
                .justifyContent(FlexAlign.Center)
                .alignItems(VerticalAlign.Center)

                Column({ space: DesignConstants.SPACING.XS }) {
                  Row({ space: DesignConstants.SPACING.SM }) {
                    Text(this.getAccountDisplayName(acc))
                      .fontSize(DesignConstants.FONT.SIZE.LG)
                      .fontColor(this.getCurrentThemePalette().textPrimary)
                    if (this.currentUserId === acc.user_id) {
                      Text('当前')
                        .fontSize(DesignConstants.FONT.SIZE.SM)
                        .fontColor(this.getCurrentThemePalette().accentText)
                        .backgroundColor(this.getCurrentThemePalette().accentBackground)
                        .borderRadius(DesignConstants.SHAPE.CHIP)
                        .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.HALF, bottom: DesignConstants.SPACING.HALF })
                    }
                  }
                  Text(`ID: ${acc.user_id}`)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                Row({ space: DesignConstants.SPACING.SM }) {
                      SystemButton({
                        label: '切换',
                        variant: 'outline',
                        buttonSize: 'small',
                        onPress: () => { this.switchTo(acc); }
                      })

                      SystemButton({
                        label: '删除',
                        variant: 'primary',
                        buttonSize: 'small',
                        onPress: () => { this.remove(acc); }
                      })
                }
              }
              .width('100%')
              .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
            }
            .backgroundColor(this.getSurfaceBackground())
            .borderRadius(DesignConstants.RADIUS.LG)
            .shadow(this.getCurrentThemePalette().elevatedShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: DesignConstants.SPACING.MD3 })
          }, (acc: AuthData) => acc.user_id)
        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .padding({
          top: DesignConstants.SPACING.SM,
          bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
        })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
        )
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }
}

@Builder
export function AccountsPageBuilder() {
  AccountsPage()
}
