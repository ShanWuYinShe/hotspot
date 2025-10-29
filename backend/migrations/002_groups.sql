-- 群组表
CREATE TABLE groups (
    group_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_type VARCHAR(20) NOT NULL CHECK (group_type IN ('public_group', 'private_group')),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    avatar_url TEXT,
    owner_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    location_name VARCHAR(255) NOT NULL,
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    password_hash VARCHAR(255),
    max_members INTEGER DEFAULT 500,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 群组索引
CREATE INDEX idx_groups_name ON groups(name);
CREATE INDEX idx_groups_owner ON groups(owner_id);
CREATE INDEX idx_groups_location ON groups USING GIST(location);
CREATE INDEX idx_groups_type ON groups(group_type);

-- 自动更新 groups location 字段的触发器
CREATE OR REPLACE FUNCTION update_group_location()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)::geography;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_group_location
BEFORE INSERT OR UPDATE ON groups
FOR EACH ROW
EXECUTE FUNCTION update_group_location();

-- 群组成员表
CREATE TABLE group_members (
    id BIGSERIAL PRIMARY KEY,
    group_id UUID NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_muted BOOLEAN DEFAULT FALSE,
    muted_until TIMESTAMP WITH TIME ZONE,
    UNIQUE (group_id, user_id)
);

-- 群组成员索引
CREATE INDEX idx_group_members_group ON group_members(group_id);
CREATE INDEX idx_group_members_user ON group_members(user_id);
CREATE INDEX idx_group_members_role ON group_members(group_id, role);

-- 群组邀请表
CREATE TABLE group_invites (
    invite_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    inviter_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    invitee_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    message TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    responded_at TIMESTAMP WITH TIME ZONE
);

-- 群组邀请索引
CREATE INDEX idx_group_invites_invitee ON group_invites(invitee_id, status);
CREATE INDEX idx_group_invites_group ON group_invites(group_id);

-- 群组申请表
CREATE TABLE group_applications (
    application_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(group_id) ON DELETE CASCADE,
    applicant_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    message TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewer_id BIGINT REFERENCES users(user_id),
    review_reason TEXT
);

-- 群组申请索引
CREATE INDEX idx_group_applications_group ON group_applications(group_id, status);
CREATE INDEX idx_group_applications_applicant ON group_applications(applicant_id);

