'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { Loader2, MapPin, LogIn } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from '@/components/ui/form';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Card, CardContent } from '@/components/ui/card';
import { MomentService } from '@/lib/api/services/moments';
import { PrivacyLevel } from '@/types/api';
import { useAuth } from '@/hooks/use-auth';
import { useLocation } from '@/hooks/use-location';
import { useLoginModal } from '@/providers/login-modal-provider';

const createSchema = (message: string) =>
  z.object({
    description: z.string().min(1, message),
    privacy_level: z.enum(['public', 'friends', 'private']),
    tags: z.string().optional(),
  });

export default function CreateMomentPage() {
  const router = useRouter();
  const [isLoading, setIsLoading] = useState(false);
  const { isAuthenticated } = useAuth();
  const { setOpen: openLoginModal } = useLoginModal();
  const { currentLocation, requestLocation } = useLocation();
  const t = useTranslations('moments.create');
  const tCommon = useTranslations('common');

  const formSchema = createSchema(tCommon('empty'));

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      description: '',
      privacy_level: 'public',
      tags: '',
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (!isAuthenticated) {
      toast.error(tCommon('retry'));
      openLoginModal(true);
      return;
    }

    setIsLoading(true);
    try {
      if (!currentLocation) {
        toast.error(tCommon('retry'));
        return;
      }
      const tags = values.tags
        ? values.tags.split(',').map((tag) => tag.trim()).filter(Boolean)
        : [];

      const response = await MomentService.create({
        description: values.description,
        privacy_level: values.privacy_level as PrivacyLevel,
        location: currentLocation,
        tags: tags,
        image_file_ids: [],
      });

      if (response.code === 0) {
        toast.success(tCommon('saved'));
        router.push('/feed');
      } else {
        toast.error(response.message || tCommon('retry'));
      }
    } finally {
      setIsLoading(false);
    }
  }

  if (!isAuthenticated) {
    return (
      <div className="max-w-xl mx-auto py-8">
        <h1 className="text-2xl font-bold mb-6">{t('title')}</h1>
        <Card>
          <CardContent className="py-12 text-center">
            <LogIn className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
            <h2 className="text-lg font-semibold mb-2">{t('loginTitle')}</h2>
            <p className="text-muted-foreground mb-6">{t('loginMessage')}</p>
            <Button onClick={() => openLoginModal(true)} size="lg">
              {t('loginAction')}
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-xl mx-auto py-8">
      <h1 className="text-2xl font-bold mb-6">{t('title')}</h1>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
          <FormField
            control={form.control}
            name="description"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('description')}</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder={t('description')}
                    className="min-h-[120px] resize-none"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="flex items-center space-x-2 text-sm text-muted-foreground p-3 bg-muted rounded-md">
            <MapPin className="h-4 w-4" />
            <span>{currentLocation?.location_name || tCommon('loading')}</span>
            {!currentLocation ? (
              <Button variant="ghost" size="sm" onClick={requestLocation}>
                {tCommon('confirm')}
              </Button>
            ) : null}
          </div>

          <FormField
            control={form.control}
            name="tags"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('tags')}</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder="travel, food, fun"
                    className="min-h-[40px] resize-none"
                    {...field}
                  />
                </FormControl>
                <FormDescription>{t('tagsExample')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="privacy_level"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('privacy')}</FormLabel>
                <Select onValueChange={field.onChange} defaultValue={field.value}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder={t('privacyPlaceholder')} />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value="public">{t('privacyOptions.public')}</SelectItem>
                    <SelectItem value="friends">{t('privacyOptions.friends')}</SelectItem>
                    <SelectItem value="private">{t('privacyOptions.private')}</SelectItem>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="flex gap-4 justify-end">
            <Button variant="outline" type="button" onClick={() => router.back()}>
              {tCommon('cancel')}
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t('submit')}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  );
}
