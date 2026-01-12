-- Add ban fields to users and dismiss reason to group_chats

ALTER TABLE users ADD COLUMN ban_reason TEXT;
ALTER TABLE users ADD COLUMN banned_until TIMESTAMPTZ;

ALTER TABLE group_chats ADD COLUMN dismiss_reason TEXT;
