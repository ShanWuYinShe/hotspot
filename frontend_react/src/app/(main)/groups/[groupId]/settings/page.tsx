/**
 * 群组设置页面 (Server Component)
 */

import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { GroupSettingsForm } from './group-settings-form';
import type { GroupDetailData } from '@/types/api';

interface GroupSettingsPageProps {
  params: Promise<{ groupId: string }>;
}

export default async function GroupSettingsPage({ params }: GroupSettingsPageProps) {
  const { groupId } = await params;
  const response = await serverApi.get<GroupDetailData>(
    ENDPOINTS.GROUP_CHATS.DETAIL(groupId),
    undefined,
    { tags: [`group-${groupId}`], revalidate: 60 }
  );

  if (!isServerSuccess(response)) {
    return (
      <div className="max-w-2xl mx-auto">
        <LoginRequiredCard actionName="管理群组" description="登录后才能管理群组设置。" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      <Card>
        <CardHeader>
          <CardTitle>群组设置</CardTitle>
        </CardHeader>
        <CardContent>
          <GroupSettingsForm group={response.data} />
        </CardContent>
      </Card>
    </div>
  );
}
