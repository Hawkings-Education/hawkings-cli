package cli

import (
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
