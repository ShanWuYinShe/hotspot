import { BaseService } from './base.service'
import { PRIVATE_MESSAGES } from '~/config/api'
import type {
  PaginatedData,
  SendPrivateMessageRequest,
  PrivateMessagesHistoryRequest,
  PrivateMessageItem,
  UpdatePrivateMessageRequest,
  RecallPrivateMessageRequest,
  DeletePrivateMessageRequest,
  MarkPrivateMessagesReadRequest,
  AddPrivateMessageReactionRequest,
  RemovePrivateMessageReactionRequest,
  PrivateMessageReactionsRequest,
  MessageReactionItem
} from '~/types/api'

class PrivateMessagesService extends BaseService {
  async send(request: SendPrivateMessageRequest) {
    return this.post<PrivateMessageItem>(PRIVATE_MESSAGES.SEND, request)
  }

  async getPrivateMessages(request: PrivateMessagesHistoryRequest) {
    return this.post<PaginatedData<PrivateMessageItem>>(PRIVATE_MESSAGES.HISTORY, request)
  }

  async update(request: UpdatePrivateMessageRequest) {
    return this.post<void>(PRIVATE_MESSAGES.UPDATE, request)
  }

  async recall(request: RecallPrivateMessageRequest) {
    return this.post<void>(PRIVATE_MESSAGES.RECALL, request)
  }

  async deletePrivateMessage(request: DeletePrivateMessageRequest) {
    return this.post<void>(PRIVATE_MESSAGES.DELETE, request)
  }

  async markPrivateMessagesAsRead(request: MarkPrivateMessagesReadRequest) {
    return this.post<void>(PRIVATE_MESSAGES.MARK_AS_READ, request)
  }

  async addReaction(request: AddPrivateMessageReactionRequest) {
    return this.post<void>(PRIVATE_MESSAGES.ADD_REACTION, request)
  }

  async removeReaction(request: RemovePrivateMessageReactionRequest) {
    return this.post<void>(PRIVATE_MESSAGES.REMOVE_REACTION, request)
  }

  async getReactions(request: PrivateMessageReactionsRequest) {
    return this.post<PaginatedData<MessageReactionItem>>(PRIVATE_MESSAGES.REACTIONS, request)
  }
}

export const privateMessagesService = new PrivateMessagesService()
export function usePrivateMessagesService() { return privateMessagesService }
