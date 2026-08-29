/**
 * ChatActionsSheet —— 聊天页「会话设置」底部操作面板（查看主页/回到底部/
 * 清空输入/举报/关闭）。从 ChatPage.buildConversationActionsSheet 抽取；
 * 业务动作（导航、滚动、举报提交）经回调上抛，组件保持纯展示。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { SoftDivider } from '../../../components/SoftDivider';
import { SystemButton } from '../../../components/SystemButton';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';

@Component
export struct ChatActionsSheet {
  @Prop isGroupChat: boolean = false;
  /** 私聊对端 user_id（控制「查看个人主页」可见性） */
  @Prop targetId: string = '';
  palette: AppThemePalette | null = null;
  surfaceBackground: string = '';
  surfaceBorder: string | Resource = '';
  onOpenProfile: () => void = () => {};
  onScrollToBottom: () => void = () => {};
  onClearInput: () => void = () => {};
  onReport: () => void = () => {};
  onClose: () => void = () => {};

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Text('会话设置')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(this.palette?.textPrimary)

      Text(this.isGroupChat ? '群聊会话' : '私聊会话')
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(this.palette?.textSecondary)

      SoftDivider({ color: this.surfaceBorder })

      if (!this.isGroupChat && this.targetId) {
        SystemButton({
          label: '查看个人主页',
          variant: 'surface',
          buttonSize: 'sheet',
          fullWidth: true,
          onPress: () => { this.onOpenProfile(); }
        })
      }

      SystemButton({
        label: '回到最新消息',
        variant: 'surface',
        buttonSize: 'sheet',
        fullWidth: true,
        onPress: () => { this.onScrollToBottom(); }
      })

      SystemButton({
        label: '清空输入框',
        variant: 'surface',
        buttonSize: 'sheet',
        fullWidth: true,
        onPress: () => { this.onClearInput(); }
      })

      SoftDivider({ color: this.surfaceBorder })

      SystemButton({
        label: '举报',
        variant: 'surface-danger',
        buttonSize: 'sheet',
        fullWidth: true,
        onPress: () => { this.onReport(); }
      })

      SystemButton({
        label: '关闭',
        variant: 'surface-accent',
        buttonSize: 'sheet',
        fullWidth: true,
        onPress: () => { this.onClose(); }
      })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.XXL,
      right: DesignConstants.SPACING.XXL,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XXL
    })
    .backgroundColor(this.surfaceBackground)
    .borderRadius(DesignConstants.SHAPE.SHEET_TOP_RADIUS)
  }
}
