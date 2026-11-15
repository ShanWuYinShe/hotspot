-- outbox inline 领取按 topic 白名单精确定位（第 82 轮审计残留）：inline 领取此前用
-- "topic <> ALL(外部 topic)" 反向过滤，外部 topic（media.purge_objects）极端积压时
-- 每轮领取都要按 event_id 顺序扫描并丢弃这些行（O(积压)，实测 10 万积压 10.5ms、
-- Rows Removed by Filter=10 万），慢 topic 的积压会拖慢所有实时 topic 的投递。
-- 领取条件改为 "topic = ANY(inline topic)" 白名单后，需要一条按 topic 定位的索引，
-- 否则仍会退化为主键/全表顺序扫描；外部 topic 积压不再进入 inline 领取的扫描路径。
--
-- 部分索引谓词与领取查询完全一致（只有 processed_at IS NULL 的行才是待投递行，
-- 已处理行属保留期数据、不进索引），三列分别支撑：topic 白名单数组匹配、
-- available_at 过滤尚未到期的退避重试行（作为索引条件，不产生额外堆访问）、
-- event_id 提供与领取 SQL 一致的确定性顺序。
CREATE INDEX IF NOT EXISTS idx_outbox_topic_due
    ON outbox_events (topic, available_at, event_id) WHERE processed_at IS NULL;
