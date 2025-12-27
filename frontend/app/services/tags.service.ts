import { BaseService } from './base.service'
import { TAGS } from '~/config/api'
import type {
  TrendingTagsRequest,
  TrendingTagsData,
  SuggestTagsRequest,
  TagSuggestData
} from '~/types/api'

class TagsService extends BaseService {
  async getTrending(request: TrendingTagsRequest = {}) {
    return this.publicPost<TrendingTagsData>(TAGS.TRENDING, request)
  }

  async suggest(request: SuggestTagsRequest) {
    return this.publicPost<TagSuggestData>(TAGS.SUGGEST, request)
  }
}

export const tagsService = new TagsService()
export function useTagsService() { return tagsService }
