// Composer store — 发布器全局开关（FAB/页内 CTA 共用唯一入口）+ 草稿。
// ComposerContext（runes 模块单例）。
// 草稿字段放 store 而非 CreateSheet 组件内：弹层关闭即卸载组件，组件级
// $state 会丢稿（移动端把手下拉 64px 误触即丢）；store 单例跨挂载存活，
// 关闭重开草稿仍在，发布成功后 clearDraft() 清空。

// 'friends' 档已移除：spec PrivacyLevel 无对应档位（曾静默存为 private，见 CreateSheet 注释）。
export type ComposerAudience = 'nearby' | 'private';

let composerOpen = $state(false);

let draftText = $state('');
let draftMoods = $state<string[]>([]);
let draftImages = $state<File[]>([]);
let draftAudience = $state<ComposerAudience>('nearby');

export function openComposer(): void {
  composerOpen = true;
}

export function closeComposer(): void {
  composerOpen = false;
}

export function toggleComposer(): void {
  composerOpen = !composerOpen;
}

/** 草稿是否有实质内容（关闭挽留/徽标提示用）。 */
export function hasDraft(): boolean {
  return draftText.trim().length > 0 || draftMoods.length > 0 || draftImages.length > 0;
}

/** 发布成功后清空草稿。 */
export function clearDraft(): void {
  draftText = '';
  draftMoods = [];
  draftImages = [];
  draftAudience = 'nearby';
}

export const composer = {
  get composerOpen() { return composerOpen; },
};

// draft：发布器草稿（可绑定 getter/setter 对，组件 bind:value 直接读写）。
export const draft = {
  get text() { return draftText; },
  set text(v: string) { draftText = v; },
  get moods() { return draftMoods; },
  set moods(v: string[]) { draftMoods = v; },
  get images() { return draftImages; },
  set images(v: File[]) { draftImages = v; },
  get audience() { return draftAudience; },
  set audience(v: ComposerAudience) { draftAudience = v; },
};
