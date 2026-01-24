/**
 * 我的群组列表 (Server Component)
 */

import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { GroupCard } from '@/components/cards/group-card';
import { EmptyState } from '@/components/common/empty-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { Users } from '@/components/icons';
import type { JoinedGroupItem, PaginatedData } from '@/types/api';

export default async function MyGroupsPage() {
  const response = await serverApi.get<PaginatedData<JoinedGroupItem>>(
    ENDPOINTS.USER.ME_GROUPS,
    { limit: 50 },
    { tags: ['profile-groups'], revalidate: 60 }
  );

  if (!isServerSuccess(response)) {
    return (
      <div className="max-w-2xl mx-auto">
        <LoginRequiredCard actionName="查看我的群组" description="登录后才能查看已加入的群组。" />
      </div>
    );
  }

  const items = response.data?.items ?? [];

  if (items.length === 0) {
    return (
      <div className="max-w-2xl mx-auto">
        <EmptyState icon={Users} title="暂无群组" description="加入或创建群组开始交流" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-3">
      {items.map((group) => (
        <GroupCard key={group.group_chat_id} group={group} showJoinButton={false} />
      ))}
    </div>
  );
}
