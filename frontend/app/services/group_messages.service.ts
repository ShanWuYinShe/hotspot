import { BaseService } from './base.service'
import { GROUP_MESSAGES } from '~/config/api'
import type {
  PaginatedData,
  SendGroupMessageRequest,
  GroupMessagesHistoryRequest,
  GroupMessageItem,
  UpdateGroupMessageRequest,
  RecallGroupMessageRequest,
  DeleteGroupMessageRequest,
  MarkGroupMessagesReadRequest,
  MessageReadStatusRequest,
  MessageReadItem,
  AddGroupMessageReactionRequest,
  RemoveGroupMessageReactionRequest,
  GroupMessageReactionsRequest,
  MessageReactionItem
} from '~/types/api'

class GroupMessagesService extends BaseService {
  async send(request: SendGroupMessageRequest) {
    return this.post<GroupMessageItem>(GROUP_MESSAGES.SEND, request)
  }

  async getGroupMessages(request: GroupMessagesHistoryRequest) {
    return this.post<PaginatedData<GroupMessageItem>>(GROUP_MESSAGES.HISTORY, request)
  }

  async update(request: UpdateGroupMessageRequest) {
    return this.post<void>(GROUP_MESSAGES.UPDATE, request)
  }

  async recall(request: RecallGroupMessageRequest) {
    return this.post<void>(GROUP_MESSAGES.RECALL, request)
  }

  async deleteMessage(request: DeleteGroupMessageRequest) {
    return this.post<void>(GROUP_MESSAGES.DELETE, request)
  }

  async markAsRead(request: MarkGroupMessagesReadRequest) {
    return this.post<void>(GROUP_MESSAGES.MARK_AS_READ, request)
  }

  async getReadStatus(request: MessageReadStatusRequest) {
    return this.post<PaginatedData<MessageReadItem>>(GROUP_MESSAGES.READ_STATUS, request)
  }

  async addReaction(request: AddGroupMessageReactionRequest) {
    return this.post<void>(GROUP_MESSAGES.ADD_REACTION, request)
  }

  async removeReaction(request: RemoveGroupMessageReactionRequest) {
    return this.post<void>(GROUP_MESSAGES.REMOVE_REACTION, request)
  }

  async getReactions(request: GroupMessageReactionsRequest) {
    return this.post<PaginatedData<MessageReactionItem>>(GROUP_MESSAGES.REACTIONS, request)
  }
}

export const groupMessagesService = new GroupMessagesService()
export function useGroupMessagesService() { return groupMessagesService }
