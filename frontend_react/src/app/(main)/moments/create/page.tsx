/**
 * 创建瞬间页面 (Server Component)
 * 服务端获取翻译，客户端处理表单
 */

import { getTranslations } from 'next-intl/server';
import { CreateMomentForm } from '@/components/moments';

export default async function CreateMomentPage() {
  const t = await getTranslations('moments');

  return (
    <CreateMomentForm
      translations={{
        create: t('create'),
      }}
    />
  );
}
