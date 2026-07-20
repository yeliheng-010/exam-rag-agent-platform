package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestExamClassSettingsUpdateOwnerOnly(t *testing.T) {
	repo := newFakeExamClassRepo()
	seedExamClass(repo, "class", 1, "owner", "CODE")
	seedExamClassMember(repo, "class", 1, "owner", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(repo, "class", 1, "assistant", types.ExamClassRoleAssistant, types.ExamClassMemberStatusActive)
	seedExamClassMember(repo, "class", 1, "student", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	svc := newExamClassServiceForTest(repo, newFakeExamTeacherAccess())

	updated, err := svc.UpdateClass(context.Background(), 1, "owner", "class", &types.UpdateExamClassRequest{
		Name: "  高三一班  ", Description: "  冲刺班  ", MemberLimit: 60,
	})
	require.NoError(t, err)
	require.Equal(t, "高三一班", updated.Name)
	require.Equal(t, "冲刺班", updated.Description)
	require.Equal(t, 60, updated.MemberLimit)

	for _, userID := range []string{"assistant", "student"} {
		_, err = svc.UpdateClass(context.Background(), 1, userID, "class", &types.UpdateExamClassRequest{
			Name: "无权更新", MemberLimit: 60,
		})
		require.ErrorIs(t, err, ErrExamPermissionDenied)
	}
}

func TestExamClassSettingsUpdateValidation(t *testing.T) {
	repo := newFakeExamClassRepo()
	seedExamClass(repo, "class", 1, "owner", "CODE")
	seedExamClassMember(repo, "class", 1, "owner", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	svc := newExamClassServiceForTest(repo, newFakeExamTeacherAccess())

	tests := []struct {
		name    string
		classID string
		req     *types.UpdateExamClassRequest
	}{
		{name: "nil request", classID: "class"},
		{name: "blank class id", classID: " ", req: &types.UpdateExamClassRequest{Name: "班级", MemberLimit: 50}},
		{name: "blank name", classID: "class", req: &types.UpdateExamClassRequest{Name: " ", MemberLimit: 50}},
		{name: "name over 255 characters", classID: "class", req: &types.UpdateExamClassRequest{Name: strings.Repeat("班", 256), MemberLimit: 50}},
		{name: "description over 2000 characters", classID: "class", req: &types.UpdateExamClassRequest{Name: "班级", Description: strings.Repeat("a", 2001), MemberLimit: 50}},
		{name: "negative member limit", classID: "class", req: &types.UpdateExamClassRequest{Name: "班级", MemberLimit: -1}},
		{name: "member limit over 1000", classID: "class", req: &types.UpdateExamClassRequest{Name: "班级", MemberLimit: 1001}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateClass(context.Background(), 1, "owner", tt.classID, tt.req)
			require.ErrorIs(t, err, ErrExamInvalidRequest)
		})
	}
}

func TestExamClassSettingsListAndArchivedDetail(t *testing.T) {
	repo := newFakeExamClassRepo()
	seedExamClass(repo, "active", 1, "owner", "ACTIVE")
	archived := seedExamClass(repo, "archived", 1, "owner", "ARCHIVED")
	archived.Status = types.ExamClassStatusArchived
	seedExamClassMember(repo, "active", 1, "student", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(repo, "archived", 1, "student", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	svc := newExamClassServiceForTest(repo, newFakeExamTeacherAccess())

	classes, err := svc.ListClasses(context.Background(), 1, "student", types.ListExamClassesFilter{})
	require.NoError(t, err)
	require.Equal(t, []string{"active"}, examClassServiceIDs(classes))
	classes, err = svc.ListClasses(context.Background(), 1, "student", types.ListExamClassesFilter{IncludeArchived: true})
	require.NoError(t, err)
	require.Equal(t, []string{"active", "archived"}, examClassServiceIDs(classes))

	detail, err := svc.GetClass(context.Background(), 1, "student", "archived")
	require.NoError(t, err)
	require.Equal(t, types.ExamClassStatusArchived, detail.Status)
	_, err = svc.GetClass(context.Background(), 1, "outsider", "archived")
	require.ErrorIs(t, err, ErrExamPermissionDenied)
	_, err = svc.GetClass(context.Background(), 2, "student", "archived")
	require.ErrorIs(t, err, ErrExamNotFound)
}

func TestExamClassSettingsArchiveRestoreAndConflict(t *testing.T) {
	repo := newFakeExamClassRepo()
	seedExamClass(repo, "class", 1, "owner", "CODE")
	seedExamClassMember(repo, "class", 1, "owner", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(repo, "class", 1, "assistant", types.ExamClassRoleAssistant, types.ExamClassMemberStatusActive)
	svc := newExamClassServiceForTest(repo, newFakeExamTeacherAccess())

	archived, err := svc.ArchiveClass(context.Background(), 1, "owner", "class")
	require.NoError(t, err)
	require.Equal(t, types.ExamClassStatusArchived, archived.Status)
	_, err = svc.ArchiveClass(context.Background(), 1, "owner", "class")
	require.ErrorIs(t, err, ErrExamStateConflict)
	_, err = svc.RestoreClass(context.Background(), 1, "assistant", "class")
	require.ErrorIs(t, err, ErrExamPermissionDenied)
	restored, err := svc.RestoreClass(context.Background(), 1, "owner", "class")
	require.NoError(t, err)
	require.Equal(t, types.ExamClassStatusActive, restored.Status)

	repo.transitionClassErr = repository.ErrExamClassStateConflict
	_, err = svc.ArchiveClass(context.Background(), 1, "owner", "class")
	require.ErrorIs(t, err, ErrExamStateConflict)
	repo.transitionClassErr = nil
	repo.updateClassMetadataErr = repository.ErrExamClassMemberLimitConflict
	_, err = svc.UpdateClass(context.Background(), 1, "owner", "class", &types.UpdateExamClassRequest{Name: "班级", MemberLimit: 1})
	require.ErrorIs(t, err, ErrExamStateConflict)
}

func TestExamClassSettingsApprovalMapsLimitConflict(t *testing.T) {
	repo := newFakeExamClassRepo()
	class := seedExamClass(repo, "class", 1, "owner", "CODE")
	class.MemberLimit = 1
	seedExamClassMember(repo, "class", 1, "owner", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(repo, "class", 1, "student", types.ExamClassRoleStudent, types.ExamClassMemberStatusPending)
	svc := newExamClassServiceForTest(repo, newFakeExamTeacherAccess())

	_, err := svc.ApproveClassMember(context.Background(), 1, "owner", "class", "student")
	require.True(t, errors.Is(err, ErrExamStateConflict))
	member, memberErr := repo.GetAnyMember(context.Background(), "class", 1, "student")
	require.NoError(t, memberErr)
	require.Equal(t, types.ExamClassMemberStatusPending, member.Status)
}

func examClassServiceIDs(classes []*types.ExamClass) []string {
	ids := make([]string, 0, len(classes))
	for _, class := range classes {
		ids = append(ids, class.ID)
	}
	return ids
}
