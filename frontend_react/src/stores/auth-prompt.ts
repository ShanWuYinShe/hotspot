/**
 * 登录提示弹窗状态
 */

import { create } from 'zustand';

export interface AuthPromptOptions {
  actionName?: string;
  description?: string;
  redirectPath?: string;
}

interface AuthPromptState extends AuthPromptOptions {
  isOpen: boolean;
  openPrompt: (options?: AuthPromptOptions) => void;
  closePrompt: () => void;
}

export const useAuthPromptStore = create<AuthPromptState>((set) => ({
  isOpen: false,
  actionName: undefined,
  description: undefined,
  redirectPath: undefined,
  openPrompt: (options) =>
    set({
      isOpen: true,
      actionName: options?.actionName,
      description: options?.description,
      redirectPath: options?.redirectPath,
    }),
  closePrompt: () =>
    set({
      isOpen: false,
      actionName: undefined,
      description: undefined,
      redirectPath: undefined,
    }),
}));
