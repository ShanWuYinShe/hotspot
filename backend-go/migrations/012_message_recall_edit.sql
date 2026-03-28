-- 012_message_recall_edit.sql
-- Add recall/edit tracking columns to message tables

ALTER TABLE group_messages ADD COLUMN IF NOT EXISTS is_recalled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE group_messages ADD COLUMN IF NOT EXISTS recalled_at TIMESTAMPTZ;
ALTER TABLE group_messages ADD COLUMN IF NOT EXISTS content_prev TEXT;
ALTER TABLE group_messages ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ;

ALTER TABLE private_messages ADD COLUMN IF NOT EXISTS is_recalled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE private_messages ADD COLUMN IF NOT EXISTS recalled_at TIMESTAMPTZ;
ALTER TABLE private_messages ADD COLUMN IF NOT EXISTS content_prev TEXT;
ALTER TABLE private_messages ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ;
