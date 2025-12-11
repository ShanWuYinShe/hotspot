import { BaseService } from './base.service'
import { PRIVATE_MESSAGES } from '~/config/api'
import type {
  PaginatedData,
  SendPrivateMessageRequest,
  SendPrivateMessageData,
  GetPrivateMessagesRequest,
  PrivateMessageItem,
  UpdatePrivateMessageRequest,
  RecallPrivateMessageRequest,
  DeletePrivateMessageRequest,
  MarkPrivateMessagesReadRequest,
  AddPrivateMessageReactionRequest,
  RemovePrivateMessageReactionRequest,
  GetPrivateMessageReactionsRequest,
  MessageReactionItem
} from '~/types/api'

class PrivateMessagesService extends BaseService {
  async send(request: SendPrivateMessageRequest) {
    return this.post<SendPrivateMessageData>(PRIVATE_MESSAGES.SEND, request)
  }

  async getPrivateMessages(request: GetPrivateMessagesRequest) {
    return this.post<PaginatedData<PrivateMessageItem>>(PRIVATE_MESSAGES.GET_HISTORY, request)
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

  async getReactions(request: GetPrivateMessageReactionsRequest) {
    return this.post<PaginatedData<MessageReactionItem>>(PRIVATE_MESSAGES.GET_REACTIONS_LIST, request)
  }
}

export const privateMessagesService = new PrivateMessagesService()
export function usePrivateMessagesService() { return privateMessagesService }
