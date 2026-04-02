ALTER TABLE private_messages ADD COLUMN IF NOT EXISTS file_id TEXT;
ALTER TABLE private_messages ADD COLUMN IF NOT EXISTS reply_to_message_id TEXT NULL REFERENCES private_messages(message_id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_private_messages_chat_key_reply_to
    ON private_messages (chat_key, reply_to_message_id);
