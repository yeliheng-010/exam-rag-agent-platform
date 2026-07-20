package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExamClassRepositorySettings(t *testing.T) {
	t.Run("filters default and management lists", testExamClassRepositorySettingsLists)
	t.Run("reads archived classes within tenant", testExamClassRepositorySettingsArchivedReads)
	t.Run("updates metadata atomically", testExamClassRepositorySettingsMetadata)
	t.Run("transitions status from expected state", testExamClassRepositorySettingsTransition)
	t.Run("approves members within limit", testExamClassRepositorySettingsApproval)
	t.Run("serializes concurrent approvals", testExamClassRepositorySettingsConcurrentApproval)
}

func testExamClassRepositorySettingsLists(t *testing.T) {
	db := newExamClassSettingsTestDB(t)
	repo := NewExamClassRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)

	seedExamClassSettingsClass(t, db, "active", 1, "owner", types.ExamClassStatusActive, 50, base.Add(time.Minute))
	seedExamClassSettingsClass(t, db, "archived", 1, "owner", types.ExamClassStatusArchived, 50, base)
	seedExamClassSettingsClass(t, db, "other-tenant", 2, "owner", types.ExamClassStatusActive, 50, base.Add(2*time.Minute))
	seedExamClassSettingsMember(t, db, "active", 1, "user", types.ExamClassMemberStatusActive, base)
	seedExamClassSettingsMember(t, db, "archived", 1, "user", types.ExamClassMemberStatusActive, base)
	seedExamClassSettingsMember(t, db, "other-tenant", 2, "user", types.ExamClassMemberStatusActive, base)

	classes, err := repo.ListByUser(ctx, 1, "user")
	require.NoError(t, err)
	require.Equal(t, []string{"active"}, examClassSettingsIDs(classes))

	classes, err = repo.ListByUserWithStatuses(ctx, 1, "user", []types.ExamClassStatus{
		types.ExamClassStatusActive,
		types.ExamClassStatusArchived,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"active", "archived"}, examClassSettingsIDs(classes))
}

func testExamClassRepositorySettingsArchivedReads(t *testing.T) {
	db := newExamClassSettingsTestDB(t)
	repo := NewExamClassRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	seedExamClassSettingsClass(t, db, "archived", 1, "owner", types.ExamClassStatusArchived, 50, base)

	_, err := repo.GetByIDAndTenant(ctx, "archived", 1)
	require.ErrorIs(t, err, ErrExamClassNotFound)

	class, err := repo.GetByIDAndTenantIncludingArchived(ctx, "archived", 1)
	require.NoError(t, err)
	require.Equal(t, types.ExamClassStatusArchived, class.Status)

	_, err = repo.GetByIDAndTenantIncludingArchived(ctx, "archived", 2)
	require.ErrorIs(t, err, ErrExamClassNotFound)

	classes, err := repo.ListByIDsAndTenantIncludingArchived(ctx, 1, []string{"archived", "missing"})
	require.NoError(t, err)
	require.Equal(t, []string{"archived"}, examClassSettingsIDs(classes))
}

func testExamClassRepositorySettingsMetadata(t *testing.T) {
	db := newExamClassSettingsTestDB(t)
	repo := NewExamClassRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	seedExamClassSettingsClass(t, db, "class", 1, "owner", types.ExamClassStatusActive, 50, base)
	seedExamClassSettingsMember(t, db, "class", 1, "owner", types.ExamClassMemberStatusActive, base)
	seedExamClassSettingsMember(t, db, "class", 1, "student", types.ExamClassMemberStatusActive, base)

	err := repo.UpdateClassMetadata(ctx, 1, "class", "owner", "新名称", "新描述", 2, base.Add(time.Hour))
	require.NoError(t, err)
	class, err := repo.GetByIDAndTenantIncludingArchived(ctx, "class", 1)
	require.NoError(t, err)
	require.Equal(t, "新名称", class.Name)
	require.Equal(t, "新描述", class.Description)
	require.Equal(t, 2, class.MemberLimit)

	err = repo.UpdateClassMetadata(ctx, 1, "class", "owner", "不应保存", "不应保存", 1, base.Add(2*time.Hour))
	require.ErrorIs(t, err, ErrExamClassMemberLimitConflict)
	class, err = repo.GetByIDAndTenantIncludingArchived(ctx, "class", 1)
	require.NoError(t, err)
	require.Equal(t, "新名称", class.Name)
	require.Equal(t, 2, class.MemberLimit)

	err = repo.UpdateClassMetadata(ctx, 1, "class", "other", "无权更新", "", 2, base.Add(3*time.Hour))
	require.ErrorIs(t, err, ErrExamClassStateConflict)
	require.NoError(t, repo.TransitionClassStatus(ctx, 1, "class", "owner", types.ExamClassStatusActive, types.ExamClassStatusArchived, base.Add(4*time.Hour)))
	err = repo.UpdateClassMetadata(ctx, 1, "class", "owner", "归档更新", "", 2, base.Add(5*time.Hour))
	require.ErrorIs(t, err, ErrExamClassStateConflict)
}

func testExamClassRepositorySettingsTransition(t *testing.T) {
	db := newExamClassSettingsTestDB(t)
	repo := NewExamClassRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	seedExamClassSettingsClass(t, db, "class", 1, "owner", types.ExamClassStatusActive, 50, base)

	err := repo.TransitionClassStatus(ctx, 1, "class", "owner", types.ExamClassStatusActive, types.ExamClassStatusArchived, base.Add(time.Hour))
	require.NoError(t, err)
	err = repo.TransitionClassStatus(ctx, 1, "class", "owner", types.ExamClassStatusActive, types.ExamClassStatusArchived, base.Add(2*time.Hour))
	require.ErrorIs(t, err, ErrExamClassStateConflict)
	err = repo.TransitionClassStatus(ctx, 1, "class", "other", types.ExamClassStatusArchived, types.ExamClassStatusActive, base.Add(3*time.Hour))
	require.ErrorIs(t, err, ErrExamClassStateConflict)
	err = repo.TransitionClassStatus(ctx, 2, "class", "owner", types.ExamClassStatusArchived, types.ExamClassStatusActive, base.Add(4*time.Hour))
	require.ErrorIs(t, err, ErrExamClassNotFound)
	require.NoError(t, repo.TransitionClassStatus(ctx, 1, "class", "owner", types.ExamClassStatusArchived, types.ExamClassStatusActive, base.Add(5*time.Hour)))
}

func testExamClassRepositorySettingsApproval(t *testing.T) {
	db := newExamClassSettingsTestDB(t)
	repo := NewExamClassRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	seedExamClassSettingsClass(t, db, "limited", 1, "owner", types.ExamClassStatusActive, 2, base)
	seedExamClassSettingsMember(t, db, "limited", 1, "owner", types.ExamClassMemberStatusActive, base)
	seedExamClassSettingsMember(t, db, "limited", 1, "student-1", types.ExamClassMemberStatusPending, base)
	seedExamClassSettingsMember(t, db, "limited", 1, "student-2", types.ExamClassMemberStatusPending, base)

	member, err := repo.ApproveMemberWithinLimit(ctx, 1, "limited", "student-1", base.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, types.ExamClassMemberStatusActive, member.Status)
	errMember, err := repo.ApproveMemberWithinLimit(ctx, 1, "limited", "student-2", base.Add(2*time.Hour))
	require.Nil(t, errMember)
	require.ErrorIs(t, err, ErrExamClassMemberLimitConflict)
	pending, err := repo.GetAnyMember(ctx, "limited", 1, "student-2")
	require.NoError(t, err)
	require.Equal(t, types.ExamClassMemberStatusPending, pending.Status)

	seedExamClassSettingsClass(t, db, "unlimited", 1, "owner", types.ExamClassStatusActive, 0, base)
	seedExamClassSettingsMember(t, db, "unlimited", 1, "student", types.ExamClassMemberStatusPending, base)
	_, err = repo.ApproveMemberWithinLimit(ctx, 1, "unlimited", "student", base.Add(time.Hour))
	require.NoError(t, err)
}

func testExamClassRepositorySettingsConcurrentApproval(t *testing.T) {
	db := newExamClassSettingsTestDB(t)
	repo := NewExamClassRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	seedExamClassSettingsClass(t, db, "class", 1, "owner", types.ExamClassStatusActive, 2, base)
	seedExamClassSettingsMember(t, db, "class", 1, "owner", types.ExamClassMemberStatusActive, base)
	seedExamClassSettingsMember(t, db, "class", 1, "student-1", types.ExamClassMemberStatusPending, base)
	seedExamClassSettingsMember(t, db, "class", 1, "student-2", types.ExamClassMemberStatusPending, base)

	errorsByUser := make(chan error, 2)
	var wg sync.WaitGroup
	for _, userID := range []string{"student-1", "student-2"} {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			_, err := repo.ApproveMemberWithinLimit(ctx, 1, "class", userID, base.Add(time.Hour))
			errorsByUser <- err
		}(userID)
	}
	wg.Wait()
	close(errorsByUser)

	successes := 0
	limitConflicts := 0
	for err := range errorsByUser {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrExamClassMemberLimitConflict):
			limitConflicts++
		default:
			t.Fatalf("unexpected approval error: %v", err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, limitConflicts)
}

func newExamClassSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(&types.ExamClass{}, &types.ExamClassMember{}))
	return db
}

func seedExamClassSettingsClass(t *testing.T, db *gorm.DB, id string, tenantID uint64, ownerID string, status types.ExamClassStatus, memberLimit int, createdAt time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&types.ExamClass{
		ID: id, TenantID: tenantID, OwnerUserID: ownerID, SpaceID: "space-" + id,
		Name: id, MemberLimit: memberLimit, Status: status, CreatedAt: createdAt, UpdatedAt: createdAt,
	}).Error)
}

func seedExamClassSettingsMember(t *testing.T, db *gorm.DB, classID string, tenantID uint64, userID string, status types.ExamClassMemberStatus, createdAt time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&types.ExamClassMember{
		ID: classID + "-" + userID, ClassID: classID, UserID: userID, TenantID: tenantID,
		Role: types.ExamClassRoleStudent, Status: status, JoinedAt: createdAt, CreatedAt: createdAt, UpdatedAt: createdAt,
	}).Error)
}

func examClassSettingsIDs(classes []*types.ExamClass) []string {
	ids := make([]string, 0, len(classes))
	for _, class := range classes {
		ids = append(ids, class.ID)
	}
	return ids
}
