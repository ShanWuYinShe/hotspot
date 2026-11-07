-- 通用事务外盒（outbox）：业务事务内与状态变更同事务写入事件行，后台
-- dispatcher 在 COMMIT 后按 topic 投递（实时广播/外部副作用）；进程在
-- COMMIT 与投递之间崩溃时，事件仍在表中，重启后重试，消除「库里已生效、
-- 实时/通知未发出」的幻影落差。
CREATE TABLE IF NOT EXISTS outbox_events (
  event_id     BIGSERIAL PRIMARY KEY,
  topic        TEXT NOT NULL,
  payload      JSONB NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  attempts     INT NOT NULL DEFAULT 0,
  last_error   TEXT,
  processed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_outbox_due
  ON outbox_events (available_at, event_id) WHERE processed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_outbox_processed
  ON outbox_events (processed_at) WHERE processed_at IS NOT NULL;
