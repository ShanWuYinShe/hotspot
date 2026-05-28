import { useState, useCallback } from "react";
import { momentsService } from "@/lib/api/services/moments";
import { isSuccess } from "@/lib/api/client";
import type { GeoPoint } from "@/lib/types/api";

interface CreatePostResult {
  submitting: boolean;
  createPost: (params: {
    description: string;
    point: GeoPoint;
    locationName: string;
    tags?: string[];
  }) => Promise<boolean>;
}

export function useCreatePost(): CreatePostResult {
  const [submitting, setSubmitting] = useState(false);

  const createPost = useCallback(
    async (params: {
      description: string;
      point: GeoPoint;
      locationName: string;
      tags?: string[];
    }): Promise<boolean> => {
      setSubmitting(true);
      try {
        const res = await momentsService.create({
          description: params.description,
          location: {
            point: params.point,
            location_name: params.locationName,
          },
          tags: params.tags,
        });
        return isSuccess(res);
      } catch {
        return false;
      } finally {
        setSubmitting(false);
      }
    },
    [],
  );

  return { submitting, createPost };
}
