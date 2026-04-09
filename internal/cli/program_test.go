package cli

import (
	"encoding/json"
	"testing"

	"hawkings-cli/internal/api"
)

func TestSortProgramSummariesByStatusPriorityThenName(t *testing.T) {
	programs := []api.ProgramSummary{
		{ID: 1, Name: "Zeta", Status: ptr("processed")},
		{ID: 2, Name: "Beta", Status: ptr("completed")},
		{ID: 3, Name: "Alpha", Status: ptr("courses_created")},
		{ID: 4, Name: "Alpha", Status: ptr("completed")},
	}

	sorted := sortProgramSummaries(programs, "status;name", "completed,processed,courses-created;ASC")

	got := []int{sorted[0].ID, sorted[1].ID, sorted[2].ID, sorted[3].ID}
	want := []int{4, 2, 1, 3}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected sort order: got %v want %v", got, want)
		}
	}
}

func TestNormalizeProgramDetailCanonicalizesLegacyStatus(t *testing.T) {
	program := normalizeProgramDetail(api.ProgramDetail{
		ID:     42,
		Status: ptr("courses_created"),
	})

	if got := stringPtrValue(program.Status); got != "courses-created" {
		t.Fatalf("unexpected canonical status: got %q", got)
	}
}

func TestParseProgramCourseReorderPayload(t *testing.T) {
	payload := map[string]any{
		"selected": []any{json.Number("33967"), json.Number("33968"), json.Number("33971")},
		"order": map[string]any{
			"33967": json.Number("1"),
			"33968": json.Number("2"),
			"33971": json.Number("3"),
		},
	}

	reorder, err := parseProgramCourseReorderPayload(payload)
	if err != nil {
		t.Fatalf("parseProgramCourseReorderPayload returned error: %v", err)
	}

	wantSelected := []int{33967, 33968, 33971}
	for i := range wantSelected {
		if reorder.Selected[i] != wantSelected[i] {
			t.Fatalf("unexpected selected order: got %v want %v", reorder.Selected, wantSelected)
		}
	}

	if got := reorder.Order[33971]; got != 3 {
		t.Fatalf("unexpected order for 33971: got %d want 3", got)
	}
}

func TestParseProgramCourseReorderPayloadRejectsInconsistentSelectedOrder(t *testing.T) {
	payload := map[string]any{
		"selected": []any{json.Number("33968"), json.Number("33967")},
		"order": map[string]any{
			"33967": json.Number("1"),
			"33968": json.Number("2"),
		},
	}

	if _, err := parseProgramCourseReorderPayload(payload); err == nil {
		t.Fatal("expected parseProgramCourseReorderPayload to fail")
	}
}

func TestValidateProgramCourseSetMatchesCurrent(t *testing.T) {
	if err := validateProgramCourseSetMatchesCurrent([]int{1, 2, 3}, []int{3, 2, 1}); err != nil {
		t.Fatalf("expected matching sets to pass, got %v", err)
	}

	if err := validateProgramCourseSetMatchesCurrent([]int{1, 2, 3}, []int{1, 2, 4}); err == nil {
		t.Fatal("expected mismatched sets to fail")
	}
}
