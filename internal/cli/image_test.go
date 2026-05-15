package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProgramImageGeneratePostsForceAndAsync(t *testing.T) {
	var posted map[string]any
	var handlerErr string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost || r.URL.Path != "/v1/course-program/410/image/generate" {
			handlerErr = "unexpected request: " + r.Method + " " + r.URL.String()
			http.Error(w, handlerErr, http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			handlerErr = "decode posted payload: " + err.Error()
			http.Error(w, handlerErr, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{
			"id": 410,
			"name": "Programa",
			"image": "https://cdn.example/program.png",
			"status": "processed"
		}`))
	}))
	defer server.Close()

	configPath := testConfig(t, server.URL)
	cmd := NewRootCommand()
	cmd.SetArgs([]string{
		"--config", configPath,
		"--profile", "test",
		"--output", "json",
		"program", "image", "generate", "410",
		"--force",
		"--async",
		"--queue", "high",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute program image generate: %v", err)
	}
	if handlerErr != "" {
		t.Fatal(handlerErr)
	}
	if got := posted["force"]; got != true {
		t.Fatalf("unexpected force: got %#v", got)
	}
	if got := posted["async"]; got != true {
		t.Fatalf("unexpected async: got %#v", got)
	}
	if got := posted["queue"]; got != "high" {
		t.Fatalf("unexpected queue: got %#v", got)
	}
}

func TestCourseImageGenerateDefaultsToSyncPayload(t *testing.T) {
	var posted map[string]any
	var handlerErr string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost || r.URL.Path != "/v1/course/35572/image/generate" {
			handlerErr = "unexpected request: " + r.Method + " " + r.URL.String()
			http.Error(w, handlerErr, http.StatusNotFound)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			handlerErr = "decode posted payload: " + err.Error()
			http.Error(w, handlerErr, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{
			"id": 35572,
			"name": "Curso",
			"image": "https://cdn.example/course.png",
			"status": "processed"
		}`))
	}))
	defer server.Close()

	configPath := testConfig(t, server.URL)
	cmd := NewRootCommand()
	cmd.SetArgs([]string{
		"--config", configPath,
		"--profile", "test",
		"--output", "json",
		"course", "image", "generate", "35572",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute course image generate: %v", err)
	}
	if handlerErr != "" {
		t.Fatal(handlerErr)
	}
	if got := posted["force"]; got != false {
		t.Fatalf("unexpected force: got %#v", got)
	}
	if got := posted["async"]; got != false {
		t.Fatalf("unexpected async: got %#v", got)
	}
	if _, ok := posted["queue"]; ok {
		t.Fatalf("queue should be omitted by default, got %#v", posted["queue"])
	}
}

func TestImageGenerateRejectsInvalidQueue(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{
		"--config", testConfig(t, "http://127.0.0.1:1"),
		"--profile", "test",
		"program", "image", "generate", "410",
		"--queue", "medium",
		"--dry-run",
	})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected invalid queue to fail")
	}
}
