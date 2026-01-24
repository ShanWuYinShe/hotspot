/**
 * 个人中心内容 (Server Component)
 * 展示个人资料与入口
 */

import Link from 'next/link';
import {
  MapPin,
  Calendar,
  Settings,
  Edit,
  Users,
  Heart,
  Bookmark,
  MessageCircle,
  ChevronRight,
  Camera,
} from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Separator } from '@/components/ui/separator';
import { formatDateTime } from '@/lib/utils';
import type { MyProfileData } from '@/types/api';

interface ProfileContentProps {
  profile: MyProfileData;
}

export function ProfileContent({ profile }: ProfileContentProps) {
  const menuItems = [
    {
      icon: Heart,
      label: '我的关注',
      description: `${profile.stats.following_count} 人`,
      href: '/profile/following' as const,
    },
    {
      icon: Users,
      label: '我的粉丝',
      description: `${profile.stats.followers_count} 人`,
      href: '/profile/followers' as const,
    },
    {
      icon: Users,
      label: '我的好友',
      description: `${profile.stats.friends_count} 人`,
      href: '/profile/friends' as const,
    },
    {
      icon: Bookmark,
      label: '我的收藏',
      description: '查看收藏的瞬间',
      href: '/profile/favorites' as const,
    },
    {
      icon: MessageCircle,
      label: '我的群组',
      description: `${profile.stats.groups_count} 个`,
      href: '/profile/groups' as const,
    },
  ];

  return (
    <div className="space-y-6">
      {/* 个人信息卡片 */}
      <Card>
        <CardContent className="p-6">
          <div className="flex items-start gap-6">
            {/* 头像 */}
            <div className="relative group">
              <Avatar className="h-24 w-24">
                {profile.avatar_url && (
                  <AvatarImage src={profile.avatar_url} alt={profile.nickname} />
                )}
                <AvatarFallback className="text-2xl">
                  {profile.nickname.slice(0, 2).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <Link
                href="/settings"
                className="absolute inset-0 flex items-center justify-center rounded-full bg-black/50 opacity-0 transition-opacity group-hover:opacity-100"
              >
                <Camera className="h-6 w-6 text-white" />
              </Link>
            </div>

            {/* 用户信息 */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-3 mb-2">
                <h1 className="text-2xl font-bold truncate">{profile.nickname}</h1>
                <Badge variant="outline" className="text-xs">
                  {profile.is_online ? '在线' : '离线'}
                </Badge>
              </div>

              {profile.bio && (
                <p className="text-muted-foreground mb-3">{profile.bio}</p>
              )}

              {/* 标签 */}
              {profile.tags && profile.tags.length > 0 && (
                <div className="flex flex-wrap gap-1 mb-3">
                  {profile.tags.map((tag) => (
                    <Badge key={tag} variant="secondary">
                      {tag}
                    </Badge>
                  ))}
                </div>
              )}

              {/* 元信息 */}
              <div className="flex items-center gap-4 text-sm text-muted-foreground">
                <span className="flex items-center gap-1">
                  <MapPin className="h-4 w-4" />
                  {profile.location.location_name || '未设置位置'}
                </span>
                <span className="flex items-center gap-1">
                  <Calendar className="h-4 w-4" />
                  {formatDateTime(profile.created_at)}
                </span>
              </div>
            </div>

            {/* 编辑按钮 */}
            <Button variant="outline" asChild>
              <Link href="/settings">
                <Edit className="h-4 w-4 mr-2" />
                编辑
              </Link>
            </Button>
          </div>

          {/* 统计数据 */}
          <div className="grid grid-cols-4 gap-4 mt-6 pt-6 border-t">
            <Link href="/profile/moments" className="text-center hover:bg-accent/50 py-2 rounded-lg transition-colors">
              <p className="text-2xl font-bold">{profile.stats.moments_count}</p>
              <p className="text-sm text-muted-foreground">瞬间</p>
            </Link>
            <Link href="/profile/friends" className="text-center hover:bg-accent/50 py-2 rounded-lg transition-colors">
              <p className="text-2xl font-bold">{profile.stats.friends_count}</p>
              <p className="text-sm text-muted-foreground">好友</p>
            </Link>
            <Link href="/profile/following" className="text-center hover:bg-accent/50 py-2 rounded-lg transition-colors">
              <p className="text-2xl font-bold">{profile.stats.following_count}</p>
              <p className="text-sm text-muted-foreground">关注</p>
            </Link>
            <Link href="/profile/followers" className="text-center hover:bg-accent/50 py-2 rounded-lg transition-colors">
              <p className="text-2xl font-bold">{profile.stats.followers_count}</p>
              <p className="text-sm text-muted-foreground">粉丝</p>
            </Link>
          </div>
        </CardContent>
      </Card>

      {/* 菜单列表 */}
      <Card>
        <CardContent className="p-0">
          {menuItems.map((item, index) => (
            <div key={item.href}>
              <Link
                href={item.href}
                className="flex items-center gap-4 px-6 py-4 hover:bg-accent/50 transition-colors"
              >
                <div className="flex items-center justify-center h-10 w-10 rounded-lg bg-primary/10">
                  <item.icon className="h-5 w-5 text-primary" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-medium">{item.label}</p>
                  <p className="text-sm text-muted-foreground">{item.description}</p>
                </div>
                <ChevronRight className="h-5 w-5 text-muted-foreground" />
              </Link>
              {index < menuItems.length - 1 && <Separator />}
            </div>
          ))}
        </CardContent>
      </Card>

      {/* 设置入口 */}
      <Card>
        <CardContent className="p-0">
          <Link
            href="/settings"
            className="flex items-center gap-4 px-6 py-4 hover:bg-accent/50 transition-colors"
          >
            <div className="flex items-center justify-center h-10 w-10 rounded-lg bg-muted">
              <Settings className="h-5 w-5" />
            </div>
            <div className="flex-1">
              <p className="font-medium">设置</p>
              <p className="text-sm text-muted-foreground">账号安全、隐私、通知等</p>
            </div>
            <ChevronRight className="h-5 w-5 text-muted-foreground" />
          </Link>
        </CardContent>
      </Card>
    </div>
  );
}
