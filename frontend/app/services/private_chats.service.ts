import { BaseService } from './base.service'
import { PRIVATE_CHATS } from '~/config/api'
import type {
  PaginatedData,
  CreatePrivateChatRequest,
  CreatePrivateChatData,
  MyPrivateChatsRequest,
  PrivateChatListItem,
  PrivateChatDetailData,
  PrivateChatOpenTheDoorRequest,
  UpdatePrivateChatSettingsRequest,
  DeletePrivateChatRequest
} from '~/types/api'

class PrivateChatsService extends BaseService {
  async create(request: CreatePrivateChatRequest) {
    return this.post<CreatePrivateChatData>(PRIVATE_CHATS.CREATE, request)
  }

  async getPrivateChats(request: MyPrivateChatsRequest = {}) {
    return this.post<PaginatedData<PrivateChatListItem>>(PRIVATE_CHATS.ME, request)
  }

  async openTheDoor(request: PrivateChatOpenTheDoorRequest) {
    return this.post<PrivateChatDetailData>(PRIVATE_CHATS.OPEN_THE_DOOR, request)
  }

  async updateSettings(request: UpdatePrivateChatSettingsRequest) {
    return this.post<void>(PRIVATE_CHATS.UPDATE_SETTINGS, request)
  }

  async deletePrivateChat(request: DeletePrivateChatRequest) {
    return this.post<void>(PRIVATE_CHATS.DELETE, request)
  }
}

export const privateChatsService = new PrivateChatsService()
export function usePrivateChatsService() { return privateChatsService }
