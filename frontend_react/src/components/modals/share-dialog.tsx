'use client';

import { useState } from 'react';
import { Link2, Twitter, Facebook } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

interface ShareDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  momentId: string;
  title?: string;
}

export function ShareDialog({ open, onOpenChange, momentId, title = 'Share Moment' }: ShareDialogProps) {
  const [isCopied, setIsCopied] = useState(false);
  const shareUrl = typeof window !== 'undefined' 
    ? `${window.location.origin}/moments/${momentId}` 
    : '';

  async function copyToClipboard() {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setIsCopied(true);
      toast.success('Link copied to clipboard');
      setTimeout(() => setIsCopied(false), 2000);
    } catch {
      toast.error('Failed to copy link');
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            Share this moment with others
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <h3 className="font-semibold mb-2">Copy Link</h3>
            <p className="text-sm text-muted-foreground mb-3">
              {shareUrl}
            </p>
            <Button
              onClick={copyToClipboard}
              variant={isCopied ? 'default' : 'outline'}
              className="w-full"
            >
              {isCopied ? 'Copied!' : 'Copy Link'}
            </Button>
          </div>

          <div>
            <h3 className="font-semibold mb-2">Share to Social Media</h3>
            <p className="text-sm text-muted-foreground mb-3">
              Share to your favorite platforms
            </p>
            <div className="grid grid-cols-3 gap-3">
              <Button
                variant="outline"
                size="icon"
                onClick={() => window.open(`https://twitter.com/intent/tweet?text=${encodeURIComponent(shareUrl)}`, '_blank')}
              >
                <Twitter className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="icon"
                onClick={() => window.open(`https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(shareUrl)}`, '_blank')}
              >
                <Facebook className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="icon"
                onClick={() => window.open(`https://wa.me/?text=${encodeURIComponent(shareUrl)}`, '_blank')}
              >
                <Link2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
