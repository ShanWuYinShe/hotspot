import { post, type RequestOptions } from "../client";
import { SEARCH } from "../endpoints";
import type {
  SearchSuggestResponse,
  GeoPoint,
  SearchSuggestType,
} from "@/lib/types/api";

export const searchService = {
  suggest(
    params: {
      prefix: string;
      reference_point: GeoPoint;
      types?: SearchSuggestType[];
      radius?: number;
    },
    options?: RequestOptions,
  ) {
    return post<SearchSuggestResponse>(SEARCH.SUGGEST, params, options);
  },
};
