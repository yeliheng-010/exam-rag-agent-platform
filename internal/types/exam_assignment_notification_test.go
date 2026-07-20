package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExamAssignmentNotificationContract(t *testing.T) {
	require.Equal(t, "exam_assignment_notifications", (ExamAssignmentNotification{}).TableName())
	require.Equal(t, ExamAssignmentNotificationKind("reminder"), ExamAssignmentNotificationKindReminder)
	result := &SendExamAssignmentReminderResult{
		SentCount:   3,
		SentUserIDs: []string{"student-1", "student-2", "student-3"},
	}
	require.Equal(t, result.SentCount, len(result.SentUserIDs))
}
