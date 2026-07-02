-- Migration: 000065_exam_platform_core
-- Description: Exam platform core domain, space, class, and question-bank tables.

CREATE TABLE IF NOT EXISTS exam_domains (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS exam_subjects (
    id VARCHAR(36) PRIMARY KEY,
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(domain_id, code)
);

CREATE TABLE IF NOT EXISTS question_types (
    id VARCHAR(36) PRIMARY KEY,
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    answer_mode VARCHAR(64) NOT NULL DEFAULT 'objective',
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_question_types_unique
    ON question_types(domain_id, COALESCE(subject_id, ''), code);

CREATE TABLE IF NOT EXISTS knowledge_points (
    id VARCHAR(36) PRIMARY KEY,
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    parent_id VARCHAR(36) REFERENCES knowledge_points(id),
    code VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    path TEXT NOT NULL DEFAULT '',
    level INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_knowledge_points_unique
    ON knowledge_points(domain_id, COALESCE(subject_id, ''), code);

CREATE TABLE IF NOT EXISTS exam_spaces (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    owner_user_id VARCHAR(36),
    space_type VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_spaces_tenant_type ON exam_spaces(tenant_id, space_type);
CREATE INDEX IF NOT EXISTS idx_exam_spaces_owner ON exam_spaces(owner_user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_exam_spaces_personal_unique
    ON exam_spaces(tenant_id, owner_user_id)
    WHERE space_type = 'personal' AND owner_user_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS exam_classes (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    owner_user_id VARCHAR(36) NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    domain_id VARCHAR(36) REFERENCES exam_domains(id),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    invite_code VARCHAR(32) UNIQUE,
    invite_code_expires_at TIMESTAMPTZ,
    member_limit INTEGER NOT NULL DEFAULT 50,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_classes_tenant_owner ON exam_classes(tenant_id, owner_user_id);
CREATE INDEX IF NOT EXISTS idx_exam_classes_space ON exam_classes(space_id);
CREATE INDEX IF NOT EXISTS idx_exam_classes_domain ON exam_classes(domain_id);

CREATE TABLE IF NOT EXISTS exam_class_members (
    id VARCHAR(36) PRIMARY KEY,
    class_id VARCHAR(36) NOT NULL REFERENCES exam_classes(id) ON DELETE CASCADE,
    user_id VARCHAR(36) NOT NULL,
    tenant_id BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'student',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(class_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_class_members_user ON exam_class_members(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_exam_class_members_class ON exam_class_members(class_id, status);

CREATE TABLE IF NOT EXISTS question_banks (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_type VARCHAR(64) NOT NULL DEFAULT 'manual',
    review_status VARCHAR(32) NOT NULL DEFAULT 'private',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_question_banks_space ON question_banks(space_id);
CREATE INDEX IF NOT EXISTS idx_question_banks_tenant_creator ON question_banks(tenant_id, created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_question_banks_domain_subject ON question_banks(domain_id, subject_id);

CREATE TABLE IF NOT EXISTS questions (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id),
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    question_type_id VARCHAR(36) REFERENCES question_types(id),
    stem TEXT NOT NULL,
    difficulty VARCHAR(32) NOT NULL DEFAULT 'unknown',
    source_year INTEGER,
    source_region VARCHAR(128) NOT NULL DEFAULT '',
    review_status VARCHAR(32) NOT NULL DEFAULT 'private',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_questions_bank ON questions(question_bank_id);
CREATE INDEX IF NOT EXISTS idx_questions_domain_subject_type ON questions(domain_id, subject_id, question_type_id);
CREATE INDEX IF NOT EXISTS idx_questions_tenant_status ON questions(tenant_id, status);

CREATE TABLE IF NOT EXISTS question_options (
    id VARCHAR(36) PRIMARY KEY,
    question_id VARCHAR(36) NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    option_key VARCHAR(16) NOT NULL,
    content TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE(question_id, option_key)
);

CREATE TABLE IF NOT EXISTS question_answers (
    id VARCHAR(36) PRIMARY KEY,
    question_id VARCHAR(36) NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    answer_text TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS question_explanations (
    id VARCHAR(36) PRIMARY KEY,
    question_id VARCHAR(36) NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    explanation_text TEXT NOT NULL,
    source_type VARCHAR(64) NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS question_knowledge_points (
    question_id VARCHAR(36) NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    knowledge_point_id VARCHAR(36) NOT NULL REFERENCES knowledge_points(id),
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(question_id, knowledge_point_id)
);

CREATE TABLE IF NOT EXISTS question_chunk_refs (
    question_id VARCHAR(36) NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    chunk_id VARCHAR(36) NOT NULL,
    ref_type VARCHAR(64) NOT NULL DEFAULT 'evidence',
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(question_id, chunk_id, ref_type)
);

INSERT INTO exam_domains (id, code, name, description)
VALUES
    ('00000000-0000-0000-0000-000000000101', 'gaokao', '高考', '中国高考考试域'),
    ('00000000-0000-0000-0000-000000000102', 'ielts', '雅思', 'IELTS 考试域')
ON CONFLICT (code) DO NOTHING;

INSERT INTO exam_subjects (id, domain_id, code, name, sort_order)
VALUES
    ('00000000-0000-0000-0000-000000001101', '00000000-0000-0000-0000-000000000101', 'chinese', '语文', 10),
    ('00000000-0000-0000-0000-000000001102', '00000000-0000-0000-0000-000000000101', 'math', '数学', 20),
    ('00000000-0000-0000-0000-000000001103', '00000000-0000-0000-0000-000000000101', 'english', '英语', 30),
    ('00000000-0000-0000-0000-000000001201', '00000000-0000-0000-0000-000000000102', 'reading', 'Reading', 10),
    ('00000000-0000-0000-0000-000000001202', '00000000-0000-0000-0000-000000000102', 'listening', 'Listening', 20),
    ('00000000-0000-0000-0000-000000001203', '00000000-0000-0000-0000-000000000102', 'writing', 'Writing', 30),
    ('00000000-0000-0000-0000-000000001204', '00000000-0000-0000-0000-000000000102', 'speaking', 'Speaking', 40)
ON CONFLICT (domain_id, code) DO NOTHING;
