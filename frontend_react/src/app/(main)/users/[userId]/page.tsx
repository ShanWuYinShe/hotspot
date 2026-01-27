'use client';

import { useQuery } from '@tanstack/react-query';
import { useParams } from 'next/navigation';
import { Loader2, MapPin, Calendar } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { UserService } from '@/lib/api/services/user';
import { useAuth } from '@/hooks/use-auth';

export default function UserProfilePage() {
  const params = useParams();
  const userId = parseInt(params.userId as string);
  const { user: currentUser } = useAuth();
  const t = useTranslations('profile');
  const tCommon = useTranslations('common');

  const { data: profile, isLoading } = useQuery({
    queryKey: ['user', userId],
    queryFn: async () => {
      if (userId === currentUser?.user_id) {
        const res = await UserService.getProfile();
        if (res.code !== 0) throw new Error(res.message);
        return res.data!;
      }
      const res = await UserService.openTheDoor(userId);
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    enabled: !!userId,
  });

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!profile) {
    return <div className="text-center py-12">{tCommon('retry')}</div>;
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader className="flex flex-row gap-6 items-start">
          <Avatar className="h-24 w-24">
            <AvatarImage src={profile.avatar_url} alt={profile.nickname} />
            <AvatarFallback className="text-2xl">
              {profile.nickname.slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <div className="flex-1">
            <h1 className="text-2xl font-bold">{profile.nickname}</h1>
            <p className="text-muted-foreground mt-1">{profile.bio ?? tCommon('empty')}</p>
            <div className="flex items-center gap-4 mt-2 text-sm text-muted-foreground">
              <span className="flex items-center gap-1">
                <MapPin className="h-4 w-4" />
                {profile.location?.location_name ?? tCommon('empty')}
              </span>
              {'created_at' in profile && profile.created_at && (
                <span className="flex items-center gap-1">
                  <Calendar className="h-4 w-4" />
                  {new Date(profile.created_at).toLocaleDateString()}
                </span>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold">{profile.stats?.moments_count ?? 0}</div>
              <div className="text-sm text-muted-foreground">{t('moments')}</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{profile.stats?.friends_count ?? 0}</div>
              <div className="text-sm text-muted-foreground">{t('friends')}</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{profile.stats?.followers_count ?? 0}</div>
              <div className="text-sm text-muted-foreground">{t('followers')}</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{profile.stats?.following_count ?? 0}</div>
              <div className="text-sm text-muted-foreground">{t('following')}</div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('tags')}</CardTitle>
        </CardHeader>
        <CardContent>
          {profile.tags && profile.tags.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {profile.tags.map((tag) => (
                <span
                  key={tag}
                  className="px-3 py-1 bg-secondary rounded-full text-sm"
                >
                  {tag}
                </span>
              ))}
            </div>
          ) : (
            <p className="text-muted-foreground">{tCommon('empty')}</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
