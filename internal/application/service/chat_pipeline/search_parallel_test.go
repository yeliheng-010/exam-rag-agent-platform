package chatpipeline

import "testing"

func TestNewPluginSearchParallelPassesQuestionRepoToChunkSearchPlugin(t *testing.T) {
	repo := &stubChatExamQuestionRepo{}

	plugin := NewPluginSearchParallel(
		NewEventManager(),
		nil,
		nil,
		nil,
		repo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	if plugin.searchPlugin == nil {
		t.Fatal("expected internal chunk search plugin")
	}
	if plugin.searchPlugin.questionRepo != repo {
		t.Fatal("expected question repo to be passed into internal chunk search plugin")
	}
}
