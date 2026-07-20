package session

import "testing"

func TestAppendInlineQueryImages(t *testing.T) {
	query := `
![first](local://42/exports/formula.png)
![duplicate](local://42/exports/formula.png)
![cross tenant](local://7/exports/other.png)
![not an image](local://42/exports/notes.txt)
plain local://42/exports/not-markdown.png
![second](local://42/exports/figure.webp "figure")`
	existing := []ImageAttachment{{URL: "local://42/exports/existing.jpg"}}

	got := appendInlineQueryImages(query, 42, existing)
	want := []string{
		"local://42/exports/existing.jpg",
		"local://42/exports/formula.png",
		"local://42/exports/figure.webp",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d images, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].URL != want[i] {
			t.Fatalf("image %d URL = %q, want %q", i, got[i].URL, want[i])
		}
	}
}
