import { BaseService } from './base.service'
import { TAGS } from '~/config/api'
import type {
  GetTrendingTagsRequest,
  TrendingTagsData,
  SuggestTagsRequest,
  TagSuggestData
} from '~/types/api'

class TagsService extends BaseService {
  async getTrending(request: GetTrendingTagsRequest = {}) {
    return this.publicPost<TrendingTagsData>(TAGS.GET_TRENDING, request)
  }

  async suggest(request: SuggestTagsRequest) {
    return this.publicPost<TagSuggestData>(TAGS.SUGGEST, request)
  }
}

export const tagsService = new TagsService()
export function useTagsService() { return tagsService }
