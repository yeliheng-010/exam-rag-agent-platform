package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const examClassMemberTestDDL = `
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    tenant_id INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE TABLE exam_class_members (
    id VARCHAR(36) PRIMARY KEY,
    class_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    tenant_id BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'student',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    joined_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
`

func TestExamClassRepositoryListMembersAddsDisplayFields(t *testing.T) {
	db := newExamClassMemberTestDB(t)
	repo := NewExamClassRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 7, 8, 15, 4, 5, 0, time.UTC)

	seedExamClassTestUser(t, db, "student-1", "张三", "student1@example.com", base)
	seedExamClassTestUser(t, db, "student-2", "张三", "student2@example.com", base)
	seedExamClassTestUser(t, db, "student-3", "李四", "student3@example.com", base.Add(time.Second))
	seedExamClassTestMember(t, db, "member-1", "student-1", types.ExamClassMemberStatusPending, base.Add(10*time.Millisecond))
	seedExamClassTestMember(t, db, "member-2", "student-2", types.ExamClassMemberStatusPending, base.Add(20*time.Millisecond))
	seedExamClassTestMember(t, db, "member-3", "student-3", types.ExamClassMemberStatusActive, base.Add(30*time.Millisecond))

	members, err := repo.ListMembers(ctx, "class-1", 10000, []types.ExamClassMemberStatus{
		types.ExamClassMemberStatusPending,
		types.ExamClassMemberStatusActive,
	})

	require.NoError(t, err)
	require.Len(t, members, 3)
	require.Equal(t, "张三", members[0].DisplayName)
	require.Equal(t, "张三", members[1].DisplayName)
	require.Equal(t, "李四", members[2].DisplayName)
	require.Equal(t, "20260708150405", members[0].DisplayID)
	require.Equal(t, "20260708150406", members[1].DisplayID)
	require.Equal(t, "20260708150407", members[2].DisplayID)
}

func newExamClassMemberTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(examClassMemberTestDDL).Error)
	return db
}

func seedExamClassTestUser(t *testing.T, db *gorm.DB, id string, username string, email string, createdAt time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(`
		INSERT INTO users (id, username, email, password_hash, tenant_id, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 'hash', 10000, true, ?, ?)
	`, id, username, email, createdAt, createdAt).Error)
}

func seedExamClassTestMember(t *testing.T, db *gorm.DB, id string, userID string, status types.ExamClassMemberStatus, createdAt time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(`
		INSERT INTO exam_class_members (id, class_id, user_id, tenant_id, role, status, joined_at, created_at, updated_at)
		VALUES (?, 'class-1', ?, 10000, 'student', ?, ?, ?, ?)
	`, id, userID, string(status), createdAt, createdAt, createdAt).Error)
}
