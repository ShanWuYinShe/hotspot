export const UserType = {
  Temp: "temp",
  Normal: "normal",
  Admin: "admin",
  System: "system",
} as const;
export type UserTypeValue = typeof UserType[keyof typeof UserType];

export const GroupRole = {
  Owner: "owner",
  Admin: "admin",
  Member: "member",
} as const;
export type GroupRoleValue = typeof GroupRole[keyof typeof GroupRole];

export const PrivacyLevel = {
  Public: "public",
  Friends: "friends",
  Private: "private",
} as const;
export type PrivacyLevelValue = typeof PrivacyLevel[keyof typeof PrivacyLevel];

export const ReactionType = {
  Like: "like",
  Love: "love",
  Laugh: "laugh",
  Angry: "angry",
  Sad: "sad",
  Wow: "wow",
} as const;
export type ReactionTypeValue = typeof ReactionType[keyof typeof ReactionType];

export const MessageType = {
  Text: "text",
  Image: "image",
  Video: "video",
  Audio: "audio",
  File: "file",
  System: "system",
  Location: "location",
} as const;
export type MessageTypeValue = typeof MessageType[keyof typeof MessageType];
