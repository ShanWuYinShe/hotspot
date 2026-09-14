/**
 * MessageActionsSheet —— 单条消息操作面板（编辑/撤回/关闭，对齐前端 MessageBubble 的
 * ⋮ 菜单语义）。点击气泡唤起；「编辑」在面板内切换为输入态（TextArea + 完成/取消），
 * 撤回/编辑提交经回调上抛，组件保持纯展示。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { SoftDivider } from '../../../components/SoftDivider';
import { SystemButton } from '../../../components/SystemButton';
import { SystemTextArea } from '../../../components/SystemTextArea';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';

const MESSAGE_EDIT_MAX: number = 5000;

@Component
export struct MessageActionsSheet {
  /** 待操作消息内容（编辑态回填初值） */
  @Prop messageContent: string = '';
  /** 可编辑（自己的文本消息且未撤回） */
  @Prop canEdit: boolean = false;
  /** 可撤回（自己的消息时限内 / 群管理员撤回他人） */
  @Prop canRecall: boolean = false;
  palette: AppThemePalette | null = null;
  surfaceBackground: string = '';
  surfaceBorder: string | Resource = '';
  onEdit: (content: string) => void = () => {};
  onRecall: () => void = () => {};
  onClose: () => void = () => {};

  /** 面板内编辑态（点「编辑」进入，完成/取消退出） */
  @State private isEditing: boolean = false;
  @State private draft: string = '';

  private startEdit(): void {
    this.draft = this.messageContent;
    this.isEditing = true;
  }

  private confirmEdit(): void {
    const trimmed = this.draft.trim();
    if (!trimmed) {
      return;
    }
    this.isEditing = false;
    this.onEdit(trimmed);
  }

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      if (this.isEditing) {
        Text('编辑消息')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.palette?.textPrimary)

        SystemTextArea({
          value: this.draft,
          placeholder: '输入修改后的内容',
          minHeight: DesignConstants.COMPONENT.PLACEHOLDER_AREA_HEIGHT,
          maxLength: MESSAGE_EDIT_MAX,
          fontSize: DesignConstants.FONT.SIZE.MD,
          onChange: (value: string) => { this.draft = value; }
        })

        Row({ space: DesignConstants.SPACING.SM }) {
          SystemButton({
            label: '取消',
            variant: 'surface',
            buttonSize: 'sheet',
            fullWidth: true,
            onPress: () => { this.isEditing = false; }
          })
          SystemButton({
            label: '完成',
            variant: 'primary',
            buttonSize: 'sheet',
            fullWidth: true,
            disabled: this.draft.trim().length === 0,
            onPress: () => { this.confirmEdit(); }
          })
        }
        .width('100%')
      } else {
        Text('消息操作')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.palette?.textPrimary)

        Text(this.messageContent)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.palette?.textSecondary)
          .maxLines(2)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')

        SoftDivider({ color: this.surfaceBorder })

        if (this.canEdit) {
          SystemButton({
            label: '编辑',
            variant: 'surface',
            buttonSize: 'sheet',
            fullWidth: true,
            onPress: () => { this.startEdit(); }
          })
        }

        if (this.canRecall) {
          // 危险操作：danger 实底（对齐前端撤回项 danger 语义）
          SystemButton({
            label: '撤回',
            variant: 'surface-danger',
            buttonSize: 'sheet',
            fullWidth: true,
            onPress: () => { this.onRecall(); }
          })
        }

        SoftDivider({ color: this.surfaceBorder })

        SystemButton({
          label: '关闭',
          variant: 'surface-accent',
          buttonSize: 'sheet',
          fullWidth: true,
          onPress: () => { this.onClose(); }
        })
      }
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
