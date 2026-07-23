package service

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestRecordKBCloneTargetUsesCreatedKnowledgeBaseID(t *testing.T) {
	progress := &types.KBCloneProgress{}
	target := &types.KnowledgeBase{ID: "kb-created"}

	recordKBCloneTarget(progress, target)

	require.Equal(t, "kb-created", progress.TargetID)
}
