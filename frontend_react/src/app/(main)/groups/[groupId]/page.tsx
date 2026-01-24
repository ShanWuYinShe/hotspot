/**
 * 群组详情页面 (Server Component)
 * 服务端获取群组详情
 */

import { Suspense } from 'react';
import { notFound } from 'next/navigation';
import { serverApi, isServerSuccess, isNotFoundError, getServerErrorMessage } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { GroupHeader } from './group-header';
import { GroupMembers } from './group-members';
import { GroupAnnouncements } from './group-announcements';
import { ServerErrorCard } from '@/components/common/server-error-card';
import type { GroupDetailData, GroupMemberItem, GroupAnnouncementItem, PaginatedData } from '@/types/api';

interface GroupPageProps {
  params: Promise<{ groupId: string }>;
}

async function GroupContent({ groupId }: { groupId: string }) {
  const [groupRes, membersRes, announcementsRes] = await Promise.all([
    serverApi.post<GroupDetailData>(
      ENDPOINTS.GROUP_CHATS.DETAIL(groupId),
      { group_chat_id: groupId },
      { tags: [`group-${groupId}`], revalidate: 60 }
    ),
    serverApi.post<PaginatedData<GroupMemberItem>>(
      ENDPOINTS.GROUP_CHATS.MEMBERS(groupId),
      { group_chat_id: groupId, limit: 20 },
      { tags: [`group-${groupId}-members`], revalidate: 60 }
    ),
    serverApi.post<PaginatedData<GroupAnnouncementItem>>(
      ENDPOINTS.GROUP_CHATS.ANNOUNCEMENTS(groupId),
      { group_chat_id: groupId, limit: 10 },
      { tags: [`group-${groupId}-announcements`], revalidate: 60 }
    ),
  ]);

  if (!isServerSuccess(groupRes)) {
    if (isNotFoundError(groupRes)) {
      notFound();
    }
    return (
      <div className="max-w-4xl mx-auto">
        <ServerErrorCard
          title="加载群组失败"
          description={getServerErrorMessage(groupRes)}
          requestId={groupRes.request_id}
        />
      </div>
    );
  }

  const group = groupRes.data;
  const members = isServerSuccess(membersRes) ? membersRes.data?.items ?? [] : [];
  const announcements = isServerSuccess(announcementsRes) ? announcementsRes.data?.items ?? [] : [];

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <GroupHeader group={group} />
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="md:col-span-2 space-y-6">
          <GroupAnnouncements 
            groupId={groupId} 
            announcements={announcements}
            isAdmin={group.my_role === 'owner' || group.my_role === 'admin'}
          />
        </div>
        <div>
          <GroupMembers 
            groupId={groupId} 
            members={members}
            totalCount={group.members_count}
          />
        </div>
      </div>
    </div>
  );
}

export default async function GroupPage({ params }: GroupPageProps) {
  const { groupId } = await params;

  return (
    <Suspense fallback={<LoadingState text="加载中..." />}>
      <GroupContent groupId={groupId} />
    </Suspense>
  );
}
