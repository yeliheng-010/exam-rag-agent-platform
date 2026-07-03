package service

import (
	"context"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type fakeRAGKBService struct {
	interfaces.KnowledgeBaseService
	kbs map[string]*types.KnowledgeBase
}

func (f *fakeRAGKBService) GetKnowledgeBasesByIDsOnly(_ context.Context, ids []string) ([]*types.KnowledgeBase, error) {
	out := make([]*types.KnowledgeBase, 0, len(ids))
	for _, id := range ids {
		if kb := f.kbs[id]; kb != nil {
			out = append(out, kb)
		}
	}
	return out, nil
}

type fakeRAGExamResourceService struct {
	interfaces.ExamResourceService
	allowed map[string]bool
}

func (f *fakeRAGExamResourceService) CanReadKnowledgeBase(_ context.Context, _ uint64, _ string, kbID string) (bool, error) {
	return f.allowed[kbID], nil
}

type fakeRAGKnowledgeService struct {
	interfaces.KnowledgeService
	items map[string]*types.Knowledge
}

func (f *fakeRAGKnowledgeService) GetKnowledgeBatchWithSharedAccess(_ context.Context, _ uint64, ids []string) ([]*types.Knowledge, error) {
	out := make([]*types.Knowledge, 0, len(ids))
	for _, id := range ids {
		if item := f.items[id]; item != nil {
			out = append(out, item)
		}
	}
	return out, nil
}

func ragCtx(role types.TenantRole, userID string) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(1))
	ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
	return ctx
}

func TestBuildSearchTargets_ViewerCannotUseSameTenantUnboundKB(t *testing.T) {
	svc := &sessionService{
		knowledgeBaseService: &fakeRAGKBService{
			kbs: map[string]*types.KnowledgeBase{
				"kb-teacher": {ID: "kb-teacher", TenantID: 1, CreatorID: "teacher-1"},
			},
		},
		examResourceService: &fakeRAGExamResourceService{},
	}

	targets, err := svc.buildSearchTargets(
		ragCtx(types.TenantRoleViewer, "student-1"),
		1,
		[]string{"kb-teacher"},
		nil,
		nil,
	)

	require.Error(t, err)
	require.Empty(t, targets)
	appErr, ok := apperrors.IsAppError(err)
	require.True(t, ok)
	require.Equal(t, apperrors.ErrForbidden, appErr.Code)
}

func TestBuildSearchTargets_ViewerCanUseClassBoundKB(t *testing.T) {
	svc := &sessionService{
		knowledgeBaseService: &fakeRAGKBService{
			kbs: map[string]*types.KnowledgeBase{
				"kb-class": {ID: "kb-class", TenantID: 1, CreatorID: "teacher-1"},
			},
		},
		examResourceService: &fakeRAGExamResourceService{
			allowed: map[string]bool{"kb-class": true},
		},
	}

	targets, err := svc.buildSearchTargets(
		ragCtx(types.TenantRoleViewer, "student-1"),
		1,
		[]string{"kb-class"},
		nil,
		nil,
	)

	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "kb-class", targets[0].KnowledgeBaseID)
	require.Equal(t, uint64(1), targets[0].TenantID)
}

func TestBuildSearchTargets_ViewerCannotUseFileFromUnboundKB(t *testing.T) {
	svc := &sessionService{
		knowledgeBaseService: &fakeRAGKBService{
			kbs: map[string]*types.KnowledgeBase{
				"kb-teacher": {ID: "kb-teacher", TenantID: 1, CreatorID: "teacher-1"},
			},
		},
		knowledgeService: &fakeRAGKnowledgeService{
			items: map[string]*types.Knowledge{
				"doc-1": {ID: "doc-1", TenantID: 1, KnowledgeBaseID: "kb-teacher"},
			},
		},
		examResourceService: &fakeRAGExamResourceService{},
	}

	targets, err := svc.buildSearchTargets(
		ragCtx(types.TenantRoleViewer, "student-1"),
		1,
		nil,
		[]string{"doc-1"},
		nil,
	)

	require.Error(t, err)
	require.Empty(t, targets)
	appErr, ok := apperrors.IsAppError(err)
	require.True(t, ok)
	require.Equal(t, apperrors.ErrForbidden, appErr.Code)
}

func TestBuildSearchTargets_ContributorCanUseSameTenantKB(t *testing.T) {
	svc := &sessionService{
		knowledgeBaseService: &fakeRAGKBService{
			kbs: map[string]*types.KnowledgeBase{
				"kb-tenant": {ID: "kb-tenant", TenantID: 1, CreatorID: "teacher-1"},
			},
		},
		examResourceService: &fakeRAGExamResourceService{},
	}

	targets, err := svc.buildSearchTargets(
		ragCtx(types.TenantRoleContributor, "teacher-2"),
		1,
		[]string{"kb-tenant"},
		nil,
		nil,
	)

	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "kb-tenant", targets[0].KnowledgeBaseID)
}
