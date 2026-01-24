/**
 * Moments 页面头部 (Server Component)
 * 标题为服务端渲染，交互按钮下沉到客户端
 */

import { getTranslations } from 'next-intl/server';
import { CreateMomentButton } from '@/components/moments';

export async function MomentsHeader() {
  const t = await getTranslations('moments');

  return (
    <div className="flex items-center justify-between mb-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <CreateMomentButton label={t('create')} />
    </div>
  );
}
