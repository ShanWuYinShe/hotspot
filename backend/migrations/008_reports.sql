-- 举报表
CREATE TABLE reports (
    id BIGSERIAL PRIMARY KEY,
    report_id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    reporter_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('user', 'moment', 'comment', 'message', 'group')),
    target_id VARCHAR(255) NOT NULL,
    reason VARCHAR(100) NOT NULL,
    description TEXT,
    status report_status NOT NULL DEFAULT 'pending'::report_status CHECK (status != 'any'::report_status),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    handled_at TIMESTAMP WITH TIME ZONE,
    handler_id BIGINT REFERENCES users(user_id),
    handler_note TEXT
);

-- 举报索引
CREATE INDEX idx_reports_id ON reports(id DESC);
CREATE UNIQUE INDEX idx_reports_report_id ON reports(report_id);
CREATE INDEX idx_reports_reporter ON reports(reporter_id);
CREATE INDEX idx_reports_target ON reports(target_type, target_id);
CREATE INDEX idx_reports_status ON reports(status, id DESC);

