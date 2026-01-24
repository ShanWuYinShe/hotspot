/**
 * 搜索结果组件 (Server Component)
 * 服务端获取搜索结果
 */

import { Users, UsersRound, Compass, MapPin } from '@/components/icons';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { EmptyState } from '@/components/common/empty-state';
import { UserCard } from '@/components/cards/user-card';
import { GroupCard } from '@/components/cards/group-card';
import { MomentCard } from '@/components/cards/moment-card';
import type { UserItem, GroupItem, MomentItem, PaginatedData, GeoPoint } from '@/types/api';

type SearchTab = 'users' | 'groups' | 'moments';

interface SearchResultsProps {
  keyword: string;
  tab: SearchTab;
  referencePoint?: GeoPoint;
}

export async function SearchResults({ keyword, tab, referencePoint }: SearchResultsProps) {
  if (!keyword) {
    return (
      <EmptyState
        icon={Users}
        title="开始搜索"
        description="输入关键词搜索用户、群组或瞬间"
      />
    );
  }

  if (!referencePoint) {
    return (
      <EmptyState
        icon={MapPin}
        title="需要位置信息"
        description="请允许定位以获得更准确的搜索结果"
      />
    );
  }

  // 根据 tab 类型获取对应结果
  if (tab === 'users') {
    const response = await serverApi.post<PaginatedData<UserItem>>(
      ENDPOINTS.USER.SEARCH_BY_KEYWORD,
      { kw: keyword, reference_point: referencePoint, limit: 20 },
      { revalidate: 60 }
    );
    
    const users = isServerSuccess(response) ? response.data?.items ?? [] : [];
    
    if (users.length === 0) {
      return (
        <EmptyState icon={Users} title="暂无结果" description="试试其他关键词" />
      );
    }

    return (
      <div className="space-y-3">
        {users.map((user) => (
          <UserCard key={user.user_id} user={user} />
        ))}
      </div>
    );
  }

  if (tab === 'groups') {
    const response = await serverApi.post<PaginatedData<GroupItem>>(
      ENDPOINTS.GROUP_CHATS.SEARCH_BY_KEYWORD,
      { kw: keyword, reference_point: referencePoint, limit: 20 },
      { revalidate: 60 }
    );
    
    const groups = isServerSuccess(response) ? response.data?.items ?? [] : [];
    
    if (groups.length === 0) {
      return (
        <EmptyState icon={UsersRound} title="暂无结果" description="试试其他关键词" />
      );
    }

    return (
      <div className="space-y-3">
        {groups.map((group) => (
          <GroupCard key={group.group_chat_id} group={group} />
        ))}
      </div>
    );
  }

  // moments
  const response = await serverApi.post<PaginatedData<MomentItem>>(
    ENDPOINTS.MOMENTS.SEARCH_BY_KEYWORD,
    { kw: keyword, reference_point: referencePoint, limit: 20 },
    { revalidate: 60 }
  );
  
  const moments = isServerSuccess(response) ? response.data?.items ?? [] : [];
  
  if (moments.length === 0) {
    return (
      <EmptyState icon={Compass} title="暂无结果" description="试试其他关键词" />
    );
  }

  return (
    <div className="space-y-4">
      {moments.map((moment) => (
        <MomentCard key={moment.moment_id} moment={moment} />
      ))}
    </div>
  );
}
