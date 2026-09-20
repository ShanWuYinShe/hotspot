-- 群加入申请与举报的并发重复兜底唯一索引（审查 L-1 / L-2）。
--
-- 两处「查重-插入」均为 check-then-act：并发双击 join 会产生两条 pending
-- 申请（管理员收到双份通知），并发举报会落多行。服务层查重保留（返回
-- 已有记录的自然去重语义），数据库侧补部分唯一索引作竞态兜底。
--
-- 部分索引（WHERE status='pending'）：同群同人可存在多条历史申请
-- （approved/rejected），只约束「至多一条 pending」。

-- 存量清理：保留每组 (group_id, applicant_user_id) 最早一条 pending，删并发产生的重复。
DELETE FROM group_applications a
USING group_applications b
WHERE a.application_id <> b.application_id
  AND a.group_id = b.group_id
  AND a.applicant_user_id = b.applicant_user_id
  AND a.status = 'pending'
  AND b.status = 'pending'
  AND (a.created_at > b.created_at
       OR (a.created_at = b.created_at AND a.application_id > b.application_id));

CREATE UNIQUE INDEX IF NOT EXISTS uq_group_applications_pending
  ON group_applications(group_id, applicant_user_id) WHERE status = 'pending';

-- 举报：同人同目标至多一条（重复举报按重复语义返回既有 report_id）。
DELETE FROM reports a
USING reports b
WHERE a.report_id <> b.report_id
  AND a.creator_user_id = b.creator_user_id
  AND a.target_type = b.target_type
  AND a.target_id = b.target_id
  AND (a.created_at > b.created_at
       OR (a.created_at = b.created_at AND a.report_id > b.report_id));

CREATE UNIQUE INDEX IF NOT EXISTS uq_reports_creator_target
  ON reports(creator_user_id, target_type, target_id);
