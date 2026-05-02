package cli

import "testing"

func TestPromptImagePayloadUsesInstructions(t *testing.T) {
	payload := promptImagePayload(&promptImageOptions{
		instructions: "Haz una imagen",
		format:       "url",
		service:      "openai",
	})

	if got := payload["instructions"]; got != "Haz una imagen" {
		t.Fatalf("unexpected instructions: got %#v", got)
	}
	if _, ok := payload["instructions_final"]; ok {
		t.Fatal("instructions_final should not be present when instructions is used")
	}
}

func TestPromptImagePayloadUsesInstructionsFinal(t *testing.T) {
	payload := promptImagePayload(&promptImageOptions{
		instructionsFinal: "Prompt exacto",
		format:            "url",
		service:           "openai",
	})

	if got := payload["instructions"]; got != "Prompt exacto" {
		t.Fatalf("unexpected instructions: got %#v", got)
	}
	if got := payload["instructions_final"]; got != true {
		t.Fatalf("unexpected instructions_final: got %#v", got)
	}
}

func TestValidatePromptImageInstructionsRequiresExactlyOne(t *testing.T) {
	if err := validatePromptImageInstructions(&promptImageOptions{}); err == nil {
		t.Fatal("expected missing instructions to fail")
	}

	if err := validatePromptImageInstructions(&promptImageOptions{
		instructions:      "base",
		instructionsFinal: "final",
	}); err == nil {
		t.Fatal("expected both instruction flags to fail")
	}

	if err := validatePromptImageInstructions(&promptImageOptions{instructions: "base"}); err != nil {
		t.Fatalf("expected instructions to pass, got %v", err)
	}

	if err := validatePromptImageInstructions(&promptImageOptions{instructionsFinal: "final"}); err != nil {
		t.Fatalf("expected instructions_final to pass, got %v", err)
	}
}
