'use client';

import { useState } from 'react';
import { Send } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { CommentService } from '@/lib/api/services/comments';
import { toast } from 'sonner';

interface CommentInputProps {
  momentId: string;
  parentCommentId?: string;
  placeholder?: string;
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function CommentInput({ 
  momentId, 
  parentCommentId, 
  placeholder,
  onSuccess,
  onCancel
}: CommentInputProps) {
  const [isLoading, setIsLoading] = useState(false);
  const t = useTranslations('comments');
  const tCommon = useTranslations('common');

  const formSchema = z.object({
    content: z.string().min(1, tCommon('empty')),
  });
  
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      content: '',
    },
  });

  const onSubmit = async (values: z.infer<typeof formSchema>) => {
    setIsLoading(true);
    try {
      const response = await CommentService.create({
        moment_id: momentId,
        content: values.content,
        parent_id: parentCommentId,
      });

      if (response.code === 0) {
        toast.success(t('posted'));
        form.reset();
        onSuccess?.();
       } else {
        toast.error(response.message || tCommon('retry'));
      }
    } finally {
      setIsLoading(false);
    }

  };

  return (
    <div className="space-y-2">
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex gap-2">
        <Textarea
          placeholder={placeholder ?? t('placeholder')}
          {...form.register('content')}
          className="min-h-[60px] resize-none flex-1"
          disabled={isLoading}
        />
        <div className="flex flex-col gap-2">
          <Button 
            type="submit" 
            size="sm" 
            disabled={isLoading || !form.formState.isValid}
            className="mt-auto"
          >
            {isLoading ? (
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-t-transparent" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
          {onCancel && (
            <Button 
              type="button" 
              variant="ghost" 
              size="sm"
              onClick={onCancel}
            >
              Cancel
            </Button>
          )}
        </div>
      </form>
    </div>
  );
}
