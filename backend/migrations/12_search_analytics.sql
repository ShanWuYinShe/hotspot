-- ============================================================================
-- 12. 搜索分析系统表
-- ============================================================================
-- 
-- 功能职责：
-- - 记录用户搜索行为和统计数据
-- - 支持热门搜索词和搜索趋势分析
-- - 提供搜索性能监控和优化建议
-- - 管理搜索建议和自动补全
--
-- 依赖关系：
-- - 依赖：01_extensions.sql, 02_enums.sql, 03_base_functions.sql, 04_users.sql
--
-- ============================================================================

-- 搜索日志表
CREATE TABLE search_logs (
    id BIGSERIAL PRIMARY KEY,                   -- 日志ID（自增主键，用于高性能分页）
    search_id TEXT UNIQUE NOT NULL DEFAULT generate_unique_id(), -- 搜索唯一标识符
    user_id BIGINT,                            -- 搜索用户ID（可为NULL，支持匿名搜索）
    query TEXT NOT NULL,                       -- 搜索查询词
    search_type TEXT NOT NULL,                 -- 搜索类型：'user'|'group'|'moment'|'unified'
    results_count INTEGER NOT NULL DEFAULT 0,  -- 搜索结果数量
    clicked_result_id TEXT,                    -- 点击的结果ID
    search_duration_ms BIGINT NOT NULL DEFAULT 0, -- 搜索耗时（毫秒）
    -- 搜索上下文
    user_location GEOGRAPHY(POINT, 4326),      -- 搜索时的用户位置
    search_radius_km DOUBLE PRECISION,         -- 搜索半径（公里）
    -- 元数据
    user_agent TEXT,                           -- 用户代理信息
    ip_address INET,                           -- 用户IP地址
    metadata JSONB NOT NULL DEFAULT '{}',     -- 其他搜索元数据
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    
    -- 外键约束
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- 搜索统计表
CREATE TABLE search_statistics (
    id BIGSERIAL PRIMARY KEY,                   -- 统计ID
    query TEXT UNIQUE NOT NULL,                -- 搜索查询词
    search_count BIGINT NOT NULL DEFAULT 0,    -- 搜索次数
    click_count BIGINT NOT NULL DEFAULT 0,     -- 点击次数
    last_searched_at TIMESTAMP WITH TIME ZONE, -- 最后搜索时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 热门搜索表
CREATE TABLE popular_searches (
    id BIGSERIAL PRIMARY KEY,                   -- 热门搜索ID
    query TEXT UNIQUE NOT NULL,                -- 搜索查询词
    search_count BIGINT NOT NULL DEFAULT 0,    -- 搜索次数
    trend_score DOUBLE PRECISION NOT NULL DEFAULT 0.0, -- 趋势分数
    category TEXT NOT NULL DEFAULT 'general',  -- 搜索分类
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否活跃
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- 更新时间
);

-- 搜索建议表
CREATE TABLE search_suggestions (
    id BIGSERIAL PRIMARY KEY,                   -- 建议ID
    prefix TEXT NOT NULL,                      -- 搜索前缀
    suggestion TEXT NOT NULL,                  -- 建议内容
    score DOUBLE PRECISION NOT NULL DEFAULT 0.0, -- 建议分数
    category TEXT NOT NULL DEFAULT 'general',  -- 建议分类
    usage_count BIGINT NOT NULL DEFAULT 0,     -- 使用次数
    is_active BOOLEAN NOT NULL DEFAULT true,   -- 是否活跃
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- 更新时间
    
    UNIQUE(prefix, suggestion) -- 防止重复建议
);

-- ============================================================================
-- 索引创建
-- ============================================================================

-- 搜索日志索引
CREATE INDEX idx_search_logs_user_id ON search_logs(user_id);
CREATE INDEX idx_search_logs_query ON search_logs(query);
CREATE INDEX idx_search_logs_search_type ON search_logs(search_type);
CREATE INDEX idx_search_logs_created_at ON search_logs(created_at);
CREATE INDEX idx_search_logs_query_created_at ON search_logs(query, created_at);

-- 搜索统计索引
CREATE INDEX idx_search_statistics_search_count ON search_statistics(search_count DESC);
CREATE INDEX idx_search_statistics_last_searched_at ON search_statistics(last_searched_at DESC);

-- 热门搜索索引
CREATE INDEX idx_popular_searches_trend_score ON popular_searches(trend_score DESC);
CREATE INDEX idx_popular_searches_category ON popular_searches(category);
CREATE INDEX idx_popular_searches_is_active ON popular_searches(is_active);
CREATE INDEX idx_popular_searches_category_active ON popular_searches(category, is_active);

-- 搜索建议索引
CREATE INDEX idx_search_suggestions_prefix ON search_suggestions(prefix);
CREATE INDEX idx_search_suggestions_score ON search_suggestions(score DESC);
CREATE INDEX idx_search_suggestions_category ON search_suggestions(category);
CREATE INDEX idx_search_suggestions_prefix_active ON search_suggestions(prefix, is_active);

-- ============================================================================
-- 触发器创建
-- ============================================================================

-- 自动更新搜索统计的触发器
CREATE OR REPLACE FUNCTION update_search_statistics_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- 更新搜索统计
    INSERT INTO search_statistics (query, search_count, last_searched_at, updated_at)
    VALUES (NEW.query, 1, NEW.created_at, NEW.created_at)
    ON CONFLICT (query) 
    DO UPDATE SET 
        search_count = search_statistics.search_count + 1,
        last_searched_at = NEW.created_at,
        updated_at = NEW.created_at;
    
    -- 如果有点击，更新点击统计
    IF NEW.clicked_result_id IS NOT NULL THEN
        UPDATE search_statistics 
        SET click_count = click_count + 1, updated_at = NEW.created_at
        WHERE query = NEW.query;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建触发器
CREATE TRIGGER trigger_update_search_statistics
    AFTER INSERT ON search_logs
    FOR EACH ROW
    EXECUTE FUNCTION update_search_statistics_trigger();

-- 自动更新搜索建议使用次数的触发器
CREATE OR REPLACE FUNCTION update_suggestion_usage_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- 更新建议使用次数
    UPDATE search_suggestions 
    SET usage_count = usage_count + 1, updated_at = NEW.created_at
    WHERE suggestion = NEW.query AND is_active = true;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建触发器
CREATE TRIGGER trigger_update_suggestion_usage
    AFTER INSERT ON search_logs
    FOR EACH ROW
    EXECUTE FUNCTION update_suggestion_usage_trigger();

-- ============================================================================
-- 分区表创建（可选，用于大数据量场景）
-- ============================================================================

-- 按月分区搜索日志表（可选）
-- CREATE TABLE search_logs_y2024m01 PARTITION OF search_logs
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- ============================================================================
-- 初始数据插入
-- ============================================================================

-- 插入一些默认的搜索建议
INSERT INTO search_suggestions (prefix, suggestion, score, category) VALUES
    ('美', '美食', 90.0, 'food'),
    ('美', '美景', 85.0, 'travel'),
    ('美', '美妆', 80.0, 'lifestyle'),
    ('运', '运动', 95.0, 'sports'),
    ('运', '运动健身', 90.0, 'sports'),
    ('运', '运动装备', 85.0, 'sports'),
    ('旅', '旅行', 95.0, 'travel'),
    ('旅', '旅游', 90.0, 'travel'),
    ('旅', '旅行攻略', 85.0, 'travel'),
    ('学', '学习', 90.0, 'education'),
    ('学', '学生', 85.0, 'education'),
    ('学', '学校', 80.0, 'education'),
    ('工', '工作', 90.0, 'career'),
    ('工', '工具', 85.0, 'general'),
    ('工', '工程', 80.0, 'career'),
    ('生', '生活', 95.0, 'lifestyle'),
    ('生', '生日', 85.0, 'lifestyle'),
    ('生', '生活技巧', 90.0, 'lifestyle'),
    ('电', '电影', 95.0, 'entertainment'),
    ('电', '电视', 85.0, 'entertainment'),
    ('电', '电子', 80.0, 'technology'),
    ('音', '音乐', 95.0, 'entertainment'),
    ('音', '音响', 80.0, 'technology'),
    ('音', '音频', 75.0, 'technology');

-- 插入一些默认的热门搜索词
INSERT INTO popular_searches (query, search_count, trend_score, category) VALUES
    ('附近的人', 1000, 95.0, 'social'),
    ('美食推荐', 800, 90.0, 'food'),
    ('运动健身', 600, 85.0, 'sports'),
    ('旅行攻略', 500, 80.0, 'travel'),
    ('摄影技巧', 400, 75.0, 'photography'),
    ('学习交流', 350, 70.0, 'education'),
    ('电影推荐', 300, 65.0, 'entertainment'),
    ('音乐分享', 250, 60.0, 'music'),
    ('宠物日常', 200, 55.0, 'pets'),
    ('生活技巧', 150, 50.0, 'lifestyle');

-- ============================================================================
-- 表注释
-- ============================================================================

COMMENT ON TABLE search_logs IS '搜索日志表，记录用户的搜索行为和结果';
COMMENT ON TABLE search_statistics IS '搜索统计表，汇总搜索词的使用频次和趋势';
COMMENT ON TABLE popular_searches IS '热门搜索表，存储计算后的热门搜索词和趋势分数';
COMMENT ON TABLE search_suggestions IS '搜索建议表，提供自动补全和搜索建议功能';

-- 列注释
COMMENT ON COLUMN search_logs.search_id IS '搜索唯一标识符，用于跟踪搜索会话';
COMMENT ON COLUMN search_logs.query IS '用户输入的搜索查询词';
COMMENT ON COLUMN search_logs.search_type IS '搜索类型，区分不同的搜索场景';
COMMENT ON COLUMN search_logs.results_count IS '搜索返回的结果数量';
COMMENT ON COLUMN search_logs.clicked_result_id IS '用户点击的搜索结果ID，用于计算点击率';
COMMENT ON COLUMN search_logs.search_duration_ms IS '搜索处理耗时，用于性能监控';

COMMENT ON COLUMN search_statistics.search_count IS '搜索词的总搜索次数';
COMMENT ON COLUMN search_statistics.click_count IS '搜索词对应结果的总点击次数';

COMMENT ON COLUMN popular_searches.trend_score IS '趋势分数，综合考虑搜索频次、时间衰减、增长率等因素';
COMMENT ON COLUMN popular_searches.category IS '搜索分类，用于分类展示热门搜索';

COMMENT ON COLUMN search_suggestions.prefix IS '搜索前缀，用于匹配用户输入';
COMMENT ON COLUMN search_suggestions.suggestion IS '建议的完整搜索词';
COMMENT ON COLUMN search_suggestions.score IS '建议的相关性分数，影响排序';
COMMENT ON COLUMN search_suggestions.usage_count IS '建议被使用的次数，用于优化建议质量';
