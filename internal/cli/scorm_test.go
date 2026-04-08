package cli

import "testing"

func TestSanitizeScormPayloadRemovesLegacyFields(t *testing.T) {
	payload := map[string]any{
		"name":        "Tema 1",
		"user_id":     12,
		"language_id": 3,
	}

	sanitized, ignored := sanitizeScormPayload(payload)

	if _, ok := sanitized["user_id"]; ok {
		t.Fatalf("user_id should have been removed")
	}
	if _, ok := sanitized["language_id"]; ok {
		t.Fatalf("language_id should have been removed")
	}
	if sanitized["name"] != "Tema 1" {
		t.Fatalf("expected name to be preserved")
	}
	if len(ignored) != 2 || ignored[0] != "user_id" || ignored[1] != "language_id" {
		t.Fatalf("unexpected ignored fields: %#v", ignored)
	}
	if _, ok := payload["user_id"]; !ok {
		t.Fatalf("original payload should not be mutated")
	}
}
