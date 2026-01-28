'use client';

import * as React from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import * as groupsService from '@/lib/api/services/groups';
import { isSuccess } from '@/lib/api/client';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Input } from '@/components/ui/input';
import {
  ArrowLeft,
  Loader2,
  MapPin,
  Users,
  Lock,
  Globe,
  UserPlus,
  LogOut,
  Shield,
  MessageCircle,
} from 'lucide-react';

export default function GroupDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const groupId = params.id as string;

  const [password, setPassword] = React.useState('');

  // Fetch group
  const groupQuery = useQuery({
    queryKey: ['group', groupId],
    queryFn: async () => {
      const res = await groupsService.getGroup(groupId);
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
  });

  // Fetch members (limited)
  const membersQuery = useQuery({
    queryKey: ['group-members', groupId],
    queryFn: async () => {
      const res = await groupsService.getMembers({
        group_chat_id: groupId,
        limit: 10,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    enabled: !!groupQuery.data,
  });

  // Join mutation
  const joinMutation = useMutation({
    mutationFn: async () => {
      const res = await groupsService.joinGroup(groupId, password || undefined);
      if (!isSuccess(res)) throw new Error(res.message);
      return res;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['group', groupId] });
      queryClient.invalidateQueries({ queryKey: ['group-members', groupId] });
    },
  });

  // Leave mutation
  const leaveMutation = useMutation({
    mutationFn: async () => {
      const res = await groupsService.leaveGroup(groupId);
      if (!isSuccess(res)) throw new Error(res.message);
      return res;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['group', groupId] });
    },
  });

  const group = groupQuery.data;
  const members = membersQuery.data?.items ?? [];
  const isJoined = group?.my_state?.is_joined ?? false;
  const myRole = group?.my_state?.role;

  if (groupQuery.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
      </div>
    );
  }

  if (groupQuery.error || !group) {
    return (
      <Card className="max-w-2xl mx-auto shadow-soft">
        <CardContent className="py-12 text-center">
          <h2 className="text-lg font-semibold mb-2">群组不存在</h2>
          <p className="text-muted-foreground mb-4">该群组可能已被解散</p>
          <Button variant="outline" onClick={() => router.back()}>
            返回
          </Button>
        </CardContent>
      </Card>
    );
  }

  const handleJoin = () => {
    if (!isAuthenticated) {
      router.push(`/login?redirect=/groups/${groupId}`);
      return;
    }
    joinMutation.mutate();
  };

  const renderJoinSection = () => {
    if (!isAuthenticated) {
      return (
        <div className="p-4 rounded-lg bg-muted/50 text-center">
          <p className="text-muted-foreground text-sm mb-3">
            登录后即可加入群组
          </p>
          <Button asChild className="rounded-full">
            <Link href={`/login?redirect=/groups/${groupId}`}>
              登录
            </Link>
          </Button>
        </div>
      );
    }

    if (isJoined) {
      return (
        <div className="flex items-center justify-between p-4 rounded-lg bg-green-50 border border-green-200">
          <div className="flex items-center gap-2 text-green-700">
            <Users className="h-5 w-5" />
            <span className="font-medium">
              您已加入此群组
              {myRole === 'owner' && ' (群主)'}
              {myRole === 'admin' && ' (管理员)'}
            </span>
          </div>
          {myRole !== 'owner' && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => leaveMutation.mutate()}
              disabled={leaveMutation.isPending}
              className="text-red-600 hover:text-red-700 hover:bg-red-50"
            >
              {leaveMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <>
                  <LogOut className="h-4 w-4 mr-1" />
                  退出群组
                </>
              )}
            </Button>
          )}
        </div>
      );
    }

    return (
      <div className="space-y-4">
        {group.has_password && (
          <div className="space-y-2">
            <label className="text-sm font-medium">群组密码</label>
            <Input
              type="password"
              placeholder="请输入群组密码"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="rounded-lg"
            />
          </div>
        )}
        <Button
          onClick={handleJoin}
          disabled={joinMutation.isPending}
          className="w-full rounded-full"
        >
          {joinMutation.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              加入中...
            </>
          ) : (
            <>
              <UserPlus className="mr-2 h-4 w-4" />
              加入群组
            </>
          )}
        </Button>
        {joinMutation.error && (
          <p className="text-sm text-destructive text-center">
            {(joinMutation.error as Error).message}
          </p>
        )}
      </div>
    );
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Back Button */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => router.back()}
        className="gap-2"
      >
        <ArrowLeft className="h-4 w-4" />
        返回
      </Button>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Group Info Card */}
          <Card className="shadow-soft overflow-hidden">
            {/* Cover */}
            <div className="h-32 bg-gradient-to-br from-primary/30 via-primary/20 to-primary/10" />

            <CardContent className="relative pt-0 pb-6">
              {/* Avatar */}
              <div className="absolute -top-12 left-6">
                <Avatar className="h-24 w-24 ring-4 ring-background shadow-xl">
                  <AvatarImage src={group.avatar_url} alt={group.name} />
                  <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-2xl font-semibold">
                    {group.name.slice(0, 2).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
              </div>

              {/* Group Info */}
              <div className="mt-16 space-y-4">
                <div className="flex items-start justify-between">
                  <div>
                    <h1 className="text-2xl font-bold flex items-center gap-2">
                      {group.name}
                      {group.has_password ? (
                        <Lock className="h-5 w-5 text-muted-foreground" />
                      ) : (
                        <Globe className="h-5 w-5 text-green-500" />
                      )}
                    </h1>
                    {group.description && (
                      <p className="text-muted-foreground mt-1">
                        {group.description}
                      </p>
                    )}
                  </div>
                </div>

                {/* Meta Info */}
                <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
                  <div className="flex items-center gap-1.5">
                    <Users className="h-4 w-4" />
                    <span>{group.members_count} 成员</span>
                  </div>
                  {group.location && (
                    <div className="flex items-center gap-1.5">
                      <MapPin className="h-4 w-4" />
                      <span>{group.location.location_name}</span>
                    </div>
                  )}
                </div>

                {/* Tags */}
                {group.tags && group.tags.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {group.tags.map((tag) => (
                      <span
                        key={tag}
                        className="px-3 py-1 rounded-full bg-primary/10 text-primary text-sm"
                      >
                        #{tag}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Join / Chat Section */}
          <Card className="shadow-soft">
            <CardContent className="p-6">
              {renderJoinSection()}

              {/* Chat Preview for Members */}
              {isJoined && (
                <div className="mt-6 pt-6 border-t">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="font-semibold">群聊</h3>
                    <Button asChild variant="outline" size="sm" className="rounded-full">
                      <Link href={`/chat/group/${groupId}`}>
                        <MessageCircle className="h-4 w-4 mr-1" />
                        进入聊天
                      </Link>
                    </Button>
                  </div>
                  <div className="text-center py-8 text-muted-foreground">
                    <MessageCircle className="h-10 w-10 mx-auto mb-3 opacity-50" />
                    <p className="text-sm">点击进入聊天查看消息</p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Members Card */}
          <Card className="shadow-soft">
            <CardContent className="p-5">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold">成员</h3>
                <span className="text-sm text-muted-foreground">
                  {group.members_count}
                </span>
              </div>

              {membersQuery.isLoading ? (
                <div className="flex justify-center py-4">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <div className="space-y-3">
                  {members.map((member) => (
                    <Link
                      key={member.user.user_id}
                      href={`/users/${member.user.user_id}`}
                      className="flex items-center gap-3 group"
                    >
                      <Avatar className="h-9 w-9">
                        <AvatarImage src={member.user.avatar_url} alt={member.user.nickname} />
                        <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-xs">
                          {member.user.nickname.slice(0, 2).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium truncate group-hover:text-primary transition-colors">
                          {member.user.nickname}
                        </p>
                        <div className="flex items-center gap-1">
                          {member.role === 'owner' && (
                            <span className="text-xs text-primary flex items-center gap-0.5">
                              <Shield className="h-3 w-3" />
                              群主
                            </span>
                          )}
                          {member.role === 'admin' && (
                            <span className="text-xs text-blue-500 flex items-center gap-0.5">
                              <Shield className="h-3 w-3" />
                              管理员
                            </span>
                          )}
                        </div>
                      </div>
                    </Link>
                  ))}

                  {group.members_count > members.length && (
                    <Button variant="ghost" className="w-full text-sm" asChild>
                      <Link href={`/groups/${groupId}/members`}>
                        查看全部成员
                      </Link>
                    </Button>
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Group Info Card */}
          <Card className="shadow-soft">
            <CardContent className="p-5">
              <h3 className="font-semibold mb-4">群组信息</h3>
              <dl className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">创建时间</dt>
                  <dd>{new Date(group.created_at).toLocaleDateString('zh-CN')}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-muted-foreground">加入方式</dt>
                  <dd>
                    {group.has_password ? '需要密码' : '公开加入'}
                  </dd>
                </div>
              </dl>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
