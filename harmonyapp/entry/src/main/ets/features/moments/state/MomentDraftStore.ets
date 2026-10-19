/**
 * 发布瞬间草稿（模块级单例）——对齐 web `stores/composer.svelte.ts` 的草稿字段语义。
 *
 * 为什么不能放组件内：`CreateMomentSheet` 由 `bindSheet` 挂载，Sheet 关闭或切走即卸载，
 * 组件级 `@State` 随之销毁。**2026-10-20 端上实测**：在发布 Sheet 内填好正文后点
 * 「创建群组」（`onSwitchToGroup` → `AppRouter.push(CREATE_GROUP)`），再返回首页并重新
 * 打开发布 Sheet —— 正文、氛围、图片全部丢失，用户没有任何提示。
 *
 * 因此草稿必须活在组件之外：Sheet 每次挂载时回填，字段变更时写透，**发布成功才清空**。
 * web 端同款机制（弹层关闭不丢稿、发布成功 clearDraft）是本次对齐的基准。
 */
export class MomentDraftStore {
  private static instance: MomentDraftStore | null = null;

  /** 正文（与组件内 DESCRIPTION_MAX_LENGTH 截断后的值一致） */
  description: string = '';
  /** 已选图片的本地路径 */
  imageUrls: string[] = [];
  /** 已选氛围标签 */
  moods: string[] = [];
  /** 可见范围档位（'nearby' | 'friends' | 'private'） */
  audience: string = 'nearby';

  static getInstance(): MomentDraftStore {
    if (MomentDraftStore.instance === null) {
      MomentDraftStore.instance = new MomentDraftStore();
    }
    return MomentDraftStore.instance;
  }

  /** 草稿是否有实质内容（用于"是否值得保留/提示"的判断） */
  hasContent(): boolean {
    return this.description.trim().length > 0 || this.imageUrls.length > 0 || this.moods.length > 0;
  }

  /** 写透当前草稿（每次字段变更调用，开销极小：字符串与短数组） */
  save(description: string, imageUrls: string[], moods: string[], audience: string): void {
    this.description = description;
    this.imageUrls = imageUrls.slice();
    this.moods = moods.slice();
    this.audience = audience;
  }

  /** 发布成功后清空（与 web clearDraft 对应） */
  clear(): void {
    this.description = '';
    this.imageUrls = [];
    this.moods = [];
    this.audience = 'nearby';
  }
}
