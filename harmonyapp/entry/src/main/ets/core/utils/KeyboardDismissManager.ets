import { UIContext } from '@kit.ArkUI';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF35;
const TAG: string = 'KeyboardDismissMgr';

export class KeyboardDismissManager {
  private static instance: KeyboardDismissManager = new KeyboardDismissManager();
  private uiContext: UIContext | null = null;
  private hiddenTargetId: string = 'GlobalKeyboardDismissTarget';

  private constructor() {}

  static getInstance(): KeyboardDismissManager {
    return KeyboardDismissManager.instance;
  }

  init(context: UIContext | null, hiddenTargetId?: string): void {
    if (!context) {
      hilog.warn(DOMAIN_NUMBER, TAG, 'init 被调用但 UIContext 为空');
      return;
    }

    this.uiContext = context;
    if (hiddenTargetId) {
      this.hiddenTargetId = hiddenTargetId;
    }
  }

  setHiddenTargetId(id: string): void {
    if (id) {
      this.hiddenTargetId = id;
    }
  }

  dismiss(): void {
    const focusController = this.uiContext?.getFocusController();
    if (!focusController) {
      return;
    }

    try {
      focusController.requestFocus(this.hiddenTargetId);
    } catch (error) {
      hilog.warn(DOMAIN_NUMBER, TAG, `requestFocus(${this.hiddenTargetId}) 失败: ${JSON.stringify(error)}`);
    }
  }
}
