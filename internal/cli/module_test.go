package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"hawkings-cli/internal/api"
)

func TestActiveLearningPlatformIDUsesProfilePlatform(t *testing.T) {
	id, ok := activeLearningPlatformID(api.Profile{
		LearningPlatform: &api.LearningPlatform{
			ID:   21,
			UUID: "profile-uuid",
		},
	}, []api.LearningPlatform{
		{ID: 99, UUID: "profile-uuid"},
	}, "profile-uuid")

	if !ok {
		t.Fatal("expected learning platform ID to resolve")
	}
	if id != 21 {
		t.Fatalf("unexpected learning platform ID: got %d want 21", id)
	}
}

func TestActiveLearningPlatformIDFallsBackToConfiguredUUID(t *testing.T) {
	id, ok := activeLearningPlatformID(api.Profile{}, []api.LearningPlatform{
		{ID: 20, UUID: "other-uuid"},
		{ID: 21, UUID: "active-uuid"},
	}, "active-uuid")

	if !ok {
		t.Fatal("expected learning platform ID to resolve")
	}
	if id != 21 {
		t.Fatalf("unexpected learning platform ID: got %d want 21", id)
	}
}

func TestActiveLearningPlatformIDRequiresKnownPlatform(t *testing.T) {
	if id, ok := activeLearningPlatformID(api.Profile{}, []api.LearningPlatform{
		{ID: 20, UUID: "other-uuid"},
	}, "active-uuid"); ok {
		t.Fatalf("expected learning platform ID not to resolve, got %d", id)
	}
}

func TestModuleCreateIncludesLearningPlatformID(t *testing.T) {
	var posted map[string]any
	var handlerErr string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/course-section/336710":
			_, _ = w.Write([]byte(`{
				"id": 336710,
				"course": {"id": 36598},
				"course_modules": [{"id": 1, "order": 2}]
			}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/profile":
			_, _ = w.Write([]byte(`{
				"id": 18979,
				"name": "Hawkings CLI",
				"learning_platform": {"id": 21, "uuid": "active-uuid", "name": "Learning House"}
			}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/course-module":
			if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
				handlerErr = "decode posted payload: " + err.Error()
				http.Error(w, handlerErr, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"id": 123,
				"name": "Intro",
				"type": "markdown",
				"order": 3,
				"status": "empty",
				"enabled": true
			}`))
		default:
			handlerErr = "unexpected request: " + r.Method + " " + r.URL.String()
			http.Error(w, handlerErr, http.StatusNotFound)
		}
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "hawkings.toml")
	configBody := `version = 1
profile = "test"

[profiles.test]
base_url = "` + server.URL + `"
x_api_key = "test-token"
platform_uuid = "active-uuid"
timeout = "5s"
`
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := NewRootCommand()
	cmd.SetArgs([]string{
		"--config", configPath,
		"--profile", "test",
		"--output", "json",
		"module", "create",
		"--section-id", "336710",
		"--name", "Intro",
		"--type", "markdown",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute module create: %v", err)
	}
	if handlerErr != "" {
		t.Fatal(handlerErr)
	}
	if posted == nil {
		t.Fatal("expected module create payload to be posted")
	}

	if got := int(posted["learning_platform_id"].(float64)); got != 21 {
		t.Fatalf("unexpected learning_platform_id: got %d want 21", got)
	}
	if got := int(posted["course_id"].(float64)); got != 36598 {
		t.Fatalf("unexpected course_id: got %d want 36598", got)
	}
	if got := int(posted["order"].(float64)); got != 3 {
		t.Fatalf("unexpected order: got %d want 3", got)
	}
}

func TestModuleActivityReadsActivityDetail(t *testing.T) {
	var sawModule bool
	var sawActivity bool
	var handlerErr string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/course-module/77":
			sawModule = true
			if got := r.URL.Query()["with[]"]; len(got) != 1 || got[0] != "activity" {
				handlerErr = "expected module request with[]=activity"
				http.Error(w, handlerErr, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"id": 77,
				"name": "Practice",
				"type": "activity",
				"status": "processed",
				"activity": {
					"id": 88,
					"uuid": "act-uuid",
					"type": "quiz",
					"title": "Quiz",
					"status": "processed",
					"description": "Check understanding"
				}
			}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/activity/act-uuid":
			sawActivity = true
			if got := r.URL.Query()["with[]"]; len(got) != 1 || got[0] != "activityQuestions" {
				handlerErr = "expected activity request with[]=activityQuestions"
				http.Error(w, handlerErr, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"id": 88,
				"uuid": "act-uuid",
				"type": "quiz",
				"title": "Quiz",
				"status": "processed",
				"description": "Check understanding",
				"content": {"questions": [{"question": "One?", "options": ["A"], "correct_answer": 0}]},
				"questions": [{"id": 1, "uuid": "question-uuid"}]
			}`))
		default:
			handlerErr = "unexpected request: " + r.Method + " " + r.URL.String()
			http.Error(w, handlerErr, http.StatusNotFound)
		}
	}))
	defer server.Close()

	configPath := testConfig(t, server.URL)
	cmd := NewRootCommand()
	cmd.SetArgs([]string{
		"--config", configPath,
		"--profile", "test",
		"--output", "json",
		"module", "activity", "77",
		"--questions",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute module activity: %v", err)
	}
	if handlerErr != "" {
		t.Fatal(handlerErr)
	}
	if !sawModule || !sawActivity {
		t.Fatalf("expected module and activity requests, got module=%v activity=%v", sawModule, sawActivity)
	}
}

func TestModuleSetActivityMergesCurrentActivityBeforePatch(t *testing.T) {
	var patched map[string]any
	var handlerErr string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/course-module/77":
			_, _ = w.Write([]byte(`{
				"id": 77,
				"name": "Practice",
				"type": "activity",
				"status": "processed",
				"activity": {"id": 88, "uuid": "act-uuid", "type": "quiz", "title": "Old title"}
			}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/activity/act-uuid":
			_, _ = w.Write([]byte(`{
				"id": 88,
				"uuid": "act-uuid",
				"type": "quiz",
				"title": "Old title",
				"status": "processed",
				"description": "Old description",
				"content": {"questions": [{"question": "Old?", "options": ["A"], "correct_answer": 0}]}
			}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/activity/act-uuid":
			if err := json.NewDecoder(r.Body).Decode(&patched); err != nil {
				handlerErr = "decode patched payload: " + err.Error()
				http.Error(w, handlerErr, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"id": 88,
				"uuid": "act-uuid",
				"type": "quiz",
				"title": "New title",
				"status": "processed",
				"description": "Old description",
				"content": {"questions": [{"question": "New?", "options": ["A"], "correct_answer": 0}]}
			}`))
		default:
			handlerErr = "unexpected request: " + r.Method + " " + r.URL.String()
			http.Error(w, handlerErr, http.StatusNotFound)
		}
	}))
	defer server.Close()

	configPath := testConfig(t, server.URL)
	cmd := NewRootCommand()
	cmd.SetArgs([]string{
		"--config", configPath,
		"--profile", "test",
		"--output", "json",
		"module", "set-activity", "77",
		"--title", "New title",
		"--json", `{"content":{"questions":[{"question":"New?","options":["A"],"correct_answer":0}]}}`,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute module set-activity: %v", err)
	}
	if handlerErr != "" {
		t.Fatal(handlerErr)
	}
	if patched == nil {
		t.Fatal("expected activity patch payload")
	}
	if got := patched["title"]; got != "New title" {
		t.Fatalf("unexpected title: got %#v", got)
	}
	if got := patched["description"]; got != "Old description" {
		t.Fatalf("unexpected description: got %#v", got)
	}
	content, ok := patched["content"].(map[string]any)
	if !ok {
		t.Fatalf("expected content object, got %#v", patched["content"])
	}
	questions, ok := content["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("expected one question in content, got %#v", content["questions"])
	}
}

func TestModuleGenerateActivityPostsToModuleActivityGenerate(t *testing.T) {
	var posted map[string]any
	var handlerErr string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/course-module/77/activity/generate":
			if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
				handlerErr = "decode posted payload: " + err.Error()
				http.Error(w, handlerErr, http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"id": 77,
				"name": "Practice",
				"type": "activity",
				"status": "queued",
				"activity": null
			}`))
		default:
			handlerErr = "unexpected request: " + r.Method + " " + r.URL.String()
			http.Error(w, handlerErr, http.StatusNotFound)
		}
	}))
	defer server.Close()

	configPath := testConfig(t, server.URL)
	cmd := NewRootCommand()
	cmd.SetArgs([]string{
		"--config", configPath,
		"--profile", "test",
		"--output", "json",
		"module", "generate-activity", "77",
		"--force",
		"--priority", "low",
		"--cache=true",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute module generate-activity: %v", err)
	}
	if handlerErr != "" {
		t.Fatal(handlerErr)
	}
	if posted == nil {
		t.Fatal("expected activity generate payload")
	}
	if got := posted["async"]; got != true {
		t.Fatalf("unexpected async: got %#v", got)
	}
	if got := posted["force"]; got != true {
		t.Fatalf("unexpected force: got %#v", got)
	}
	if got := posted["priority"]; got != "low" {
		t.Fatalf("unexpected priority: got %#v", got)
	}
	if got := posted["cache"]; got != true {
		t.Fatalf("unexpected cache: got %#v", got)
	}
}

func testConfig(t *testing.T, serverURL string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "hawkings.toml")
	configBody := `version = 1
profile = "test"

[profiles.test]
base_url = "` + serverURL + `"
x_api_key = "test-token"
platform_uuid = "active-uuid"
timeout = "5s"
`
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}
