import { AlertTriangle } from '@/components/icons';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface ServerErrorCardProps {
  title?: string;
  description?: string;
  requestId?: string;
  className?: string;
}

export function ServerErrorCard({
  title = '加载失败',
  description = '当前无法获取数据，请稍后再试。',
  requestId,
  className,
}: ServerErrorCardProps) {
  return (
    <Card className={cn('border-dashed', className)}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <AlertTriangle className="h-5 w-5 text-destructive" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2 text-sm text-muted-foreground">
        <p>{description}</p>
        {requestId && <p className="text-xs text-muted-foreground">request_id: {requestId}</p>}
      </CardContent>
    </Card>
  );
}
