import { useState, useCallback } from 'react';
import { momentsService } from '@/lib/api/services/moments';

interface FeedActions {
  likedIds: Set<string>;
  savedIds: Set<string>;
  toggleLike: (id: string) => void;
  toggleSave: (id: string) => void;
}

export function useFeedActions(): FeedActions {
  const [likedIds, setLikedIds] = useState<Set<string>>(new Set());
  const [savedIds, setSavedIds] = useState<Set<string>>(new Set());

  const toggleLike = useCallback(async (id: string) => {
    const wasLiked = likedIds.has(id);
    setLikedIds(p => { const n = new Set(p); wasLiked ? n.delete(id) : n.add(id); return n; });
    try {
      await (wasLiked ? momentsService.unlike({ moment_id: id }) : momentsService.like({ moment_id: id }));
    } catch {
      setLikedIds(p => { const n = new Set(p); wasLiked ? n.add(id) : n.delete(id); return n; });
    }
  }, [likedIds]);

  const toggleSave = useCallback(async (id: string) => {
    const wasSaved = savedIds.has(id);
    setSavedIds(p => { const n = new Set(p); wasSaved ? n.delete(id) : n.add(id); return n; });
    try {
      await (wasSaved ? momentsService.unfavorite({ moment_id: id }) : momentsService.favorite({ moment_id: id }));
    } catch {
      setSavedIds(p => { const n = new Set(p); wasSaved ? n.add(id) : n.delete(id); return n; });
    }
  }, [savedIds]);

  return { likedIds, savedIds, toggleLike, toggleSave };
}
