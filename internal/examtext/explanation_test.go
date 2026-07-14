package examtext

import (
	"strings"
	"testing"
)

func TestBuildBaselineExplanationUsesFoldedEvidence(t *testing.T) {
	explanation := BuildBaselineExplanation(
		"A",
		[]ExplanationOption{{Key: "A", Content: "Sonder Lüm Hotel"}},
		"Right across the street, Sonder Lum Hotel offers spacious rooms.",
	)

	if !strings.Contains(explanation, "Sonder Lüm Hotel") {
		t.Fatalf("missing answer option: %q", explanation)
	}
	if !strings.Contains(explanation, "Sonder Lum Hotel offers spacious rooms") {
		t.Fatalf("missing folded evidence: %q", explanation)
	}
}

func TestBuildBaselineExplanationMatchesPartialEvidenceTokens(t *testing.T) {
	explanation := BuildBaselineExplanation(
		"C",
		[]ExplanationOption{{Key: "C", Content: "Obtain a parking pass"}},
		"SoFi Stadium requires guests to enter through the gate indicated on their digital parking pass.",
	)

	if !strings.Contains(explanation, "Correct answer: C") {
		t.Fatalf("missing answer: %q", explanation)
	}
	if !strings.Contains(explanation, "digital parking pass") {
		t.Fatalf("missing partial-token evidence: %q", explanation)
	}
}
