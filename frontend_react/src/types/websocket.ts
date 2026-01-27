export interface WebSocketEvent<TPayload = unknown> {
  type: string;
  payload: TPayload;
}

export type AnyWebSocketEvent = WebSocketEvent<unknown>;
