'use client';

import * as React from 'react';
import Link from 'next/link';
import { useQuery, useInfiniteQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import * as usersService from '@/lib/api/services/users';
import * as momentsService from '@/lib/api/services/moments';
import { isSuccess } from '@/lib/api/client';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Loader2,
  Settings,
  Edit3,
  MapPin,
  Users,
  UserPlus,
  Heart,
  Bookmark,
  Image as ImageIcon,
  MessageCircle,
  Grid3X3,
  Shield,
} from 'lucide-react';
import type { UserMomentListItem } from '@/types/api';

type TabType = 'moments' | 'liked' | 'favorites';

export default function ProfilePage() {
  const { user } = useAuthStore();
  const [activeTab, setActiveTab] = React.useState<TabType>('moments');

  // Fetch full profile
  const profileQuery = useQuery({
    queryKey: ['my-profile'],
    queryFn: async () => {
      const res = await usersService.getMyProfile();
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
  });

  // Fetch moments based on tab
  const momentsQuery = useInfiniteQuery({
    queryKey: ['my-moments', activeTab],
    queryFn: async ({ pageParam }) => {
      let res;
      if (activeTab === 'moments') {
        res = await momentsService.getMyMoments({ cursor: pageParam as string | undefined, limit: 12 });
      } else if (activeTab === 'liked') {
        res = await momentsService.getMyLikedMoments({ cursor: pageParam as string | undefined, limit: 12 });
      } else {
        res = await momentsService.getMyFavorites({ cursor: pageParam as string | undefined, limit: 12 });
      }
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
  });

  const profile = profileQuery.data || user;
  const moments = momentsQuery.data?.pages.flatMap((page) => page?.items ?? []) ?? [];

  const tabs = [
    { value: 'moments', label: '瞬间', icon: Grid3X3 },
    { value: 'liked', label: '喜欢', icon: Heart },
    { value: 'favorites', label: '收藏', icon: Bookmark },
  ] as const;

  const quickLinks = [
    { href: '/profile/friends', label: '好友', icon: Users, count: profile?.stats.friends_count, color: 'text-blue-500 bg-blue-50' },
    { href: '/profile/followers', label: '粉丝', icon: UserPlus, count: profile?.stats.followers_count, color: 'text-green-500 bg-green-50' },
    { href: '/profile/following', label: '关注', icon: Heart, count: profile?.stats.following_count, color: 'text-red-500 bg-red-50' },
    { href: '/profile/blocked', label: '黑名单', icon: Shield, count: undefined, color: 'text-gray-500 bg-gray-50' },
  ];

  if (profileQuery.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Profile Card */}
      <Card className="shadow-soft overflow-hidden">
        {/* Cover */}
        <div className="h-32 bg-gradient-to-br from-primary/30 via-primary/20 to-primary/10" />

        <CardContent className="relative pt-0 pb-6">
          {/* Avatar */}
          <div className="absolute -top-16 left-6">
            <Avatar className="h-32 w-32 ring-4 ring-background shadow-xl">
              <AvatarImage src={profile?.avatar_url} alt={profile?.nickname} />
              <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-3xl font-semibold">
                {profile?.nickname?.slice(0, 2).toUpperCase() || 'U'}
              </AvatarFallback>
            </Avatar>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-4">
            <Button asChild variant="outline" size="sm" className="rounded-full gap-2">
              <Link href="/settings/profile">
                <Edit3 className="h-4 w-4" />
                编辑资料
              </Link>
            </Button>
            <Button asChild variant="ghost" size="icon" className="rounded-full">
              <Link href="/settings">
                <Settings className="h-4 w-4" />
              </Link>
            </Button>
          </div>

          {/* User Info */}
          <div className="mt-6 space-y-4">
            <div>
              <h1 className="text-2xl font-bold">{profile?.nickname}</h1>
              {profile?.bio && (
                <p className="text-muted-foreground mt-1">{profile.bio}</p>
              )}
            </div>

            {/* Meta Info */}
            <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
              {profile?.location && (
                <div className="flex items-center gap-1.5">
                  <MapPin className="h-4 w-4" />
                  <span>{profile.location.location_name}</span>
                </div>
              )}
              {profile?.email && (
                <span>{profile.email}</span>
              )}
            </div>

            {/* Tags */}
            {profile?.tags && profile.tags.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {profile.tags.map((tag) => (
                  <span
                    key={tag}
                    className="px-3 py-1 rounded-full bg-primary/10 text-primary text-sm"
                  >
                    #{tag}
                  </span>
                ))}
              </div>
            )}

            {/* Stats */}
            <div className="flex items-center gap-6 pt-4 border-t">
              <div className="text-center">
                <p className="text-2xl font-bold">{profile?.stats.moments_count}</p>
                <p className="text-sm text-muted-foreground">瞬间</p>
              </div>
              <Link href="/profile/followers" className="text-center hover:text-primary transition-colors">
                <p className="text-2xl font-bold">{profile?.stats.followers_count}</p>
                <p className="text-sm text-muted-foreground">粉丝</p>
              </Link>
              <Link href="/profile/following" className="text-center hover:text-primary transition-colors">
                <p className="text-2xl font-bold">{profile?.stats.following_count}</p>
                <p className="text-sm text-muted-foreground">关注</p>
              </Link>
              <Link href="/profile/friends" className="text-center hover:text-primary transition-colors">
                <p className="text-2xl font-bold">{profile?.stats.friends_count}</p>
                <p className="text-sm text-muted-foreground">好友</p>
              </Link>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Quick Links */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {quickLinks.map((link) => (
          <Link key={link.href} href={link.href}>
            <Card className="group hover:shadow-lg hover:border-primary/30 transition-all cursor-pointer">
              <CardContent className="flex items-center gap-3 py-4 px-4">
                <div className={cn('w-10 h-10 rounded-xl flex items-center justify-center group-hover:scale-110 transition-transform', link.color)}>
                  <link.icon className="h-5 w-5" />
                </div>
                <div>
                  <p className="font-medium">{link.label}</p>
                  {link.count !== undefined && (
                    <p className="text-sm text-muted-foreground">{link.count}</p>
                  )}
                </div>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>

      {/* Moments Section */}
      <Card className="shadow-soft">
        <CardContent className="p-6">
          {/* Tabs */}
          <div className="flex items-center gap-2 mb-6">
            {tabs.map((tab) => (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                className={cn(
                  'flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium transition-all',
                  activeTab === tab.value
                    ? 'bg-primary text-primary-foreground shadow-md'
                    : 'bg-muted/50 text-muted-foreground hover:bg-muted'
                )}
              >
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </button>
            ))}
          </div>

          {/* Content */}
          {momentsQuery.isLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : moments.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <ImageIcon className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>
                {activeTab === 'moments'
                  ? '还没有发布瞬间'
                  : activeTab === 'liked'
                  ? '还没有喜欢的瞬间'
                  : '还没有收藏的瞬间'}
              </p>
            </div>
          ) : (
            <>
              {/* Grid Layout for Images */}
              <div className="grid grid-cols-3 gap-2">
                {moments
                  .filter((m) => m.images && m.images.length > 0)
                  .map((moment) => (
                    <Link
                      key={moment.moment_id}
                      href={`/moments/${moment.moment_id}`}
                      className="relative aspect-square rounded-lg overflow-hidden bg-muted group"
                    >
                      <img
                        src={moment.images?.[0]?.url}
                        alt=""
                        className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                      />
                      <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
                        <div className="flex items-center gap-4 text-white">
                          <div className="flex items-center gap-1">
                            <Heart className="h-4 w-4" />
                            <span className="text-sm">{moment.likes_count}</span>
                          </div>
                          <div className="flex items-center gap-1">
                            <MessageCircle className="h-4 w-4" />
                            <span className="text-sm">{moment.comments_count}</span>
                          </div>
                        </div>
                      </div>
                    </Link>
                  ))}
              </div>

              {/* Text-only Moments */}
              <div className="mt-4 space-y-2">
                {moments
                  .filter((m) => !m.images || m.images.length === 0)
                  .map((moment) => (
                    <MomentPreview key={moment.moment_id} moment={moment} />
                  ))}
              </div>

              {/* Load More */}
              {momentsQuery.hasNextPage && (
                <div className="flex justify-center pt-4">
                  <Button
                    variant="outline"
                    onClick={() => momentsQuery.fetchNextPage()}
                    disabled={momentsQuery.isFetchingNextPage}
                    className="rounded-full"
                  >
                    {momentsQuery.isFetchingNextPage ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        加载中...
                      </>
                    ) : (
                      '查看更多'
                    )}
                  </Button>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

interface MomentPreviewProps {
  moment: UserMomentListItem;
}

function MomentPreview({ moment }: MomentPreviewProps) {
  return (
    <Link
      href={`/moments/${moment.moment_id}`}
      className="block p-4 rounded-lg bg-muted/50 hover:bg-muted transition-colors"
    >
      <p className="line-clamp-2 text-sm">{moment.description}</p>
      <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
        <span>{new Date(moment.created_at).toLocaleDateString('zh-CN')}</span>
        <div className="flex items-center gap-1">
          <Heart className="h-3 w-3" />
          <span>{moment.likes_count}</span>
        </div>
        <div className="flex items-center gap-1">
          <MessageCircle className="h-3 w-3" />
          <span>{moment.comments_count}</span>
        </div>
      </div>
    </Link>
  );
}
