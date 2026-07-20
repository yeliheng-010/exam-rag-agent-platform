package types_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

func TestExamClassSettingsContract(t *testing.T) {
	req := types.UpdateExamClassRequest{
		Name:        "高三一班",
		Description: "冲刺班",
		MemberLimit: 60,
	}
	require.Equal(t, "高三一班", req.Name)
	require.Equal(t, "冲刺班", req.Description)
	require.Equal(t, 60, req.MemberLimit)
	require.True(t, types.ListExamClassesFilter{IncludeArchived: true}.IncludeArchived)

	repositoryType := reflect.TypeOf((*interfaces.ExamClassRepository)(nil)).Elem()
	serviceType := reflect.TypeOf((*interfaces.ExamClassService)(nil)).Elem()

	requireInterfaceMethod(t, repositoryType, "ListByUserWithStatuses", (func(context.Context, uint64, string, []types.ExamClassStatus) ([]*types.ExamClass, error))(nil))
	requireInterfaceMethod(t, repositoryType, "GetByIDAndTenantIncludingArchived", (func(context.Context, string, uint64) (*types.ExamClass, error))(nil))
	requireInterfaceMethod(t, repositoryType, "ListByIDsAndTenantIncludingArchived", (func(context.Context, uint64, []string) ([]*types.ExamClass, error))(nil))
	requireInterfaceMethod(t, repositoryType, "UpdateClassMetadata", (func(context.Context, uint64, string, string, string, string, int, time.Time) error)(nil))
	requireInterfaceMethod(t, repositoryType, "TransitionClassStatus", (func(context.Context, uint64, string, string, types.ExamClassStatus, types.ExamClassStatus, time.Time) error)(nil))
	requireInterfaceMethod(t, repositoryType, "ApproveMemberWithinLimit", (func(context.Context, uint64, string, string, time.Time) (*types.ExamClassMember, error))(nil))

	requireInterfaceMethod(t, serviceType, "ListClasses", (func(context.Context, uint64, string, types.ListExamClassesFilter) ([]*types.ExamClass, error))(nil))
	requireInterfaceMethod(t, serviceType, "UpdateClass", (func(context.Context, uint64, string, string, *types.UpdateExamClassRequest) (*types.ExamClass, error))(nil))
	requireInterfaceMethod(t, serviceType, "ArchiveClass", (func(context.Context, uint64, string, string) (*types.ExamClass, error))(nil))
	requireInterfaceMethod(t, serviceType, "RestoreClass", (func(context.Context, uint64, string, string) (*types.ExamClass, error))(nil))
}

func requireInterfaceMethod(t *testing.T, interfaceType reflect.Type, name string, signature any) {
	t.Helper()
	method, ok := interfaceType.MethodByName(name)
	require.Truef(t, ok, "%s must declare %s", interfaceType.Name(), name)
	require.Equal(t, reflect.TypeOf(signature), method.Type)
}
