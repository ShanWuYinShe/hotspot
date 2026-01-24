/**
 * 瞬间编辑页面 (Server Component)
 */

import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { MomentEditForm } from './moment-edit-form';
import type { MomentItem } from '@/types/api';

interface MomentEditPageProps {
  params: Promise<{ momentId: string }>;
}

export default async function MomentEditPage({ params }: MomentEditPageProps) {
  const { momentId } = await params;
  const response = await serverApi.get<MomentItem>(
    ENDPOINTS.MOMENTS.DETAIL(momentId),
    undefined,
    { tags: [`moment-${momentId}`], revalidate: 60 }
  );

  if (!isServerSuccess(response)) {
    return (
      <div className="max-w-2xl mx-auto">
        <LoginRequiredCard actionName="编辑瞬间" description="登录后才能编辑瞬间。" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      <Card>
        <CardHeader>
          <CardTitle>编辑瞬间</CardTitle>
        </CardHeader>
        <CardContent>
          <MomentEditForm moment={response.data} />
        </CardContent>
      </Card>
    </div>
  );
}
